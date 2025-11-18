# Modular Architecture Design

**Project**: Stablecoin Payment Gateway
**Architecture Pattern**: Modular Monolith
**Created**: 2025-11-18
**Status**: Implementation Plan

---

## 🎯 Overview

This document describes the modular architecture for the payment gateway. We're implementing a **modular monolith** - a single deployable application with well-defined module boundaries that can be extracted into microservices later.

### Why Modular Monolith?

✅ **Benefits**:
- **Simple Deployment**: Single binary, no distributed system complexity
- **Development Speed**: No network overhead, easy debugging
- **Clear Boundaries**: Enforced module separation prevents coupling
- **Future-Proof**: Easy migration to microservices when needed
- **Team Scalability**: Different teams can own different modules
- **Testing**: Modules can be tested independently

❌ **Traditional Monolith Problems We Avoid**:
- Tight coupling between components
- Unclear dependencies
- Difficult to extract services
- Hard to parallelize development

---

## 🏗️ Architecture Layers

### Layer 1: Modules (Business Domains)

Each module is a **bounded context** representing a business capability:

```
internal/modules/
├── payment/          # Payment creation, confirmation, lifecycle
├── merchant/         # Merchant registration, KYC, management
├── payout/           # Withdrawal requests, approvals, execution
├── blockchain/       # Multi-chain listeners, wallet management
├── compliance/       # AML, Travel Rule, transaction screening
├── ledger/           # Double-entry accounting, balance calculation
└── notification/     # Webhooks, emails, real-time updates
```

### Layer 2: Shared Kernel

Common utilities used across all modules:

```
internal/shared/
├── events/           # Event bus for inter-module communication
├── types/            # Common value objects (Money, Address, etc.)
├── errors/           # Standard error types
└── interfaces/       # Cross-module interfaces
```

### Layer 3: Infrastructure

Technical capabilities (unchanged from current structure):

```
internal/pkg/
├── database/         # PostgreSQL client
├── cache/            # Redis client
├── logger/           # Structured logging
├── storage/          # S3/MinIO file storage
└── ...               # Other utilities
```

---

## 📦 Module Structure

Each module follows this standardized structure:

```
modules/{module_name}/
├── domain/
│   ├── models.go              # Entities (Payment, Merchant, etc.)
│   ├── value_objects.go       # Value objects (PaymentStatus, etc.)
│   ├── events.go              # Domain events
│   └── errors.go              # Module-specific errors
│
├── service/
│   ├── service.go             # Main service interface
│   ├── {feature}_service.go   # Feature implementations
│   └── service_test.go        # Service tests
│
├── repository/
│   ├── repository.go          # Repository interface
│   ├── postgres.go            # PostgreSQL implementation
│   ├── cache.go               # Redis caching layer
│   └── repository_test.go     # Repository tests
│
├── handler/
│   ├── http.go                # HTTP handlers
│   ├── dto.go                 # Request/Response DTOs
│   ├── routes.go              # Route registration
│   └── handler_test.go        # Handler tests
│
├── events/
│   ├── publisher.go           # Event publishing
│   ├── subscriber.go          # Event handling
│   └── handlers.go            # Event handler implementations
│
└── module.go                  # Module initialization
```

### Module Initialization Pattern

Each module exposes a `Module` struct that encapsulates all dependencies:

```go
// modules/payment/module.go
package payment

import (
    "database/sql"
    "github.com/redis/go-redis/v9"
    "stable_payment_gateway/internal/shared/events"
)

// Module encapsulates all payment domain dependencies
type Module struct {
    Service    Service
    Repository Repository
    Handler    *Handler
    publisher  events.Publisher
}

// Config holds module configuration
type Config struct {
    DB        *sql.DB
    Cache     *redis.Client
    EventBus  events.Publisher
    Logger    *logrus.Logger
}

// NewModule initializes the payment module
func NewModule(cfg Config) (*Module, error) {
    // Initialize repository
    repo := NewRepository(cfg.DB, cfg.Cache)

    // Initialize service
    svc := NewService(ServiceConfig{
        Repository: repo,
        Logger:     cfg.Logger,
        EventBus:   cfg.EventBus,
    })

    // Initialize HTTP handler
    handler := NewHandler(svc, cfg.Logger)

    // Initialize event subscribers
    subscriber := NewEventSubscriber(svc, cfg.Logger)
    cfg.EventBus.Subscribe(subscriber)

    return &Module{
        Service:    svc,
        Repository: repo,
        Handler:    handler,
        publisher:  cfg.EventBus,
    }, nil
}

// RegisterRoutes registers HTTP routes for this module
func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
    m.Handler.RegisterRoutes(router)
}

// Shutdown gracefully shuts down the module
func (m *Module) Shutdown() error {
    // Cleanup resources
    return nil
}
```

