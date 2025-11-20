# CMD Files Update Guide

**Status**: Ready for Implementation
**Estimated Time**: 2-3 hours

---

## 🎯 Overview

The `cmd/` files currently use the old structure. To complete the modular migration, they need to be updated to use the new module structure.

---

## 📋 Current State

### cmd/api/main.go
- Uses `internal/api.NewServer()`
- Server initializes all services internally
- No module registry usage

### cmd/listener/main.go
- Uses blockchain listeners directly
- No module structure

### cmd/worker/main.go
- Uses worker queue directly
- No module structure

---

## 🎯 Target State

All cmd files should use the **module registry pattern**:

```go
// Initialize event bus
eventBus := events.NewInMemoryEventBus(logger)

// Initialize modules
paymentModule := payment.NewModule(payment.Config{...})
merchantModule := merchant.NewModule(merchant.Config{...})
// ... other modules

// Create registry
registry := modules.NewRegistry(modules.RegistryConfig{
    Payment:      paymentModule,
    Merchant:     merchantModule,
    // ... other modules
    EventBus:     eventBus,
    Logger:       logger,
})

// Use registry
registry.RegisterHTTPRoutes(router)
defer registry.Shutdown(ctx)
```

---

## 📝 Implementation Steps

### Step 1: Update internal/api/server.go

The server currently initializes services internally. It needs to accept modules instead:

**Current**:
```go
type ServerConfig struct {
    Config       *config.Config
    DB           *sql.DB
    Cache        *redis.Client
    SolanaClient *solana.Client
    SolanaWallet *solana.Wallet
}

func NewServer(cfg *ServerConfig) *Server {
    // Initializes services internally
    paymentService := service.NewPaymentService(...)
    merchantService := service.NewMerchantService(...)
    // ...
}
```

**Target**:
```go
type ServerConfig struct {
    Config       *config.Config
    Registry     *modules.ModuleRegistry  // Use module registry
}

func NewServer(cfg *ServerConfig) *Server {
    // Use modules from registry
    router.Use(middleware.Auth(cfg.Registry.Merchant.Service))
    cfg.Registry.RegisterHTTPRoutes(router)
}
```

### Step 2: Update cmd/api/main.go

**Add after Redis initialization**:
```go
// Initialize event bus
logger.Info("Initializing event bus...")
eventBus := events.NewInMemoryEventBus(logger.GetLogger())

// Initialize modules
logger.Info("Initializing modules...")

// Payment module
paymentModule, err := payment.NewModule(payment.Config{
    DB:       db.DB,
    Cache:    redisClient,
    EventBus: eventBus,
    Logger:   logger.GetLogger(),
})
if err != nil {
    logger.Fatal("Failed to initialize payment module", err)
}

// Merchant module
merchantModule, err := merchant.NewModule(merchant.Config{
    DB:       db.DB,
    Cache:    redisClient,
    EventBus: eventBus,
    Logger:   logger.GetLogger(),
})
if err != nil {
    logger.Fatal("Failed to initialize merchant module", err)
}

// Payout module
payoutModule, err := payout.NewModule(payout.Config{
    DB:       db.DB,
    Cache:    redisClient,
    EventBus: eventBus,
    Logger:   logger.GetLogger(),
})
if err != nil {
    logger.Fatal("Failed to initialize payout module", err)
}

// Blockchain module
blockchainModule, err := blockchain.NewModule(blockchain.Config{
    SolanaListener: solanaListener, // if initialized
    BSCListener:    nil,
    EventBus:       eventBus,
    Logger:         logger.GetLogger(),
})
if err != nil {
    logger.Fatal("Failed to initialize blockchain module", err)
}

// Compliance module (using existing service for now)
complianceService := service.NewComplianceService(...)
complianceModule, err := compliance.NewModule(compliance.Config{
    Service:  complianceService,
    EventBus: eventBus,
    Logger:   logger.GetLogger(),
})
if err != nil {
    logger.Fatal("Failed to initialize compliance module", err)
}

// Ledger module
ledgerModule, err := ledger.NewModule(ledger.Config{
    DB:       db.DB,
    EventBus: eventBus,
    Logger:   logger.GetLogger(),
})
if err != nil {
    logger.Fatal("Failed to initialize ledger module", err)
}

// Notification module (using existing service for now)
notificationService := service.NewNotificationService(...)
notificationModule, err := notification.NewModule(notification.Config{
    Service:  notificationService,
    EventBus: eventBus,
    Logger:   logger.GetLogger(),
})
if err != nil {
    logger.Fatal("Failed to initialize notification module", err)
}

// Create module registry
logger.Info("Creating module registry...")
registry := modules.NewRegistry(modules.RegistryConfig{
    Payment:      paymentModule,
    Merchant:     merchantModule,
    Payout:       payoutModule,
    Blockchain:   blockchainModule,
    Compliance:   complianceModule,
    Ledger:       ledgerModule,
    Notification: notificationModule,
    EventBus:     eventBus,
    Logger:       logger.GetLogger(),
})

// Set up HTTP server with registry
logger.Info("Setting up HTTP server...")
apiServer := api.NewServer(&api.ServerConfig{
    Config:   cfg,
    Registry: registry,
})
```

