package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	solanasdk "github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	auditrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/audit/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/bsc"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/solana"
	complianceservice "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
	infrastructurerepository "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/repository"
	merchantrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/repository"
	notificationservice "github.com/hxuan190/stable_payment_gateway/internal/modules/notification/service"
	paymentrepo "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/repository"
	paymentport "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/port"
	paymentservice "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
	walletDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/wallet/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func main() {
	// Initialize logger
	appLogger := logger.New(logger.Config{
		Level:        "info",
		Format:       "json",
		Output:       os.Stdout,
		ReportCaller: true,
	})
	appLogger.Info("Starting Blockchain Listener Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to load configuration")
	}

	appLogger.WithFields(logger.Fields{
		"environment": cfg.Environment,
		"solana_rpc":  cfg.Solana.RPCURL,
	}).Info("Configuration loaded")

	db, err := database.InitGORM(&cfg.Database, cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to database")
	}
	appLogger.Info("Database connection established")

	balanceRepo := infrastructurerepository.NewWalletBalanceRepository(db)
	merchantRepo := merchantrepository.NewMerchantRepository(db)
	auditRepo := auditrepository.NewAuditRepository(db)

	// Initialize services
	notifService := notificationservice.NewNotificationService(notificationservice.NotificationServiceConfig{
		Logger:      appLogger.Logger,
		HTTPTimeout: 30 * time.Second,
		EmailSender: cfg.Email.FromEmail,
	})

	// Initialize AML service for compliance screening
	amlService := initializeAMLService(cfg, auditRepo, appLogger.Logger)

	appLogger.Info("AML screening service initialized")

	// Initialize compliance alert service (skip for now - repo type mismatch)
	var complianceAlertService *complianceservice.ComplianceAlertService

	appLogger.Info("Compliance alert service initialized (skipped)")

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

	// Initialize payment service for transaction confirmation
	paymentService := initializePaymentService(cfg, db, merchantRepo, solanaWallet, appLogger.Logger)

	// Create payment confirmation callback with AML screening
	confirmationCallback := createPaymentConfirmationCallback(
		paymentService,
		amlService,
		complianceAlertService,
		appLogger.Logger,
	)

	// Configure supported token mints for Solana
	supportedTokenMints := createSupportedTokenMints(cfg, appLogger.Logger)

	// Create Solana Client for listener
	solanaClient, err := solana.NewClientWithURL(cfg.Solana.RPCURL)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create Solana client for listener")
	}

	// Initialize and start Solana blockchain listener
	solanaListener, err := solana.NewTransactionListener(solana.ListenerConfig{
		Client:               solanaClient,
		Wallet:               solanaWallet,
		WSURL:                cfg.Solana.WSURL, // Use configured WebSocket URL (auto-derives from RPC if empty)
		ConfirmationCallback: confirmationCallback,
		SupportedTokenMints:  supportedTokenMints,
		PollInterval:         10 * time.Second,
		MaxRetries:           3,
	})
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create Solana transaction listener")
	}

	appLogger.WithFields(logger.Fields{
		"wallet_address":   solanaListener.GetWalletAddress(),
		"supported_tokens": len(supportedTokenMints),
	}).Info("Solana transaction listener configured")

	// Start the listener
	if err := solanaListener.Start(); err != nil {
		appLogger.WithError(err).Fatal("Failed to start Solana transaction listener")
	}

	appLogger.Info("Solana transaction listener started successfully")

	// Initialize and start BSC blockchain listener (if configured)
	var bscListener *bsc.TransactionListener
	if cfg.BSC.WalletPrivateKey != "" && cfg.BSC.RPCURL != "" {
		appLogger.Info("Initializing BSC blockchain listener...")

		// Initialize BSC wallet
		bscWallet, err := bsc.LoadWallet(cfg.BSC.WalletPrivateKey, cfg.BSC.RPCURL)
		if err != nil {
			appLogger.WithError(err).Fatal("Failed to load BSC wallet")
		}

		appLogger.WithField("wallet_address", bscWallet.GetAddress()).Info("BSC wallet loaded")

		// Verify BSC wallet health
		if err := bscWallet.HealthCheck(ctx); err != nil {
			appLogger.WithError(err).Fatal("BSC wallet health check failed")
		}

		appLogger.Info("BSC wallet health check passed")

		// Configure supported token contracts for BSC
		supportedBSCTokens := createSupportedBSCTokenContracts(cfg, appLogger.Logger)

		// Create BSC Client for listener
		bscClient, err := bsc.NewClientWithURL(cfg.BSC.RPCURL)
		if err != nil {
			appLogger.WithError(err).Fatal("Failed to create BSC client for listener")
		}

		// Initialize BSC blockchain listener
		bscListener, err = bsc.NewTransactionListener(bsc.ListenerConfig{
			Client:                  bscClient,
			Wallet:                  bscWallet,
			ConfirmationCallback:    confirmationCallback,
			SupportedTokenContracts: supportedBSCTokens,
			PollInterval:            10 * time.Second,
			RequiredConfirmations:   15, // BSC finality
			MaxRetries:              3,
		})
		if err != nil {
			appLogger.WithError(err).Fatal("Failed to create BSC transaction listener")
		}

		appLogger.WithFields(logger.Fields{
			"wallet_address":    bscListener.GetWalletAddress(),
			"supported_tokens":  len(supportedBSCTokens),
			"required_confirms": 15,
		}).Info("BSC transaction listener configured")

		// Start the BSC listener
		if err := bscListener.Start(); err != nil {
			appLogger.WithError(err).Fatal("Failed to start BSC transaction listener")
		}

		appLogger.Info("BSC transaction listener started successfully")
	} else {
		appLogger.Info("BSC configuration not found, skipping BSC listener initialization")
	}

	// Start wallet balance monitor
	monitorConfig := solana.WalletMonitorConfig{
		CheckInterval:   5 * time.Minute,
		MinThresholdUSD: decimal.NewFromInt(1000),      // $1,000 minimum
		MaxThresholdUSD: decimal.NewFromInt(10000),     // $10,000 maximum
		AlertEmails:     []string{cfg.Email.FromEmail}, // TODO: Configure ops team emails
		Network:         getNetworkFromString(cfg.Solana.Network),
	}

	walletMonitor := solana.NewWalletMonitor(
		solanaWallet,
		monitorConfig,
		appLogger.Logger,
		notifService,
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

	// Stop Solana listener
	if err := solanaListener.Stop(); err != nil {
		appLogger.WithError(err).Error("Error stopping Solana listener")
	}

	// Stop BSC listener if running
	if bscListener != nil {
		if err := bscListener.Stop(); err != nil {
			appLogger.WithError(err).Error("Error stopping BSC listener")
		}
	}

	// Stop wallet monitor
	walletMonitor.Stop()

	appLogger.Info("Blockchain listener stopped successfully")
	fmt.Println("Blockchain listener stopped successfully")
}

