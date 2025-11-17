package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db            *sql.DB
	cache         cache.Cache
	solanaClient  *solana.Client
	solanaWallet  *solana.Wallet
	version       string
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(
	db *sql.DB,
	cache cache.Cache,
	solanaClient *solana.Client,
	solanaWallet *solana.Wallet,
	version string,
) *HealthHandler {
	return &HealthHandler{
		db:           db,
		cache:        cache,
		solanaClient: solanaClient,
		solanaWallet: solanaWallet,
		version:      version,
	}
}

// Health returns a basic health check response
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, dto.HealthResponse{
		Status:  "ok",
		Message: "Payment Gateway API is running",
	})
}

// Status returns detailed system status
// GET /api/v1/status
func (h *HealthHandler) Status(c *gin.Context) {
	ctx := c.Request.Context()

	response := dto.StatusResponse{
		Status:  "healthy",
		Version: h.version,
		Services: dto.ServiceStatuses{
			Database:   h.checkDatabase(ctx),
			Redis:      h.checkRedis(ctx),
			Blockchain: h.checkBlockchain(ctx),
		},
	}

	// Add wallet status if Solana wallet is configured
	if h.solanaWallet != nil {
		walletStatus := h.checkWallet(ctx)
		response.Wallet = walletStatus
	}

	// Determine overall status
	if response.Services.Database.Status == "unhealthy" ||
		response.Services.Redis.Status == "unhealthy" {
		response.Status = "unhealthy"
	} else if response.Services.Database.Status == "degraded" ||
		response.Services.Redis.Status == "degraded" ||
		response.Services.Blockchain.Status == "degraded" {
		response.Status = "degraded"
	}

	// Return appropriate HTTP status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if response.Status == "degraded" {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	c.JSON(statusCode, dto.SuccessResponse(response))
}

// checkDatabase checks database connectivity and latency
func (h *HealthHandler) checkDatabase(ctx context.Context) dto.ServiceStatus {
	start := time.Now()

	// Try to ping database with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := h.db.PingContext(pingCtx)
	latency := time.Since(start)

	if err != nil {
		logger.Error("Database health check failed", err)
		return dto.ServiceStatus{
			Status:  "unhealthy",
			Message: "Database connection failed",
			Latency: latency.String(),
		}
	}

	// Check if latency is too high (> 100ms is concerning)
	if latency > 100*time.Millisecond {
		return dto.ServiceStatus{
			Status:  "degraded",
			Message: "Database latency is high",
			Latency: latency.String(),
		}
	}

	return dto.ServiceStatus{
		Status:  "healthy",
		Message: "Database connection is healthy",
		Latency: latency.String(),
	}
}

// checkRedis checks Redis connectivity and latency
func (h *HealthHandler) checkRedis(ctx context.Context) dto.ServiceStatus {
	start := time.Now()

	// Try to ping Redis with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := h.cache.Ping(pingCtx)
	latency := time.Since(start)

	if err != nil {
		logger.Error("Redis health check failed", err)
		return dto.ServiceStatus{
			Status:  "unhealthy",
			Message: "Redis connection failed",
			Latency: latency.String(),
		}
	}

	// Check if latency is too high (> 50ms is concerning)
	if latency > 50*time.Millisecond {
		return dto.ServiceStatus{
			Status:  "degraded",
			Message: "Redis latency is high",
			Latency: latency.String(),
		}
	}

	return dto.ServiceStatus{
		Status:  "healthy",
		Message: "Redis connection is healthy",
		Latency: latency.String(),
	}
}

// checkBlockchain checks Solana RPC connectivity
func (h *HealthHandler) checkBlockchain(ctx context.Context) dto.ServiceStatus {
	if h.solanaClient == nil {
		return dto.ServiceStatus{
			Status:  "unhealthy",
			Message: "Solana client not configured",
		}
	}

	start := time.Now()

	// Check if Solana RPC is healthy
	err := h.solanaClient.HealthCheck(ctx)
	latency := time.Since(start)

	if err != nil {
		logger.Error("Blockchain health check failed", err)
		return dto.ServiceStatus{
			Status:  "unhealthy",
			Message: "Solana RPC connection failed",
			Latency: latency.String(),
		}
	}

	// Check if latency is too high (> 500ms is concerning for blockchain RPC)
	if latency > 500*time.Millisecond {
		return dto.ServiceStatus{
			Status:  "degraded",
			Message: "Solana RPC latency is high",
			Latency: latency.String(),
		}
	}

	return dto.ServiceStatus{
		Status:  "healthy",
		Message: "Solana RPC connection is healthy",
		Latency: latency.String(),
	}
}

// checkWallet checks hot wallet status and balances
func (h *HealthHandler) checkWallet(ctx context.Context) *dto.WalletStatus {
	if h.solanaWallet == nil {
		return nil
	}

	walletStatus := &dto.WalletStatus{
		Address:     h.solanaWallet.GetAddress(),
		LastChecked: time.Now().UTC().Format(time.RFC3339),
	}

	// Get SOL balance
	solBalance, err := h.solanaWallet.GetSOLBalance(ctx)
	if err != nil {
		logger.Error("Failed to get SOL balance", err)
	} else {
		walletStatus.BalanceSOL = fmt.Sprintf("%.4f SOL", solBalance)
	}

	// Note: Token balances are fetched asynchronously and may not be critical for health checks
	// If token balance fetching fails, we log a warning but don't fail the health check
	// The actual token mint addresses should be configured via environment variables

	return walletStatus
}
