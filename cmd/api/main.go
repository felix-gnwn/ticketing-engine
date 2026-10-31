package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/felix-gnwn/ticketing-engine/internal/database"
	"github.com/felix-gnwn/ticketing-engine/internal/middleware"
	"github.com/felix-gnwn/ticketing-engine/internal/handler"
	postgres "github.com/felix-gnwn/ticketing-engine/internal/repository/postgres"
	"github.com/felix-gnwn/ticketing-engine/internal/usecase"
)

func main() {
	// INIT STRUCTURED LOGGING (slog)
	// JSON logs are mandatory for indexing and searching.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Display DEBUG, INFO, WARN, and ERROR
	}))
	slog.SetDefault(logger)

	// LOAD ENVIRONMENT VARIABLES
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, relying on system environment variables")
	}

	// JWT SECRET
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET is not set in environment")
		os.Exit(1)
	}

	// INIT DATABASE CONNECTION POOL
	connString := os.Getenv("DB_CONNECTION_STRING")
	if connString == "" {
		slog.Error("DB_CONNECTION_STRING is not set in environment")
		os.Exit(1)
	}

	dbPool, err := database.NewPostgresPool(connString)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// WIRING DEPENDENCY INJECTION - TICKET MODULE
	ticketRepo := postgres.NewTicketRepository(dbPool)
	ticketUC := usecase.NewTicketUseCase(ticketRepo)
	ticketHandler := handler.NewTicketHandler(ticketUC)

	// WIRING DEPENDENCY INJECTION - TICKET MODULE
	userRepo := postgres.NewUserRepository(dbPool)
	userUC := usecase.NewUserUseCase(userRepo, jwtSecret)
	userHandler := handler.NewUserHandler(userUC)

	// SETUP WORKER POOL
	const numWorkers = 3
	jobs := make(chan string, 100)
	var wg sync.WaitGroup

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg)
	}

	// SETUP HTTP ROUTER & SERVER
	mux := http.DefaultServeMux

	// --- ROUTES: AUTH ---
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)

	// --- ROUTES: TICKET ---
	mux.HandleFunc("POST /api/v1/events", ticketHandler.CreateEvent)
	mux.HandleFunc("GET /api/v1/events", ticketHandler.BrowseAvailable)
	mux.HandleFunc("POST /api/v1/tickets/checkout", middleware.RequireAuth(ticketHandler.CheckoutTicket))
	mux.HandleFunc("/api/ticket", func(w http.ResponseWriter, r *http.Request) {
		orderID := fmt.Sprintf("TICKET-%d", time.Now().UnixNano())

		jobs <- orderID

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(fmt.Sprintf("Order %s has been accepted and queued for processing.\n", orderID)))
	})

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
		Addr:    ":8080",
		Handler: mux,
	}

	// Run HTTP server asynchronously
	go func() {
		slog.Info("Ticketing Engine is starting", 
			"port", 8080, 
			"pprof_url", "http://localhost:8080/debug/pprof")
			
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

	// Close job channel and wait for workers to complete current tasks
	close(jobs)
	slog.Info("Waiting for background workers to finish remaining jobs...")
	wg.Wait()

	slog.Info("All workers stopped. System shutdown complete.")
}

// WORKER LOGIC
func worker(id int, jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for orderID := range jobs {
		slog.Info("Worker started processing", 
			"worker_id", id, 
			"order_id", orderID)
		
		// Simulate heavy database transaction or external API call
		time.Sleep(2 * time.Second) 
		
		slog.Info("Worker finished processing", 
			"worker_id", id, 
			"order_id", orderID)
	}
	slog.Debug("Worker stopped (no more jobs in queue)", "worker_id", id)
}