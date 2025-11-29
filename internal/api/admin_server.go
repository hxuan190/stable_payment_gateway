package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/api/handler"
	"github.com/hxuan190/stable_payment_gateway/internal/api/middleware"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	auditrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/audit/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/solana"
	compliancerepository "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/repository"
	complianceservice "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
	infrastructurerepository "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/repository"
	ledgerrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/repository"
	merchanthandler "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/handler"
	merchantrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/repository"
	merchantservice "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/service"
	paymentrepo "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/repository"
	payoutrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/repository"
	payoutservice "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/service"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	jwtpkg "github.com/hxuan190/stable_payment_gateway/internal/pkg/jwt"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/storage"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// kycStorageAdapter adapts storage.StorageService to handler.StorageService
type kycStorageAdapter struct {
	storage storage.StorageService
}

func (a *kycStorageAdapter) UploadKYCDocument(ctx context.Context, merchantID string, docType string, filename string, content []byte) (string, error) {
	return a.storage.UploadKYCDocument(ctx, merchantID, docType, filename, bytes.NewReader(content), "application/octet-stream")
}

func (a *kycStorageAdapter) DeleteFile(ctx context.Context, fileURL string) error {
	return a.storage.DeleteKYCDocument(ctx, fileURL)
}

// walletBalanceGetterAdapter adapts Solana wallet to WalletBalanceGetter interface
type walletBalanceGetterAdapter struct {
	wallet *solana.Wallet
}

func (a *walletBalanceGetterAdapter) GetBalance() (decimal.Decimal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenMints := map[string]string{
		"USDT": "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
		"USDC": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
	}

	walletBalance, err := a.wallet.GetWalletBalance(ctx, tokenMints)
	if err != nil {
		return decimal.Zero, err
	}

	totalUSD := decimal.Zero
	if usdtInfo, ok := walletBalance.Tokens["USDT"]; ok {
		totalUSD = totalUSD.Add(usdtInfo.Balance)
	}
	if usdcInfo, ok := walletBalance.Tokens["USDC"]; ok {
		totalUSD = totalUSD.Add(usdcInfo.Balance)
	}

	return totalUSD, nil
}

// AdminServer represents the admin HTTP server
type AdminServer struct {
	config       *config.Config
	router       *gin.Engine
	httpServer   *http.Server
	gormDB       *gorm.DB
	cache        cache.Cache
	jwtManager   *jwtpkg.Manager
	solanaClient *solana.Client
	solanaWallet *solana.Wallet
}

// AdminServerConfig holds dependencies for the admin server
type AdminServerConfig struct {
	Config       *config.Config
	GormDB       *gorm.DB
	Cache        cache.Cache
	SolanaClient *solana.Client
	SolanaWallet *solana.Wallet
}

// NewAdminServer creates a new admin HTTP server instance
func NewAdminServer(cfg *AdminServerConfig) *AdminServer {
	// Set Gin mode based on environment
	if cfg.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize JWT manager
	jwtManager := jwtpkg.NewManager(cfg.Config.JWT.Secret, cfg.Config.JWT.ExpirationHours)

	server := &AdminServer{
		config:       cfg.Config,
		gormDB:       cfg.GormDB,
		cache:        cfg.Cache,
		jwtManager:   jwtManager,
		solanaClient: cfg.SolanaClient,
		solanaWallet: cfg.SolanaWallet,
	}

	// Initialize router
	server.setupRouter()

	return server
}

// setupRouter configures the Gin router with all middleware and routes
func (s *AdminServer) setupRouter() {
	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())             // Recover from panics
	router.Use(middleware.RequestLogger()) // Log all requests
	router.Use(s.corsMiddleware())         // CORS configuration

	// Set up routes
	s.setupRoutes(router)

	s.router = router
}

// corsMiddleware configures CORS settings for admin panel
func (s *AdminServer) corsMiddleware() gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins: s.config.API.AllowedOrigins,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Request-ID",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(config)
}

