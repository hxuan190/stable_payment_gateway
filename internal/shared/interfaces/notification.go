package interfaces

import "context"

// WebhookEvent represents a webhook event to be delivered
type WebhookEvent struct {
	MerchantID string
	EventType  string
	Payload    map[string]interface{}
}

// EmailRequest represents an email notification request
type EmailRequest struct {
	To       string
	Subject  string
	Template string
	Data     map[string]interface{}
}

// NotificationSender provides notification capabilities
type NotificationSender interface {
	// SendWebhook sends a webhook notification to merchant
	SendWebhook(ctx context.Context, event WebhookEvent) error

	// SendEmail sends an email notification
	SendEmail(ctx context.Context, req EmailRequest) error
}
