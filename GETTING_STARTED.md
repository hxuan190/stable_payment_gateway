# Getting Started - Implementation Guide

**For engineering team starting development**

---

## 📋 Pre-Development Checklist

### Business Setup
- [ ] Legal entity registered
- [ ] Bank account opened (for VND settlements)
- [ ] OTC partner identified (get 2-3 quotes)
- [ ] 3-5 pilot merchants lined up
- [ ] KYC requirements documented

### Technical Setup
- [ ] GitHub repo created
- [ ] Team has access
- [ ] Development machines ready (Go 1.21+)
- [ ] Cloud account (DO/AWS/GCP)
- [ ] Domain purchased (e.g., `gateway.vn`)

### Team Alignment
- [ ] Everyone read REQUIREMENTS.md
- [ ] Tech stack approved (Golang + Solana + BSC)
- [ ] Architecture understood
- [ ] Timeline agreed (flexible, but aim for 6 weeks)

---

## 🚀 Day 1: Project Setup

### 1. Initialize Go Project

```bash
# Create project directory
mkdir stable-payment-gateway
cd stable-payment-gateway

# Initialize Go module
go mod init github.com/yourusername/stable-payment-gateway

# Create directory structure
mkdir -p cmd/{api,listener,worker,admin}
mkdir -p internal/{api/{handler,middleware,routes},blockchain/{solana,bsc,types},service,repository,model,config,pkg/{crypto,validator,logger,errors}}
mkdir -p web/{dashboard,payment,admin}
mkdir -p migrations
mkdir -p scripts
mkdir -p docker
mkdir -p docs

# Create main files
touch cmd/api/main.go
touch cmd/listener/main.go
touch internal/config/config.go
touch .env.example
touch Makefile
touch docker-compose.yml
```

---

### 2. Install Core Dependencies

```bash
# HTTP framework
go get github.com/gin-gonic/gin

# Database
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/golang-migrate/migrate/v4

# Redis
go get github.com/redis/go-redis/v9

# Blockchain
go get github.com/gagliardetto/solana-go
go get github.com/ethereum/go-ethereum

# Utilities
go get github.com/google/uuid
go get github.com/shopspring/decimal
go get github.com/spf13/viper
go get go.uber.org/zap
go get github.com/go-playground/validator/v10

# Background jobs
go get github.com/hibiken/asynq

# Testing
go get github.com/stretchr/testify
```

---

### 3. Setup Environment Variables

Create `.env.example`:
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=payment_gateway

# Redis
REDIS_HOST=localhost:6379
REDIS_PASSWORD=

# Blockchain (Testnet first!)
SOLANA_RPC_URL=https://api.devnet.solana.com
SOLANA_WALLET_PRIVATE_KEY=your_private_key_here
SOLANA_WALLET_ADDRESS=your_wallet_address_here

BSC_RPC_URL=https://data-seed-prebsc-1-s1.binance.org:8545
BSC_WALLET_PRIVATE_KEY=your_private_key_here
BSC_WALLET_ADDRESS=your_wallet_address_here

# API
API_PORT=8080
JWT_SECRET=your_jwt_secret_here
API_RATE_LIMIT=100

# Exchange Rate API
EXCHANGE_RATE_API=https://api.coingecko.com/api/v3

# Webhook
WEBHOOK_RETRY_MAX=5

# Environment
ENV=development
```

Copy to `.env`:
```bash
cp .env.example .env
# Edit .env with your actual values
```

**⚠️ IMPORTANT**: Add `.env` to `.gitignore`!

---

### 4. Setup Docker Compose (Development)

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: payment_gateway_db
    environment:
      POSTGRES_DB: payment_gateway
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    container_name: payment_gateway_redis
    ports:
      - "6379:6379"

  adminer:
    image: adminer
    container_name: payment_gateway_adminer
    ports:
      - "8081:8080"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

Start services:
```bash
docker-compose up -d
```

---

### 5. Create Makefile

```makefile
.PHONY: help run-api run-listener migrate-up migrate-down test lint