---

## 🔄 Inter-Module Communication

### Rule: Modules MUST NOT Import Each Other Directly

❌ **Bad**: `payment` module imports `merchant` module directly
✅ **Good**: `payment` module uses shared interfaces and events

### Communication Patterns

#### 1. **Shared Interfaces** (Synchronous)

Define interfaces in `shared/interfaces/` for cross-module dependencies:

```go
// shared/interfaces/merchant.go
package interfaces

type MerchantReader interface {
    GetByID(ctx context.Context, id string) (*MerchantInfo, error)
    IsKYCApproved(ctx context.Context, id string) (bool, error)
}

type MerchantInfo struct {
    ID          string
    KYCStatus   string
    KYCTier     int
    IsActive    bool
}
```

Payment module uses the interface:

```go
// modules/payment/service/service.go
package service

import "stable_payment_gateway/internal/shared/interfaces"

type Service struct {
    repo           Repository
    merchantReader interfaces.MerchantReader  // Interface, not direct import
    eventBus       events.Publisher
}

func (s *Service) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*Payment, error) {
    // Use interface to check merchant status
    merchant, err := s.merchantReader.GetByID(ctx, req.MerchantID)
    if err != nil {
        return nil, err
    }

    if !merchant.IsActive {
        return nil, ErrMerchantInactive
    }

    // ... rest of logic
}
```

Merchant module implements the interface:

```go
// modules/merchant/service/service.go
package service

// Ensure MerchantService implements the shared interface
var _ interfaces.MerchantReader = (*Service)(nil)

func (s *Service) GetByID(ctx context.Context, id string) (*interfaces.MerchantInfo, error) {
    merchant, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Convert domain model to shared interface type
    return &interfaces.MerchantInfo{
        ID:        merchant.ID,
        KYCStatus: merchant.KYCStatus,
        KYCTier:   merchant.KYCTier,
        IsActive:  merchant.IsActive,
    }, nil
}
```

#### 2. **Event Bus** (Asynchronous)

For eventual consistency and decoupled communication:

```go
// modules/payment/domain/events.go
package domain

type PaymentConfirmedEvent struct {
    PaymentID   string
    MerchantID  string
    AmountVND   decimal.Decimal
    AmountCrypto decimal.Decimal
    Chain       string
    TxHash      string
    ConfirmedAt time.Time
}

func (e PaymentConfirmedEvent) Name() string {
    return "payment.confirmed"
}
```

Payment module publishes event:

```go
// modules/payment/service/payment_service.go
func (s *Service) ConfirmPayment(ctx context.Context, paymentID string) error {
    // ... confirm payment logic

    // Publish event
    event := domain.PaymentConfirmedEvent{
        PaymentID:    payment.ID,
        MerchantID:   payment.MerchantID,
        AmountVND:    payment.AmountVND,
        AmountCrypto: payment.AmountCrypto,
        Chain:        payment.Chain,
        TxHash:       payment.TxHash,
        ConfirmedAt:  time.Now(),
    }

    if err := s.eventBus.Publish(ctx, event); err != nil {
        s.logger.Error("Failed to publish PaymentConfirmedEvent", err)
        // Don't fail the operation, just log
    }

    return nil
}
```

Ledger module subscribes to event:

```go
// modules/ledger/events/subscriber.go
package events

type Subscriber struct {
    ledgerService *service.Service
    logger        *logrus.Logger
}

func (s *Subscriber) HandlePaymentConfirmed(ctx context.Context, event domain.PaymentConfirmedEvent) error {
    // Create ledger entries when payment is confirmed
    return s.ledgerService.RecordPaymentConfirmed(ctx, RecordPaymentRequest{
        PaymentID:    event.PaymentID,
        MerchantID:   event.MerchantID,
        AmountVND:    event.AmountVND,
        AmountCrypto: event.AmountCrypto,
    })
}

// Subscribe registers all event handlers
func (s *Subscriber) Subscribe(bus events.EventBus) {
    bus.On("payment.confirmed", s.HandlePaymentConfirmed)
}
```

---

## 📋 Domain Module Boundaries

### Payment Module

**Responsibilities**:
- Create payment requests
- Generate QR codes
- Validate payment confirmations
- Manage payment lifecycle (created → pending → confirmed → completed)
- Calculate exchange rates

**Dependencies**:
- `interfaces.MerchantReader` - Check merchant status
- `interfaces.ComplianceChecker` - Validate transactions
- `events.Publisher` - Emit payment events

