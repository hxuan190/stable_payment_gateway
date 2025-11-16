package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
}

// Fields type for structured logging
type Fields map[string]interface{}

// ContextKey type for context values
type contextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey contextKey = "correlation_id"
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "request_id"
)

var (
	// defaultLogger is the global logger instance
	defaultLogger *Logger
)

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, text
	Output     io.Writer
	ReportCaller bool
}

// New creates a new logger instance
func New(cfg Config) *Logger {
	log := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set output format
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "caller",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	}

	// Set output
	if cfg.Output != nil {
		log.SetOutput(cfg.Output)
	} else {
		log.SetOutput(os.Stdout)
	}

	// Set caller reporting
	log.SetReportCaller(cfg.ReportCaller)

	return &Logger{Logger: log}
}

// Init initializes the default logger
func Init(cfg Config) {
	defaultLogger = New(cfg)
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		// Initialize with default config if not set
		Init(Config{
			Level:  "info",
			Format: "json",
		})
	}
	return defaultLogger
}

// WithFields creates a new logger entry with fields
func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

// WithContext creates a new logger entry with context values
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithContext(ctx)

	// Add correlation ID if present
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		entry = entry.WithField("correlation_id", correlationID)
	}

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	return entry
}

// WithError creates a new logger entry with error
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Helper methods for structured logging

// Debug logs a debug message
func Debug(msg string, fields ...Fields) {
	entry := GetLogger().Logger
	if len(fields) > 0 {
		entry = GetLogger().WithFields(fields[0]).Logger
	}
	entry.Debug(msg)
}

// Info logs an info message
func Info(msg string, fields ...Fields) {
	entry := GetLogger().Logger
	if len(fields) > 0 {
		entry = GetLogger().WithFields(fields[0]).Logger
	}
	entry.Info(msg)
}

// Warn logs a warning message
func Warn(msg string, fields ...Fields) {
	entry := GetLogger().Logger
	if len(fields) > 0 {
		entry = GetLogger().WithFields(fields[0]).Logger
	}
	entry.Warn(msg)
}

// Error logs an error message
func Error(msg string, err error, fields ...Fields) {
	entry := GetLogger().WithError(err)
	if len(fields) > 0 {
		entry = entry.WithFields(logrus.Fields(fields[0]))
	}
	entry.Error(msg)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, err error, fields ...Fields) {
	entry := GetLogger().WithError(err)
	if len(fields) > 0 {
		entry = entry.WithFields(logrus.Fields(fields[0]))
	}
	entry.Fatal(msg)
}

// WithContext logs with context
func WithContext(ctx context.Context) *logrus.Entry {
	return GetLogger().WithContext(ctx)
}

// WithFields logs with fields
func WithFields(fields Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// Security-related logging helpers

// LogAuthFailure logs authentication failures
func LogAuthFailure(ctx context.Context, reason string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event"] = "auth_failure"
	fields["reason"] = reason
	GetLogger().WithContext(ctx).WithFields(logrus.Fields(fields)).Warn("Authentication failed")
}

// LogAuthSuccess logs successful authentication
func LogAuthSuccess(ctx context.Context, userID string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event"] = "auth_success"
	fields["user_id"] = userID
	GetLogger().WithContext(ctx).WithFields(logrus.Fields(fields)).Info("Authentication successful")
}

// LogPaymentCreated logs payment creation
func LogPaymentCreated(ctx context.Context, paymentID, merchantID string, amountVND string) {
	GetLogger().WithContext(ctx).WithFields(logrus.Fields{
		"event":       "payment_created",
		"payment_id":  paymentID,
		"merchant_id": merchantID,
		"amount_vnd":  amountVND,
	}).Info("Payment created")
}

// LogPaymentCompleted logs payment completion
func LogPaymentCompleted(ctx context.Context, paymentID, txHash string, amountCrypto string) {
	GetLogger().WithContext(ctx).WithFields(logrus.Fields{
		"event":         "payment_completed",
		"payment_id":    paymentID,
		"tx_hash":       txHash,
		"amount_crypto": amountCrypto,
	}).Info("Payment completed")
}

// LogPaymentFailed logs payment failure
func LogPaymentFailed(ctx context.Context, paymentID, reason string) {
	GetLogger().WithContext(ctx).WithFields(logrus.Fields{
		"event":      "payment_failed",
		"payment_id": paymentID,
		"reason":     reason,
	}).Error("Payment failed")
}

// LogPayoutRequested logs payout request
func LogPayoutRequested(ctx context.Context, payoutID, merchantID string, amountVND string) {
	GetLogger().WithContext(ctx).WithFields(logrus.Fields{
		"event":       "payout_requested",
		"payout_id":   payoutID,
		"merchant_id": merchantID,
		"amount_vnd":  amountVND,
	}).Info("Payout requested")
}

// LogPayoutCompleted logs payout completion
func LogPayoutCompleted(ctx context.Context, payoutID, referenceNumber string) {
	GetLogger().WithContext(ctx).WithFields(logrus.Fields{
		"event":            "payout_completed",
		"payout_id":        payoutID,
		"reference_number": referenceNumber,
	}).Info("Payout completed")
}

// LogKYCApproved logs KYC approval
func LogKYCApproved(ctx context.Context, merchantID, approvedBy string) {
	GetLogger().WithContext(ctx).WithFields(logrus.Fields{
		"event":       "kyc_approved",
		"merchant_id": merchantID,
		"approved_by": approvedBy,
	}).Info("KYC approved")
}

// LogKYCRejected logs KYC rejection
func LogKYCRejected(ctx context.Context, merchantID, rejectedBy, reason string) {
	GetLogger().WithContext(ctx).WithFields(logrus.Fields{
		"event":       "kyc_rejected",
		"merchant_id": merchantID,
		"rejected_by": rejectedBy,
		"reason":      reason,
	}).Warn("KYC rejected")
}

// LogSuspiciousActivity logs suspicious activities
func LogSuspiciousActivity(ctx context.Context, activityType string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event"] = "suspicious_activity"
	fields["activity_type"] = activityType
	GetLogger().WithContext(ctx).WithFields(logrus.Fields(fields)).Warn("Suspicious activity detected")
}

// LogBlockchainTransaction logs blockchain transaction events
func LogBlockchainTransaction(ctx context.Context, chain, txHash string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event"] = "blockchain_transaction"
	fields["chain"] = chain
	fields["tx_hash"] = txHash
	GetLogger().WithContext(ctx).WithFields(logrus.Fields(fields)).Info("Blockchain transaction processed")
}

// SanitizeFields removes sensitive data from log fields
func SanitizeFields(fields Fields) Fields {
	sanitized := make(Fields)
	sensitiveKeys := []string{
		"password", "private_key", "secret", "token", "api_key",
		"credit_card", "ssn", "tax_id",
	}

	for k, v := range fields {
		// Check if key contains sensitive information
		isSensitive := false
		for _, sensitive := range sensitiveKeys {
			if contains(k, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