// setupRoutes configures all admin API routes
func (s *AdminServer) setupRoutes(router *gin.Engine) {
	// Initialize repositories
	merchantRepo := merchantrepository.NewMerchantRepository(s.gormDB)
	newPaymentRepo := paymentrepo.NewPostgresPaymentRepository(s.gormDB)
	payoutRepo := payoutrepository.NewPayoutRepository(s.gormDB)
	balanceRepo := ledgerrepository.NewBalanceRepository(s.gormDB)
	auditRepo := auditrepository.NewAuditRepository(s.gormDB)

	// Travel Rule repo requires encryption cipher - skip for now
	var travelRuleRepo compliancerepository.TravelRuleRepository

	kycDocumentRepo := infrastructurerepository.NewKYCDocumentRepository(s.gormDB)

	// AML Rule repo requires GORM - skip for now
	var amlRuleRepo *compliancerepository.AMLRuleRepository

	// Initialize TRM Labs client (for AML screening)
	var trmClient complianceservice.TRMLabsClient
	if s.config.TRMLabs.APIKey != "" {
		trmClient = trmlabs.NewClient(s.config.TRMLabs.APIKey, s.config.TRMLabs.BaseURL)
	} else {
		trmClient = trmlabs.NewMockClient() // Use mock for development
		logger.Warn("TRM Labs API key not configured, using mock client")
	}

	// Initialize storage service
	var storageService storage.StorageService
	if s.config.AWS.S3Bucket != "" {
		s3Cfg := storage.S3Config{
			Region:      s.config.AWS.Region,
			KYCBucket:   s.config.AWS.S3Bucket,
			AuditBucket: s.config.AWS.S3Bucket, // Using same bucket for now
			Encryption:  "AES256",
		}
		ctx := context.Background()
		var err error
		storageService, err = storage.NewS3Storage(ctx, s3Cfg)
		if err != nil {
			logger.Error("Failed to initialize S3 storage", err)
			storageService = storage.NewMockStorage() // Fallback to mock
		}
	} else {
		storageService = storage.NewMockStorage() // Use mock for development
		logger.Warn("S3 not configured, using mock storage")
	}

	// Initialize services
	merchantService := merchantservice.NewMerchantService(merchantRepo, s.gormDB)

	// Initialize compliance services
	ruleEngine := complianceservice.NewRuleEngine(amlRuleRepo, newPaymentRepo, logger.GetLogger())

	var redisClient *redis.Client
	if rc, ok := s.cache.(*cache.RedisCache); ok {
		redisClient = rc.GetClient()
	}

	amlService := complianceservice.NewAMLService(
		trmClient,
		auditRepo,
		newPaymentRepo,
		ruleEngine,
		redisClient,
		logger.GetLogger(),
	)

	complianceService := complianceservice.NewComplianceService(
		amlService,
		merchantRepo,
		travelRuleRepo,
		kycDocumentRepo,
		newPaymentRepo,
		auditRepo,
		logger.GetLogger(),
	)

	payoutService := payoutservice.NewPayoutService(
		*payoutRepo,
		s.gormDB,
	)

	// Health check handler (no auth required)
	healthHandler := handler.NewHealthHandler(
		s.gormDB,
		s.cache,
		s.solanaClient,
		s.solanaWallet,
		"admin-v1.0.0", // Version string
	)
	router.GET("/health", healthHandler.Health)

	// Admin API v1 routes
	adminV1 := router.Group("/api/admin/v1")
	{
		// Public endpoints (no authentication)
		auth := adminV1.Group("/auth")
		{
			// Admin login endpoint
			auth.POST("/login", s.handleAdminLogin)
			// Note: In production, you'd want proper admin user management
			// For MVP, this can be a simple credentials check against environment variables
		}

		// Protected endpoints (require JWT authentication)
		protected := adminV1.Group("")
		protected.Use(middleware.JWTAuth(s.jwtManager))
		{
			// Initialize admin handler
			// Create wallet balance getter adapter (defined in server.go)
			walletBalanceGetter := &walletBalanceGetterAdapter{wallet: s.solanaWallet}

			adminHandler := handler.NewAdminHandler(
				merchantService,
				payoutService,
				complianceService,
				merchantRepo,
				payoutRepo,
				newPaymentRepo,
				balanceRepo,
				travelRuleRepo,
				kycDocumentRepo,
				walletBalanceGetter,
			)

			// Create storage adapter for KYC handler
			storageAdapter := &kycStorageAdapter{storage: storageService}
			kycHandler := merchanthandler.NewKYCHandler(storageAdapter, kycDocumentRepo)

			// Merchant management routes
			merchants := protected.Group("/merchants")
			{
				merchants.GET("", adminHandler.ListMerchants)   // List all merchants
				merchants.GET("/:id", adminHandler.GetMerchant) // Get merchant details
			}

			// KYC management routes
			kyc := protected.Group("/kyc")
			{
				kyc.POST("/:id/approve", adminHandler.ApproveKYC)       // Approve KYC
				kyc.POST("/:id/reject", adminHandler.RejectKYC)         // Reject KYC
				kyc.DELETE("/documents/:id", kycHandler.DeleteDocument) // Delete KYC document
			}

			// Payout management routes
			payouts := protected.Group("/payouts")
			{
				payouts.GET("", adminHandler.ListPayouts)                  // List all payouts
				payouts.GET("/:id", adminHandler.GetPayout)                // Get payout details
				payouts.POST("/:id/approve", adminHandler.ApprovePayout)   // Approve payout
				payouts.POST("/:id/reject", adminHandler.RejectPayout)     // Reject payout
				payouts.POST("/:id/complete", adminHandler.CompletePayout) // Mark as completed
			}

			// System monitoring routes
			system := protected.Group("/system")
			{
				system.GET("/stats", adminHandler.GetStats)            // System-wide statistics
				system.GET("/stats/daily", adminHandler.GetDailyStats) // Daily statistics
			}

			// Compliance routes
			compliance := protected.Group("/compliance")
			{
				compliance.GET("/travel-rule", adminHandler.GetTravelRuleData)             // Travel rule data
				compliance.GET("/metrics/:merchant_id", adminHandler.GetComplianceMetrics) // Compliance metrics
			}
		}
	}
}

