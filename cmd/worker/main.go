package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/worker"
)

func main() {
	// Initialize logger
	logger.Init(logger.Config{
		Level:  "info",
		Format: "json",
	})

	logger.Info("Starting Background Worker Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", err)
	}

	logger.Info("Configuration loaded successfully", logger.Fields{
		"environment": cfg.Environment,
	})

	// Initialize database connection
	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database connection", err)
	}

	logger.Info("Database connection established")

	// Wait for database to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.WaitForConnection(ctx, 5); err != nil {
		logger.Fatal("Database connection failed", err)
	}

	// Check database health
	if err := db.HealthCheck(context.Background()); err != nil {
		logger.Fatal("Database health check failed", err)
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

	// Initialize Solana client and wallet (optional, for balance monitoring)
	var solanaClient *solana.Client
	var solanaWallet *solana.Wallet

	if cfg.Solana.RPCURL != "" && cfg.Solana.WalletAddress != "" {
		logger.Info("Initializing Solana client...")
		solanaClient, err = solana.NewClientWithURL(cfg.Solana.RPCURL)
		if err != nil {
			logger.Warn("Failed to initialize Solana client (non-fatal for worker)", logger.Fields{
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
					logger.Warn("Failed to load Solana wallet", logger.Fields{
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

	// Create worker server
	logger.Info("Setting up worker server...")
	workerServer := worker.NewServer(&worker.ServerConfig{
		RedisAddr:                fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		RedisPassword:            cfg.Redis.Password,
		RedisDB:                  cfg.Redis.DB,
		DB:                       db.DB,
		Cache:                    redisClient,
		SolanaClient:             solanaClient,
		SolanaWallet:             solanaWallet,
		Concurrency:              10, // Process up to 10 jobs concurrently
		ExchangeRatePrimaryAPI:   cfg.ExchangeRate.PrimaryAPI,
		ExchangeRateSecondaryAPI: cfg.ExchangeRate.SecondaryAPI,
		ExchangeRateCacheTTL:     time.Duration(cfg.ExchangeRate.CacheTTL) * time.Second,
		ExchangeRateTimeout:      time.Duration(cfg.ExchangeRate.Timeout) * time.Second,
		Queues: map[string]int{
			"webhooks":       5, // Highest priority
			"webhooks_retry": 3,
			"periodic":       4,
			"monitoring":     2,
			"reports":        1, // Lowest priority
		},
	})

	logger.Info("Worker server configured successfully")

	// Start worker server in a goroutine
	go func() {
		logger.Info("Starting worker server...")
		if err := workerServer.Start(); err != nil {
			logger.Fatal("Worker server failed", err)
		}
	}()

	logger.Info("Worker service started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Received shutdown signal, gracefully shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown worker server
	if err := workerServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down worker server", err)
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

	fmt.Println("Worker service stopped")
}
