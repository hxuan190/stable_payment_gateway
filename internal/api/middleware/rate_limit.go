package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow checks if a request is allowed and returns the current state
	Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitState, error)
}

// RateLimitState holds the current state of rate limiting for a key
type RateLimitState struct {
	Allowed   bool  // Whether the request is allowed
	Limit     int   // Maximum allowed requests
	Remaining int   // Remaining requests in current window
	RetryIn   int64 // Seconds until retry is allowed (0 if allowed)
}

// RedisRateLimiter implements sliding window rate limiting using Redis
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
	}
}

// Allow checks if a request is allowed using sliding window algorithm
// This implementation uses Redis sorted sets with timestamps as scores
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitState, error) {
	now := time.Now()

	// Redis Lua script for atomic sliding window rate limiting
	// This ensures thread-safety and reduces round trips
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local windowStart = now - window

		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', windowStart)

		-- Count current requests in window
		local count = redis.call('ZCARD', key)

		-- Check if limit exceeded
		if count >= limit then
			-- Get the oldest timestamp in the window
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local retryIn = 0
			if #oldest > 0 then
				retryIn = math.ceil(tonumber(oldest[2]) + window - now)
			end
			return {0, limit, 0, retryIn} -- not allowed
		end

		-- Add current request
		redis.call('ZADD', key, now, now)

		-- Set expiration to cleanup old keys (window + 1 minute buffer)
		redis.call('EXPIRE', key, window + 60)

		-- Return: allowed, limit, remaining, retryIn
		return {1, limit, limit - count - 1, 0}
	`)

	// Execute Lua script
	result, err := script.Run(
		ctx,
		r.client,
		[]string{key},
		now.Unix(),
		int64(window.Seconds()),
		limit,
	).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	// Parse result
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 4 {
		return nil, fmt.Errorf("unexpected rate limit script result")
	}

	allowed := resultSlice[0].(int64) == 1
	limitVal := int(resultSlice[1].(int64))
	remaining := int(resultSlice[2].(int64))
	retryIn := resultSlice[3].(int64)

	return &RateLimitState{
		Allowed:   allowed,
		Limit:     limitVal,
		Remaining: remaining,
		RetryIn:   retryIn,
	}, nil
}

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	Limiter           RateLimiter
	APIKeyLimit       int           // Requests per minute per API key
	IPLimit           int           // Requests per minute per IP
	Window            time.Duration // Time window (default: 1 minute)
	SkipSuccessHeader bool          // Skip rate limit headers on successful requests
}

// RateLimit returns a Gin middleware that implements rate limiting
// It enforces both per-API-key and per-IP limits
func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	// Set defaults
	if config.APIKeyLimit == 0 {
		config.APIKeyLimit = 100 // Default: 100 requests/minute per API key
	}
	if config.IPLimit == 0 {
		config.IPLimit = 1000 // Default: 1000 requests/minute per IP
	}
	if config.Window == 0 {
		config.Window = 1 * time.Minute
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Check IP-based rate limit first (global limit)
		clientIP := c.ClientIP()
		ipKey := fmt.Sprintf("ratelimit:ip:%s", clientIP)

		ipState, err := config.Limiter.Allow(ctx, ipKey, config.IPLimit, config.Window)
		if err != nil {
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"error": err.Error(),
				"ip":    clientIP,
				"key":   ipKey,
			}).Error("Rate limit check failed")

			// On error, allow the request but log the issue
			c.Next()
			return
		}

		if !ipState.Allowed {
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"ip":        clientIP,
				"limit":     ipState.Limit,
				"retry_in":  ipState.RetryIn,
			}).Warn("IP rate limit exceeded")

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(ipState.Limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix()+ipState.RetryIn, 10))
			c.Header("Retry-After", strconv.FormatInt(ipState.RetryIn, 10))

			c.JSON(429, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": fmt.Sprintf("Too many requests. Please retry after %d seconds", ipState.RetryIn),
				},
			})
			c.Abort()
			return
		}

		// Check API key rate limit if authenticated
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			apiKey := extractBearerTokenForRateLimit(authHeader)
			if apiKey != "" {
				apiKeyLimitKey := fmt.Sprintf("ratelimit:apikey:%s", apiKey)

				apiKeyState, err := config.Limiter.Allow(ctx, apiKeyLimitKey, config.APIKeyLimit, config.Window)
				if err != nil {
					logger.WithContext(ctx).WithFields(logrus.Fields{
						"error":   err.Error(),
						"api_key": maskAPIKeyForRateLimit(apiKey),
						"key":     apiKeyLimitKey,
					}).Error("API key rate limit check failed")

					// On error, allow the request but log the issue
					c.Next()
					return
				}

				if !apiKeyState.Allowed {
					logger.WithContext(ctx).WithFields(logrus.Fields{
						"api_key":   maskAPIKeyForRateLimit(apiKey),
						"limit":     apiKeyState.Limit,
						"retry_in":  apiKeyState.RetryIn,
					}).Warn("API key rate limit exceeded")

					// Set rate limit headers
					c.Header("X-RateLimit-Limit", strconv.Itoa(apiKeyState.Limit))
					c.Header("X-RateLimit-Remaining", "0")
					c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix()+apiKeyState.RetryIn, 10))
					c.Header("Retry-After", strconv.FormatInt(apiKeyState.RetryIn, 10))

					c.JSON(429, gin.H{
						"error": gin.H{
							"code":    "API_KEY_RATE_LIMIT_EXCEEDED",
							"message": fmt.Sprintf("API key rate limit exceeded. Please retry after %d seconds", apiKeyState.RetryIn),
						},
					})
					c.Abort()
					return
				}

				// Set rate limit headers for API key (more restrictive than IP)
				if !config.SkipSuccessHeader {
					c.Header("X-RateLimit-Limit", strconv.Itoa(apiKeyState.Limit))
					c.Header("X-RateLimit-Remaining", strconv.Itoa(apiKeyState.Remaining))
					c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))
				}
			}
		} else {
			// No API key - use IP limit headers
			if !config.SkipSuccessHeader {
				c.Header("X-RateLimit-Limit", strconv.Itoa(ipState.Limit))
				c.Header("X-RateLimit-Remaining", strconv.Itoa(ipState.Remaining))
				c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))
			}
		}

		c.Next()
	}
}

// extractBearerTokenForRateLimit extracts the token from "Bearer <token>" format
// This is a simplified version that doesn't validate format strictly
func extractBearerTokenForRateLimit(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// maskAPIKeyForRateLimit masks the API key for logging
func maskAPIKeyForRateLimit(apiKey string) string {
	if len(apiKey) <= 12 {
		return "***"
	}
	return apiKey[:8] + "..." + apiKey[len(apiKey)-4:]
}
