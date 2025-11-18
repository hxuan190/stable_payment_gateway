# Modules

This directory contains the business domain modules for the payment gateway. Each module represents a **bounded context** and is independently testable and deployable.

## Module Structure

Each module follows a consistent layered architecture:

```
{module_name}/
├── domain/          # Domain models, events, value objects
├── service/         # Business logic (core of the module)
├── repository/      # Data access layer
├── handler/         # HTTP handlers (API endpoints)
├── events/          # Event subscribers (handle events from other modules)
└── module.go        # Module initialization and wiring
```

## Available Modules

### 1. Payment Module (`payment/`)

**Responsibilities**:
- Create payment requests
- Generate QR codes for crypto payments
- Validate payment confirmations
- Manage payment lifecycle
- Calculate exchange rates

**Dependencies**:
- `interfaces.MerchantReader` - Verify merchant status
- `interfaces.ComplianceChecker` - Screen transactions
- `events.EventBus` - Publish payment events

**Events Published**:
- `payment.created` - When a new payment is created
- `payment.confirmed` - When blockchain transaction is confirmed
- `payment.expired` - When payment expires (30 min timeout)
- `payment.failed` - When payment processing fails

**Events Subscribed**:
- `blockchain.transaction_detected` - From blockchain module
- `compliance.transaction_approved` - From compliance module
- `compliance.transaction_rejected` - From compliance module

---

### 2. Merchant Module (`merchant/`)

**Responsibilities**:
- Merchant registration and onboarding
- KYC document management
- API key generation and management
- Webhook configuration

**Dependencies**:
- `interfaces.StorageService` - Store KYC documents
- `events.EventBus` - Publish merchant events

**Events Published**:
- `merchant.registered`
- `merchant.kyc_submitted`
- `merchant.kyc_approved`
- `merchant.kyc_rejected`

---

### 3. Payout Module (`payout/`)

**Responsibilities**:
- Payout request creation
- Approval workflow management
- Bank transfer tracking
- Balance validation

**Dependencies**:
- `interfaces.MerchantReader` - Get merchant details
- `interfaces.LedgerReader` - Check available balance
- `events.EventBus` - Publish payout events

**Events Published**:
- `payout.requested`
- `payout.approved`
- `payout.rejected`
- `payout.completed`
- `payout.failed`

**Events Subscribed**:
- `payment.confirmed` - Update available balance
- `ledger.balance_updated` - Revalidate payout eligibility

---

### 4. Blockchain Module (`blockchain/`)

**Responsibilities**:
- Listen to blockchain networks (Solana, BSC)
- Detect incoming transactions
- Validate transaction amounts and memos
- Manage hot wallet operations
- Sign and broadcast transactions

**Dependencies**:
- `interfaces.PaymentReader` - Validate payment exists
- `events.EventBus` - Publish blockchain events

**Events Published**:
- `blockchain.transaction_detected`
- `blockchain.transaction_confirmed`
- `blockchain.transaction_finalized`

**Events Subscribed**:
- `payout.approved` - Execute blockchain payout

---

### 5. Compliance Module (`compliance/`)

**Responsibilities**:
- AML screening via TRM Labs
- FATF Travel Rule validation
- Transaction limit enforcement
- Risk scoring and analysis
- Sanctions list checking

**Dependencies**:
- `interfaces.MerchantReader` - Get KYC tier
- `interfaces.TransactionReader` - Get transaction history
- `events.EventBus` - Publish compliance events

**Events Published**:
- `compliance.transaction_approved`
- `compliance.transaction_rejected`
- `compliance.high_risk_detected`

**Events Subscribed**:
- `payment.created` - Screen new payments
- `payout.requested` - Screen payout requests

---

### 6. Ledger Module (`ledger/`)

**Responsibilities**:
- Double-entry bookkeeping
- Balance calculations
- Fee recording
- Ledger entry immutability
- Financial reconciliation

**Dependencies**:
- `events.EventBus` - Publish ledger events

**Events Published**:
- `ledger.entry_created`
- `ledger.balance_updated`

**Events Subscribed**:
- `payment.confirmed` - Credit merchant balance
- `payout.completed` - Debit merchant balance
- `payment.created` - Reserve funds

---

### 7. Notification Module (`notification/`)

**Responsibilities**:
- Webhook delivery with retries
- Email notifications
- Real-time WebSocket updates
- Notification templates

**Dependencies**:
- `interfaces.MerchantWebhookProvider` - Get webhook config
- `events.EventBus` - Publish notification events

**Events Published**:
- `notification.sent`
- `notification.failed`

**Events Subscribed**:
- `payment.confirmed` - Notify merchant
- `payout.completed` - Notify merchant
- `merchant.kyc_approved` - Send email

---

## Module Communication Rules

### Rule #1: No Direct Module Imports

