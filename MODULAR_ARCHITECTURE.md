# Modular Architecture

**Last Updated**: 2025-11-23
**Status**: Active Implementation

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Architecture Principles](#architecture-principles)
3. [Directory Structure](#directory-structure)
4. [Module Structure](#module-structure)
5. [Communication Patterns](#communication-patterns)
6. [Current Status](#current-status)
7. [Migration Guide](#migration-guide)
8. [Best Practices](#best-practices)

---

## Overview

This codebase follows a **modular monolith** architecture, where the application is organized into independent modules representing business domains. Each module is:

- **Self-contained**: Contains all layers (domain, service, repository, handler)
- **Loosely coupled**: Communicates through interfaces and events
- **Independently testable**: Can be tested in isolation with mocks
- **Extractable**: Can be converted to a microservice with minimal changes

### Why Modular Architecture?

✅ **Clear boundaries**: Each module has well-defined responsibilities
✅ **Team scalability**: Different teams can work on different modules
✅ **Maintainability**: Changes are localized to specific modules
✅ **Future-proof**: Easy path to microservices if needed
✅ **Better testing**: Modules can be tested independently

---

## Architecture Principles

### 1. Hexagonal Architecture (Ports & Adapters)

Modules follow hexagonal architecture principles:

```
┌─────────────────────────────────────────┐
│         Payment Module                   │
├─────────────────────────────────────────┤
│                                          │
│  ┌────────────────────────────────┐    │
│  │       Domain Layer              │    │
│  │  (Business Logic & Entities)    │    │
│  │                                 │    │
│  │  - domain/payment.go            │    │
│  │  - domain/events.go             │    │
│  │  - domain/errors.go             │    │
│  │  - domain/repository.go (port)  │    │
│  └────────────────────────────────┘    │
│              ▲                          │
│              │                          │
│  ┌───────────┴──────────────┐          │
│  │    Service Layer          │          │
│  │  (Use Cases)              │          │
│  │                           │          │
│  │  - service/payment_service.go       │
│  └───────────┬──────────────┘          │
│              │                          │
│         ┌────┴─────┐                   │
│    ┌────▼────┐  ┌──▼───────┐          │
│    │ Adapter │  │ Adapter  │          │
│    │  (HTTP) │  │  (DB)    │          │
│    └─────────┘  └──────────┘          │
│                                         │
│  adapter/http/         adapter/repository/
│  - handler.go          - postgres.go    │
│  - dto.go                               │
└─────────────────────────────────────────┘
```

**Key Principles**:
- **Domain layer** contains core business logic (framework-agnostic)
- **Ports** define interfaces for external interactions
- **Adapters** implement ports (HTTP, database, external APIs)
- Dependencies point **inward** (adapters → domain, never outward)

### 2. Domain-Driven Design (DDD)

- Each module represents a **bounded context**
- **Ubiquitous language**: Code reflects business terminology
- **Aggregates**: Payment, Merchant, Payout are aggregates
- **Domain events**: Modules communicate via events

### 3. Dependency Inversion

Modules depend on **interfaces**, not concrete implementations:

```go
// ✅ GOOD: Payment module depends on interface
type PaymentService struct {
    merchantReader MerchantReader // interface
    repository     PaymentRepository // interface
}

// ❌ BAD: Direct dependency on another module
import "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant"
type PaymentService struct {
    merchantService *merchant.Service // concrete type
}
```

---

## Directory Structure

### Overall Structure

```
stable_payment_gateway/
├── cmd/                          # Application entry points
│   ├── api/                      # REST API server
│   ├── listener/                 # Blockchain listener service
│   ├── worker/                   # Background worker
│   └── admin/                    # Admin server
│
├── internal/
│   ├── modules/                  # 🆕 NEW: Business domain modules
│   │   ├── payment/              # Payment module
│   │   ├── merchant/             # Merchant module
│   │   ├── payout/               # Payout module
│   │   ├── ledger/               # Ledger module
│   │   ├── compliance/           # Compliance module
│   │   ├── blockchain/           # Blockchain module
│   │   ├── notification/         # Notification module
│   │   ├── README.md             # Module documentation
│   │   └── registry.go           # Module registry
│   │
│   ├── shared/                   # Shared kernel
│   │   ├── events/               # Event bus
│   │   ├── interfaces/           # Shared interfaces
│   │   └── types/                # Common value objects
│   │
│   ├── pkg/                      # Shared utilities
│   │   ├── database/             # Database connection
│   │   ├── cache/                # Redis cache
│   │   ├── logger/               # Logging
│   │   ├── crypto/               # Encryption
│   │   └── jwt/                  # JWT handling
│   │
│   ├── api/                      # HTTP server setup
│   │   ├── middleware/           # HTTP middleware
│   │   ├── websocket/            # WebSocket handlers
│   │   └── server.go             # Server initialization
│   │
│   ├── config/                   # Configuration
│   ├── model/                    # Shared database models
│   │
│   ├── service/                  # ⚠️  LEGACY: To be migrated
│   ├── repository/               # ⚠️  LEGACY: To be migrated
│   └── blockchain/               # ⚠️  LEGACY: To be migrated
│
├── migrations/                   # Database migrations
├── docs/                         # Documentation
├── web/                          # Frontend applications
└── scripts/                      # Utility scripts
```

### Module Structure

Each module follows this standard structure:

```
modules/{module_name}/
├── domain/                # Domain layer (core business logic)
│   ├── {entity}.go       # Domain entities/aggregates
│   ├── events.go         # Domain events
│   ├── errors.go         # Domain-specific errors
│   ├── repository.go     # Repository interface (port)
│   └── common.go         # Value objects, enums
│
├── service/              # Application layer (use cases)
│   └── {module}_service.go
│
├── adapter/              # Adapter layer (infrastructure)
│   ├── http/             # HTTP adapter (inbound)
│   │   ├── handler.go
│   │   └── dto.go
│   │
│   ├── repository/       # Database adapter (outbound)
│   │   └── postgres.go
│   │
│   └── legacy/           # ⚠️  Temporary adapters for old code
│       ├── merchant_adapter.go
│       └── compliance_adapter.go
│
├── port/                 # Port interfaces (optional)
│   └── service.go
│
├── events/               # Event subscribers (optional)
│   └── subscriber.go
│
├── handler/              # HTTP handlers (layered architecture variant)
│   └── {module}_handler.go
│
├── repository/           # Data access (layered architecture variant)
│   └── {module}_repository.go
│
└── module.go             # Module initialization & wiring
```

**Note**: Some modules use **hexagonal architecture** (payment) with `adapter/` and `port/`, while others use **layered architecture** (merchant, payout) with `handler/` and `repository/`. Both are valid; payment module is being refactored to hexagonal as an example.

---

## Module Structure

### Available Modules

#### 1. Payment Module (`modules/payment/`)

**Architecture**: Hexagonal (Ports & Adapters)

**Responsibilities**:
- Create payment requests
- Generate QR codes for crypto payments
- Validate payment confirmations
- Calculate exchange rates
- Manage payment lifecycle

**Structure**:
```
payment/
├── domain/                    # Core business logic
│   ├── payment.go            # Payment entity
│   ├── events.go             # PaymentCreated, PaymentConfirmed, etc.
│   ├── errors.go             # ErrPaymentNotFound, ErrInvalidAmount, etc.
│   ├── repository.go         # PaymentRepository interface
│   └── common.go             # PaymentStatus, Chain, Token enums
│
├── service/                   # Use cases
│   └── payment_service.go    # CreatePayment, ConfirmPayment, etc.
│
├── adapter/
│   ├── http/                 # Inbound adapter (API)
│   │   ├── handler.go        # HTTP handlers
│   │   └── dto.go            # Request/response DTOs
│   │
│   ├── repository/           # Outbound adapter (Database)
│   │   └── postgres.go       # PostgreSQL implementation
│   │
│   └── legacy/               # Temporary adapters for old code
│       ├── merchant_adapter.go
│       ├── compliance_adapter.go
│       ├── exchange_rate_adapter.go
│       └── payment_repository_adapter.go
│
├── port/                      # Port interfaces
│   └── service.go
│
└── module.go                  # Module initialization
```

**Dependencies (via interfaces)**:
- `MerchantReader` - Read merchant data
- `ExchangeRateProvider` - Get exchange rates
- `ComplianceChecker` - Screen transactions
- `AMLService` - AML compliance checks

**Events Published**:
- `payment.created` - New payment created
- `payment.confirmed` - Blockchain transaction confirmed
- `payment.expired` - Payment expired (30 min timeout)
- `payment.failed` - Payment processing failed

**Events Subscribed**:
- `blockchain.transaction_detected`
- `compliance.transaction_approved`
- `compliance.transaction_rejected`

---

#### 2. Merchant Module (`modules/merchant/`)

**Architecture**: Layered

**Responsibilities**:
- Merchant registration
- KYC management
- API key generation
- Webhook configuration

**Structure**:
```
merchant/
├── domain/
│   ├── merchant.go
│   └── events.go
├── service/
│   └── merchant_service.go
├── repository/
│   └── merchant_repository.go
├── handler/
│   └── merchant_handler.go
├── events/
│   └── subscriber.go
└── module.go
```

**Dependencies**: `StorageService`, `EventBus`

**Events Published**:
- `merchant.registered`
- `merchant.kyc_submitted`
- `merchant.kyc_approved`
- `merchant.kyc_rejected`

---

#### 3. Payout Module (`modules/payout/`)

**Responsibilities**:
- Payout request creation
- Approval workflow
- Bank transfer tracking
- Balance validation

**Dependencies**: `MerchantReader`, `LedgerReader`, `EventBus`

**Events Published**:
- `payout.requested`
- `payout.approved`
- `payout.rejected`
- `payout.completed`

**Events Subscribed**:
- `payment.confirmed` - Update available balance
- `ledger.balance_updated`

---

#### 4. Blockchain Module (`modules/blockchain/`)

**Responsibilities**:
- Listen to blockchain networks (Solana, BSC, TRON)
- Detect incoming transactions
- Validate amounts and memos
- Hot wallet management

**Structure**:
```
blockchain/
├── solana/               # Solana-specific code
│   ├── listener.go
│   ├── client.go
│   ├── parser.go
│   └── wallet.go
├── bsc/                  # BSC-specific code
│   ├── listener.go
│   └── client.go
├── tron/                 # TRON-specific code (planned)
└── module.go
```

**Events Published**:
- `blockchain.transaction_detected`
- `blockchain.transaction_confirmed`
- `blockchain.transaction_finalized`

---

#### 5. Compliance Module (`modules/compliance/`)

**Responsibilities**:
- AML screening (TRM Labs)
- FATF Travel Rule validation
- Transaction limit enforcement
- Risk scoring

**Dependencies**: `MerchantReader`, `TransactionReader`, `EventBus`

**Events Published**:
- `compliance.transaction_approved`
- `compliance.transaction_rejected`
- `compliance.high_risk_detected`

**Events Subscribed**:
- `payment.created`
- `payout.requested`

---

#### 6. Ledger Module (`modules/ledger/`)

**Responsibilities**:
- Double-entry bookkeeping
- Balance calculations
- Fee recording
- Immutable ledger entries

**Events Published**:
- `ledger.entry_created`
- `ledger.balance_updated`

**Events Subscribed**:
- `payment.confirmed`
- `payout.completed`

---

#### 7. Notification Module (`modules/notification/`)

**Responsibilities**:
- Webhook delivery
- Email notifications
- Real-time WebSocket updates
- Notification templates

**Events Subscribed**:
- `payment.confirmed`
- `payout.completed`
- `merchant.kyc_approved`

---

## Communication Patterns

### 1. Synchronous: Interfaces

For operations requiring immediate response:

```go
// Define interface in shared/interfaces/
type MerchantReader interface {
    GetByID(ctx context.Context, id string) (*Merchant, error)
    GetByAPIKey(ctx context.Context, apiKey string) (*Merchant, error)
}

// Inject in module initialization
func NewPaymentService(merchantReader MerchantReader, ...) *PaymentService {
    return &PaymentService{
        merchantReader: merchantReader,
    }
}

// Use in service
merchant, err := s.merchantReader.GetByID(ctx, merchantID)
```

### 2. Asynchronous: Events

For eventual consistency and decoupling:

```go
// Domain event
type PaymentConfirmedEvent struct {
    events.BaseEvent
    PaymentID    string
    MerchantID   string
    AmountVND    decimal.Decimal
    AmountCrypto decimal.Decimal
    TxHash       string
}

// Publisher (Payment module)
event := domain.NewPaymentConfirmedEvent(payment, txHash)
s.eventBus.Publish(ctx, event)

// Subscriber (Ledger module)
eventBus.Subscribe("payment.confirmed", func(ctx context.Context, event events.Event) error {
    // Handle payment confirmation
    ledgerService.RecordPayment(ctx, event.(*PaymentConfirmedEvent))
    return nil
})
```

### 3. Module Communication Rules

#### ❌ NEVER Do This:

```go
// Direct module import
import "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant"

type PaymentService struct {
    merchantService *merchant.Service // ❌ Tight coupling
}
```

#### ✅ ALWAYS Do This:

```go
// Use interface
import "github.com/hxuan190/stable_payment_gateway/internal/shared/interfaces"

type PaymentService struct {
    merchantReader interfaces.MerchantReader // ✅ Loose coupling
}
```

---

## Current Status

### ✅ Completed

- [x] Payment module (hexagonal architecture)
- [x] Module registry for centralized management
- [x] Event bus for async communication
- [x] Module README documentation
- [x] Payment module initialization
- [x] Legacy adapters for backward compatibility

### 🚧 In Progress

- [ ] Migrating handlers from `internal/api/handler/` to modules
- [ ] Cleaning up duplicate code in `internal/service/`
- [ ] Cleaning up duplicate code in `internal/repository/`
- [ ] Consolidating blockchain code to `modules/blockchain/`

### 📋 Planned

- [ ] Migrate all modules to hexagonal architecture
- [ ] Complete event subscriptions for all modules
- [ ] Add integration tests for cross-module communication
- [ ] Extract modules into microservices (future)

---

## Migration Guide

### From Old Structure to Modules

#### Step 1: Identify the Module

Determine which module the code belongs to:
- Payment creation/confirmation → `payment`
- Merchant registration/KYC → `merchant`
- Payout requests/approvals → `payout`
- Ledger entries → `ledger`
- AML screening → `compliance`

#### Step 2: Create Module Structure

```bash
mkdir -p internal/modules/{module_name}/{domain,service,adapter/http,adapter/repository}
```

#### Step 3: Move Code

```bash
# Move domain models
mv internal/model/payment.go internal/modules/payment/domain/payment.go

# Move service
mv internal/service/payment.go internal/modules/payment/service/payment_service.go

# Move repository
mv internal/repository/payment.go internal/modules/payment/adapter/repository/postgres.go

# Move handler
mv internal/api/handler/payment.go internal/modules/payment/adapter/http/handler.go
```

#### Step 4: Extract Interfaces

```go
// In modules/payment/domain/repository.go
type PaymentRepository interface {
    Create(ctx context.Context, payment *Payment) error
    GetByID(ctx context.Context, id string) (*Payment, error)
    Update(ctx context.Context, payment *Payment) error
}
```

#### Step 5: Create module.go

```go
// modules/payment/module.go
package payment

type Module struct {
    Service    *service.PaymentService
    Repository domain.PaymentRepository
    Handler    *http.PaymentHandler
}

func NewModule(cfg Config) (*Module, error) {
    repository := repository.NewPostgresPaymentRepository(cfg.DB)
    service := service.NewPaymentService(repository, cfg.MerchantReader, ...)
    handler := http.NewPaymentHandler(service, ...)

    return &Module{
        Service:    service,
        Repository: repository,
        Handler:    handler,
    }, nil
}
```

#### Step 6: Update Imports

Update all references to use the new module:

```go
// Old
import "github.com/hxuan190/stable_payment_gateway/internal/service"
svc := service.NewPaymentService(...)

// New
import paymentservice "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
svc := paymentservice.NewPaymentService(...)
```

#### Step 7: Register in Module Registry

```go
// internal/modules/registry.go
registry := modules.NewRegistry(modules.RegistryConfig{
    PaymentService: paymentModule.Service,
    PaymentHandler: paymentModule.Handler,
    // ...
})
```

---

## Best Practices

### 1. Keep Domain Layer Pure

```go
// ✅ GOOD: Pure business logic
type Payment struct {
    ID     string
    Amount decimal.Decimal
    Status PaymentStatus
}

func (p *Payment) Confirm(txHash string) error {
    if p.Status != PaymentStatusPending {
        return ErrInvalidStatus
    }
    p.Status = PaymentStatusConfirmed
    p.TxHash = txHash
    p.ConfirmedAt = time.Now()
    return nil
}

// ❌ BAD: Infrastructure concerns in domain
type Payment struct {
    ID     string `json:"id" gorm:"primaryKey"`
}
```

### 2. Use Value Objects

```go
// Value object for type safety
type PaymentStatus string

const (
    PaymentStatusCreated   PaymentStatus = "created"
    PaymentStatusPending   PaymentStatus = "pending"
    PaymentStatusConfirmed PaymentStatus = "confirmed"
)
```

### 3. Validate at Boundaries

```go
// Validate in adapter layer (HTTP)
func (h *Handler) CreatePayment(c *gin.Context) {
    var req CreatePaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse(err))
        return
    }

    // Validate business rules in domain/service
    payment, err := h.service.CreatePayment(ctx, req.ToCommand())
    if err != nil {
        c.JSON(400, ErrorResponse(err))
        return
    }

    c.JSON(200, toDTO(payment))
}
```

### 4. Use Events for Side Effects

```go
// Don't call other modules directly
// ❌ BAD
func (s *PaymentService) ConfirmPayment(...) error {
    // ...
    ledgerService.RecordPayment(...) // Direct call
    notificationService.SendWebhook(...) // Direct call
    return nil
}

// ✅ GOOD: Publish event
func (s *PaymentService) ConfirmPayment(...) error {
    // ...
    event := NewPaymentConfirmedEvent(payment)
    s.eventBus.Publish(ctx, event) // Ledger & Notification subscribe
    return nil
}
```

### 5. Test in Isolation

```go
// Mock dependencies
func TestPaymentService_CreatePayment(t *testing.T) {
    mockRepo := &MockPaymentRepository{}
    mockMerchantReader := &MockMerchantReader{}
    mockEventBus := &MockEventBus{}

    svc := NewPaymentService(mockRepo, mockMerchantReader, mockEventBus, ...)

    payment, err := svc.CreatePayment(ctx, cmd)

    assert.NoError(t, err)
    assert.NotNil(t, payment)
    mockEventBus.AssertPublished("payment.created")
}
```

---

## Resources

- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Domain-Driven Design](https://www.domainlanguage.com/ddd/)
- [Module Pattern in Go](https://threedots.tech/post/modular-monolith-primer/)
- [Event-Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html)

---

**Questions?** See `/internal/modules/README.md` for module-specific details.

**Last Updated**: 2025-11-23
