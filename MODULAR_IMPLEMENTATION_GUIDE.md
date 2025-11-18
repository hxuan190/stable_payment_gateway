# Modular Architecture Implementation Guide

**Status**: ✅ Implemented
**Date**: 2025-11-18

---

## Overview

The payment gateway now uses a **modular monolith** architecture. All existing code continues to work, but it's now organized into clear module boundaries that make it easy to extract into microservices later.

### Key Components

1. **Module Registry** (`internal/modules/registry.go`) - Central registry for all modules
2. **Event Bus** (`internal/shared/events/`) - Inter-module communication
3. **Shared Interfaces** (`internal/shared/interfaces/`) - Cross-module contracts
4. **Existing Services** - Unchanged, wrapped by modules

---

## Quick Start

### Using the Module Registry

```go
package main

import (
    "github.com/hxuan190/stable_payment_gateway/internal/modules"
    "github.com/hxuan190/stable_payment_gateway/internal/shared/events"
)

func main() {
    // Initialize event bus
    eventBus := events.NewInMemoryEventBus(logger)

    // Initialize existing services (as before)
    paymentService := service.NewPaymentService(/* ... */)
    merchantService := service.NewMerchantService(/* ... */)
    // ... other services

    // Initialize existing handlers (as before)
    paymentHandler := handler.NewPaymentHandler(/* ... */)
    merchantHandler := handler.NewMerchantHandler(/* ... */)
    // ... other handlers

    // Create module registry
    registry := modules.NewRegistry(modules.RegistryConfig{
        PaymentService:   paymentService,
        MerchantService:  merchantService,
        PayoutService:    payoutService,
        ComplianceService: complianceService,
        LedgerService:    ledgerService,
        NotificationService: notificationService,

        PaymentHandler:   paymentHandler,
        MerchantHandler:  merchantHandler,
        PayoutHandler:    payoutHandler,

        SolanaListener:   solanaListener,
        BSCListener:      bscListener,

        EventBus:         eventBus,
        Logger:           logger,
    })

    // Modules are now organized and event-driven!
    // Use modules via registry:
    payment := registry.Payment.Service
    merchant := registry.Merchant.Service

    // Graceful shutdown
    defer registry.Shutdown(context.Background())
}
```

---

## Module Structure

### Current Modules

| Module | Components | Responsibilities |
|--------|-----------|-----------------|
| **Payment** | Service, Handler | Payment creation, confirmation, lifecycle |
| **Merchant** | Service, Handler | Registration, KYC, API keys |
| **Payout** | Service, Handler | Withdrawal requests, approvals |
| **Blockchain** | Solana Listener, BSC Listener | Transaction detection, confirmation |
| **Compliance** | Service | AML screening, Travel Rule |
| **Ledger** | Service | Double-entry accounting |
| **Notification** | Service | Webhooks, emails |

### Module Dependencies

```
┌─────────────────────────────────────────────────┐
│               Module Registry                    │
│  (Organizes all modules, manages lifecycle)     │
└─────────────────────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
    ┌────▼────┐   ┌───▼────┐   ┌───▼────┐
    │ Payment │   │Merchant│   │ Payout │
    └────┬────┘   └───┬────┘   └───┬────┘
         │            │            │
         └────────────┼────────────┘
                      │
         ┌────────────┼────────────┐
         │            │            │
    ┌────▼────┐  ┌───▼───┐   ┌───▼────────┐
    │Blockchain│  │Ledger │   │Notification│
    └─────────┘  └───────┘   └────────────┘
```

Modules communicate via **Event Bus** (no direct imports!)

---

## Event-Driven Communication

### How It Works

```go
// Payment module publishes event
eventBus.Publish(ctx, PaymentConfirmedEvent{
    PaymentID:  "pay-123",
    MerchantID: "merchant-456",
    Amount:     decimal.NewFromInt(1000000),
})

// Ledger module subscribes and handles asynchronously
eventBus.Subscribe("payment.confirmed", func(ctx context.Context, event Event) error {
    // Update ledger entries
    return ledgerService.RecordPayment(ctx, event)
})
```

### Configured Events

The registry automatically sets up these event subscriptions:

- `payment.confirmed` → Ledger (update balance) + Notification (send webhook)
- `merchant.kyc_approved` → Payment (enable payment acceptance)
- `payout.completed` → Ledger (debit balance)

---

## Migration Path to Microservices

When you're ready to extract a module into a separate service:

### Step 1: Extract Module Code

```bash
# Example: Extract payment module
mkdir payment-service
cp -r internal/modules/payment payment-service/
cp -r internal/service/payment.go payment-service/service/
cp -r internal/repository/payment.go payment-service/repository/
cp -r internal/api/handler/payment.go payment-service/handler/
```

### Step 2: Replace Event Bus

```go
// Replace in-memory event bus with Redis/RabbitMQ
eventBus := events.NewRedisEventBus(redisClient, logger)
// OR
eventBus := events.NewRabbitMQEventBus(rabbitmqConn, logger)
```

No code changes needed - same interface!

### Step 3: Deploy Separately

```yaml
# docker-compose.yml
services:
  payment-service:
    build: ./payment-service
    ports:
      - "8081:8080"
    environment:
      - REDIS_URL=redis://redis:6379

  main-app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - payment-service
      - redis
```

