package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotificationService(t *testing.T) {
	t.Run("creates service with default values", func(t *testing.T) {
		config := NotificationServiceConfig{}
		service := NewNotificationService(config)

		assert.NotNil(t, service)
		assert.NotNil(t, service.logger)
		assert.NotNil(t, service.httpClient)
		assert.Equal(t, 30*time.Second, service.httpClient.Timeout)
	})

	t.Run("creates service with custom config", func(t *testing.T) {
		logger := logrus.New()
		config := NotificationServiceConfig{
			Logger:      logger,
			HTTPTimeout: 10 * time.Second,
			EmailSender: "test@example.com",
			EmailAPIKey: "test-key",
		}
		service := NewNotificationService(config)

		assert.Equal(t, logger, service.logger)
		assert.Equal(t, 10*time.Second, service.httpClient.Timeout)
		assert.Equal(t, "test@example.com", service.emailSender)
		assert.Equal(t, "test-key", service.emailAPIKey)
	})
}

func TestGenerateHMACSignature(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})

	t.Run("generates consistent signature for same input", func(t *testing.T) {
		payload := []byte(`{"test": "data"}`)
		secret := "my-secret"

		sig1 := service.generateHMACSignature(payload, secret)
		sig2 := service.generateHMACSignature(payload, secret)

		assert.Equal(t, sig1, sig2)
		assert.NotEmpty(t, sig1)
	})

	t.Run("generates different signatures for different payloads", func(t *testing.T) {
		secret := "my-secret"
		payload1 := []byte(`{"test": "data1"}`)
		payload2 := []byte(`{"test": "data2"}`)

		sig1 := service.generateHMACSignature(payload1, secret)
		sig2 := service.generateHMACSignature(payload2, secret)

		assert.NotEqual(t, sig1, sig2)
	})

	t.Run("generates different signatures for different secrets", func(t *testing.T) {
		payload := []byte(`{"test": "data"}`)
		secret1 := "secret1"
		secret2 := "secret2"

		sig1 := service.generateHMACSignature(payload, secret1)
		sig2 := service.generateHMACSignature(payload, secret2)

		assert.NotEqual(t, sig1, sig2)
	})
}

func TestVerifyWebhookSignature(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})

	t.Run("verifies valid signature", func(t *testing.T) {
		payload := []byte(`{"test": "data"}`)
		secret := "my-secret"
		signature := service.generateHMACSignature(payload, secret)

		valid := service.VerifyWebhookSignature(payload, signature, secret)
		assert.True(t, valid)
	})

	t.Run("rejects invalid signature", func(t *testing.T) {
		payload := []byte(`{"test": "data"}`)
		secret := "my-secret"
		invalidSignature := "invalid-signature"

		valid := service.VerifyWebhookSignature(payload, invalidSignature, secret)
		assert.False(t, valid)
	})

	t.Run("rejects signature with wrong secret", func(t *testing.T) {
		payload := []byte(`{"test": "data"}`)
		secret1 := "secret1"
		secret2 := "secret2"
		signature := service.generateHMACSignature(payload, secret1)

		valid := service.VerifyWebhookSignature(payload, signature, secret2)
		assert.False(t, valid)
	})

	t.Run("rejects signature for modified payload", func(t *testing.T) {
		payload1 := []byte(`{"test": "data1"}`)
		payload2 := []byte(`{"test": "data2"}`)
		secret := "my-secret"
		signature := service.generateHMACSignature(payload1, secret)

		valid := service.VerifyWebhookSignature(payload2, signature, secret)
		assert.False(t, valid)
	})
}