// getNetworkFromString converts string network to walletDomain.Network
func getNetworkFromString(network string) walletDomain.Network {
	switch network {
	case "mainnet":
		return walletDomain.NetworkMainnet
	case "testnet":
		return walletDomain.NetworkTestnet
	case "devnet":
		return walletDomain.NetworkDevnet
	default:
		return walletDomain.NetworkDevnet
	}
}

// initializeAMLService creates an AML service instance
func initializeAMLService(
	cfg *config.Config,
	auditRepo *auditrepository.AuditRepository,
	appLogger *logrus.Logger,
) complianceservice.AMLService {
	var trmClient complianceservice.TRMLabsClient

	if cfg.TRM.APIKey != "" && cfg.TRM.BaseURL != "" {
		appLogger.Info("Initializing TRM Labs client (production mode)")
		client := initTRMLabsClient(cfg.TRM.APIKey, cfg.TRM.BaseURL)
		trmClient = client
	} else {
		appLogger.Info("TRM Labs API key not configured, using mock client")
		trmClient = createMockTRMClient()
	}

	return complianceservice.NewAMLService(
		trmClient,
		auditRepo,
		nil,
		nil,
		nil,
		logger.GetLogger(),
	)
}

func initializePaymentService(
	cfg *config.Config,
	db *gorm.DB,
	merchantRepo *merchantrepository.MerchantRepository,
	solanaWallet *solana.Wallet,
	appLogger *logrus.Logger,
) *paymentservice.PaymentService {
	newPaymentRepo := paymentrepo.NewPostgresPaymentRepository(db)
	return paymentservice.NewPaymentService(
		newPaymentRepo,
		merchantRepo,
		nil,
		nil,
		nil,
		paymentservice.PaymentServiceConfig{
			DefaultChain:    "solana",
			DefaultCurrency: "USDT",
			WalletAddress:   solanaWallet.GetAddress(),
			FeePercentage:   0.01,
			ExpiryMinutes:   30,
			RedisClient:     nil,
		},
		appLogger,
	)
}