help:
	@echo "Available commands:"
	@echo "  make run-api       - Run API server"
	@echo "  make run-listener  - Run blockchain listener"
	@echo "  make migrate-up    - Run database migrations"
	@echo "  make migrate-down  - Rollback database migrations"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"

run-api:
	go run cmd/api/main.go

run-listener:
	go run cmd/listener/main.go

migrate-up:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/payment_gateway?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/payment_gateway?sslmode=disable" down

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

build-api:
	CGO_ENABLED=0 GOOS=linux go build -o bin/api cmd/api/main.go

build-listener:
	CGO_ENABLED=0 GOOS=linux go build -o bin/listener cmd/listener/main.go

docker-build:
	docker build -t payment-gateway-api -f docker/Dockerfile.api .
	docker build -t payment-gateway-listener -f docker/Dockerfile.listener .
```

---

## 📊 Week 1-2: Core Implementation

### Day 2-3: Database Schema

Create first migration:
```bash
migrate create -ext sql -dir migrations -seq create_merchants_table
```

Edit `migrations/000001_create_merchants_table.up.sql`:
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE merchants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    business_name VARCHAR(255) NOT NULL,
    tax_id VARCHAR(50),

    kyc_status VARCHAR(50) DEFAULT 'pending',
    kyc_data JSONB,

    api_key VARCHAR(255) UNIQUE,
    webhook_url VARCHAR(500),
    webhook_secret VARCHAR(255),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_merchants_email ON merchants(email);
CREATE INDEX idx_merchants_api_key ON merchants(api_key);
```

Create similar migrations for:
- `payments`
- `payouts`
- `ledger_entries`
- `merchant_balances`
- `audit_logs`
- `blockchain_transactions`

Apply migrations:
```bash
make migrate-up
```

---

### Day 4-5: Define Models

Create `internal/model/merchant.go`:
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Merchant struct {
    ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
    Email         string     `gorm:"uniqueIndex;not null"`
    BusinessName  string     `gorm:"not null"`
    TaxID         string

    KYCStatus     string     `gorm:"default:pending"`
    KYCData       string     `gorm:"type:jsonb"`

    APIKey        string     `gorm:"uniqueIndex"`
    WebhookURL    string
    WebhookSecret string

    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type Payment struct {
    ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
    MerchantID    uuid.UUID       `gorm:"type:uuid;not null;index"`

    AmountVND     decimal.Decimal `gorm:"type:decimal(15,2);not null"`
    AmountCrypto  decimal.Decimal `gorm:"type:decimal(20,8)"`
    CryptoCurrency string
    Chain         string

    OrderID       string          `gorm:"index"`

    WalletAddress string
    TxHash        string          `gorm:"uniqueIndex"`

    Status        string          `gorm:"default:created;index"`

    CallbackURL   string
    Metadata      string          `gorm:"type:jsonb"`

    ExpiresAt     *time.Time
    ConfirmedAt   *time.Time
    CreatedAt     time.Time
}
```

---

### Day 6-8: Build API Server

Create `cmd/api/main.go`:
```go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/yourusername/stable-payment-gateway/internal/config"
    "github.com/yourusername/stable-payment-gateway/internal/api/routes"
)

func main() {
    // Load config
    cfg := config.Load()

    // Setup router
    r := gin.Default()

    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // Setup routes
    routes.SetupPaymentRoutes(r)
    routes.SetupMerchantRoutes(r)

    // Start server
    log.Printf("API server starting on port %s", cfg.APIPort)
    r.Run(":" + cfg.APIPort)
}
```

Create `internal/api/handler/payment.go`:
```go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/yourusername/stable-payment-gateway/internal/service"
)

type PaymentHandler struct {
    paymentService *service.PaymentService
}

