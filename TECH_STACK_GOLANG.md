# Tech Stack - Golang Implementation

**Updated**: Golang-based architecture with multi-chain support (Solana + BNB Chain)

---

## 🎯 Core Tech Stack

### Backend (Golang)

**Framework & Core**
```
- Go 1.21+
- Gin (HTTP framework) or Fiber (high performance)
- PostgreSQL 15 (database)
- Redis 7 (cache, rate limiting, queues)
- GORM (ORM) or sqlx (if prefer raw SQL)
```

**Why Golang?**
- ✅ High performance (important for blockchain listener)
- ✅ Great concurrency (goroutines for multi-chain monitoring)
- ✅ Strong typing (reduce bugs in financial logic)
- ✅ Easy deployment (single binary)
- ✅ Excellent blockchain libraries (Solana, Ethereum/BSC)

---

### Blockchain Libraries

**Solana**
```go
import (
    "github.com/gagliardetto/solana-go"
    "github.com/gagliardetto/solana-go/rpc"
    "github.com/gagliardetto/solana-go/programs/token"
)
```

**BNB Chain (BSC)**
```go
import (
    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
)
```

---

### Frontend

**Merchant Dashboard**
```
- Next.js 14 (App Router)
- TypeScript
- TailwindCSS
- shadcn/ui components
- React Query (data fetching)
```

**Payment Page**
```
- Next.js (static generation)
- QR code generation (qrcode.react)
- WebSocket (real-time payment status)
```

---

### Infrastructure

**Containerization**
```
- Docker + Docker Compose
- Multi-stage builds (optimize Go binary size)
```

**Database**
```
- PostgreSQL 15 (primary database)
- Redis 7 (cache, job queue)
- Potential: TimescaleDB extension (time-series data for analytics)
```

**Message Queue (Phase 2)**
```
- Redis Streams (built-in, simple)
- OR RabbitMQ (if need more features)
```

**Monitoring**
```
- Prometheus (metrics)
- Grafana (dashboards)
- Loki (logs)
- Alertmanager (alerts)
```

---

## 🏗️ Project Structure (Golang Monorepo)

```
stable_payment_gateway/
├── cmd/
│   ├── api/              # REST API server
│   │   └── main.go
│   ├── listener/         # Blockchain listener service
│   │   └── main.go
│   ├── worker/           # Background jobs (payouts, etc.)
│   │   └── main.go
│   └── admin/            # Admin API
│       └── main.go
│
├── internal/
│   ├── api/
│   │   ├── handler/      # HTTP handlers
│   │   ├── middleware/   # Auth, rate limit, logging
│   │   └── routes/       # Route definitions
│   │
│   ├── blockchain/
│   │   ├── solana/       # Solana listener
│   │   │   ├── listener.go
│   │   │   ├── wallet.go
│   │   │   └── transaction.go
│   │   ├── bsc/          # BSC listener
│   │   │   ├── listener.go
│   │   │   ├── wallet.go
│   │   │   └── erc20.go
│   │   └── types/        # Common blockchain types
│   │
│   ├── service/
│   │   ├── payment.go    # Payment business logic
│   │   ├── merchant.go   # Merchant management
│   │   ├── payout.go     # Payout logic
│   │   ├── ledger.go     # Double-entry ledger
│   │   ├── webhook.go    # Webhook dispatcher
│   │   └── exchange.go   # Exchange rate fetching
│   │
│   ├── repository/
│   │   ├── payment.go    # Payment DB operations
│   │   ├── merchant.go
│   │   ├── payout.go
│   │   └── ledger.go
│   │
│   ├── model/
│   │   ├── payment.go    # Domain models
│   │   ├── merchant.go
│   │   ├── payout.go
│   │   └── ledger.go
│   │
│   ├── config/
│   │   └── config.go     # Configuration loading
│   │
│   └── pkg/              # Shared utilities
│       ├── crypto/       # Encryption, hashing
│       ├── validator/    # Input validation
│       ├── logger/       # Structured logging
│       └── errors/       # Custom error types
│
├── web/                  # Frontend (Next.js)
│   ├── dashboard/        # Merchant dashboard
│   ├── payment/          # Payment page
│   └── admin/            # Admin panel
│
├── migrations/           # Database migrations
│   ├── 001_create_merchants.up.sql
│   ├── 001_create_merchants.down.sql
│   └── ...
│
├── scripts/
│   ├── deploy.sh
│   └── setup_dev.sh
│
├── docker/
│   ├── Dockerfile.api
│   ├── Dockerfile.listener
│   └── docker-compose.yml
│
├── docs/
│   ├── api.md
│   └── deployment.md
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 🔧 Key Go Packages

### HTTP Framework Options

**Option 1: Gin (Recommended for MVP)**
```go
// Fast, simple, well-documented
import "github.com/gin-gonic/gin"

