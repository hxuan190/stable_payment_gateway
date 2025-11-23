package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/api"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
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

	// Initialize Redis connection
	logger.Info("Initializing Redis connection...")
	redisClient, err := cache.NewRedisCache(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Fatal("Failed to initialize Redis connection", err)
	}

	// Test Redis connection
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := redisClient.Ping(pingCtx); err != nil {
		logger.Fatal("Redis connection failed", err)
	}
	logger.Info("Redis connection established")

	// Initialize Solana client (optional for API server, but useful for health checks)
	var solanaClient *solana.Client
	var solanaWallet *solana.Wallet

	if cfg.Solana.RPCURL != "" && cfg.Solana.WalletAddress != "" {
		logger.Info("Initializing Solana client...")
		solanaClient, err = solana.NewClientWithURL(cfg.Solana.RPCURL)
		if err != nil {
			logger.Warn("Failed to initialize Solana client (non-fatal for API server)", logger.Fields{
				"error": err.Error(),
			})
		} else {
			logger.Info("Solana client initialized", logger.Fields{
				"network": cfg.Solana.Network,
				"rpc_url": cfg.Solana.RPCURL,
			})

			// Initialize wallet if private key is provided
			if cfg.Solana.WalletPrivateKey != "" {
				solanaWallet, err = solana.LoadWallet(cfg.Solana.WalletPrivateKey, cfg.Solana.RPCURL)
				if err != nil {
					logger.Warn("Failed to load Solana wallet (non-fatal for API server)", logger.Fields{
						"error": err.Error(),
					})
				} else {
					logger.Info("Solana wallet loaded", logger.Fields{
						"address": solanaWallet.GetAddress(),
					})
				}
			}
		}
	}

	// TODO: Initialize repositories
	// TODO: Initialize services

	// Set up HTTP server
	logger.Info("Setting up HTTP server...")
	apiServer := api.NewServer(&api.ServerConfig{
		Config:       cfg,
		DB:           db.DB,
		Cache:        redisClient,
		SolanaClient: solanaClient,
		SolanaWallet: solanaWallet,
	})

	// Start HTTP server
	if err := apiServer.Start(); err != nil {
		logger.Fatal("Failed to start HTTP server", err)
	}

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

	// Shutdown HTTP server
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down HTTP server", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		logger.Error("Error closing Redis connection", err)
	}

	// Close database connection
	if err := db.Close(); err != nil {
		logger.Error("Error closing database connection", err)
	}

	<-shutdownCtx.Done()
	if shutdownCtx.Err() == context.DeadlineExceeded {
		logger.Warn("Shutdown deadline exceeded, forcing exit")
	}

	fmt.Println("API server stopped")
}