func TestSendWebhook_Success(t *testing.T) {
	// Create a test server that always returns 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Signature"))
		assert.Equal(t, "payment.completed", r.Header.Get("X-Webhook-Event"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Timestamp"))
		assert.Equal(t, "1", r.Header.Get("X-Webhook-Attempt"))
		assert.Equal(t, "StablecoinGateway-Webhook/1.0", r.Header.Get("User-Agent"))

		// Verify request body
		var payload WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, WebhookEventPaymentCompleted, payload.Event)
		assert.Equal(t, "payment-123", payload.Data["payment_id"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"received": true}`))
	}))
	defer server.Close()

	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	data := map[string]interface{}{
		"payment_id": "payment-123",
		"amount":     "1000000",
	}

	err := service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, data)
	assert.NoError(t, err)
}

func TestSendWebhook_Retry(t *testing.T) {
	// Create a test server that fails first 2 attempts, then succeeds
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "temporary failure"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"received": true}`))
	}))
	defer server.Close()

	service := NewNotificationService(NotificationServiceConfig{
		HTTPTimeout: 5 * time.Second,
	})
	ctx := context.Background()

	data := map[string]interface{}{
		"payment_id": "payment-123",
	}

	startTime := time.Now()
	err := service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, data)
	duration := time.Since(startTime)

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))

	// Verify exponential backoff (1s + 2s = 3s minimum)
	// Allow some tolerance for test execution time
	assert.GreaterOrEqual(t, duration, 3*time.Second)
	assert.Less(t, duration, 5*time.Second)
}