r := gin.Default()
r.POST("/api/v1/payments", createPayment)
```

**Option 2: Fiber (If need extreme performance)**
```go
// Express-like, very fast
import "github.com/gofiber/fiber/v2"

app := fiber.New()
app.Post("/api/v1/payments", createPayment)
```

**Recommendation**: **Gin** for MVP (better docs, larger community)

---

### Database

**Option 1: GORM (ORM)**
```go
import "gorm.io/gorm"

type Payment struct {
    ID          uuid.UUID
    MerchantID  uuid.UUID
    AmountVND   decimal.Decimal
    Status      string
    CreatedAt   time.Time
}

db.Create(&payment)
```

**Option 2: sqlx (raw SQL with helpers)**
```go
import "github.com/jmoiron/sqlx"

type Payment struct {
    ID          uuid.UUID       `db:"id"`
    MerchantID  uuid.UUID       `db:"merchant_id"`
    AmountVND   decimal.Decimal `db:"amount_vnd"`
}

db.Get(&payment, "SELECT * FROM payments WHERE id = $1", id)
```

**Recommendation**: **GORM** for MVP (faster development), migrate to sqlx if need performance

---

### Migration

```go
import "github.com/golang-migrate/migrate/v4"

// Use SQL files for migrations (see migrations/ folder)
```

---

### Decimal Handling (IMPORTANT for money)

```go
import "github.com/shopspring/decimal"

// NEVER use float64 for money!
amountVND := decimal.NewFromFloat(2300000)
amountUSDT := decimal.NewFromFloat(100)

fee := amountVND.Mul(decimal.NewFromFloat(0.01)) // 1% fee
```

---

### Background Jobs

**Option 1: Redis-based (Simple)**
```go
import "github.com/hibiken/asynq"

// Enqueue job
client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
task := asynq.NewTask("payment:confirm", payload)
client.Enqueue(task)

// Process job
server := asynq.NewServer(...)
mux := asynq.NewServeMux()
mux.HandleFunc("payment:confirm", handlePaymentConfirm)
```

**Recommendation**: **asynq** for MVP

---

### Configuration

```go
import "github.com/spf13/viper"

viper.SetConfigFile(".env")
viper.AutomaticEnv()
viper.ReadInConfig()

dbHost := viper.GetString("DB_HOST")
```

---

### Logging

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Payment created",
    zap.String("payment_id", paymentID),
    zap.String("merchant_id", merchantID),
)
```

---

## 🔗 Multi-Chain Architecture

### Concurrent Blockchain Listening

```go
func main() {
    var wg sync.WaitGroup

    // Start Solana listener
    wg.Add(1)
    go func() {
        defer wg.Done()
        solanaListener := blockchain.NewSolanaListener(config.SolanaRPC)
        solanaListener.Start()
    }()

    // Start BSC listener
    wg.Add(1)
    go func() {
        defer wg.Done()
        bscListener := blockchain.NewBSCListener(config.BSCRPC)
        bscListener.Start()
    }()

    wg.Wait()
}
```

---

### Unified Payment Handler

```go
type BlockchainListener interface {
    Start() error
    Stop() error
    GetChain() string
}

type SolanaListener struct {
    rpc    *rpc.Client
    wallet solana.PublicKey
}

func (l *SolanaListener) Start() error {
    // Subscribe to transactions
    sub, err := l.rpc.AccountSubscribe(l.wallet, rpc.CommitmentFinalized)

    for {
        select {
        case tx := <-sub.Response():
            // Parse transaction
            // Extract memo (payment_id)
            // Call payment service
            paymentService.ConfirmPayment(paymentID, txHash, amount, chain)
        }
    }
}

type BSCListener struct {
    client *ethclient.Client
    wallet common.Address
}

func (l *BSCListener) Start() error {
    // Listen to USDT contract Transfer events
    usdtAddress := common.HexToAddress("0x55d398326f99059fF775485246999027B3197955")
    query := ethereum.FilterQuery{
        Addresses: []common.Address{usdtAddress},
    }

    logs := make(chan types.Log)
    sub, _ := l.client.SubscribeFilterLogs(context.Background(), query, logs)

    for {
        select {
        case vLog := <-logs:
            // Parse Transfer event
            // Check if recipient is our wallet
            // Extract amount, tx hash
            // Call payment service
            paymentService.ConfirmPayment(paymentID, txHash, amount, chain)
        }
    }
}
```

---

### Payment Service (Chain-Agnostic)