// createPaymentConfirmationCallback creates a callback function with AML screening
func createPaymentConfirmationCallback(
	paymentService *paymentservice.PaymentService,
	amlService complianceservice.AMLService,
	complianceAlertService *complianceservice.ComplianceAlertService,
	appLogger *logrus.Logger,
) func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
	return func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
		ctx := context.Background()

		appLogger.WithFields(logrus.Fields{
			"payment_id": paymentID,
			"tx_hash":    txHash,
			"amount":     amount,
			"token":      tokenMint,
		}).Info("Processing payment confirmation")

		// Get payment to extract sender address
		payment, err := paymentService.GetPaymentStatus(ctx, paymentID)
		if err != nil {
			appLogger.WithError(err).Error("Failed to get payment for AML screening")
			return fmt.Errorf("failed to get payment: %w", err)
		}

		// Extract from_address from payment (it will be set by blockchain parser)
		// For now, we'll screen after confirmation when from_address is available
		// In a full implementation, the listener would pass from_address to this callback

		// Create confirmation request
		confirmReq := paymentport.ConfirmPaymentRequest{
			PaymentID:     paymentID,
			TxHash:        txHash,
			ActualAmount:  amount,
			Confirmations: 1,
			FromAddress:   "",
		}

		// Confirm the payment
		confirmedPayment, err := paymentService.ConfirmPayment(ctx, confirmReq)
		if err != nil {
			appLogger.WithError(err).Error("Payment confirmation failed")
			return fmt.Errorf("payment confirmation failed: %w", err)
		}

		// AML SCREENING: Screen the sender address after payment is confirmed
		if confirmedPayment.FromAddress.Valid && confirmedPayment.FromAddress.String != "" {
			fromAddress := confirmedPayment.FromAddress.String

			appLogger.WithFields(logrus.Fields{
				"payment_id":   paymentID,
				"from_address": fromAddress,
				"chain":        "solana",
			}).Info("Screening sender address for AML compliance")

			// Screen the wallet address
			result, err := amlService.ScreenWalletAddress(ctx, fromAddress, "solana")
			if err != nil {
				appLogger.WithError(err).Warn("AML screening failed (non-fatal)")
				// Don't fail payment if AML screening service has issues
				// Log the error and continue
			} else {
				// Record screening result
				paymentUUID, parseErr := uuid.Parse(payment.ID)
				if parseErr != nil {
					appLogger.WithError(parseErr).Warn("Failed to parse payment ID for screening result")
				} else {
					err = amlService.RecordScreeningResult(ctx, paymentUUID, result)
					if err != nil {
						appLogger.WithError(err).Warn("Failed to record AML screening result")
					}
				}

				// Check if address is sanctioned or high-risk
				if result.IsSanctioned {
					appLogger.WithFields(logrus.Fields{
						"payment_id":   paymentID,
						"from_address": fromAddress,
						"risk_score":   result.RiskScore,
						"flags":        result.Flags,
					}).Error("SANCTIONED ADDRESS DETECTED - Manual review required")

					// Create compliance alert for sanctioned address
					paymentUUID, err := uuid.Parse(paymentID)
					if err != nil {
						appLogger.WithError(err).Error("Failed to parse payment ID for compliance alert")
					} else {
						// Prepare evidence from AML screening
						evidence := map[string]interface{}{
							"screening_provider": "TRM Labs",
							"risk_score":         result.RiskScore,
							"flags":              result.Flags,
							"sanctioned":         result.IsSanctioned,
							"screened_at":        time.Now().Format(time.RFC3339),
						}

						// Create sanctioned address alert
						alert, err := complianceAlertService.CreateSanctionedAddressAlert(
							ctx,
							paymentUUID,
							fromAddress,
							string(payment.Chain),
							result.RiskScore,
							result.Flags,
							evidence,
						)
						if err != nil {
							appLogger.WithError(err).Error("Failed to create sanctioned address alert")
						} else {
							appLogger.WithFields(logrus.Fields{
								"alert_id":   alert.ID,
								"payment_id": paymentID,
							}).Warn("Sanctioned address alert created and compliance team notified")
						}
					}
				} else if result.RiskScore >= 80 {
					appLogger.WithFields(logrus.Fields{
						"payment_id":   paymentID,
						"from_address": fromAddress,
						"risk_score":   result.RiskScore,
						"flags":        result.Flags,
					}).Warn("HIGH-RISK ADDRESS DETECTED - Review recommended")

					// Create compliance alert for high-risk address
					paymentUUID, err := uuid.Parse(paymentID)
					if err != nil {
						appLogger.WithError(err).Error("Failed to parse payment ID for compliance alert")
					} else {
						// Prepare evidence from AML screening
						evidence := map[string]interface{}{
							"screening_provider": "TRM Labs",
							"risk_score":         result.RiskScore,
							"flags":              result.Flags,
							"sanctioned":         result.IsSanctioned,
							"screened_at":        time.Now().Format(time.RFC3339),
						}

						// Create high-risk address alert
						alert, err := complianceAlertService.CreateHighRiskAlert(
							ctx,
							paymentUUID,
							fromAddress,
							string(payment.Chain),
							result.RiskScore,
							result.Flags,
							evidence,
						)
						if err != nil {
							appLogger.WithError(err).Error("Failed to create high-risk address alert")
						} else {
							appLogger.WithFields(logrus.Fields{
								"alert_id":   alert.ID,
								"payment_id": paymentID,
								"severity":   alert.Severity,
							}).Info("High-risk address alert created")
						}
					}
				} else {
					appLogger.WithFields(logrus.Fields{
						"payment_id":   paymentID,
						"from_address": fromAddress,
						"risk_score":   result.RiskScore,
					}).Info("AML screening passed - clean address")
				}
			}
		}

		appLogger.WithField("payment_id", paymentID).Info("Payment confirmed successfully")
		return nil
	}
}