func TestSendWebhook_MaxRetriesExceeded(t *testing.T) {
	// Create a test server that always fails
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "permanent failure"}`))
	}))
	defer server.Close()

	service := NewNotificationService(NotificationServiceConfig{
		HTTPTimeout: 5 * time.Second,
	})
	ctx := context.Background()

	data := map[string]interface{}{
		"payment_id": "payment-123",
	}

	err := service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, data)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook delivery failed after 5 attempts")
	assert.Equal(t, int32(5), atomic.LoadInt32(&attemptCount))
}

func TestSendWebhook_ContextCancellation(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewNotificationService(NotificationServiceConfig{})

	// Create context that cancels after 500ms
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	data := map[string]interface{}{
		"payment_id": "payment-123",
	}

	err := service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, data)

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestSendWebhook_InvalidURL(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	data := map[string]interface{}{
		"payment_id": "payment-123",
	}

	err := service.SendWebhook(ctx, "http://invalid-url-that-does-not-exist.local", "test-secret", WebhookEventPaymentCompleted, data)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook delivery failed after 5 attempts")
}

func TestSendWebhook_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		shouldSucceed  bool
	}{
		{"200 OK", http.StatusOK, true},
		{"201 Created", http.StatusCreated, true},
		{"204 No Content", http.StatusNoContent, true},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"401 Unauthorized", http.StatusUnauthorized, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"500 Internal Server Error", http.StatusInternalServerError, false},
		{"503 Service Unavailable", http.StatusServiceUnavailable, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			service := NewNotificationService(NotificationServiceConfig{})
			ctx := context.Background()

			data := map[string]interface{}{"test": "data"}
			err := service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, data)

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestSendWebhook_PayloadStructure(t *testing.T) {
	var receivedPayload WebhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	testData := map[string]interface{}{
		"payment_id":  "payment-123",
		"amount":      "1000000",
		"merchant_id": "merchant-456",
	}

	err := service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, testData)
	require.NoError(t, err)

	// Verify payload structure
	assert.Equal(t, WebhookEventPaymentCompleted, receivedPayload.Event)
	assert.NotZero(t, receivedPayload.Timestamp)
	assert.Equal(t, "payment-123", receivedPayload.Data["payment_id"])
	assert.Equal(t, "1000000", receivedPayload.Data["amount"])
	assert.Equal(t, "merchant-456", receivedPayload.Data["merchant_id"])
}

func TestSendEmail(t *testing.T) {
	t.Run("logs email for MVP without sending", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		service := NewNotificationService(NotificationServiceConfig{
			Logger: logger,
		})

		ctx := context.Background()
		data := map[string]interface{}{
			"payment_id": "payment-123",
			"amount":     "1000000",
		}

		err := service.SendEmail(ctx, EmailTypePaymentConfirmation, "merchant@example.com", data)
		assert.NoError(t, err)
	})
}

func TestSendPaymentConfirmationEmail(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	err := service.SendPaymentConfirmationEmail(ctx, "merchant@example.com", "payment-123", "1000000 VND")
	assert.NoError(t, err)
}

func TestSendPayoutApprovedEmail(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	err := service.SendPayoutApprovedEmail(ctx, "merchant@example.com", "payout-123", "5000000 VND")
	assert.NoError(t, err)
}

func TestSendPayoutCompletedEmail(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	err := service.SendPayoutCompletedEmail(ctx, "merchant@example.com", "payout-123", "5000000 VND", "REF-2025-001")
	assert.NoError(t, err)
}

func TestSendDailySettlementReport(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	reportData := map[string]interface{}{
		"date":                 "2025-11-16",
		"total_payments_count": 150,
		"total_volume_vnd":     "500000000",
		"total_payouts_count":  5,
		"total_fees_vnd":       "5000000",
	}

	err := service.SendDailySettlementReport(ctx, "ops@example.com", reportData)
	assert.NoError(t, err)
}

func TestSendKYCApprovedEmail(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	err := service.SendKYCApprovedEmail(ctx, "merchant@example.com", "merchant-123", "sk_live_abc123xyz")
	assert.NoError(t, err)
}

func TestSendKYCRejectedEmail(t *testing.T) {
	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	err := service.SendKYCRejectedEmail(ctx, "merchant@example.com", "merchant-123", "Business license is not clear")
	assert.NoError(t, err)
}

func TestWebhookEvents(t *testing.T) {
	t.Run("webhook event constants are defined", func(t *testing.T) {
		assert.Equal(t, WebhookEvent("payment.completed"), WebhookEventPaymentCompleted)
		assert.Equal(t, WebhookEvent("payment.failed"), WebhookEventPaymentFailed)
		assert.Equal(t, WebhookEvent("payout.completed"), WebhookEventPayoutCompleted)
	})
}

func TestEmailTypes(t *testing.T) {
	t.Run("email type constants are defined", func(t *testing.T) {
		assert.Equal(t, EmailType("payment_confirmation"), EmailTypePaymentConfirmation)
		assert.Equal(t, EmailType("payout_approved"), EmailTypePayoutApproved)
		assert.Equal(t, EmailType("payout_completed"), EmailTypePayoutCompleted)
		assert.Equal(t, EmailType("daily_settlement"), EmailTypeDailySettlement)
		assert.Equal(t, EmailType("kyc_approved"), EmailTypeKYCApproved)
		assert.Equal(t, EmailType("kyc_rejected"), EmailTypeKYCRejected)
	})
}

func TestConcurrentWebhookDelivery(t *testing.T) {
	// Test that multiple webhooks can be sent concurrently
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate some processing time
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()

	numWebhooks := 10
	errors := make(chan error, numWebhooks)

	startTime := time.Now()

	// Send webhooks concurrently
	for i := 0; i < numWebhooks; i++ {
		go func(id int) {
			data := map[string]interface{}{
				"payment_id": fmt.Sprintf("payment-%d", id),
			}
			err := service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, data)
			errors <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numWebhooks; i++ {
		err := <-errors
		assert.NoError(t, err)
	}

	duration := time.Since(startTime)

	// Verify concurrent execution (should be much less than sequential)
	// Sequential would be ~1 second (10 * 100ms), concurrent should be ~100-200ms
	assert.Less(t, duration, 500*time.Millisecond)
}

func BenchmarkGenerateHMACSignature(b *testing.B) {
	service := NewNotificationService(NotificationServiceConfig{})
	payload := []byte(`{"event": "payment.completed", "data": {"payment_id": "test-123", "amount": "1000000"}}`)
	secret := "test-secret-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateHMACSignature(payload, secret)
	}
}

func BenchmarkSendWebhook(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewNotificationService(NotificationServiceConfig{})
	ctx := context.Background()
	data := map[string]interface{}{
		"payment_id": "payment-123",
		"amount":     "1000000",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.SendWebhook(ctx, server.URL, "test-secret", WebhookEventPaymentCompleted, data)
	}
}