```go
type PaymentService struct {
    repo           *repository.PaymentRepository
    ledgerService  *LedgerService
    webhookService *WebhookService
}

func (s *PaymentService) ConfirmPayment(
    paymentID string,
    txHash string,
    amount decimal.Decimal,
    chain string, // "solana" or "bsc"
) error {
    // 1. Get payment from DB
    payment, err := s.repo.GetByID(paymentID)

    // 2. Validate amount matches
    if !amount.Equal(payment.AmountCrypto) {
        return errors.New("amount mismatch")
    }

    // 3. Update payment status
    payment.Status = "completed"
    payment.TxHash = txHash
    payment.Chain = chain
    payment.ConfirmedAt = time.Now()
    s.repo.Update(payment)

    // 4. Update ledger
    s.ledgerService.RecordPayment(payment)

    // 5. Send webhook
    s.webhookService.Send(payment.MerchantID, "payment.completed", payment)

    return nil
}
```

---

## 🏪 Tourism/Hospitality Focus

### QR Code for Restaurants

**Use Case**: Customer finishes meal, waiter shows QR code on tablet/phone

```go
// API endpoint for restaurant POS
POST /api/v1/pos/payment
{
  "merchantId": "xxx",
  "tableNumber": "12",
  "billAmount": 2300000,  // VND
  "items": [
    {"name": "Phở bò", "qty": 2, "price": 80000},
    {"name": "Cà phê", "qty": 2, "price": 35000}
  ]
}

Response:
{
  "paymentId": "pay_xxx",
  "qrCode": "data:image/png...",  // Large QR for tablet display
  "paymentUrl": "https://pay.gateway.com/pay_xxx",
  "expiresIn": 600  // 10 minutes
}
```

**Display Options**:
1. Tablet at table (show QR on screen)
2. Printed on bill (thermal printer integration)
3. SMS to customer (if have phone number)

---

### Hotel Check-in Integration

**Use Case**: Guest checks in, receptionist creates payment for room deposit

```go
POST /api/v1/hotels/reservation
{
  "merchantId": "xxx",
  "reservationId": "RES-12345",
  "guestName": "John Doe",
  "roomNumber": "302",
  "amountVND": 5000000,  // Deposit
  "checkInDate": "2025-11-20",
  "checkOutDate": "2025-11-25"
}

// Generate payment link → send via email/SMS to guest
```

**Features for Hotels**:
- Link payment to reservation system
- Multiple payment links per reservation (deposit, final payment, extras)
- Auto-settlement after checkout
- Guest payment history

---

### Multi-Stablecoin Support

For tourism industry, support multiple stablecoins:

**Solana Network**
- USDT (Tether): Most popular
- USDC (Circle): Common in US

**BNB Chain (BSC)**
- USDT (BEP20): Popular in Asia
- BUSD (Binance USD): Binance users

```go
type SupportedToken struct {
    Chain    string  // "solana", "bsc"
    Symbol   string  // "USDT", "USDC", "BUSD"
    Contract string  // Token contract address (for BSC)
    Decimals int     // 6 for USDT on Solana, 18 for BSC
}

var supportedTokens = []SupportedToken{
    {Chain: "solana", Symbol: "USDT", Contract: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Decimals: 6},
    {Chain: "solana", Symbol: "USDC", Contract: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", Decimals: 6},
    {Chain: "bsc", Symbol: "USDT", Contract: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18},
    {Chain: "bsc", Symbol: "BUSD", Contract: "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", Decimals: 18},
}
```

**When creating payment**:
```go
POST /api/v1/payments
{
  "amountVND": 2300000,
  "acceptedTokens": ["USDT", "USDC"]  // Merchant chooses which to accept
}

Response:
{
  "paymentId": "pay_xxx",
  "paymentOptions": [
    {
      "chain": "solana",
      "token": "USDT",
      "amount": "100",
      "wallet": "8xK7...",
      "qrCode": "..."
    },
    {
      "chain": "solana",
      "token": "USDC",
      "amount": "100",
      "wallet": "8xK7...",
      "qrCode": "..."
    },
    {
      "chain": "bsc",
      "token": "USDT",
      "amount": "100",
      "wallet": "0xABC...",
      "qrCode": "..."
    }
  ]
}
```

---

## 🚀 Deployment (Golang)

### Docker Multi-Stage Build

```dockerfile
# Dockerfile.api
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

**Build size**: ~15-20MB (vs 200MB+ Node.js)

---

### Docker Compose (Development)

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: payment_gateway
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  api:
    build:
      context: .
      dockerfile: docker/Dockerfile.api
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      REDIS_HOST: redis
      SOLANA_RPC: https://api.mainnet-beta.solana.com
      BSC_RPC: https://bsc-dataseed.binance.org
    depends_on:
      - postgres
      - redis

  listener:
    build:
      context: .
      dockerfile: docker/Dockerfile.listener
    environment:
      DB_HOST: postgres
      REDIS_HOST: redis
      SOLANA_RPC: https://api.mainnet-beta.solana.com
      BSC_RPC: https://bsc-dataseed.binance.org
    depends_on:
      - postgres
      - redis

  dashboard:
    build:
      context: ./web/dashboard
    ports:
      - "3000:3000"
    depends_on:
      - api
```