// handleAdminLogin handles admin login and JWT generation
func (s *AdminServer) handleAdminLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	// Validate admin credentials
	// For MVP: Check against environment variables
	// In production: Check against admin users table in database
	adminEmail := s.config.Admin.Email
	adminPassword := s.config.Admin.Password

	if req.Email != adminEmail || req.Password != adminPassword {
		logger.Warn("Failed admin login attempt", logger.Fields{
			"email": req.Email,
		})
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid email or password",
			},
		})
		return
	}

	// Generate JWT token
	adminID := "admin-1" // For MVP, use a fixed admin ID
	role := "super_admin"

	token, err := s.jwtManager.GenerateToken(adminID, req.Email, role)
	if err != nil {
		logger.Error("Failed to generate JWT token", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "TOKEN_GENERATION_FAILED",
				"message": "Failed to generate authentication token",
			},
		})
		return
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.ExpirationHours) * time.Hour)

	// Log successful login
	logger.Info("Admin logged in successfully", logger.Fields{
		"admin_email": req.Email,
		"admin_id":    adminID,
	})

	// Return token
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"token":      token,
			"expires_at": expiresAt,
			"admin": gin.H{
				"id":    adminID,
				"email": req.Email,
				"role":  role,
			},
		},
	})
}

// Start starts the HTTP server
func (s *AdminServer) Start() error {
	address := fmt.Sprintf("%s:%d", s.config.Admin.Host, s.config.Admin.Port)

	s.httpServer = &http.Server{
		Addr:           address,
		Handler:        s.router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	logger.Info("Admin server starting", logger.Fields{
		"address": address,
		"host":    s.config.Admin.Host,
		"port":    s.config.Admin.Port,
	})

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Admin server failed to start", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *AdminServer) Shutdown(ctx context.Context) error {
	logger.Info("Admin server shutting down...")

	if s.httpServer == nil {
		return nil
	}

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("admin server shutdown error: %w", err)
	}

	logger.Info("Admin server stopped successfully")
	return nil
}

// GetRouter returns the Gin router (useful for testing)
func (s *AdminServer) GetRouter() *gin.Engine {
	return s.router
}
