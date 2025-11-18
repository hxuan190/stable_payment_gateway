package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/hxuan190/stable_payment_gateway/internal/api/handler"
	"github.com/hxuan190/stable_payment_gateway/internal/api/middleware"
	"github.com/hxuan190/stable_payment_gateway/internal/api/websocket"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

// Server represents the HTTP server
type Server struct {
	config       *config.Config
	router       *gin.Engine
	httpServer   *http.Server
	db           *sql.DB
	cache        cache.Cache
	solanaClient *solana.Client
	solanaWallet *solana.Wallet
}

// ServerConfig holds dependencies for the server
type ServerConfig struct {
	Config       *config.Config
	DB           *sql.DB
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
	merchantRepo := s.initMerchantRepository()
	paymentRepo := s.initPaymentRepository()
	payoutRepo := s.initPayoutRepository()
	balanceRepo := s.initBalanceRepository()
	ledgerRepo := s.initLedgerRepository()
	auditRepo := s.initAuditLogRepository()
	travelRuleRepo := s.initTravelRuleRepository()
	kycDocumentRepo := s.initKYCDocumentRepository()

	// Initialize services
	exchangeRateService := s.initExchangeRateService()
	ledgerService := s.initLedgerService(ledgerRepo)

	// Initialize merchant service for admin
	merchantService := s.initMerchantService(merchantRepo, balanceRepo)

	// Initialize compliance services (must be before payment service)
	amlService := s.initAMLService(auditRepo, paymentRepo)
	complianceService := s.initComplianceService(amlService, merchantRepo, travelRuleRepo, kycDocumentRepo, paymentRepo, auditRepo)

	// Initialize payment service with compliance
	paymentService := s.initPaymentService(paymentRepo, merchantRepo, exchangeRateService, complianceService)
	payoutService := s.initPayoutService(payoutRepo, merchantRepo, balanceRepo, ledgerService)

	// Initialize handlers
	// Use storage base URL or construct from API config
	baseURL := s.config.Storage.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", s.config.API.Host, s.config.API.Port)
	}
	paymentHandler := handler.NewPaymentHandler(paymentService, complianceService, exchangeRateService, baseURL)
	payoutHandler := handler.NewPayoutHandler(payoutService)
	kycHandler := handler.NewKYCHandler(kycDocumentRepo, merchantRepo, complianceService, s.config.Storage)
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
		s.solanaWallet,
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
	adminGroup := router.Group("/api/admin")
	adminGroup.Use(middleware.JWTAuth(s.config.JWT.Secret))
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

// Repository initialization helpers

func (s *Server) initMerchantRepository() *repository.MerchantRepository {
	return repository.NewMerchantRepository(s.db, logger.GetLogger())
}

func (s *Server) initPaymentRepository() *repository.PaymentRepository {
	return repository.NewPaymentRepository(s.db, logger.GetLogger())
}

func (s *Server) initPayoutRepository() *repository.PayoutRepository {
	return repository.NewPayoutRepository(s.db, logger.GetLogger())
}

func (s *Server) initBalanceRepository() *repository.BalanceRepository {
	return repository.NewBalanceRepository(s.db, logger.GetLogger())
}

func (s *Server) initLedgerRepository() *repository.LedgerRepository {
	return repository.NewLedgerRepository(s.db, logger.GetLogger())
}

// Service initialization helpers

func (s *Server) initExchangeRateService() *service.ExchangeRateService {
	return service.NewExchangeRateService(
		s.cache,
		logger.GetLogger(),
		service.ExchangeRateConfig{
			CacheTTL: 5 * time.Minute,
		},
	)
}

