package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// Job type constants
const (
	TypeWebhookDelivery       = "webhook:delivery"
	TypePaymentExpiry         = "payment:expiry"
	TypeBalanceCheck          = "wallet:balance_check"
	TypeDailySettlementReport = "report:daily_settlement"
)

// Job priority levels
const (
	PriorityCritical = 3
	PriorityHigh     = 2
	PriorityMedium   = 1
	PriorityLow      = 0
)

// Queue manages job queue operations
type Queue struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

// QueueConfig holds configuration for the job queue
type QueueConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// NewQueue creates a new job queue client
func NewQueue(cfg *QueueConfig) (*Queue, error) {
	redisOpts := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	client := asynq.NewClient(redisOpts)
	inspector := asynq.NewInspector(redisOpts)

	return &Queue{
		client:    client,
		inspector: inspector,
	}, nil
}

// Close closes the queue client
func (q *Queue) Close() error {
	return q.client.Close()
}

// WebhookDeliveryPayload represents the payload for webhook delivery jobs
type WebhookDeliveryPayload struct {
	MerchantID string                 `json:"merchant_id"`
	Event      string                 `json:"event"`
	Payload    map[string]interface{} `json:"payload"`
	Attempt    int                    `json:"attempt"`
}

// PaymentExpiryPayload represents the payload for payment expiry jobs
type PaymentExpiryPayload struct {
	RunAt time.Time `json:"run_at"`
}

// BalanceCheckPayload represents the payload for balance check jobs
type BalanceCheckPayload struct {
	RunAt time.Time `json:"run_at"`
}

// DailySettlementReportPayload represents the payload for daily settlement report jobs
type DailySettlementReportPayload struct {
	Date time.Time `json:"date"`
}

// EnqueueWebhookDelivery enqueues a webhook delivery job
func (q *Queue) EnqueueWebhookDelivery(ctx context.Context, payload *WebhookDeliveryPayload) error {
	taskPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	task := asynq.NewTask(TypeWebhookDelivery, taskPayload)

	// Set options: retry up to 5 times with exponential backoff
	opts := []asynq.Option{
		asynq.MaxRetry(5),
		asynq.Queue("webhooks"),
		asynq.Timeout(30 * time.Second),
		// Exponential backoff: 1s, 2s, 4s, 8s, 16s
		asynq.RetryDelay(func(n int, err error, task *asynq.Task) time.Duration {
			return time.Duration(1<<uint(n)) * time.Second
		}),
	}

	// Enqueue with priority based on attempt number
	if payload.Attempt > 0 {
		opts = append(opts, asynq.Queue("webhooks_retry"))
	}

	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue webhook delivery job: %w", err)
	}

	logger.Info("Webhook delivery job enqueued", logger.Fields{
		"task_id":     info.ID,
		"merchant_id": payload.MerchantID,
		"event":       payload.Event,
		"queue":       info.Queue,
	})

	return nil
}

// EnqueuePaymentExpiry enqueues a payment expiry check job
func (q *Queue) EnqueuePaymentExpiry(ctx context.Context, payload *PaymentExpiryPayload) error {
	taskPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payment expiry payload: %w", err)
	}

	task := asynq.NewTask(TypePaymentExpiry, taskPayload)

	opts := []asynq.Option{
		asynq.MaxRetry(3),
		asynq.Queue("periodic"),
		asynq.Timeout(5 * time.Minute),
	}

	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue payment expiry job: %w", err)
	}

	logger.Info("Payment expiry job enqueued", logger.Fields{
		"task_id": info.ID,
		"run_at":  payload.RunAt,
	})

	return nil
}

// EnqueueBalanceCheck enqueues a balance check job
func (q *Queue) EnqueueBalanceCheck(ctx context.Context, payload *BalanceCheckPayload) error {
	taskPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal balance check payload: %w", err)
	}

	task := asynq.NewTask(TypeBalanceCheck, taskPayload)

	opts := []asynq.Option{
		asynq.MaxRetry(3),
		asynq.Queue("monitoring"),
		asynq.Timeout(2 * time.Minute),
	}

	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue balance check job: %w", err)
	}

	logger.Info("Balance check job enqueued", logger.Fields{
		"task_id": info.ID,
		"run_at":  payload.RunAt,
	})

	return nil
}

// EnqueueDailySettlementReport enqueues a daily settlement report job
func (q *Queue) EnqueueDailySettlementReport(ctx context.Context, payload *DailySettlementReportPayload) error {
	taskPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal daily settlement report payload: %w", err)
	}

	task := asynq.NewTask(TypeDailySettlementReport, taskPayload)

	opts := []asynq.Option{
		asynq.MaxRetry(3),
		asynq.Queue("reports"),
		asynq.Timeout(10 * time.Minute),
		asynq.ProcessAt(payload.Date), // Schedule for specific time
	}

	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue daily settlement report job: %w", err)
	}

	logger.Info("Daily settlement report job enqueued", logger.Fields{
		"task_id": info.ID,
		"date":    payload.Date,
	})

	return nil
}

// GetQueueStats returns statistics for all queues
func (q *Queue) GetQueueStats(ctx context.Context) (map[string]*asynq.QueueInfo, error) {
	queues := []string{"webhooks", "webhooks_retry", "periodic", "monitoring", "reports"}
	stats := make(map[string]*asynq.QueueInfo)

	for _, queueName := range queues {
		info, err := q.inspector.GetQueueInfo(queueName)
		if err != nil {
			// Queue might not exist yet, skip
			continue
		}
		stats[queueName] = info
	}

	return stats, nil
}

// GetTaskInfo returns information about a specific task
func (q *Queue) GetTaskInfo(queue, taskID string) (*asynq.TaskInfo, error) {
	return q.inspector.GetTaskInfo(queue, taskID)
}

// PurgeQueue removes all tasks from a specific queue
func (q *Queue) PurgeQueue(queueName string) error {
	n, err := q.inspector.DeleteAllPendingTasks(queueName)
	if err != nil {
		return fmt.Errorf("failed to purge queue %s: %w", queueName, err)
	}

	logger.Info("Queue purged", logger.Fields{
		"queue":        queueName,
		"tasks_deleted": n,
	})

	return nil
}
