package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appConfig "github.com/felix-gnwn/ticketing-engine/internal/config"
	"github.com/felix-gnwn/ticketing-engine/internal/database"
	"github.com/felix-gnwn/ticketing-engine/internal/handler"
	"github.com/felix-gnwn/ticketing-engine/internal/middleware"
	cache "github.com/felix-gnwn/ticketing-engine/internal/repository/cache"
	"github.com/felix-gnwn/ticketing-engine/internal/repository/messaging"
	postgres "github.com/felix-gnwn/ticketing-engine/internal/repository/postgres"
	storage "github.com/felix-gnwn/ticketing-engine/internal/repository/storage"
	"github.com/felix-gnwn/ticketing-engine/internal/usecase"
)

func main() {
	// INIT STRUCTURED LOGGING (slog)
	// JSON logs are mandatory for indexing and searching.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Display DEBUG, INFO, WARN, and ERROR
	}))
	slog.SetDefault(logger)

	// LOAD ENTERPRISE CONFIGURATION (Fail-Fast & Centralized)
	cfg, err := appConfig.Load()
	if err != nil {
		slog.Error("Configuration initialization failed", "error", err)
		os.Exit(1)
	}

	// INIT DATABASE CONNECTION POOL
	dbPool, err := database.NewPostgresPool(cfg.DBConnString)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// INIT REDIS CONNECTION POOL
	redisClient, err := database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		slog.Error("Failed to initialize Redis client", "error", err)
		os.Exit(1)
	}

	defer redisClient.Close()
	slog.Info("Successfully connected to Redis connection pool", "host", cfg.RedisHost, "port", cfg.RedisPort)

	// WIRING DEPENDENCY INJECTION - TICKET MODULE (With Redis Cache-Aside Decorator)
	pgTicketRepo := postgres.NewTicketRepository(dbPool)
	ticketRepo := cache.NewCachedTicketRepository(pgTicketRepo, redisClient)
	ticketUC := usecase.NewTicketUseCase(ticketRepo)
	ticketHandler := handler.NewTicketHandler(ticketUC)

	// WIRING DEPENDENCY INJECTION - USER MODULE
	userRepo := postgres.NewUserRepository(dbPool)
	userUC := usecase.NewUserUseCase(userRepo, cfg.JWTSecret)
	userHandler := handler.NewUserHandler(userUC)

	// WIRING DEPENDENCY INJECTION - ORDER MODULE
	eventProducer := messaging.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopicOrderCreated)
	defer func() {
		if err := eventProducer.Close(); err != nil {
			slog.Error("Failed to close Kafka producer connection gracefully", slog.String("error", err.Error()))
		} else {
			slog.Info("Kafka Producer connection closed gracefully")
		}
	}()

	orderRepo := postgres.NewOrderRepository(dbPool)
	idempotencyRepo := cache.NewRedisIdempotencyRepository(redisClient)
	orderUC := usecase.NewOrderUseCase(orderRepo, ticketRepo, idempotencyRepo, eventProducer)
	orderHandler := handler.NewOrderHandler(orderUC)

	// INIT AWS/MINIO CONFIG
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           cfg.MinIOEndpoint,
			SigningRegion: cfg.MinIORegion,
		}, nil
	})

	sdkCfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithRegion(cfg.MinIORegion),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.MinIOAccessKey, cfg.MinIOSecretKey, "")),
		awsConfig.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		slog.Error("Failed to load MinIO configuration", "error", err)
		os.Exit(1)
	}

	s3Client := s3.NewFromConfig(sdkCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// WIRING DEPENDENCY INJECTION - ASSET MODULE
	assetRepo := postgres.NewAssetRepository(dbPool)
	storageRepo := storage.NewS3StorageRepository(s3Client, cfg.MinIOBucket)
	assetUC := usecase.NewAssetUseCase(assetRepo, storageRepo)
	assetHandler := handler.NewAssetHandler(assetUC)

	// SETUP HTTP ROUTER & SERVER
	mux := http.DefaultServeMux

	// --- ROUTES: AUTH ---
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)

	// --- ROUTES: TICKET ---
	mux.HandleFunc("POST /api/v1/events", ticketHandler.CreateEvent)
	mux.HandleFunc("GET /api/v1/events", ticketHandler.BrowseAvailable)
	mux.HandleFunc("POST /api/v1/tickets/checkout", middleware.RequireAuth(ticketHandler.CheckoutTicket))

	// --- ROUTES: ASSET ---
	mux.HandleFunc("POST /api/v1/assets/upload-url", middleware.RequireAuth(assetHandler.RequestUploadURL))
	mux.HandleFunc("PUT /api/v1/assets/confirm", middleware.RequireAuth(assetHandler.ConfirmUpload))

	// --- ROUTES: ORDER ---
	mux.HandleFunc("POST /api/v1/orders", middleware.RequireAuth(orderHandler.CreateOrder))
	mux.HandleFunc("GET /api/v1/orders", middleware.RequireAuth(orderHandler.GetUserOrders))
	mux.HandleFunc("GET /api/v1/orders/{id}", middleware.RequireAuth(orderHandler.GetOrderByID))

	// Health Probe (Liveness)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness Probe
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := dbPool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("DATABASE_UNAVAILABLE"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: mux,
	}

	// Run HTTP server asynchronously
	go func() {
		slog.Info("Ticketing Engine is starting",
			"port", cfg.AppPort,
			"pprof_url", fmt.Sprintf("http://localhost:%s/debug/pprof", cfg.AppPort))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server crashed", "error", err)
			os.Exit(1)
		}
	}()

	// GRACEFUL SHUTDOWN
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit // Block until signal is received
	slog.Warn("Shutdown signal received, initiating graceful shutdown...")

	// Shutdown HTTP Server (Reject new incoming requests)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Failed to gracefully shutdown HTTP server", "error", err)
	} else {
		slog.Info("HTTP server stopped accepting new requests")
	}
}