**Events Published**:
- `payment.created`
- `payment.confirmed`
- `payment.expired`
- `payment.failed`

**Events Subscribed**:
- `blockchain.transaction_detected`
- `compliance.transaction_approved`
- `compliance.transaction_rejected`

---

### Merchant Module

**Responsibilities**:
- Merchant registration
- KYC document upload and review
- Merchant authentication (API keys, JWT)
- Merchant settings management

**Dependencies**:
- `interfaces.StorageService` - Store KYC documents
- `events.Publisher` - Emit merchant events

**Events Published**:
- `merchant.registered`
- `merchant.kyc_submitted`
- `merchant.kyc_approved`
- `merchant.kyc_rejected`

**Events Subscribed**:
- (None - root aggregate)

---

### Payout Module

**Responsibilities**:
- Payout request creation
- Payout approval workflow
- Payout execution tracking
- Bank account validation

**Dependencies**:
- `interfaces.MerchantReader` - Get merchant details
- `interfaces.LedgerReader` - Check available balance
- `events.Publisher` - Emit payout events

**Events Published**:
- `payout.requested`
- `payout.approved`
- `payout.rejected`
- `payout.completed`
- `payout.failed`

**Events Subscribed**:
- `payment.confirmed` - Update available balance
- `ledger.balance_updated` - Validate payout eligibility

---

### Blockchain Module

**Responsibilities**:
- Listen to Solana, BSC, other chains
- Detect incoming transactions
- Validate transaction details (amount, memo)
- Manage hot wallet operations
- Sign and broadcast transactions

**Dependencies**:
- `interfaces.PaymentReader` - Validate payment exists
- `events.Publisher` - Emit blockchain events

**Events Published**:
- `blockchain.transaction_detected`
- `blockchain.transaction_confirmed`
- `blockchain.transaction_finalized`

**Events Subscribed**:
- `payout.approved` - Execute blockchain payout

---

### Compliance Module

**Responsibilities**:
- AML screening (TRM Labs integration)
- FATF Travel Rule validation
- Transaction limits enforcement
- Risk scoring
- Sanctions list checking

**Dependencies**:
- `interfaces.MerchantReader` - Get KYC tier
- `interfaces.TransactionReader` - Get transaction history
- `events.Publisher` - Emit compliance events

**Events Published**:
- `compliance.transaction_approved`
- `compliance.transaction_rejected`
- `compliance.high_risk_detected`

**Events Subscribed**:
- `payment.created` - Screen new payments
- `payout.requested` - Screen payout requests

---

### Ledger Module

**Responsibilities**:
- Double-entry accounting
- Balance calculations
- Fee recording
- Ledger entry immutability
- Balance reconciliation

**Dependencies**:
- `events.Publisher` - Emit ledger events

**Events Published**:
- `ledger.entry_created`
- `ledger.balance_updated`

**Events Subscribed**:
- `payment.confirmed` - Credit merchant balance
- `payout.completed` - Debit merchant balance
- `payment.created` - Reserve funds

---

### Notification Module

**Responsibilities**:
- Webhook delivery with retries
- Email notifications
- Real-time WebSocket updates
- Notification templates

**Dependencies**:
- `interfaces.MerchantReader` - Get webhook URLs
- `events.Publisher` - Emit notification events

**Events Published**:
- `notification.sent`
- `notification.failed`

**Events Subscribed**:
- `payment.confirmed` - Notify merchant
- `payout.completed` - Notify merchant
- `merchant.kyc_approved` - Send email

---

## 🔌 Event Bus Implementation

### In-Memory Event Bus (MVP)

For the modular monolith, we'll use an in-memory event bus:

```go
// shared/events/bus.go
package events

import (
    "context"
    "sync"
)

type Event interface {
    Name() string
}

type Handler func(ctx context.Context, event Event) error

type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventName string, handler Handler)
    Publisher
    Subscriber
}

type Publisher interface {
    Publish(ctx context.Context, event Event) error
}

type Subscriber interface {
    Subscribe(eventName string, handler Handler)
}

type InMemoryEventBus struct {
    handlers map[string][]Handler
    mu       sync.RWMutex
    logger   *logrus.Logger
}

func NewInMemoryEventBus(logger *logrus.Logger) *InMemoryEventBus {
    return &InMemoryEventBus{
        handlers: make(map[string][]Handler),
        logger:   logger,
    }
}

func (b *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
    b.mu.RLock()
    handlers, exists := b.handlers[event.Name()]
    b.mu.RUnlock()

    if !exists {
        b.logger.Debugf("No handlers for event: %s", event.Name())
        return nil
    }

    // Execute handlers asynchronously
    for _, handler := range handlers {
        go func(h Handler) {
            if err := h(ctx, event); err != nil {
                b.logger.Errorf("Event handler failed for %s: %v", event.Name(), err)
            }
        }(handler)
    }

    return nil
}

func (b *InMemoryEventBus) Subscribe(eventName string, handler Handler) {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.handlers[eventName] = append(b.handlers[eventName], handler)
    b.logger.Infof("Subscribed handler to event: %s", eventName)
}
```