// initTRMLabsClient initializes a real TRM Labs client
func initTRMLabsClient(apiKey, baseURL string) complianceservice.TRMLabsClient {
	client := trmlabs.NewClient(apiKey, baseURL)
	return client
}

// createMockTRMClient creates a mock TRM Labs client for development
func createMockTRMClient() complianceservice.TRMLabsClient {
	return trmlabs.NewMockClient()
}

// createSupportedTokenMints creates a map of supported token mints from config
func createSupportedTokenMints(cfg *config.Config, appLogger *logrus.Logger) map[string]solana.TokenMintInfo {
	supportedTokens := make(map[string]solana.TokenMintInfo)

	// Add USDT if configured
	if cfg.Solana.USDTMint != "" {
		usdtMint, err := solanasdk.PublicKeyFromBase58(cfg.Solana.USDTMint)
		if err != nil {
			appLogger.WithFields(logrus.Fields{
				"error": err.Error(),
				"mint":  cfg.Solana.USDTMint,
			}).Warn("Invalid USDT mint address, skipping")
		} else {
			supportedTokens["USDT"] = solana.TokenMintInfo{
				MintAddress: usdtMint,
				Symbol:      "USDT",
				Decimals:    6, // USDT uses 6 decimals on Solana
			}
			appLogger.WithField("mint", cfg.Solana.USDTMint).Info("USDT support configured")
		}
	}

	// Add USDC if configured
	if cfg.Solana.USDCMint != "" {
		usdcMint, err := solanasdk.PublicKeyFromBase58(cfg.Solana.USDCMint)
		if err != nil {
			appLogger.WithFields(logrus.Fields{
				"error": err.Error(),
				"mint":  cfg.Solana.USDCMint,
			}).Warn("Invalid USDC mint address, skipping")
		} else {
			supportedTokens["USDC"] = solana.TokenMintInfo{
				MintAddress: usdcMint,
				Symbol:      "USDC",
				Decimals:    6, // USDC uses 6 decimals on Solana
			}
			appLogger.WithField("mint", cfg.Solana.USDCMint).Info("USDC support configured")
		}
	}

	if len(supportedTokens) == 0 {
		appLogger.Warn("No supported token mints configured - listener will not process any payments")
	}

	return supportedTokens
}

// createSupportedBSCTokenContracts creates a map of supported BEP20 token contracts from config
func createSupportedBSCTokenContracts(cfg *config.Config, appLogger *logrus.Logger) map[string]bsc.TokenContractInfo {
	supportedTokens := make(map[string]bsc.TokenContractInfo)

	// Add USDT if configured
	if cfg.BSC.USDTContract != "" {
		if !common.IsHexAddress(cfg.BSC.USDTContract) {
			appLogger.WithFields(logrus.Fields{
				"contract": cfg.BSC.USDTContract,
			}).Warn("Invalid USDT contract address, skipping")
		} else {
			usdtContract := common.HexToAddress(cfg.BSC.USDTContract)
			supportedTokens["USDT"] = bsc.TokenContractInfo{
				ContractAddress: usdtContract,
				Symbol:          "USDT",
				Decimals:        18, // USDT uses 18 decimals on BSC
			}
			appLogger.WithField("contract", cfg.BSC.USDTContract).Info("BSC USDT support configured")
		}
	}

	// Add BUSD if configured
	if cfg.BSC.BUSDContract != "" {
		if !common.IsHexAddress(cfg.BSC.BUSDContract) {
			appLogger.WithFields(logrus.Fields{
				"contract": cfg.BSC.BUSDContract,
			}).Warn("Invalid BUSD contract address, skipping")
		} else {
			busdContract := common.HexToAddress(cfg.BSC.BUSDContract)
			supportedTokens["BUSD"] = bsc.TokenContractInfo{
				ContractAddress: busdContract,
				Symbol:          "BUSD",
				Decimals:        18, // BUSD uses 18 decimals on BSC
			}
			appLogger.WithField("contract", cfg.BSC.BUSDContract).Info("BSC BUSD support configured")
		}
	}

	if len(supportedTokens) == 0 {
		appLogger.Warn("No supported BSC token contracts configured - BSC listener will not process any payments")
	}

	return supportedTokens
}