func NewPaymentHandler(ps *service.PaymentService) *PaymentHandler {
    return &PaymentHandler{paymentService: ps}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
    var req CreatePaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Validate merchant API key
    merchantID := c.GetString("merchant_id")

    // Create payment
    payment, err := h.paymentService.CreatePayment(merchantID, req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, payment)
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
    paymentID := c.Param("id")

    payment, err := h.paymentService.GetPayment(paymentID)
    if err != nil {
        c.JSON(404, gin.H{"error": "payment not found"})
        return
    }

    c.JSON(200, payment)
}
```

---

### Day 9-12: Blockchain Listeners

Create `internal/blockchain/solana/listener.go`:
```go
package solana

import (
    "context"
    "log"

    "github.com/gagliardetto/solana-go"
    "github.com/gagliardetto/solana-go/rpc"
    "github.com/gagliardetto/solana-go/rpc/ws"
)

type Listener struct {
    rpcClient *rpc.Client
    wsClient  *ws.Client
    wallet    solana.PublicKey
}

func NewListener(rpcURL string, wsURL string, walletAddr string) *Listener {
    wallet, _ := solana.PublicKeyFromBase58(walletAddr)

    return &Listener{
        rpcClient: rpc.New(rpcURL),
        wsClient:  ws.Connect(context.Background(), wsURL),
        wallet:    wallet,
    }
}

func (l *Listener) Start() error {
    log.Printf("Solana listener starting, monitoring wallet: %s", l.wallet)

    // Subscribe to account changes
    sub, err := l.wsClient.AccountSubscribe(
        l.wallet,
        rpc.CommitmentFinalized,
    )
    if err != nil {
        return err
    }

    for {
        got, err := sub.Recv()
        if err != nil {
            log.Printf("Error receiving: %v", err)
            continue
        }

        // Handle transaction
        l.handleTransaction(got)
    }
}

func (l *Listener) handleTransaction(data interface{}) {
    // Parse transaction
    // Extract memo (payment_id)
    // Verify amount
    // Call payment service to confirm payment

    log.Printf("New transaction detected: %+v", data)
}
```

Similar implementation for BSC in `internal/blockchain/bsc/listener.go`

---

### Day 13-14: Payment Service

Create `internal/service/payment.go`:
```go
package service

import (
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "github.com/yourusername/stable-payment-gateway/internal/model"
    "github.com/yourusername/stable-payment-gateway/internal/repository"
)

type PaymentService struct {
    paymentRepo *repository.PaymentRepository
    exchangeService *ExchangeService
    ledgerService *LedgerService
}

func (s *PaymentService) CreatePayment(merchantID string, req CreatePaymentRequest) (*model.Payment, error) {
    // 1. Get current exchange rate
    rate, err := s.exchangeService.GetUSDTVNDRate()
    if err != nil {
        return nil, err
    }

    // 2. Calculate crypto amount
    amountCrypto := decimal.NewFromFloat(req.AmountVND).Div(rate)

    // 3. Create payment
    payment := &model.Payment{
        ID:            uuid.New(),
        MerchantID:    uuid.MustParse(merchantID),
        AmountVND:     decimal.NewFromFloat(req.AmountVND),
        AmountCrypto:  amountCrypto,
        CryptoCurrency: "USDT",
        OrderID:       req.OrderID,
        Status:        "created",
        ExpiresAt:     time.Now().Add(30 * time.Minute),
        CreatedAt:     time.Now(),
    }

    // 4. Save to database
    err = s.paymentRepo.Create(payment)
    if err != nil {
        return nil, err
    }

    return payment, nil
}