### Future: External Message Broker

When migrating to microservices, swap in Redis Streams, RabbitMQ, or Kafka:

```go
// shared/events/redis_bus.go
package events

type RedisEventBus struct {
    client *redis.Client
    logger *logrus.Logger
}

func (b *RedisEventBus) Publish(ctx context.Context, event Event) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    return b.client.Publish(ctx, event.Name(), data).Err()
}

func (b *RedisEventBus) Subscribe(eventName string, handler Handler) {
    pubsub := b.client.Subscribe(context.Background(), eventName)

    go func() {
        for msg := range pubsub.Channel() {
            var event Event
            if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
                b.logger.Error(err)
                continue
            }

            if err := handler(context.Background(), event); err != nil {
                b.logger.Error(err)
            }
        }
    }()
}
```

---

## 🚀 Migration Path to Microservices

### Step 1: Modular Monolith (Current)
- All modules in single codebase
- In-memory event bus
- Single database
- Single deployment

### Step 2: Extract First Service (e.g., Blockchain Listener)
1. Copy `modules/blockchain/` to new repository
2. Replace in-memory event bus with Redis/RabbitMQ
3. Keep shared database initially (shared schema)
4. Deploy blockchain service separately
5. Update main app to call blockchain service via HTTP/gRPC

### Step 3: Database Per Service
1. Extract blockchain-related tables to separate database
2. Use events for data synchronization
3. Implement Saga pattern for distributed transactions

### Step 4: Repeat for Other Modules
- Extract compliance module (heavy external API calls)
- Extract notification module (high-volume, separate concerns)
- Keep core modules (payment, merchant, payout) together longer

---

## 📂 File Organization

### Current Structure
```
internal/
├── api/
│   ├── handler/
│   │   ├── payment.go
│   │   ├── merchant.go
│   │   ├── payout.go
│   └── middleware/
├── service/
│   ├── payment.go
│   ├── merchant.go
│   ├── payout.go
├── repository/
│   ├── payment.go
│   ├── merchant.go
│   ├── payout.go
└── model/
    ├── payment.go
    ├── merchant.go
    ├── payout.go
```

### Target Modular Structure
```
internal/
├── modules/
│   ├── payment/
│   │   ├── domain/
│   │   │   ├── payment.go              # Payment entity
│   │   │   ├── payment_status.go        # Value object
│   │   │   ├── events.go               # Domain events
│   │   │   └── errors.go               # Payment-specific errors
│   │   ├── service/
│   │   │   ├── service.go              # Interface
│   │   │   ├── payment_service.go      # Implementation
│   │   │   └── service_test.go
│   │   ├── repository/
│   │   │   ├── repository.go           # Interface
│   │   │   ├── postgres.go             # Implementation
│   │   │   └── repository_test.go
│   │   ├── handler/
│   │   │   ├── http.go                 # HTTP handlers
│   │   │   ├── dto.go                  # DTOs
│   │   │   └── routes.go
│   │   ├── events/
│   │   │   └── subscriber.go           # Event handlers
│   │   └── module.go                   # Module initialization
│   │
│   ├── merchant/
│   │   ├── domain/
│   │   ├── service/
│   │   ├── repository/
│   │   ├── handler/
│   │   ├── events/
│   │   └── module.go
│   │
│   ├── payout/
│   ├── blockchain/
│   ├── compliance/
│   ├── ledger/
│   └── notification/
│
├── shared/
│   ├── events/
│   │   ├── bus.go                      # Event bus interface
│   │   ├── inmemory.go                 # In-memory implementation
│   │   └── redis.go                    # Redis implementation (future)
│   ├── types/
│   │   ├── money.go                    # Money value object
│   │   ├── address.go                  # Address value object
│   │   └── pagination.go               # Pagination types
│   ├── errors/
│   │   ├── errors.go                   # Standard error types
│   │   └── codes.go                    # Error codes
│   └── interfaces/
│       ├── merchant.go                 # Merchant interfaces
│       ├── payment.go                  # Payment interfaces
│       ├── compliance.go               # Compliance interfaces
│       └── storage.go                  # Storage interfaces
│
└── pkg/                                # Infrastructure (unchanged)
```

