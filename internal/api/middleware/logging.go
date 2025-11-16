package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// RequestLogger returns a Gin middleware for logging HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate correlation ID for request tracing
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Generate request ID
		requestID := uuid.New().String()

		// Add IDs to context
		ctx := context.WithValue(c.Request.Context(), logger.CorrelationIDKey, correlationID)
		ctx = context.WithValue(ctx, logger.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set response headers
		c.Header("X-Correlation-ID", correlationID)
		c.Header("X-Request-ID", requestID)

		// Start timer
		startTime := time.Now()

		// Log request
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"query":          c.Request.URL.RawQuery,
			"ip":             c.ClientIP(),
			"user_agent":     c.Request.UserAgent(),
			"correlation_id": correlationID,
			"request_id":     requestID,
		}).Info("Incoming request")

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Log response
		statusCode := c.Writer.Status()
		logLevel := getLogLevel(statusCode)

		fields := logrus.Fields{
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"status":         statusCode,
			"latency_ms":     latency.Milliseconds(),
			"ip":             c.ClientIP(),
			"correlation_id": correlationID,
			"request_id":     requestID,
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		logEntry := logger.WithContext(ctx).WithFields(fields)

		switch logLevel {
		case "error":
			logEntry.Error("Request completed with error")
		case "warn":
			logEntry.Warn("Request completed with warning")
		default:
			logEntry.Info("Request completed")
		}
	}
}

// getLogLevel determines log level based on HTTP status code
func getLogLevel(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "error"
	case statusCode >= 400:
		return "warn"
	default:
		return "info"
	}
}

// Recovery returns a Gin middleware for panic recovery
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx := c.Request.Context()
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"error": err,
					"path":  c.Request.URL.Path,
				}).Error("Panic recovered")

				c.JSON(500, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