### Step 4: Update Main App

```go
// In main app, replace local service with HTTP client
paymentClient := NewPaymentHTTPClient("http://payment-service:8080")
```

**That's it!** The modular structure makes extraction trivial.

---

## Benefits Achieved

✅ **Clear Boundaries** - Each module has well-defined responsibilities

✅ **Independent Testing** - Test modules in isolation

✅ **Event-Driven** - Modules communicate asynchronously

✅ **Easy Extraction** - Path to microservices is clear

✅ **Existing Code Works** - No breaking changes!

✅ **Team Scalability** - Different teams can own different modules

---

## Code Examples

### Example 1: Adding a New Event Subscription

```go
// In registry.go, add to setupEventSubscriptions():
r.eventBus.Subscribe("payment.expired", func(ctx context.Context, event events.Event) error {
    r.logger.Info("Processing payment.expired event")

    if r.Notification != nil {
        // Send expiration notification
        return r.Notification.Service.SendPaymentExpiredEmail(ctx, event)
    }

    return nil
})
```

### Example 2: Publishing an Event from Existing Service

```go
// In internal/service/payment.go, after payment confirmation:
func (s *PaymentService) ConfirmPayment(ctx context.Context, req ConfirmPaymentRequest) (*model.Payment, error) {
    // ... existing confirmation logic ...

    // Publish event (if event bus is available)
    if s.eventBus != nil {
        event := PaymentConfirmedEvent{
            PaymentID:  payment.ID,
            MerchantID: payment.MerchantID,
            Amount:     payment.AmountVND,
        }
        s.eventBus.Publish(ctx, event)
    }

    return payment, nil
}
```

### Example 3: Accessing Modules

```go
// Via registry
payment, err := registry.Payment.Service.CreatePayment(ctx, req)
merchant, err := registry.Merchant.Service.GetByID(ctx, merchantID)

// Check module status
status := registry.GetModuleStatus()
// {"payment": true, "merchant": true, ...}
```

---

## Testing

### Unit Testing Modules

```go
func TestPaymentModule(t *testing.T) {
    // Setup
    eventBus := events.NewInMemoryEventBus(logger)
    paymentService := service.NewPaymentService(/* ... */)

    registry := modules.NewRegistry(modules.RegistryConfig{
        PaymentService: paymentService,
        EventBus:       eventBus,
        Logger:         logger,
    })

    // Test
    payment, err := registry.Payment.Service.CreatePayment(ctx, req)
    assert.NoError(t, err)
}
```

### Integration Testing (Event-Driven)

```go
func TestPaymentConfirmation_UpdatesLedger(t *testing.T) {
    eventBus := events.NewInMemoryEventBus(logger)

    // Initialize modules
    registry := modules.NewRegistry(/* ... */)

    // Trigger event
    payment, _ := registry.Payment.Service.ConfirmPayment(ctx, req)

    // Wait for async processing
    time.Sleep(100 * time.Millisecond)

    // Verify ledger was updated
    balance, _ := registry.Ledger.Service.GetBalance(ctx, merchantID)
    assert.Equal(t, expectedBalance, balance)
}
```

---

## Troubleshooting

### Issue: Events not being processed

**Solution**: Ensure event bus is initialized and modules are registered:

```go
// Check module status
status := registry.GetModuleStatus()
log.Printf("Module status: %+v", status)

// Check event bus handler count
handlerCount := eventBus.HandlerCount("payment.confirmed")
log.Printf("Handlers for payment.confirmed: %d", handlerCount)
```

### Issue: Circular dependencies

**Solution**: Use shared interfaces, never import modules directly:

```go
// ❌ Bad
import "internal/modules/merchant"
merchant := merchantModule.GetByID(id)

// ✅ Good
import "internal/shared/interfaces"
type PaymentService struct {
    merchantReader interfaces.MerchantReader
}
```

---

## Next Steps

### Phase 1: Current (✅ Done)
- Module registry implemented
- Event bus configured
- Existing code organized into modules

### Phase 2: Enhance Event-Driven (Recommended Next)
- Add event publishing to all services
- Implement event subscribers for all modules
- Add event sourcing for audit trail

### Phase 3: Extract First Microservice (Future)
- Extract blockchain listener module
- Deploy as separate service
- Use Redis for event bus

### Phase 4: Full Microservices (Future)
- Extract remaining modules
- Implement API gateway
- Add service mesh (Istio)

---

## FAQ

**Q: Does this change break existing functionality?**
A: No! All existing code works as before. The module registry is just an organization layer.

**Q: Do I need to refactor all code now?**
A: No. The modular structure works with existing code. Refactoring can happen gradually.

**Q: How do I add a new module?**
A: 1) Create service/handler, 2) Add to registry config, 3) Register in NewRegistry().

**Q: Can I extract just one module to a microservice?**
A: Yes! That's the beauty of this architecture. Extract any module independently.

**Q: What about database transactions?**
A: For now, use the same database. When extracting to microservices, use the Saga pattern for distributed transactions.

---

## References

- [Modular Architecture Design](./MODULAR_ARCHITECTURE.md)
- [Event Bus Documentation](./internal/shared/README.md)
- [Shared Interfaces](./internal/shared/interfaces/)

---

**Questions?** Check the module registry code: `internal/modules/registry.go`

**Last Updated**: 2025-11-18