func (s *PaymentService) ConfirmPayment(paymentID, txHash string, amount decimal.Decimal, chain string) error {
    // 1. Get payment
    payment, err := s.paymentRepo.GetByID(paymentID)
    if err != nil {
        return err
    }

    // 2. Validate amount
    if !amount.Equal(payment.AmountCrypto) {
        return errors.New("amount mismatch")
    }

    // 3. Update payment
    payment.Status = "completed"
    payment.TxHash = txHash
    payment.Chain = chain
    now := time.Now()
    payment.ConfirmedAt = &now

    err = s.paymentRepo.Update(payment)
    if err != nil {
        return err
    }

    // 4. Update ledger
    s.ledgerService.RecordPayment(payment)

    // 5. Send webhook (async)
    go s.webhookService.Send(payment.MerchantID, "payment.completed", payment)

    return nil
}
```

---

## 🧪 Testing

### Unit Tests

Create `internal/service/payment_test.go`:
```go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreatePayment(t *testing.T) {
    // Setup
    service := NewPaymentService(/* mock dependencies */)

    // Test
    payment, err := service.CreatePayment("merchant-123", CreatePaymentRequest{
        AmountVND: 2300000,
        OrderID: "ORDER-001",
    })

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "created", payment.Status)
    assert.NotEmpty(t, payment.ID)
}
```

Run tests:
```bash
make test
```

---

### Integration Tests (Testnet)

1. Get testnet tokens:
   - Solana: https://solfaucet.com/
   - BSC: https://testnet.binance.org/faucet-smart

2. Create test payment:
```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Authorization: Bearer test_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "amountVND": 230000,
    "orderId": "TEST-001"
  }'
```

3. Send testnet crypto to payment address

4. Verify payment confirmed in database

---

## 📦 Deployment

### Build Docker Images

Create `docker/Dockerfile.api`:
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api .
EXPOSE 8080
CMD ["./api"]
```

Build:
```bash
make docker-build
```

---

### Deploy to VPS

```bash
# 1. SSH to VPS
ssh root@your-vps-ip

# 2. Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# 3. Clone repo
git clone https://github.com/yourusername/stable-payment-gateway
cd stable-payment-gateway

# 4. Setup environment
cp .env.example .env
nano .env  # Edit with production values

# 5. Start services
docker-compose -f docker-compose.prod.yml up -d

# 6. Check logs
docker-compose logs -f
```

---

## ✅ Milestone Checklist

### Week 1
- [ ] Project setup complete
- [ ] Database schema created
- [ ] Basic API working (health check, create payment)
- [ ] Models and repositories implemented

### Week 2
- [ ] Payment creation API complete
- [ ] Exchange rate integration working
- [ ] Solana listener detecting transactions
- [ ] BSC listener detecting transactions

### Week 3
- [ ] Payment confirmation flow working end-to-end
- [ ] Ledger service implemented
- [ ] Webhook system working
- [ ] Basic merchant dashboard (Next.js)

### Week 4
- [ ] Payment page with QR code
- [ ] Real-time status updates
- [ ] Multi-chain support tested
- [ ] KYC form and approval flow

### Week 5
- [ ] Payout request system
- [ ] Admin panel for KYC/payout approval
- [ ] Email notifications
- [ ] Documentation complete

### Week 6
- [ ] Security audit
- [ ] Load testing
- [ ] Testnet → Mainnet migration
- [ ] Pilot merchant onboarding
- [ ] Production launch! 🚀

---

## 🆘 Common Issues

### Issue: Can't connect to Solana RPC
**Solution**: Use paid RPC (Helius, QuickNode) instead of public endpoint

### Issue: PostgreSQL connection refused
**Solution**: Check docker-compose is running: `docker-compose ps`

### Issue: Migration fails
**Solution**: Drop database and recreate:
```bash
docker-compose down
docker volume rm stable_payment_gateway_postgres_data
docker-compose up -d
make migrate-up
```

---

## 📚 Resources

- [Go by Example](https://gobyexample.com/)
- [Gin Documentation](https://gin-gonic.com/docs/)
- [GORM Guide](https://gorm.io/docs/)
- [Solana Go SDK](https://github.com/gagliardetto/solana-go)
- [Go Ethereum Docs](https://geth.ethereum.org/docs/developers/dapp-developer/native)

---

## 💬 Team Communication

**Daily Standup** (async in Slack):
- What did you do yesterday?
- What will you do today?
- Any blockers?

**Weekly Review** (Friday):
- Demo working features
- Review code
- Plan next week

**Code Reviews**:
- All PRs require 1 approval
- Run tests before merging
- Follow Go style guide

---

**Let's build! 🚀**
