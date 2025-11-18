package interfaces

import (
	"context"
	"time"
)

// MerchantInfo contains basic merchant information for cross-module access
type MerchantInfo struct {
	ID            string
	Email         string
	BusinessName  string
	KYCStatus     string
	KYCTier       int
	IsActive      bool
	WebhookURL    string
	WebhookSecret string
	CreatedAt     time.Time
}

// MerchantReader provides read-only access to merchant data
// Used by other modules to check merchant status without tight coupling
type MerchantReader interface {
	// GetByID retrieves merchant by ID
	GetByID(ctx context.Context, id string) (*MerchantInfo, error)

	// GetByAPIKey retrieves merchant by API key
	GetByAPIKey(ctx context.Context, apiKey string) (*MerchantInfo, error)

	// IsKYCApproved checks if merchant has approved KYC
	IsKYCApproved(ctx context.Context, merchantID string) (bool, error)

	// IsActive checks if merchant account is active
	IsActive(ctx context.Context, merchantID string) (bool, error)

	// GetKYCTier returns the merchant's KYC tier (1, 2, or 3)
	GetKYCTier(ctx context.Context, merchantID string) (int, error)
}

// MerchantWebhookProvider provides webhook configuration
type MerchantWebhookProvider interface {
	// GetWebhookConfig returns webhook URL and secret for a merchant
	GetWebhookConfig(ctx context.Context, merchantID string) (url, secret string, err error)
}