---

## 🧪 Testing Strategy

### Unit Tests (Module Level)

Each module can be tested independently:

```go
// modules/payment/service/service_test.go
func TestPaymentService_CreatePayment(t *testing.T) {
    // Arrange
    mockRepo := &MockRepository{}
    mockMerchantReader := &MockMerchantReader{}
    mockEventBus := &MockEventBus{}

    svc := NewService(ServiceConfig{
        Repository:     mockRepo,
        MerchantReader: mockMerchantReader,
        EventBus:       mockEventBus,
    })

    // Act
    payment, err := svc.CreatePayment(context.Background(), CreatePaymentRequest{
        MerchantID: "merchant-123",
        AmountVND:  decimal.NewFromInt(1000000),
    })

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, payment)
    assert.Equal(t, "created", payment.Status)
}
```

### Integration Tests (Cross-Module)

Test event-driven interactions:

```go
// tests/integration/payment_ledger_test.go
func TestPaymentConfirmation_UpdatesLedger(t *testing.T) {
    // Setup in-memory event bus
    eventBus := events.NewInMemoryEventBus(logger)

    // Initialize modules
    paymentModule := payment.NewModule(/* ... */)
    ledgerModule := ledger.NewModule(/* ... */)

    // Simulate payment confirmation
    err := paymentModule.Service.ConfirmPayment(ctx, paymentID)
    assert.NoError(t, err)

    // Wait for event processing
    time.Sleep(100 * time.Millisecond)

    // Verify ledger was updated
    balance, err := ledgerModule.Service.GetBalance(ctx, merchantID)
    assert.NoError(t, err)
    assert.Equal(t, expectedBalance, balance)
}
```

---

## 📊 Dependency Graph

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│  cmd/api, cmd/listener, cmd/worker, cmd/admin               │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
┌────────▼───────────────────────────────▼─────────────────┐
│                    Module Layer                           │
│  ┌──────────┐  ┌─────────┐  ┌────────┐  ┌──────────┐    │
│  │ Payment  │  │Merchant │  │ Payout │  │Blockchain│    │
│  └────┬─────┘  └────┬────┘  └───┬────┘  └────┬─────┘    │
│       │             │           │            │           │
│  ┌────▼─────┐  ┌───▼──────┐  ┌─▼────────┐  ┌▼───────┐  │
│  │Compliance│  │  Ledger  │  │Notification│ │ ...    │  │
│  └──────────┘  └──────────┘  └────────────┘ └────────┘  │
│                                                           │
│         (Modules communicate via Events & Interfaces)     │
└────────────────────────┬──────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
┌────────▼─────────┐           ┌────────▼──────────┐
│  Shared Kernel   │           │  Infrastructure   │
│  ┌────────────┐  │           │  ┌──────────┐     │
│  │Event Bus   │  │           │  │Database  │     │
│  │Types       │  │           │  │Cache     │     │
│  │Errors      │  │           │  │Logger    │     │
│  │Interfaces  │  │           │  │Storage   │     │
│  └────────────┘  │           │  └──────────┘     │
└──────────────────┘           └───────────────────┘
```

---

## ✅ Implementation Checklist

### Phase 1: Foundation
- [ ] Create `internal/shared/` structure
- [ ] Implement in-memory event bus
- [ ] Define shared error types
- [ ] Define common value objects (Money, Address)
- [ ] Define cross-module interfaces

### Phase 2: Refactor Core Modules
- [ ] Payment module
- [ ] Merchant module
- [ ] Payout module

### Phase 3: Refactor Supporting Modules
- [ ] Blockchain module
- [ ] Compliance module
- [ ] Ledger module
- [ ] Notification module

### Phase 4: Integration
- [ ] Update `cmd/api` to use modules
- [ ] Update `cmd/listener` to use modules
- [ ] Update `cmd/worker` to use modules
- [ ] Update `cmd/admin` to use modules

### Phase 5: Testing & Documentation
- [ ] Write integration tests
- [ ] Update API documentation
- [ ] Create module dependency diagram
- [ ] Write migration guide for developers

---

## 📚 References

- [Modular Monolith Architecture](https://www.kamilgrzybek.com/design/modular-monolith-primer/)
- [Domain-Driven Design](https://martinfowler.com/tags/domain%20driven%20design.html)
- [Event-Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html)
- [Monolith to Microservices](https://martinfowler.com/articles/break-monolith-into-microservices.html)

---

**Next Steps**: Proceed with Phase 1 implementation - Create shared kernel infrastructure
