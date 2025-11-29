package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/api/handler"
	"github.com/hxuan190/stable_payment_gateway/internal/api/middleware"
	"github.com/hxuan190/stable_payment_gateway/internal/api/websocket"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	auditrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/audit/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/solana"
	compliancehandler "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/handler"
	compliancerepository "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/repository"
	complianceservice "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
	infrastructurerepository "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/repository"
	infrastructureservice "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/service"
	ledgerrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/repository"
	merchanthandler "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/handler"
	merchantrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/repository"
	merchantservice "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/service"
	paymenthttp "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/http"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/legacy"
	paymentrepo "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/repository"
	paymentservice "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
	payouthandler "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/handler"
	payoutrepository "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/repository"
	payoutservice "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/service"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/jwt"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/storage"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
)

// Server represents the HTTP server
type Server struct {
	config       *config.Config
	router       *gin.Engine
	httpServer   *http.Server
	db           *gorm.DB
	cache        cache.Cache
	solanaClient *solana.Client
	solanaWallet *solana.Wallet
}

// ServerConfig holds dependencies for the server
type ServerConfig struct {
	Config       *config.Config
	DB           *gorm.DB
	Cache        cache.Cache
	SolanaClient *solana.Client
	SolanaWallet *solana.Wallet
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *ServerConfig) *Server {
	// Set Gin mode based on environment
	if cfg.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	server := &Server{
		config:       cfg.Config,
		db:           cfg.DB,
		cache:        cfg.Cache,
		solanaClient: cfg.SolanaClient,
		solanaWallet: cfg.SolanaWallet,
	}

	// Initialize router
	server.setupRouter()

	return server
}

// setupRouter configures the Gin router with all middleware and routes
func (s *Server) setupRouter() {
	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())             // Recover from panics
	router.Use(middleware.RequestLogger()) // Log all requests
	router.Use(s.corsMiddleware())         // CORS configuration
	router.Use(s.rateLimitMiddleware())    // Rate limiting

	// Set up routes
	s.setupRoutes(router)

	s.router = router
}

// corsMiddleware configures CORS settings
func (s *Server) corsMiddleware() gin.HandlerFunc {
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
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"Retry-After",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(config)
}

