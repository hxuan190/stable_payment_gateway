package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	solanasdk "github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/bsc"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
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
	paymentRepo := repository.NewPaymentRepository(db, appLogger)
	merchantRepo := repository.NewMerchantRepository(db, appLogger)
	auditRepo := repository.NewAuditLogRepository(db, appLogger)
	complianceAlertRepo := repository.NewComplianceAlertRepository(db)

	// Initialize services
	notificationService := service.NewNotificationService(service.NotificationServiceConfig{
		Logger:      appLogger,
		HTTPTimeout: 30 * time.Second,
		EmailSender: cfg.Email.FromEmail,
		EmailAPIKey: cfg.Email.APIKey,
	})

	// Initialize AML service for compliance screening
	amlService := initializeAMLService(cfg, auditRepo, paymentRepo, appLogger)

	appLogger.Info("AML screening service initialized")

	// Initialize compliance alert service
	complianceAlertService := service.NewComplianceAlertService(service.ComplianceAlertServiceConfig{
		ComplianceAlertRepo: complianceAlertRepo,
		PaymentRepo:         paymentRepo,
		NotificationService: notificationService,
		Logger:              appLogger,
		OpsTeamEmails:       cfg.OpsTeamEmails, // From config
	})

	appLogger.Info("Compliance alert service initialized")

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
	paymentService := initializePaymentService(cfg, paymentRepo, merchantRepo, solanaWallet, appLogger)

	// Create payment confirmation callback with AML screening
	confirmationCallback := createPaymentConfirmationCallback(
		paymentService,
		amlService,
		complianceAlertService,
		appLogger,
	)

	// Configure supported token mints for Solana
	supportedTokenMints := createSupportedTokenMints(cfg, appLogger)

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

	appLogger.WithFields(logrus.Fields{
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
		supportedBSCTokens := createSupportedBSCTokenContracts(cfg, appLogger)

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

		appLogger.WithFields(logrus.Fields{
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

// initializeAMLService creates an AML service instance
func initializeAMLService(
	cfg *config.Config,
	auditRepo *repository.AuditLogRepository,
	paymentRepo *repository.PaymentRepository,
	appLogger *logrus.Logger,
) service.AMLService {
	// Initialize TRM Labs client (or mock if no API key)
	var trmClient service.TRMLabsClient

	if cfg.TRM.APIKey != "" && cfg.TRM.BaseURL != "" {
		appLogger.Info("Initializing TRM Labs client (production mode)")
		client, err := initTRMLabsClient(cfg.TRM.APIKey, cfg.TRM.BaseURL)
		if err != nil {
			appLogger.WithError(err).Warn("Failed to initialize TRM Labs client, using mock")
			trmClient = createMockTRMClient()
		} else {
			trmClient = client
		}
	} else {
		appLogger.Info("TRM Labs API key not configured, using mock client")
		trmClient = createMockTRMClient()
	}

	return service.NewAMLService(
		trmClient,
		auditRepo,
		paymentRepo,
		appLogger,
	)
}

// initializePaymentService creates a payment service instance
func initializePaymentService(
	cfg *config.Config,
	paymentRepo *repository.PaymentRepository,
	merchantRepo *repository.MerchantRepository,
	solanaWallet *solana.Wallet,
	appLogger *logrus.Logger,
) *service.PaymentService {
	// For the listener, we don't need exchange rate service or compliance service
	// We only need to confirm payments
	return service.NewPaymentService(
		paymentRepo,
		merchantRepo,
		nil, // No exchange rate service needed for confirmation
		nil, // No compliance service needed (already checked during creation)
		service.PaymentServiceConfig{
			DefaultChain:    model.ChainSolana,
			DefaultCurrency: "USDT",
			WalletAddress:   solanaWallet.GetAddress(),
			FeePercentage:   0.01,
			ExpiryMinutes:   30,
			RedisClient:     nil, // TODO: Add Redis if available
		},
		appLogger,
	)
}

// createPaymentConfirmationCallback creates a callback function with AML screening
func createPaymentConfirmationCallback(
	paymentService *service.PaymentService,
	amlService service.AMLService,
	complianceAlertService *service.ComplianceAlertService,
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
		confirmReq := service.ConfirmPaymentRequest{
			PaymentID:     paymentID,
			TxHash:        txHash,
			ActualAmount:  amount,
			Confirmations: 1,  // Solana finalized = sufficient
			FromAddress:   "", // Will be extracted from transaction
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
				err = amlService.RecordScreeningResult(ctx, payment.IDAsUUID(), result)
				if err != nil {
					appLogger.WithError(err).Warn("Failed to record AML screening result")
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
							payment.Chain,
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
							payment.Chain,
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
func initTRMLabsClient(apiKey, baseURL string) (service.TRMLabsClient, error) {
	client, err := trmlabs.NewClient(apiKey, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create TRM Labs client: %w", err)
	}
	return client, nil
}

// createMockTRMClient creates a mock TRM Labs client for development
func createMockTRMClient() service.TRMLabsClient {
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
