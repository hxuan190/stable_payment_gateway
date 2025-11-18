# Shared Kernel

This directory contains the **shared kernel** for the modular monolith architecture. The shared kernel provides common functionality that all modules can use without creating tight coupling.

## Directory Structure

```
shared/
├── events/          # Event bus for inter-module communication
├── types/           # Common value objects (Money, Address, Pagination)
├── errors/          # Standard error types and codes
└── interfaces/      # Cross-module interfaces
```

## Components

### Events

The event bus enables asynchronous, decoupled communication between modules.

**Usage Example**:

```go
// Publishing an event
event := domain.PaymentConfirmedEvent{
    PaymentID:  "pay-123",
    MerchantID: "merchant-456",
    Amount:     decimal.NewFromInt(1000000),
}

err := eventBus.Publish(ctx, event)
```

```go
// Subscribing to an event
eventBus.Subscribe("payment.confirmed", func(ctx context.Context, event events.Event) error {
    paymentEvent := event.(domain.PaymentConfirmedEvent)
    // Handle the event
    return nil
})
```

**Implementations**:
- `InMemoryEventBus`: For modular monolith (current)
- Future: Redis, RabbitMQ, Kafka for microservices

### Types

Common value objects to ensure consistency across modules.

#### Money

```go
// Create money values
amountVND := types.VND(decimal.NewFromInt(1000000))
amountUSDT := types.USDT(decimal.NewFromFloat(43.48))

// Arithmetic operations
total, err := amountVND.Add(otherAmount)
fee := amountVND.Mul(decimal.NewFromFloat(0.01)) // 1% fee

// Comparisons
if amountVND.IsPositive() {
    // ...
}
```

#### Pagination

```go
params := types.PaginationParams{
    Page:     1,
    PageSize: 20,
}

// Get database offset
offset := params.Offset()
limit := params.GetPageSize()

// Create response metadata
meta := types.NewPaginationMeta(params, totalItems)
```

#### BlockchainAddress

```go
addr := types.NewBlockchainAddress("0x1234...", "BSC")
if err := addr.Validate(); err != nil {
    // Invalid address
}
```

### Errors

Standard error types with HTTP status codes.

**Usage Example**:

```go
// Create specific errors
return errors.NotFound("Payment")
return errors.Validation("Invalid amount")
return errors.InsufficientBalance("Not enough funds")

// Add details
err := errors.Validation("Invalid input").
    WithDetails("field", "email").
    WithDetails("value", email)

// Wrap existing errors
return errors.Wrap(dbErr, errors.ErrCodeInternal, "Database error", 500)

// Check error type
if errors.IsAppError(err) {
    appErr := errors.GetAppError(err)
    statusCode := appErr.StatusCode
}
```

### Interfaces

Cross-module interfaces enable modules to communicate without direct dependencies.

#### MerchantReader

```go
type MerchantReader interface {
    GetByID(ctx context.Context, id string) (*MerchantInfo, error)
    IsKYCApproved(ctx context.Context, merchantID string) (bool, error)
    IsActive(ctx context.Context, merchantID string) (bool, error)
}

// Usage in payment module
merchant, err := merchantReader.GetByID(ctx, merchantID)
if !merchant.IsActive {
    return errors.Forbidden("Merchant account is inactive")
}
```

#### PaymentReader

```go
type PaymentReader interface {
    GetByID(ctx context.Context, id string) (*PaymentInfo, error)
    GetByMemo(ctx context.Context, memo string) (*PaymentInfo, error)
    IsExpired(ctx context.Context, paymentID string) (bool, error)
}

// Usage in blockchain module
payment, err := paymentReader.GetByMemo(ctx, txMemo)
if payment.Status != "pending" {
    return errors.InvalidStatus("Payment already processed")
}
```

#### ComplianceChecker

```go
type ComplianceChecker interface {
    ScreenTransaction(ctx context.Context, req TransactionScreeningRequest) (*TransactionScreeningResult, error)
    CheckTransactionLimit(ctx context.Context, merchantID string, amount decimal.Decimal, currency string) error
}

// Usage in payment module
result, err := complianceChecker.ScreenTransaction(ctx, TransactionScreeningRequest{
    TransactionID: payment.ID,
    MerchantID:    payment.MerchantID,
    Amount:        payment.AmountCrypto,
    Currency:      payment.Token,
})

if !result.Approved {
    return errors.Forbidden("Transaction rejected by compliance")
}
```

#### LedgerReader/Writer

```go
type LedgerReader interface {
    GetBalance(ctx context.Context, merchantID string) (*BalanceInfo, error)
    HasSufficientBalance(ctx context.Context, merchantID string, amount decimal.Decimal) (bool, error)
}

type LedgerWriter interface {
    RecordPaymentConfirmed(ctx context.Context, req LedgerEntryRequest) error
    RecordPayoutCompleted(ctx context.Context, req LedgerEntryRequest) error
}

// Usage in payout module
hasFunds, err := ledgerReader.HasSufficientBalance(ctx, merchantID, payoutAmount)
if !hasFunds {
    return errors.InsufficientBalance("Merchant balance too low")
}
```

## Design Principles

### 1. No Module Dependencies

Modules **MUST NOT** import each other directly. Use shared interfaces and events instead.

❌ **Bad**:
```go
// payment module importing merchant module
import "stable_payment_gateway/internal/modules/merchant"

merchant := merchantModule.GetByID(id)
```

✅ **Good**:
```go
// payment module using shared interface
import "stable_payment_gateway/internal/shared/interfaces"

type PaymentService struct {
    merchantReader interfaces.MerchantReader
}

merchant, err := s.merchantReader.GetByID(ctx, id)
```

### 2. Interface Segregation

Interfaces should be small and focused. Separate read and write operations.

✅ **Good**:
```go
type PaymentReader interface {
    GetByID(ctx context.Context, id string) (*PaymentInfo, error)
}

type PaymentWriter interface {
    UpdateStatus(ctx context.Context, id string, status string) error
}
```

### 3. Value Objects for Data Transfer

Use value objects from `types/` package for consistent data representation.

```go
// Use Money instead of decimal.Decimal directly
amount := types.VND(decimal.NewFromInt(1000000))

// Use BlockchainAddress instead of string
walletAddr := types.NewBlockchainAddress(address, "SOLANA")
```

### 4. Event-Driven for Eventual Consistency

Use events for operations that don't require immediate consistency.

```go
// After confirming payment, publish event
event := PaymentConfirmedEvent{
    PaymentID:  payment.ID,
    MerchantID: payment.MerchantID,
    Amount:     payment.AmountVND,
}
eventBus.Publish(ctx, event)

// Ledger module subscribes and updates balance asynchronously
```

## Testing

All shared components have comprehensive unit tests:

```bash
# Test event bus
go test ./internal/shared/events/...

# Test error handling
go test ./internal/shared/errors/...

# Test all shared components
go test ./internal/shared/...
```

## Migration from Current Structure

When refactoring modules to use the shared kernel:

1. **Replace direct imports** with shared interfaces
2. **Use event bus** instead of direct function calls for async operations
3. **Use shared error types** for consistent error handling
4. **Use Money type** instead of decimal.Decimal for all monetary values

## Future Enhancements

### Redis Event Bus

For microservices deployment, swap in Redis-based event bus:

```go
eventBus := events.NewRedisEventBus(redisClient, logger)
```

The interface remains the same, so modules don't need changes.

### External Message Broker

For high-volume production:

```go
eventBus := events.NewKafkaEventBus(kafkaConfig, logger)
```

---

**Last Updated**: 2025-11-18
**Status**: ✅ Implemented and Tested
