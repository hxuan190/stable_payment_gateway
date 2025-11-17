# Background Worker Service

This package implements the background job processing system for the payment gateway using [asynq](https://github.com/hibiken/asynq).

## Overview

The worker service processes asynchronous tasks using Redis as the job broker. It handles four main types of jobs:

1. **Webhook Delivery** - Sends webhooks to merchants for payment events
2. **Payment Expiry** - Marks expired payments as expired (runs every 5 minutes)
3. **Balance Check** - Monitors wallet balances and alerts on thresholds (runs every 5 minutes)
4. **Daily Settlement Report** - Generates daily reports for ops team (runs daily at 8 AM)

## Architecture

### Components

- **Queue (`queue.go`)**: Client for enqueuing jobs
- **Server (`server.go`)**: Worker server that processes jobs
- **Handlers (`handlers.go`)**: Job processing logic

### Job Types

#### 1. Webhook Delivery
- **Type**: `webhook:delivery`
- **Queue**: `webhooks` or `webhooks_retry`
- **Retry**: Up to 5 times with exponential backoff
- **Timeout**: 30 seconds

Delivers webhook notifications to merchants when payment events occur.

#### 2. Payment Expiry
- **Type**: `payment:expiry`
- **Queue**: `periodic`
- **Schedule**: Every 5 minutes (cron: `*/5 * * * *`)
- **Retry**: Up to 3 times
- **Timeout**: 5 minutes

Finds payments that have been pending for more than 30 minutes and marks them as expired.

#### 3. Balance Check
- **Type**: `wallet:balance_check`
- **Queue**: `monitoring`
- **Schedule**: Every 5 minutes (cron: `*/5 * * * *`)
- **Retry**: Up to 3 times
- **Timeout**: 2 minutes

Checks USDT and USDC balances in the hot wallet and logs alerts if:
- Balance falls below $1,000 (minimum threshold)
- Balance exceeds $10,000 (maximum threshold)

#### 4. Daily Settlement Report
- **Type**: `report:daily_settlement`
- **Queue**: `reports`
- **Schedule**: Daily at 8 AM (cron: `0 8 * * *`)
- **Retry**: Up to 3 times
- **Timeout**: 10 minutes

Generates a daily report with:
- Total payments and completed payments
- Total payouts and completed payouts
- Current wallet balance
- Net revenue

## Queue Priorities

Queues are processed by priority (higher number = higher priority):

| Queue | Priority | Purpose |
|-------|----------|---------|
| `webhooks` | 5 | Immediate webhook delivery |
| `periodic` | 4 | Scheduled payment expiry checks |
| `webhooks_retry` | 3 | Retried webhook deliveries |
| `monitoring` | 2 | Wallet balance monitoring |
| `reports` | 1 | Daily settlement reports |

## Configuration

The worker service is configured via environment variables:

```bash
# Redis (used as job broker)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=payment_gateway

# Solana (for balance monitoring)
SOLANA_RPC_URL=https://api.devnet.solana.com
SOLANA_WALLET_PRIVATE_KEY=<secret>
SOLANA_WALLET_ADDRESS=<public>
```

## Running the Worker

### Development
```bash
go run cmd/worker/main.go
```

### Production
```bash
# Build
make build-worker

# Run
./bin/worker
```

### Docker
```bash
docker-compose up worker
```

## Usage

### Enqueuing Jobs

```go
import (
    "context"
    "github.com/hxuan190/stable_payment_gateway/internal/worker"
)

// Create queue client
queue, err := worker.NewQueue(&worker.QueueConfig{
    RedisAddr:     "localhost:6379",
    RedisPassword: "",
    RedisDB:       0,
})
if err != nil {
    // handle error
}
defer queue.Close()

// Enqueue webhook delivery
err = queue.EnqueueWebhookDelivery(context.Background(), &worker.WebhookDeliveryPayload{
    MerchantID: "merchant_123",
    Event:      "payment.completed",
    Payload: map[string]interface{}{
        "payment_id": "pay_456",
        "amount_vnd": "2300000",
    },
    Attempt: 0,
})
```

### Monitoring Jobs

```go
// Get queue statistics
stats, err := queue.GetQueueStats(context.Background())
for queueName, info := range stats {
    fmt.Printf("Queue %s: %d active, %d pending\n",
        queueName, info.Active, info.Pending)
}

// Get specific task info
taskInfo, err := queue.GetTaskInfo("webhooks", "task_id_123")
```

## Testing

### Unit Tests
```bash
go test ./internal/worker/...
```

### Integration Tests
Ensure Redis is running:
```bash
docker-compose up -d redis
go test ./internal/worker/... -tags=integration
```

## Monitoring

The worker logs all job processing events with structured logging:

```json
{
  "level": "info",
  "message": "Webhook delivered successfully",
  "merchant_id": "merchant_123",
  "event": "payment.completed",
  "attempt": 0,
  "timestamp": "2025-11-17T10:30:00Z"
}
```

### Key Metrics to Monitor

- **Job Success Rate**: Percentage of jobs completed successfully
- **Job Latency**: Time from enqueue to completion
- **Queue Depth**: Number of pending jobs per queue
- **Retry Rate**: Percentage of jobs requiring retries
- **Wallet Balance**: Current hot wallet balance (from balance check job)

## Error Handling

- Jobs are automatically retried with exponential backoff
- Failed jobs are moved to dead letter queue after max retries
- All errors are logged with context
- Non-critical errors (e.g., webhook delivery failures) don't crash the worker

## Future Improvements

- [ ] Implement GetByDateRange in payment/payout repositories for accurate daily reports
- [ ] Add email notifications for wallet balance alerts
- [ ] Add email delivery for daily settlement reports
- [ ] Implement more granular monitoring and alerting
- [ ] Add support for BSC wallet balance checking
- [ ] Add job result persistence for audit trail
- [ ] Implement job priority adjustment based on merchant tier

## Related Documentation

- [asynq Documentation](https://github.com/hibiken/asynq)
- [Redis Documentation](https://redis.io/documentation)
- [Project Architecture](../../ARCHITECTURE.md)
- [Backend Task Breakdown](../../BACKEND_TASK_BREAKDOWN.md)
