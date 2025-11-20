package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/pkg/email"
	"github.com/sirupsen/logrus"
)

// WebhookEvent represents the type of webhook event
type WebhookEvent string

const (
	// WebhookEventPaymentCompleted is sent when a payment is confirmed
	WebhookEventPaymentCompleted WebhookEvent = "payment.completed"
	// WebhookEventPaymentFailed is sent when a payment fails or expires
	WebhookEventPaymentFailed WebhookEvent = "payment.failed"
	// WebhookEventPayoutCompleted is sent when a payout is processed
	WebhookEventPayoutCompleted WebhookEvent = "payout.completed"
)

// WebhookPayload represents the structure of webhook data sent to merchants
type WebhookPayload struct {
	Event     WebhookEvent           `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// WebhookDeliveryAttempt represents a single webhook delivery attempt
type WebhookDeliveryAttempt struct {
	URL        string
	Attempt    int
	StatusCode int
	Error      error
	Duration   time.Duration
}

// NotificationService handles webhook and email notifications
type NotificationService struct {
	logger        *logrus.Logger
	httpClient    *http.Client
	emailProvider *email.SendGridProvider

	// Email configuration (can be nil for MVP without email)
	emailSender string
	emailAPIKey string
}

// NotificationServiceConfig holds configuration for the notification service
type NotificationServiceConfig struct {
	Logger      *logrus.Logger
	HTTPTimeout time.Duration
	EmailSender string
	EmailAPIKey string
}

// NewNotificationService creates a new notification service
func NewNotificationService(config NotificationServiceConfig) *NotificationService {
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 30 * time.Second
	}

	// Initialize email provider if API key is configured
	var emailProvider *email.SendGridProvider
	if config.EmailAPIKey != "" {
		emailProvider = email.NewSendGridProvider(email.SendGridConfig{
			APIKey:    config.EmailAPIKey,
			FromEmail: config.EmailSender,
			FromName:  "Stablecoin Payment Gateway",
			Logger:    config.Logger,
		})
		config.Logger.Info("Email provider (SendGrid) initialized")
	} else {
		config.Logger.Warn("Email API key not configured - emails will be logged only")
	}

	return &NotificationService{
		logger:        config.Logger,
		emailProvider: emailProvider,
		httpClient: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		emailSender: config.EmailSender,
		emailAPIKey: config.EmailAPIKey,
	}
}

// SendWebhook sends a webhook notification to a merchant with retry logic
// Returns error only if all retry attempts fail
func (s *NotificationService) SendWebhook(
	ctx context.Context,
	webhookURL string,
	webhookSecret string,
	event WebhookEvent,
	data map[string]interface{},
) error {
	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	// Maximum 5 retry attempts with exponential backoff
	maxAttempts := 5
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		deliveryAttempt, err := s.sendWebhookAttempt(ctx, webhookURL, webhookSecret, payload, attempt)

		s.logger.WithFields(logrus.Fields{
			"url":          webhookURL,
			"event":        event,
			"attempt":      attempt,
			"status_code":  deliveryAttempt.StatusCode,
			"duration_ms":  deliveryAttempt.Duration.Milliseconds(),
			"error":        err,
		}).Info("Webhook delivery attempt")

		if err == nil && deliveryAttempt.StatusCode >= 200 && deliveryAttempt.StatusCode < 300 {
			s.logger.WithFields(logrus.Fields{
				"url":     webhookURL,
				"event":   event,
				"attempt": attempt,
			}).Info("Webhook delivered successfully")
			return nil
		}

		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("webhook failed with status code: %d", deliveryAttempt.StatusCode)
		}

		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Don't retry on last attempt
		if attempt < maxAttempts {
			// Exponential backoff: 1s, 2s, 4s, 8s, 16s
			backoffDuration := time.Duration(1<<uint(attempt-1)) * time.Second

			s.logger.WithFields(logrus.Fields{
				"url":              webhookURL,
				"attempt":          attempt,
				"next_retry_in_ms": backoffDuration.Milliseconds(),
			}).Warn("Webhook delivery failed, will retry")

			select {
			case <-time.After(backoffDuration):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	s.logger.WithFields(logrus.Fields{
		"url":          webhookURL,
		"event":        event,
		"max_attempts": maxAttempts,
		"error":        lastErr,
	}).Error("Webhook delivery failed after all retry attempts")

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", maxAttempts, lastErr)
}

// sendWebhookAttempt performs a single webhook delivery attempt
func (s *NotificationService) sendWebhookAttempt(
	ctx context.Context,
	webhookURL string,
	webhookSecret string,
	payload WebhookPayload,
	attempt int,
) (WebhookDeliveryAttempt, error) {
	deliveryAttempt := WebhookDeliveryAttempt{
		URL:     webhookURL,
		Attempt: attempt,
	}

	startTime := time.Now()
	defer func() {
		deliveryAttempt.Duration = time.Since(startTime)
	}()

	// Serialize payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		deliveryAttempt.Error = fmt.Errorf("failed to marshal webhook payload: %w", err)
		return deliveryAttempt, deliveryAttempt.Error
	}

	// Generate HMAC signature
	signature := s.generateHMACSignature(payloadBytes, webhookSecret)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payloadBytes))
	if err != nil {
		deliveryAttempt.Error = fmt.Errorf("failed to create webhook request: %w", err)
		return deliveryAttempt, deliveryAttempt.Error
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", string(payload.Event))
	req.Header.Set("X-Webhook-Timestamp", payload.Timestamp.Format(time.RFC3339))
	req.Header.Set("X-Webhook-Attempt", fmt.Sprintf("%d", attempt))
	req.Header.Set("User-Agent", "StablecoinGateway-Webhook/1.0")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		deliveryAttempt.Error = fmt.Errorf("failed to send webhook request: %w", err)
		return deliveryAttempt, deliveryAttempt.Error
	}
	defer resp.Body.Close()

	deliveryAttempt.StatusCode = resp.StatusCode

	// Read response body (limited to 1MB to prevent abuse)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"url":    webhookURL,
			"status": resp.StatusCode,
		}).Warn("Failed to read webhook response body")
	}

	// Log response for debugging (only for non-2xx responses)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.WithFields(logrus.Fields{
			"url":          webhookURL,
			"status_code":  resp.StatusCode,
			"response":     string(respBody),
			"attempt":      attempt,
		}).Warn("Webhook returned non-success status code")
	}

	return deliveryAttempt, nil
}

// generateHMACSignature creates HMAC-SHA256 signature for webhook verification
func (s *NotificationService) generateHMACSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyWebhookSignature verifies the HMAC signature of a webhook payload
// This helper function can be used by merchants to verify webhook authenticity
func (s *NotificationService) VerifyWebhookSignature(payload []byte, signature string, secret string) bool {
	expectedSignature := s.generateHMACSignature(payload, secret)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Email notification types
type EmailType string

const (
	EmailTypePaymentConfirmation EmailType = "payment_confirmation"
	EmailTypePayoutApproved      EmailType = "payout_approved"
	EmailTypePayoutCompleted     EmailType = "payout_completed"
	EmailTypeDailySettlement     EmailType = "daily_settlement"
	EmailTypeKYCApproved         EmailType = "kyc_approved"
	EmailTypeKYCRejected         EmailType = "kyc_rejected"
)

// EmailData represents data for email templates
type EmailData struct {
	To          string
	Subject     string
	TemplateID  string
	TemplateData map[string]interface{}
}

// SendEmail sends an email notification
// Integrates with SendGrid for actual email delivery
func (s *NotificationService) SendEmail(ctx context.Context, emailType EmailType, to string, data map[string]interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"type":   emailType,
		"to":     to,
		"sender": s.emailSender,
	}).Info("Preparing email notification")

	// Map EmailType to TemplateType
	var templateType email.TemplateType
	switch emailType {
	case EmailTypePaymentConfirmation:
		templateType = email.TemplatePaymentConfirmation
	case EmailTypePayoutApproved:
		templateType = email.TemplatePayoutApproved
	case EmailTypePayoutCompleted:
		templateType = email.TemplatePayoutCompleted
	case EmailTypeDailySettlement:
		templateType = email.TemplateDailySettlement
	case EmailTypeKYCApproved:
		templateType = email.TemplateKYCApproved
	case EmailTypeKYCRejected:
		templateType = email.TemplateKYCRejected
	default:
		return fmt.Errorf("unsupported email type: %s", emailType)
	}

	// Get template
	template := email.GetTemplate(templateType)
	if template == nil {
		return fmt.Errorf("template not found for type: %s", emailType)
	}

	// Render email from template
	renderedEmail, err := email.RenderTemplate(template, data)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"type":  emailType,
			"to":    to,
			"error": err,
		}).Error("Failed to render email template")
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// Set recipient
	renderedEmail.To = to

	// If email provider is configured, send via SendGrid
	if s.emailProvider != nil {
		err := s.emailProvider.Send(ctx, renderedEmail)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"type":  emailType,
				"to":    to,
				"error": err,
			}).Error("Failed to send email via SendGrid")
			return fmt.Errorf("failed to send email: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"type":    emailType,
			"to":      to,
			"subject": renderedEmail.Subject,
		}).Info("Email sent successfully via SendGrid")
		return nil
	}

	// If no email provider, just log (development/testing mode)
	s.logger.WithFields(logrus.Fields{
		"type":    emailType,
		"to":      to,
		"subject": renderedEmail.Subject,
		"data":    data,
	}).Info("Email notification (development mode: logging only, not actually sent)")

	return nil
}

// SendPaymentConfirmationEmail sends email when payment is confirmed
func (s *NotificationService) SendPaymentConfirmationEmail(ctx context.Context, merchantEmail string, paymentID string, amount string, currency string) error {
	data := map[string]interface{}{
		"payment_id": paymentID,
		"amount":     amount,
		"currency":   currency,
		"timestamp":  email.FormatTimestamp(time.Now()),
	}
	return s.SendEmail(ctx, EmailTypePaymentConfirmation, merchantEmail, data)
}

// SendPayoutApprovedEmail sends email when payout is approved
func (s *NotificationService) SendPayoutApprovedEmail(ctx context.Context, merchantEmail string, payoutID string, amount string) error {
	data := map[string]interface{}{
		"payout_id": payoutID,
		"amount":    amount,
		"timestamp": email.FormatTimestamp(time.Now()),
	}
	return s.SendEmail(ctx, EmailTypePayoutApproved, merchantEmail, data)
}

// SendPayoutCompletedEmail sends email when payout is completed
func (s *NotificationService) SendPayoutCompletedEmail(ctx context.Context, merchantEmail string, payoutID string, amount string, referenceNumber string) error {
	data := map[string]interface{}{
		"payout_id":        payoutID,
		"amount":           amount,
		"reference_number": referenceNumber,
		"timestamp":        email.FormatTimestamp(time.Now()),
	}
	return s.SendEmail(ctx, EmailTypePayoutCompleted, merchantEmail, data)
}

// SendDailySettlementReport sends daily settlement report to ops team
func (s *NotificationService) SendDailySettlementReport(ctx context.Context, opsEmail string, reportData map[string]interface{}) error {
	// Ensure timestamp is formatted
	if _, ok := reportData["timestamp"]; !ok {
		reportData["timestamp"] = email.FormatTimestamp(time.Now())
	}
	return s.SendEmail(ctx, EmailTypeDailySettlement, opsEmail, reportData)
}

// SendKYCApprovedEmail sends email when merchant KYC is approved
func (s *NotificationService) SendKYCApprovedEmail(ctx context.Context, merchantEmail string, merchantID string, apiKey string) error {
	data := map[string]interface{}{
		"merchant_id": merchantID,
		"api_key":     apiKey,
		"timestamp":   email.FormatTimestamp(time.Now()),
	}
	return s.SendEmail(ctx, EmailTypeKYCApproved, merchantEmail, data)
}

// SendKYCRejectedEmail sends email when merchant KYC is rejected
func (s *NotificationService) SendKYCRejectedEmail(ctx context.Context, merchantEmail string, merchantID string, reason string) error {
	data := map[string]interface{}{
		"merchant_id": merchantID,
		"reason":      reason,
		"timestamp":   email.FormatTimestamp(time.Now()),
	}
	return s.SendEmail(ctx, EmailTypeKYCRejected, merchantEmail, data)
}

// SendTreasuryAlertEmail sends email when treasury threshold is triggered
func (s *NotificationService) SendTreasuryAlertEmail(ctx context.Context, opsEmails []string, alertType string, walletAddress string, balance string, threshold string, recommendedAction string) error {
	data := map[string]interface{}{
		"alert_type":         alertType,
		"wallet_address":     walletAddress,
		"balance":            balance,
		"threshold":          threshold,
		"recommended_action": recommendedAction,
		"timestamp":          email.FormatTimestamp(time.Now()),
	}

	// Send to all ops team emails
	for _, opsEmail := range opsEmails {
		// Map to treasury alert email type (need to add to EmailType)
		templateType := email.TemplateTreasuryAlert
		template := email.GetTemplate(templateType)
		if template == nil {
			return fmt.Errorf("template not found for treasury alert")
		}

		renderedEmail, err := email.RenderTemplate(template, data)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"to":    opsEmail,
				"error": err,
			}).Error("Failed to render treasury alert email template")
			continue
		}

		renderedEmail.To = opsEmail

		if s.emailProvider != nil {
			if err := s.emailProvider.Send(ctx, renderedEmail); err != nil {
				s.logger.WithFields(logrus.Fields{
					"to":    opsEmail,
					"error": err,
				}).Error("Failed to send treasury alert email")
				// Continue to send to other recipients even if one fails
				continue
			}
		} else {
			s.logger.WithFields(logrus.Fields{
				"to":      opsEmail,
				"subject": renderedEmail.Subject,
			}).Info("Treasury alert email (development mode: logging only)")
		}
	}

	return nil
}

// SendComplianceAlertEmail sends email when compliance issue is detected
func (s *NotificationService) SendComplianceAlertEmail(ctx context.Context, opsEmails []string, alertType string, merchantID string, details string, requiredAction string) error {
	data := map[string]interface{}{
		"alert_type":      alertType,
		"merchant_id":     merchantID,
		"details":         details,
		"required_action": requiredAction,
		"timestamp":       email.FormatTimestamp(time.Now()),
	}

	// Send to all ops team emails
	for _, opsEmail := range opsEmails {
		templateType := email.TemplateComplianceAlert
		template := email.GetTemplate(templateType)
		if template == nil {
			return fmt.Errorf("template not found for compliance alert")
		}

		renderedEmail, err := email.RenderTemplate(template, data)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"to":    opsEmail,
				"error": err,
			}).Error("Failed to render compliance alert email template")
			continue
		}

		renderedEmail.To = opsEmail

		if s.emailProvider != nil {
			if err := s.emailProvider.Send(ctx, renderedEmail); err != nil {
				s.logger.WithFields(logrus.Fields{
					"to":    opsEmail,
					"error": err,
				}).Error("Failed to send compliance alert email")
				// Continue to send to other recipients even if one fails
				continue
			}
		} else {
			s.logger.WithFields(logrus.Fields{
				"to":      opsEmail,
				"subject": renderedEmail.Subject,
			}).Info("Compliance alert email (development mode: logging only)")
		}
	}

	return nil
}