Modules **MUST NOT** import each other directly. Instead, use:

1. **Shared Interfaces** (for synchronous calls)
2. **Event Bus** (for asynchronous communication)

❌ **Bad**:
```go
import "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant"

merchant := merchantModule.Service.GetByID(id)
```

✅ **Good**:
```go
import "github.com/hxuan190/stable_payment_gateway/internal/shared/interfaces"

type PaymentService struct {
    merchantReader interfaces.MerchantReader
}

merchant, err := s.merchantReader.GetByID(ctx, id)
```

### Rule #2: Use Events for Eventual Consistency

When an operation doesn't require immediate consistency, use events:

```go
// After confirming payment, publish event
event := domain.NewPaymentConfirmedEvent(
    payment.ID,
    payment.MerchantID,
    payment.AmountVND,
    payment.AmountCrypto,
    payment.Chain,
    payment.Token,
    txHash,
    time.Now(),
)
s.eventBus.Publish(ctx, event)

// Ledger module will handle this asynchronously
```

### Rule #3: Domain Events Live in Domain Package

All domain events must be defined in `{module}/domain/events.go`:

```go
// modules/payment/domain/events.go
type PaymentConfirmedEvent struct {
    events.BaseEvent
    PaymentID  string
    MerchantID string
    Amount     decimal.Decimal
}

func NewPaymentConfirmedEvent(...) *PaymentConfirmedEvent {
    return &PaymentConfirmedEvent{
        BaseEvent: events.NewBaseEvent("payment.confirmed"),
        // ... fields
    }
}
```

## Module Initialization

Each module is initialized in `cmd/api/main.go` (or other main applications):

```go
// Initialize event bus
eventBus := events.NewInMemoryEventBus(logger)

// Initialize modules
merchantModule, err := merchant.NewModule(merchant.Config{
    DB:       db,
    Cache:    redisClient,
    EventBus: eventBus,
    Logger:   logger,
})

paymentModule, err := payment.NewModule(payment.Config{
    DB:                db,
    Cache:             redisClient,
    EventBus:          eventBus,
    Logger:            logger,
    MerchantReader:    merchantModule.Service,  // Inject as interface
    ComplianceChecker: complianceModule.Service,
})

// Register routes
api := router.Group("/api/v1")
paymentModule.RegisterRoutes(api)
merchantModule.RegisterRoutes(api)
```

## Testing Modules

### Unit Tests

Each layer can be tested independently:

```go
// Test service with mocks
func TestPaymentService_CreatePayment(t *testing.T) {
    mockRepo := &MockRepository{}
    mockMerchantReader := &MockMerchantReader{}
    mockEventBus := &MockEventBus{}

    svc := service.NewService(service.ServiceConfig{
        Repository:     mockRepo,
        MerchantReader: mockMerchantReader,
        EventBus:       mockEventBus,
    })

    // Test logic
}
```

### Integration Tests

Test cross-module interactions:

```go
// Test payment + ledger integration
func TestPaymentConfirmation_UpdatesLedger(t *testing.T) {
    // Initialize real event bus
    eventBus := events.NewInMemoryEventBus(logger)

    // Initialize modules
    paymentModule := payment.NewModule(/* ... */)
    ledgerModule := ledger.NewModule(/* ... */)

    // Trigger payment confirmation
    err := paymentModule.Service.ConfirmPayment(ctx, paymentID, txHash)

    // Wait for async event processing
    time.Sleep(100 * time.Millisecond)

    // Verify ledger was updated
    balance, err := ledgerModule.Service.GetBalance(ctx, merchantID)
    assert.Equal(t, expectedBalance, balance)
}
```

## Migration from Current Structure

To refactor existing code into modules:

1. **Create module structure**: `mkdir -p internal/modules/{name}/{domain,service,repository,handler,events}`
2. **Move domain models** to `domain/` package
3. **Move service logic** to `service/` package
4. **Move repository** to `repository/` package
5. **Move HTTP handlers** to `handler/` package
6. **Extract interfaces** to `shared/interfaces/`
7. **Create events** in `domain/events.go`
8. **Wire up module** in `module.go`
9. **Update main.go** to use the module

## Benefits of Modular Architecture

✅ **Clear Boundaries**: Each module has well-defined responsibilities

✅ **Independent Testing**: Modules can be tested in isolation

✅ **Team Scalability**: Different teams can work on different modules

✅ **Easy Extraction**: Modules can become microservices with minimal changes

✅ **Reduced Coupling**: Modules interact through interfaces and events

✅ **Better Organization**: Code is organized by business domain, not technical layer

---

**See Also**:
- [Modular Architecture Documentation](../../MODULAR_ARCHITECTURE.md)
- [Shared Kernel Documentation](../shared/README.md)

**Last Updated**: 2025-11-18