// rateLimitMiddleware configures rate limiting
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	// Get Redis client from cache
	redisCache, ok := s.cache.(*cache.RedisCache)
	if !ok {
		logger.Warn("Rate limiting disabled: Redis cache not available")
		// Return a no-op middleware if Redis is not available
		return func(c *gin.Context) {
			c.Next()
		}
	}

	redisClient := redisCache.GetClient()
	rateLimiter := middleware.NewRedisRateLimiter(redisClient)

	return middleware.RateLimit(middleware.RateLimitConfig{
		Limiter:     rateLimiter,
		APIKeyLimit: s.config.Security.RateLimitPerMinute,
		IPLimit:     1000, // 1000 requests/minute per IP (global)
		Window:      1 * time.Minute,
	})
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes(router *gin.Engine) {
	// Health check handler
	healthHandler := handler.NewHealthHandler(
		s.db,
		s.cache,
		s.solanaClient,
		s.solanaWallet,
		s.config.Version,
	)

	// Initialize repositories
	merchantRepo := merchantrepository.NewMerchantRepository(s.db)
	payoutRepo := payoutrepository.NewPayoutRepository(s.db)
	balanceRepo := ledgerrepository.NewBalanceRepository(s.db)
	auditRepo := auditrepository.NewAuditRepository(s.db)

	// Travel Rule repo requires encryption cipher - skip for now as it's optional
	var travelRuleRepo compliancerepository.TravelRuleRepository

	kycDocumentRepo := infrastructurerepository.NewKYCDocumentRepository(s.db)
	// AML Rule repo requires GORM - skip for now
	var amlRuleRepo *compliancerepository.AMLRuleRepository

	// Initialize NEW Payment Module (Hexagonal Architecture) - needed early for AML service
	paymentRepo := paymentrepo.NewPostgresPaymentRepository(s.db)

	// Initialize services
	exchangeRateService := infrastructureservice.NewExchangeRateService(
		s.config.ExchangeRate.PrimaryAPI,
		s.config.ExchangeRate.SecondaryAPI,
		time.Duration(s.config.ExchangeRate.CacheTTL)*time.Second,
		time.Duration(s.config.ExchangeRate.Timeout)*time.Second,
		s.cache,
		logger.GetLogger().Logger,
	)
	// Initialize merchant service for admin
	merchantService := merchantservice.NewMerchantService(merchantRepo, s.db)

	// Initialize compliance services (must be before payment service)
	trmClient := s.initTRMLabsClient()
	ruleEngine := complianceservice.NewRuleEngine(amlRuleRepo, paymentRepo, logger.GetLogger())

	var redisClient *redis.Client
	if redisCache, ok := s.cache.(*cache.RedisCache); ok {
		redisClient = redisCache.GetClient()
	}

	amlService := complianceservice.NewAMLService(
		trmClient,
		auditRepo,
		paymentRepo,
		ruleEngine,
		redisClient,
		logger.GetLogger(),
	)
	complianceService := complianceservice.NewComplianceService(
		amlService,
		merchantRepo,
		travelRuleRepo,
		kycDocumentRepo,
		paymentRepo,
		auditRepo,
		logger.GetLogger(),
	)
	// Skip merchant adapter for now - type mismatch issue
	var merchantAdapter *legacy.MerchantRepositoryAdapter
	exchangeRateAdapter := legacy.NewExchangeRateServiceAdapter(exchangeRateService)
	complianceAdapter := legacy.NewComplianceServiceAdapter(complianceService)
	amlAdapter := legacy.NewAMLServiceAdapter(amlService)

	paymentService := paymentservice.NewPaymentService(
		paymentRepo,
		merchantAdapter,
		exchangeRateAdapter,
		complianceAdapter,
		amlAdapter,
		paymentservice.PaymentServiceConfig{
			DefaultChain:    "solana",
			DefaultCurrency: "USDT",
			WalletAddress:   s.solanaWallet.GetAddress(),
			FeePercentage:   0.01,
			ExpiryMinutes:   30,
			RedisClient:     redisClient,
		},
		logger.GetLogger().Logger,
	)

	payoutService := payoutservice.NewPayoutService(
		*payoutRepo,
		s.db,
	)

	// Initialize handlers
	// Use storage base URL or construct from API config
	baseURL := s.config.Storage.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", s.config.API.Host, s.config.API.Port)
	}
	exchangeRateHTTPAdapter := legacy.NewExchangeRateHTTPAdapter(exchangeRateService)
	paymentHandler := paymenthttp.NewPaymentHandler(paymentService, complianceService, exchangeRateHTTPAdapter, baseURL)

	// Use module handlers
	payoutHandler := payouthandler.NewPayoutHandler(payoutService)

	// Create storage adapter for KYC handler
	var baseStorageService storage.StorageService
	if s.config.Storage.Bucket != "" && s.config.Storage.Region != "" {
		ctx := context.Background()
		s3Cfg := storage.S3Config{
			Region:      s.config.Storage.Region,
			KYCBucket:   s.config.Storage.Bucket,
			AuditBucket: s.config.Storage.Bucket,
			Encryption:  "AES256",
		}
		var err error
		baseStorageService, err = storage.NewS3Storage(ctx, s3Cfg)
		if err != nil {
			logger.Warn("Failed to initialize S3 storage, using mock", logger.Fields{"error": err.Error()})
			baseStorageService = storage.NewMockStorage()
		}
	} else {
		baseStorageService = storage.NewMockStorage()
	}
	kycStorageAdapter := &kycStorageAdapter{storage: baseStorageService}
	kycHandler := merchanthandler.NewKYCHandler(kycStorageAdapter, kycDocumentRepo)

	// Compliance module handlers
	amlRuleHandler := compliancehandler.NewAMLRuleHandler(amlRuleRepo)

	// Create wallet balance getter adapter
	walletBalanceGetter := &solanaWalletAdapter{wallet: s.solanaWallet}

	adminHandler := handler.NewAdminHandler(
		merchantService,
		payoutService,
		complianceService,
		merchantRepo,
		payoutRepo,
		paymentRepo,
		balanceRepo,
		travelRuleRepo,
		kycDocumentRepo,
		walletBalanceGetter,
	)

	// Public routes (no authentication required)
	router.GET("/health", healthHandler.Health)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public endpoints (no authentication required)
		v1.GET("/status", healthHandler.Status)

		// Public payment status endpoint (Payer Experience Layer)
		publicGroup := v1.Group("/public")
		{
			publicGroup.GET("/payments/:id/status", paymentHandler.GetPublicPaymentStatus)
		}

		// Payment routes (API key authentication required)
		paymentGroup := v1.Group("/payments")
		paymentGroup.Use(middleware.APIKeyAuth(middleware.APIKeyAuthConfig{
			MerchantRepo: merchantRepo,
			Cache:        s.cache,
			CacheTTL:     5 * time.Minute,
		}))
		{
			paymentGroup.POST("", paymentHandler.CreatePayment)
			paymentGroup.GET("/:id", paymentHandler.GetPayment)
			paymentGroup.GET("", paymentHandler.ListPayments)
		}

		// Merchant routes (API key authentication required)
		merchantGroup := v1.Group("/merchant")
		merchantGroup.Use(middleware.APIKeyAuth(middleware.APIKeyAuthConfig{
			MerchantRepo: merchantRepo,
			Cache:        s.cache,
			CacheTTL:     5 * time.Minute,
		}))
		{
			// TODO: merchantGroup.GET("/balance", merchantHandler.GetBalance)
			// TODO: merchantGroup.GET("/transactions", merchantHandler.GetTransactions)
			merchantGroup.POST("/payouts", payoutHandler.RequestPayout)
			merchantGroup.GET("/payouts", payoutHandler.ListPayouts)
			merchantGroup.GET("/payouts/:id", payoutHandler.GetPayout)
		}
	}

	// WebSocket routes (real-time payment status updates)
	// Initialize WebSocket handler
	redisCache, ok := s.cache.(*cache.RedisCache)
	if ok {
		wsHandler := websocket.NewPaymentWSHandler(redisCache.GetClient())
		router.GET("/ws/payments/:id", wsHandler.HandlePaymentStatus)
	} else {
		logger.Warn("WebSocket disabled: Redis cache not available")
	}

	// Merchant KYC routes (API key authentication required)
	kycGroup := v1.Group("/merchants/kyc")
	kycGroup.Use(middleware.APIKeyAuth(middleware.APIKeyAuthConfig{
		MerchantRepo: merchantRepo,
		Cache:        s.cache,
		CacheTTL:     5 * time.Minute,
	}))
	{
		kycGroup.POST("/documents", kycHandler.UploadDocument)
		kycGroup.GET("/documents", kycHandler.ListDocuments)
		kycGroup.DELETE("/documents/:id", kycHandler.DeleteDocument)
	}

	// Admin routes (JWT authentication required)
	jwtManager := jwt.NewManager(s.config.JWT.Secret, s.config.JWT.ExpirationHours)
	adminGroup := router.Group("/api/admin")
	adminGroup.Use(middleware.JWTAuth(jwtManager))
	{
		// KYC management
		adminGroup.POST("/merchants/:id/kyc/approve", adminHandler.ApproveKYC)
		adminGroup.POST("/merchants/:id/kyc/reject", adminHandler.RejectKYC)
		adminGroup.GET("/merchants", adminHandler.ListMerchants)
		adminGroup.GET("/merchants/:id", adminHandler.GetMerchant)

		// KYC Document management
		adminGroup.GET("/kyc/documents/pending", adminHandler.GetPendingKYCDocuments)
		adminGroup.POST("/kyc/documents/:id/approve", adminHandler.ApproveKYCDocument)
		adminGroup.POST("/kyc/documents/:id/reject", adminHandler.RejectKYCDocument)

		// Compliance management
		adminGroup.GET("/compliance/travel-rule", adminHandler.GetTravelRuleData)
		adminGroup.GET("/compliance/metrics/:merchant_id", adminHandler.GetComplianceMetrics)
		adminGroup.POST("/merchants/:id/upgrade-tier", adminHandler.UpgradeMerchantTier)

		// Payout management
		adminGroup.GET("/payouts", adminHandler.ListPayouts)
		adminGroup.GET("/payouts/:id", adminHandler.GetPayout)
		adminGroup.POST("/payouts/:id/approve", adminHandler.ApprovePayout)
		adminGroup.POST("/payouts/:id/reject", adminHandler.RejectPayout)
		adminGroup.POST("/payouts/:id/complete", adminHandler.CompletePayout)

		// System statistics
		adminGroup.GET("/stats", adminHandler.GetStats)
		adminGroup.GET("/stats/daily", adminHandler.GetDailyStats)

		// AML rule management
		amlRules := adminGroup.Group("/aml-rules")
		{
			amlRules.GET("", amlRuleHandler.ListRules)
			amlRules.GET("/:id", amlRuleHandler.GetRule)
			amlRules.GET("/category/:category", amlRuleHandler.GetRulesByCategory)
			amlRules.POST("", amlRuleHandler.CreateRule)
			amlRules.PUT("/:id", amlRuleHandler.UpdateRule)
			amlRules.DELETE("/:id", amlRuleHandler.DeleteRule)
			amlRules.PATCH("/:id/toggle", amlRuleHandler.ToggleRule)
		}
	}

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "The requested resource was not found",
			},
			"timestamp": time.Now().UTC(),
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.API.Host, s.config.API.Port)

	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.API.ReadTimeout,
		WriteTimeout:   s.config.API.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	logger.Info("Starting HTTP server", logger.Fields{
		"host": s.config.API.Host,
		"port": s.config.API.Port,
		"env":  s.config.Environment,
	})

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("Shutting down HTTP server...")

	if s.httpServer == nil {
		return nil
	}

	// Attempt graceful shutdown with context timeout
	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", err)
		return err
	}

	logger.Info("HTTP server stopped successfully")
	return nil
}

// GetRouter returns the Gin router (useful for testing)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// solanaWalletAdapter adapts Solana wallet to WalletBalanceGetter interface
type solanaWalletAdapter struct {
	wallet *solana.Wallet
}

func (a *solanaWalletAdapter) GetBalance() (decimal.Decimal, error) {
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

func (s *Server) initTRMLabsClient() complianceservice.TRMLabsClient {
	trmAPIKey := s.config.TRM.APIKey
	trmBaseURL := s.config.TRM.BaseURL

	if trmAPIKey != "" && trmBaseURL != "" {
		client := trmlabs.NewClient(trmAPIKey, trmBaseURL)
		logger.Info("TRM Labs client initialized (production mode)")
		return client
	}

	logger.Info("TRM Labs client initialized (mock mode)")
	return trmlabs.NewMockClient()
}
