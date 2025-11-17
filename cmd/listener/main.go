package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

func main() {
	// Initialize logger
	appLogger := logger.New()
	appLogger.Info("Starting Blockchain Listener Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to load configuration")
	}

	appLogger.WithFields(logrus.Fields{
		"environment": cfg.Environment,
		"solana_rpc":  cfg.Solana.RPCURL,
	}).Info("Configuration loaded")

	// Initialize database connection
	db, err := database.Connect(cfg.GetDatabaseDSN())
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	appLogger.Info("Database connection established")

	// Initialize repositories
	balanceRepo := repository.NewWalletBalanceRepository(db)

	// Initialize notification service
	notificationService := service.NewNotificationService(service.NotificationServiceConfig{
		Logger:      appLogger,
		HTTPTimeout: 30 * time.Second,
		EmailSender: cfg.Email.FromEmail,
		EmailAPIKey: cfg.Email.APIKey,
	})

	// Initialize Solana wallet
	if cfg.Solana.WalletPrivateKey == "" {
		appLogger.Fatal("SOLANA_WALLET_PRIVATE_KEY is required")
	}

	solanaWallet, err := solana.LoadWallet(cfg.Solana.WalletPrivateKey, cfg.Solana.RPCURL)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to load Solana wallet")
	}

	appLogger.WithField("wallet_address", solanaWallet.GetAddress()).Info("Solana wallet loaded")

	// Verify wallet health
	ctx := context.Background()
	if err := solanaWallet.HealthCheck(ctx); err != nil {
		appLogger.WithError(err).Fatal("Wallet health check failed")
	}

	appLogger.Info("Wallet health check passed")

	// TODO: Initialize blockchain clients (Solana, BSC)
	// TODO: Initialize payment service
	// TODO: Start blockchain listeners

	// Start wallet balance monitor
	monitorConfig := solana.WalletMonitorConfig{
		CheckInterval:   5 * time.Minute,
		MinThresholdUSD: decimal.NewFromInt(1000),  // $1,000 minimum
		MaxThresholdUSD: decimal.NewFromInt(10000), // $10,000 maximum
		AlertEmails:     []string{cfg.Email.FromEmail}, // TODO: Configure ops team emails
		Network:         getNetworkFromString(cfg.Solana.Network),
	}

	walletMonitor := solana.NewWalletMonitor(
		solanaWallet,
		monitorConfig,
		appLogger,
		notificationService,
		balanceRepo,
		cfg.Solana,
	)

	// Start monitor in background
	go walletMonitor.Start(ctx)

	appLogger.Info("Wallet balance monitor started")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down blockchain listener...")

	// Stop wallet monitor
	walletMonitor.Stop()

	fmt.Println("Blockchain listener stopped successfully")
}

// getNetworkFromString converts string network to model.Network
func getNetworkFromString(network string) model.Network {
	switch network {
	case "mainnet":
		return model.NetworkMainnet
	case "testnet":
		return model.NetworkTestnet
	case "devnet":
		return model.NetworkDevnet
	default:
		return model.NetworkDevnet
	}
}
