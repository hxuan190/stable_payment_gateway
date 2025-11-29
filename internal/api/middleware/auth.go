package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	merchantDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/sirupsen/logrus"
)

// Context keys for storing merchant information
const (
	MerchantContextKey = "merchant"
	MerchantIDKey      = "merchant_id"
)

// MerchantRepository defines the interface for merchant data access
// This allows for easier testing and mocking
type MerchantRepository interface {
	GetByAPIKey(apiKey string) (*merchantDomain.Merchant, error)
}

// APIKeyAuthConfig holds configuration for API key authentication
type APIKeyAuthConfig struct {
	MerchantRepo MerchantRepository
	Cache        cache.Cache
	CacheTTL     time.Duration // How long to cache merchant data
}

// APIKeyAuth returns a Gin middleware that validates API key authentication
func APIKeyAuth(config APIKeyAuthConfig) gin.HandlerFunc {
	// Set default cache TTL if not provided
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Extract API key from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"ip":   c.ClientIP(),
				"path": c.Request.URL.Path,
			}).Warn("Missing Authorization header")

			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Missing Authorization header",
				},
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		apiKey := extractBearerToken(authHeader)
		if apiKey == "" {
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"ip":   c.ClientIP(),
				"path": c.Request.URL.Path,
			}).Warn("Invalid Authorization header format")

			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid Authorization header format. Expected: Bearer <api_key>",
				},
			})
			c.Abort()
			return
		}

		// Try to get merchant from cache first
		merchant, err := getMerchantFromCacheOrDB(ctx, apiKey, config)
		if err != nil {
			if err == sql.ErrNoRows || strings.Contains(err.Error(), "not found") {
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"ip":      c.ClientIP(),
					"path":    c.Request.URL.Path,
					"api_key": maskAPIKey(apiKey),
				}).Warn("Invalid API key")

				c.JSON(401, gin.H{
					"error": gin.H{
						"code":    "INVALID_API_KEY",
						"message": "Invalid API key",
					},
				})
				c.Abort()
				return
			}

			// Database error
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"error":   err.Error(),
				"ip":      c.ClientIP(),
				"path":    c.Request.URL.Path,
				"api_key": maskAPIKey(apiKey),
			}).Error("Failed to authenticate merchant")

			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to authenticate request",
				},
			})
			c.Abort()
			return
		}

		// Validate merchant can accept payments
		if !merchant.CanAcceptPayments() {
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"merchant_id": merchant.ID,
				"kyc_status":  merchant.KYCStatus,
				"status":      merchant.Status,
				"has_api_key": merchant.APIKey.Valid,
				"ip":          c.ClientIP(),
				"path":        c.Request.URL.Path,
			}).Warn("Merchant cannot accept payments")

			// Provide specific error message based on status
			var message string
			switch {
			case merchant.KYCStatus != merchantDomain.KYCStatusApproved:
				message = fmt.Sprintf("Merchant KYC not approved (status: %s)", merchant.KYCStatus)
			case merchant.Status != merchantDomain.MerchantStatusActive:
				message = fmt.Sprintf("Merchant account not active (status: %s)", merchant.Status)
			case !merchant.APIKey.Valid:
				message = "API key not configured"
			default:
				message = "Merchant cannot accept payments"
			}

			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "MERCHANT_NOT_ALLOWED",
					"message": message,
				},
			})
			c.Abort()
			return
		}

		// Set merchant in context for downstream handlers
		c.Set(MerchantContextKey, merchant)
		c.Set(MerchantIDKey, merchant.ID)

		// Add merchant ID to request context for logging
		ctx = context.WithValue(ctx, "merchant_id", merchant.ID)
		c.Request = c.Request.WithContext(ctx)

		logger.WithContext(ctx).WithFields(logrus.Fields{
			"merchant_id":   merchant.ID,
			"business_name": merchant.BusinessName,
			"path":          c.Request.URL.Path,
		}).Debug("Merchant authenticated successfully")

		c.Next()
	}
}

// getMerchantFromCacheOrDB tries to get merchant from cache first, falls back to database
func getMerchantFromCacheOrDB(ctx context.Context, apiKey string, config APIKeyAuthConfig) (*merchantDomain.Merchant, error) {
	cacheKey := fmt.Sprintf("merchant:apikey:%s", apiKey)

	// Try cache first if available
	if config.Cache != nil {
		cachedData, err := config.Cache.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Cache hit - unmarshal merchant data
			var merchant merchantDomain.Merchant
			if err := json.Unmarshal([]byte(cachedData), &merchant); err == nil {
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"merchant_id": merchant.ID,
					"cache_key":   cacheKey,
				}).Debug("Merchant loaded from cache")
				return &merchant, nil
			}
			// If unmarshal fails, continue to database
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"error":     err.Error(),
				"cache_key": cacheKey,
			}).Warn("Failed to unmarshal cached merchant data")
		}
	}

	// Cache miss or no cache - fetch from database
	merchant, err := config.MerchantRepo.GetByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	// Store in cache if available
	if config.Cache != nil && merchant != nil {
		merchantJSON, err := json.Marshal(merchant)
		if err == nil {
			if err := config.Cache.Set(ctx, cacheKey, string(merchantJSON), config.CacheTTL); err != nil {
				// Log error but don't fail the request
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"error":       err.Error(),
					"merchant_id": merchant.ID,
					"cache_key":   cacheKey,
				}).Warn("Failed to cache merchant data")
			} else {
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"merchant_id": merchant.ID,
					"cache_key":   cacheKey,
					"ttl":         config.CacheTTL,
				}).Debug("Merchant cached successfully")
			}
		}
	}

	return merchant, nil
}

// extractBearerToken extracts the token from "Bearer <token>" format
func extractBearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// maskAPIKey masks the API key for logging (shows first 8 and last 4 characters)
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 12 {
		return "***"
	}
	return apiKey[:8] + "..." + apiKey[len(apiKey)-4:]
}

// GetMerchantFromContext retrieves the merchant from the Gin context
// This is a helper function for handlers to get the authenticated merchant
func GetMerchantFromContext(c *gin.Context) (*merchantDomain.Merchant, error) {
	merchantValue, exists := c.Get(MerchantContextKey)
	if !exists {
		return nil, fmt.Errorf("merchant not found in context")
	}

	merchant, ok := merchantValue.(*merchantDomain.Merchant)
	if !ok {
		return nil, fmt.Errorf("invalid merchant type in context")
	}

	return merchant, nil
}

// GetMerchantIDFromContext retrieves the merchant ID from the Gin context
func GetMerchantIDFromContext(c *gin.Context) (string, error) {
	merchantIDValue, exists := c.Get(MerchantIDKey)
	if !exists {
		return "", fmt.Errorf("merchant ID not found in context")
	}

	merchantID, ok := merchantIDValue.(string)
	if !ok {
		return "", fmt.Errorf("invalid merchant ID type in context")
	}

	return merchantID, nil
}
