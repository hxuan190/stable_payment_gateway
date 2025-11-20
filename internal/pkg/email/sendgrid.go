package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// SendGridProvider implements email sending using SendGrid API
type SendGridProvider struct {
	apiKey     string
	fromEmail  string
	fromName   string
	httpClient *http.Client
	logger     *logrus.Logger
	baseURL    string
}

// SendGridConfig holds configuration for SendGrid
type SendGridConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
	Logger    *logrus.Logger
	BaseURL   string // Optional, defaults to SendGrid production API
}

// NewSendGridProvider creates a new SendGrid email provider
func NewSendGridProvider(config SendGridConfig) *SendGridProvider {
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.sendgrid.com/v3"
	}

	return &SendGridProvider{
		apiKey:    config.APIKey,
		fromEmail: config.FromEmail,
		fromName:  config.FromName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  config.Logger,
		baseURL: baseURL,
	}
}

// SendGridEmail represents the SendGrid email API request
type SendGridEmail struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmailAddress      `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
}

// SendGridPersonalization represents SendGrid personalization
type SendGridPersonalization struct {
	To                  []SendGridEmailAddress `json:"to"`
	DynamicTemplateData map[string]interface{} `json:"dynamic_template_data,omitempty"`
}

// SendGridEmailAddress represents an email address
type SendGridEmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// SendGridContent represents email content
type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SendGridResponse represents the SendGrid API response
type SendGridResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// Send sends an email via SendGrid with retry logic
func (p *SendGridProvider) Send(ctx context.Context, email *Email) error {
	// Maximum 3 retry attempts with exponential backoff
	maxAttempts := 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := p.sendAttempt(ctx, email, attempt)

		if err == nil {
			p.logger.WithFields(logrus.Fields{
				"to":      email.To,
				"subject": email.Subject,
				"attempt": attempt,
			}).Info("Email sent successfully via SendGrid")
			return nil
		}

		lastErr = err

		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Don't retry on last attempt
		if attempt < maxAttempts {
			// Exponential backoff: 1s, 5s, 15s
			backoffDuration := time.Duration([]int{1, 5, 15}[attempt-1]) * time.Second

			p.logger.WithFields(logrus.Fields{
				"to":              email.To,
				"attempt":         attempt,
				"next_retry_in_s": backoffDuration.Seconds(),
				"error":           err,
			}).Warn("Email sending failed, will retry")

			select {
			case <-time.After(backoffDuration):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	p.logger.WithFields(logrus.Fields{
		"to":           email.To,
		"subject":      email.Subject,
		"max_attempts": maxAttempts,
		"error":        lastErr,
	}).Error("Email sending failed after all retry attempts")

	return fmt.Errorf("email sending failed after %d attempts: %w", maxAttempts, lastErr)
}

// sendAttempt performs a single email sending attempt
func (p *SendGridProvider) sendAttempt(ctx context.Context, email *Email, attempt int) error {
	// Build SendGrid email payload
	sgEmail := SendGridEmail{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmailAddress{
					{Email: email.To},
				},
				DynamicTemplateData: email.TemplateData,
			},
		},
		From: SendGridEmailAddress{
			Email: p.fromEmail,
			Name:  p.fromName,
		},
		Subject: email.Subject,
		Content: []SendGridContent{
			{
				Type:  "text/html",
				Value: email.HTMLBody,
			},
		},
	}

	// If plain text body is provided, add it
	if email.TextBody != "" {
		sgEmail.Content = append([]SendGridContent{
			{
				Type:  "text/plain",
				Value: email.TextBody,
			},
		}, sgEmail.Content...)
	}

	// Marshal to JSON
	payloadBytes, err := json.Marshal(sgEmail)
	if err != nil {
		return fmt.Errorf("failed to marshal SendGrid email payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/mail/send", p.baseURL),
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		return fmt.Errorf("failed to create SendGrid request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		p.logger.Warn("Failed to read SendGrid response body")
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
			"to":          email.To,
			"attempt":     attempt,
		}).Error("SendGrid API returned error")

		return fmt.Errorf("SendGrid API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Email represents an email message
type Email struct {
	To           string
	Subject      string
	HTMLBody     string
	TextBody     string
	TemplateData map[string]interface{}
}