---

### Production Deployment Options

**Option 1: Single VPS (MVP)**
- 4 CPU, 8GB RAM, 100GB SSD
- Ubuntu 22.04
- Docker + Docker Compose
- NGINX reverse proxy
- Let's Encrypt SSL

**Option 2: Kubernetes (Scale)**
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
      - name: api
        image: payment-gateway-api:latest
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

---

## 📊 Performance Expectations

### Golang vs Node.js

| Metric | Node.js | Golang | Improvement |
|--------|---------|--------|-------------|
| Request latency (p50) | 15ms | 5ms | **3x faster** |
| Request latency (p99) | 100ms | 30ms | **3x faster** |
| Memory usage | 200MB | 50MB | **4x less** |
| Concurrent connections | 5,000 | 50,000 | **10x more** |
| Container size | 200MB | 20MB | **10x smaller** |
| Cold start | 2-3s | <100ms | **20x faster** |

**For payment gateway**: Low latency = better UX (faster payment confirmation)

---

## 🔐 Security Best Practices (Go)

### Secure Wallet Key Storage

```go
import "github.com/joho/godotenv"

// Load from .env (never commit keys to git)
godotenv.Load()
privateKey := os.Getenv("SOLANA_PRIVATE_KEY")

// Or use Vault (production)
import "github.com/hashicorp/vault/api"

client, _ := api.NewClient(api.DefaultConfig())
secret, _ := client.Logical().Read("secret/data/wallets/solana")
privateKey := secret.Data["private_key"].(string)
```

---

### Input Validation

```go
import "github.com/go-playground/validator/v10"

type CreatePaymentRequest struct {
    AmountVND   float64 `json:"amountVND" validate:"required,gt=0,lte=500000000"`
    OrderID     string  `json:"orderId" validate:"required,max=100"`
    CallbackURL string  `json:"callbackUrl" validate:"omitempty,url"`
}

validate := validator.New()
err := validate.Struct(req)
```

---

### Rate Limiting

```go
import "github.com/ulule/limiter/v3"

rate := limiter.Rate{Period: 1 * time.Minute, Limit: 100}
store := memory.NewStore()
instance := limiter.New(store, rate)

middleware := limiter.NewMiddleware(instance)
r.Use(middleware)
```

---

## 📦 Go Dependencies (go.mod)

```go
module github.com/yourusername/stable-payment-gateway

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/gagliardetto/solana-go v1.8.4
    github.com/ethereum/go-ethereum v1.13.5
    github.com/shopspring/decimal v1.3.1
    gorm.io/gorm v1.25.5
    gorm.io/driver/postgres v1.5.4
    github.com/redis/go-redis/v9 v9.3.0
    github.com/spf13/viper v1.17.0
    go.uber.org/zap v1.26.0
    github.com/golang-migrate/migrate/v4 v4.16.2
    github.com/hibiken/asynq v0.24.1
    github.com/google/uuid v1.4.0
    github.com/go-playground/validator/v10 v10.15.5
)
```

---

## 🎯 Implementation Priority (Golang)

### Week 1-2: Core Setup
1. Project structure + Makefile
2. Database models (GORM)
3. API server (Gin)
4. Auth middleware
5. Basic CRUD for merchants, payments

### Week 3-4: Blockchain Integration
6. Solana listener (goroutine)
7. BSC listener (goroutine)
8. Wallet management
9. Payment confirmation flow
10. Ledger service

### Week 5: Business Logic
11. Merchant dashboard (Next.js)
12. Payment page with QR
13. Webhook dispatcher
14. Payout request system

### Week 6: Testing & Deploy
15. Integration tests
16. Load testing (go test -bench)
17. Docker build + deploy
18. Pilot merchants

---

## ✅ Golang Benefits for This Project

1. **Performance**: Handle thousands of concurrent payments
2. **Concurrency**: Monitor multiple chains simultaneously (goroutines)
3. **Type Safety**: Reduce bugs in financial calculations
4. **Deployment**: Single binary, easy to deploy
5. **Ecosystem**: Excellent blockchain libraries
6. **Cost**: Lower server costs (less memory/CPU usage)

---

## 🚀 Next Steps

1. ✅ Confirm tech stack with team
2. 🔲 Set up Go project structure
3. 🔲 Initialize database schema
4. 🔲 Implement API endpoints
5. 🔲 Build blockchain listeners
6. 🔲 Deploy to staging

---

**Ready to start building? 🚀**