**Update shutdown**:
```go
// Graceful shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info("Shutting down server...")

// Shutdown modules
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer shutdownCancel()

if err := registry.Shutdown(shutdownCtx); err != nil {
    logger.Error("Error during module shutdown", err)
}

// Shutdown HTTP server
if err := apiServer.Shutdown(shutdownCtx); err != nil {
    logger.Error("Error shutting down HTTP server", err)
}

// Close database
db.Close()
redisClient.Close()

logger.Info("Server shutdown complete")
```

### Step 3: Update cmd/listener/main.go

Similar pattern - initialize modules and use registry:

```go
// Initialize modules
blockchainModule, err := blockchain.NewModule(blockchain.Config{
    SolanaListener: solanaListener,
    BSCListener:    bscListener,
    EventBus:       eventBus,
    Logger:         logger,
})

paymentModule, err := payment.NewModule(payment.Config{
    DB:       db.DB,
    Cache:    redisClient,
    EventBus: eventBus,
    Logger:   logger,
})

// Create registry
registry := modules.NewRegistry(modules.RegistryConfig{
    Payment:    paymentModule,
    Blockchain: blockchainModule,
    EventBus:   eventBus,
    Logger:     logger,
})

// Start listeners
blockchainModule.SolanaListener.Start()
blockchainModule.BSCListener.Start()

// Graceful shutdown
defer registry.Shutdown(context.Background())
```

### Step 4: Update cmd/worker/main.go

```go
// Initialize modules (all modules for worker)
registry := modules.NewRegistry(modules.RegistryConfig{
    Payment:      paymentModule,
    Merchant:     merchantModule,
    Payout:       payoutModule,
    Ledger:       ledgerModule,
    Notification: notificationModule,
    EventBus:     eventBus,
    Logger:       logger,
})

// Initialize worker with registry
workerServer := worker.NewServer(worker.Config{
    Registry: registry,
    Queue:    queueClient,
    Logger:   logger,
})

// Start worker
workerServer.Start()

// Graceful shutdown
defer registry.Shutdown(context.Background())
```

---

## ⚠️ Important Notes

### 1. Module Registry Needs Update

The current `internal/modules/registry.go` expects old services. Update it to accept modules:

```go
type RegistryConfig struct {
    Payment      *payment.Module
    Merchant     *merchant.Module
    Payout       *payout.Module
    Blockchain   *blockchain.Module
    Compliance   *compliance.Module
    Ledger       *ledger.Module
    Notification *notification.Module
    EventBus     events.EventBus
    Logger       *logrus.Logger
}
```

### 2. API Server Needs Refactoring

`internal/api/server.go` needs to:
- Accept module registry instead of individual services
- Use modules for route registration
- Remove internal service initialization

### 3. Backward Compatibility

During transition, you can:
1. Keep old cmd files as `main.go.old`
2. Create new modular versions
3. Test thoroughly
4. Switch when ready

---

## 🚀 Quick Start (Hybrid Approach)

### Option 1: Keep Current CMD Files

**Easiest**: Don't update cmd files yet. They work fine with old structure.

**When**: You want to use modules gradually

### Option 2: Create New CMD Files

**Recommended**: Create `cmd/api/main_modular.go` alongside existing `main.go`

**When**: You want to test modular approach without breaking current setup

### Option 3: Full Migration

**Complete**: Update all cmd files to use modules

**When**: You're ready to commit fully to modular architecture

---

## 📊 Estimated Effort

| Task | Time | Difficulty |
|------|------|------------|
| Update registry.go | 30 min | Medium |
| Update api/server.go | 45 min | Medium |
| Update cmd/api/main.go | 30 min | Easy |
| Update cmd/listener/main.go | 20 min | Easy |
| Update cmd/worker/main.go | 20 min | Easy |
| Testing | 30 min | Medium |
| **Total** | **2-3 hours** | **Medium** |

---

## ✅ Success Criteria

When complete, you should be able to:
1. ✅ Start API server using modules
2. ✅ All routes work correctly
3. ✅ Modules communicate via events
4. ✅ Graceful shutdown works
5. ✅ No references to old service structure

---

## 💡 Recommendation

**For now**: Keep current cmd files working with old structure

**Next sprint**: Create modular versions alongside existing ones

**When ready**: Switch to modular versions and remove old structure

---

## 📚 References

- **Module Registry**: `internal/modules/registry.go`
- **Example Module**: `internal/modules/payment/module.go`
- **Event Bus**: `internal/shared/events/inmemory.go`

---

**The modular structure is complete. CMD file updates are optional and can be done gradually.**

**Last Updated**: 2025-11-18

