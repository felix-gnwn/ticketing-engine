package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felix-gnwn/ticketing-engine/internal/config"
	"github.com/felix-gnwn/ticketing-engine/internal/repository/messaging"
	"github.com/felix-gnwn/ticketing-engine/internal/worker"
)

func main() {
	// Setup Structured JSON Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting Ticketing Engine Worker Daemon...")

	// Load Centralized Environment Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load environment configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize Kafka Event Consumer
	kafkaConsumer := messaging.NewKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaTopicOrderCreated, cfg.KafkaConsumerGroup, cfg.KafkaMinBytes, cfg.KafkaMaxBytes)
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			slog.Error("Error closing Kafka consumer connection", slog.String("error", err.Error()))
		}
	}()

	// Initialize & Start Worker Engine
	ctx, cancel := context.WithCancel(context.Background())
	processor := worker.NewOrderProcessor(kafkaConsumer, cfg.WorkerCount, cfg.WorkerBufferSize)
	processor.Start(ctx)

	// Graceful Shutdown Signal Handler
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

	<-stopSignal
	slog.Info("Shutdown signal received, initiating graceful worker shutdown...")

	// Cancel context to stop fetching new messages
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		processor.Stop()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Worker daemon stopped successfully")
	case <-shutdownCtx.Done():
		slog.Warn("Worker shutdown timed out, forcing exit")
	}
}