func (s *Server) initPaymentService(
	paymentRepo *repository.PaymentRepository,
	merchantRepo *repository.MerchantRepository,
	exchangeRateService *service.ExchangeRateService,
	complianceService service.ComplianceService,
) *service.PaymentService {
	// Get Redis client if available
	var redisClient *redis.Client
	if redisCache, ok := s.cache.(*cache.RedisCache); ok {
		redisClient = redisCache.GetClient()
	}

	return service.NewPaymentService(
		paymentRepo,
		merchantRepo,
		exchangeRateService,
		complianceService, // Pass compliance service for pre-payment validation
		service.PaymentServiceConfig{
			DefaultChain:    model.ChainSolana,
			DefaultCurrency: "USDT",
			WalletAddress:   s.solanaWallet.GetAddress(),
			FeePercentage:   0.01, // 1% fee
			ExpiryMinutes:   30,   // 30 minutes expiry
			RedisClient:     redisClient,
		},
		logger.GetLogger(),
	)
}

func (s *Server) initLedgerService(
	ledgerRepo *repository.LedgerRepository,
) *service.LedgerService {
	return service.NewLedgerService(ledgerRepo, logger.GetLogger())
}

func (s *Server) initPayoutService(
	payoutRepo *repository.PayoutRepository,
	merchantRepo *repository.MerchantRepository,
	balanceRepo *repository.BalanceRepository,
	ledgerService *service.LedgerService,
) *service.PayoutService {
	return service.NewPayoutService(
		payoutRepo,
		merchantRepo,
		balanceRepo,
		ledgerService,
		s.db,
	)
}

func (s *Server) initMerchantService(
	merchantRepo *repository.MerchantRepository,
	balanceRepo *repository.BalanceRepository,
) *service.MerchantService {
	return service.NewMerchantService(
		merchantRepo,
		balanceRepo,
		s.db,
	)
}

func (s *Server) initAuditLogRepository() *repository.AuditLogRepository {
	return repository.NewAuditLogRepository(s.db, logger.GetLogger())
}

func (s *Server) initTravelRuleRepository() *repository.TravelRuleRepository {
	return repository.NewTravelRuleRepository(s.db, logger.GetLogger())
}

func (s *Server) initKYCDocumentRepository() *repository.KYCDocumentRepository {
	return repository.NewKYCDocumentRepository(s.db, logger.GetLogger())
}

func (s *Server) initAMLService(
	auditRepo *repository.AuditLogRepository,
	paymentRepo *repository.PaymentRepository,
) service.AMLService {
	// Initialize TRM Labs client (with mock for development)
	trmClient := s.initTRMLabsClient()

	return service.NewAMLService(
		trmClient,
		auditRepo,
		paymentRepo,
		logger.GetLogger(),
	)
}

func (s *Server) initTRMLabsClient() service.TRMLabsClient {
	// Check if we have TRM Labs API key configured
	trmAPIKey := s.config.TRM.APIKey
	trmBaseURL := s.config.TRM.BaseURL

	if trmAPIKey != "" && trmBaseURL != "" {
		// Use real TRM Labs client
		client, err := trmlabs.NewClient(trmAPIKey, trmBaseURL)
		if err != nil {
			logger.Warn("Failed to initialize TRM Labs client, using mock", logger.Fields{
				"error": err.Error(),
			})
			return trmlabs.NewMockClient()
		}
		logger.Info("TRM Labs client initialized (production mode)")
		return client
	}

	// Use mock client for development/testing
	logger.Info("TRM Labs client initialized (mock mode)")
	return trmlabs.NewMockClient()
}

func (s *Server) initComplianceService(
	amlService service.AMLService,
	merchantRepo *repository.MerchantRepository,
	travelRuleRepo *repository.TravelRuleRepository,
	kycDocumentRepo *repository.KYCDocumentRepository,
	paymentRepo *repository.PaymentRepository,
	auditRepo *repository.AuditLogRepository,
) service.ComplianceService {
	return service.NewComplianceService(
		amlService,
		merchantRepo,
		travelRuleRepo,
		kycDocumentRepo,
		paymentRepo,
		auditRepo,
		logger.GetLogger(),
	)
}
