package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

func main() {
	// Initialize logger
	logger.Init(logger.Config{
		Level:  "info",
		Format: "json",
	})

	logger.Info("Starting Payment Gateway API Server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", err)
	}

	logger.Info("Configuration loaded successfully", logger.Fields{
		"environment": cfg.Environment,
		"api_port":    cfg.API.Port,
	})

	// Initialize database connection
	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database connection", err)
	}

	logger.Info("Database connection established")

	// Wait for database to be ready (useful in Docker environments)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.WaitForConnection(ctx, 5); err != nil {
		logger.Fatal("Database connection failed", err)
	}

	// Log database pool stats
	db.LogPoolStats()

	// Check database health
	if err := db.HealthCheck(context.Background()); err != nil {
		logger.Fatal("Database health check failed", err)
	}

	// Check migration version
	version, err := db.CheckMigrationVersion(context.Background())
	if err != nil {
		logger.Warn("Failed to check migration version", logger.Fields{
			"error": err.Error(),
		})
	} else {
		logger.Info("Database migration version", logger.Fields{
			"version": version,
		})
	}

	// TODO: Initialize Redis connection
	// TODO: Initialize services
	// TODO: Set up HTTP server with Gin
	// TODO: Set up routes and middleware
	// TODO: Start HTTP server in goroutine

	logger.Info("API server started successfully", logger.Fields{
		"port": cfg.API.Port,
		"host": cfg.API.Host,
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Received shutdown signal, gracefully shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Close database connection
	if err := db.Close(); err != nil {
		logger.Error("Error closing database connection", err)
	}

	// TODO: Shutdown HTTP server
	// TODO: Close Redis connection
	// TODO: Close other resources

	<-shutdownCtx.Done()
	if shutdownCtx.Err() == context.DeadlineExceeded {
		logger.Warn("Shutdown deadline exceeded, forcing exit")
	}

	fmt.Println("API server stopped")
}
