# Tech Stack Decisions - Stablecoin Payment Gateway

**Last Updated**: 2025-11-18
**Status**: ✅ Approved
**Architecture Principle**: **Monolithic-first with Modular Design** (microservices-ready)

---

## 🎯 Architectural Philosophy

### Monolithic with Modular Boundaries

**Decision**: Start with **monolithic deployment** but enforce **strict module boundaries** internally.

**Why?**
- ✅ Faster MVP development (no inter-service communication overhead)
- ✅ Simpler deployment and debugging
- ✅ Lower operational complexity (single database transaction, no distributed tracing needed)
- ✅ Easy to extract services later (when needed)

**How?**
```
Single Go binary with:
├── internal/service/payment/      # Could become Payment Service
├── internal/service/ledger/       # Could become Ledger Service
├── internal/service/compliance/   # Could become Compliance Service
├── internal/service/analytics/    # Could become Analytics Service (v2.0)
├── internal/blockchain/solana/    # Could become Blockchain Listener Service
└── internal/worker/               # Could become Worker Service

Decoupling Strategy:
- Services only communicate via interfaces (no direct struct access)
- Clear bounded contexts (DDD principles)
- Events published to internal event bus (future: external message queue)
```

**Migration Path to Microservices (when volume > $1M/month)**:
1. Extract Blockchain Listener first (most CPU-intensive)
2. Extract Analytics Service (separate read workload)
3. Extract Worker Service (independent scaling)
4. Keep Ledger + Payment in monolith (ACID transactions critical)

---

## 🗄️ Data Layer Decisions

### 1. Primary Database: PostgreSQL 15

**Decision**: ✅ **PostgreSQL 15** with advanced features

**Features Used**:
- ✅ **Partitioning** (audit logs by year)
- ✅ **JSONB** (metadata storage)
- ✅ **Row-level security** (multi-tenancy for merchants)
- ✅ **Logical replication** (for CDC to analytics DB in v2.0)

**Partitioning Strategy for Audit Logs**:
```sql
-- 5-year retention requirement
CREATE TABLE audit_logs (
    id BIGSERIAL,
    event_time TIMESTAMP NOT NULL,
    ...
) PARTITION BY RANGE (event_time);

-- Partitions
CREATE TABLE audit_logs_2025 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
-- ... create partitions for 2026-2029

-- Auto-archive job (runs monthly)
-- Archive partitions > 2 years old to S3 Glacier
```

---

### 2. Cold Storage: AWS S3 Glacier

**Decision**: ✅ **AWS S3 with Glacier lifecycle policy**

**Use Cases**:
1. **KYC Documents** (encrypted at rest)
   - S3 Standard (frequent access during KYC review)
   - Auto-transition to Glacier after 90 days
2. **Audit Logs Archive** (5-year retention)
   - PostgreSQL: Active 2 years (OLTP)
   - S3 Glacier: 3-5 years (compliance archive)

**Why S3 Glacier?**
- ✅ Cheapest long-term storage ($0.004/GB/month)
- ✅ Compliance-ready (WORM, audit trails)
- ✅ Regional options (Singapore for Vietnam proximity)
- ❌ MinIO considered but lacks automatic lifecycle policies

**Implementation**:
```go
// internal/pkg/storage/s3.go
type S3Storage struct {
    client *s3.Client
}

// Upload KYC document
func (s *S3Storage) UploadKYC(merchantID uuid.UUID, docType string, file io.Reader) (string, error) {
    key := fmt.Sprintf("kyc/%s/%s/%s", merchantID, docType, uuid.New())
    // Upload to S3 with SSE-S3 encryption
    // Lifecycle: Standard (90d) → Glacier
}

// Archive audit logs
func (s *S3Storage) ArchiveAuditLogs(year int, data []byte) error {
    key := fmt.Sprintf("audit-logs/%d/archive.json.gz", year)
    // Upload compressed JSON to Glacier storage class
}
```

---

### 3. Cache & Pub/Sub: Redis 7

**Decision**: ✅ **Redis 7** (multiple use cases)

**Use Cases**:
1. **Caching** (exchange rates, merchant configs)
2. **Rate Limiting** (token bucket algorithm)
3. **Session Management** (admin JWT blacklist)
4. **Pub/Sub** (WebSocket real-time events - MVP v1.1)

**Pub/Sub for WebSocket** (MVP v1.1):
```go
// Publish payment event
redis.Publish(ctx, "payment_events:"+paymentID, payloadJSON)

// WebSocket handler subscribes
pubsub := redis.Subscribe(ctx, "payment_events:"+paymentID)
for msg := range pubsub.Channel() {
    websocket.WriteJSON(msg.Payload)
}
```

**Limitations**:
- ⚠️ Fire-and-forget (no message persistence)
- ⚠️ Not suitable for CDC/analytics (use NATS in v2.0)

---

### 4. Message Queue: NATS JetStream (v2.0)

**Decision**: ✅ **NATS JetStream** (added in v2.0, NOT MVP v1.1)

**Why NATS over Kafka?**
- ✅ Lightweight (20MB binary vs 500MB+ Kafka + ZooKeeper)
- ✅ Lower latency (<1ms vs ~10ms Kafka)
- ✅ Simpler ops (single binary, no ZooKeeper)
- ✅ Perfect fit for CDC event streaming (1K-10K msg/sec)
- ❌ Kafka considered but overkill for initial scale

**Use Cases (v2.0+)**:
1. **CDC Pipeline**: Debezium → NATS → TimescaleDB
2. **Event-Driven Architecture**: Payment events, ledger events, compliance events
3. **Analytics Stream**: Real-time data for SaaS dashboard

**Architecture**:
```
PostgreSQL (Ledger)
    ↓ (Debezium CDC)
NATS JetStream Topics:
    - ledger.payment.created
    - ledger.payment.completed
    - ledger.payout.requested
    ↓ (Consumers)
├── TimescaleDB Writer (analytics)
├── Webhook Dispatcher (notifications)
└── Fraud Detection Service (v3.0)
```

---

### 5. Analytics Database: TimescaleDB (v2.0)

**Decision**: ✅ **TimescaleDB** (PostgreSQL extension)

**Why TimescaleDB over ClickHouse?**
| Criteria | TimescaleDB | ClickHouse |
|----------|-------------|------------|
| **Query Language** | SQL (PostgreSQL) ✅ | Custom SQL dialect ❌ |
| **Time-series** | Native hypertables ✅ | Requires custom schema ❌ |
| **Operational Complexity** | Low (Postgres extension) ✅ | High (new DB system) ❌ |
| **Write Performance** | 100K inserts/sec ✅ | 1M+ inserts/sec (overkill) |
| **Team Knowledge** | Already know Postgres ✅ | Learning curve ❌ |

**Deployment**: Separate TimescaleDB instance (not same as OLTP PostgreSQL)

**Schema Design**:
```sql
-- Analytics hypertable
CREATE TABLE payment_analytics (
    time TIMESTAMPTZ NOT NULL,
    merchant_id UUID NOT NULL,
    amount_vnd DECIMAL(15,2),
    amount_usd DECIMAL(15,2),
    currency VARCHAR(10),
    chain VARCHAR(20),
    payer_country CHAR(2),
    status VARCHAR(50)
);

-- Convert to hypertable (time-series optimized)
SELECT create_hypertable('payment_analytics', 'time');

-- Continuous aggregates for "Giờ vàng" (peak hours)
CREATE MATERIALIZED VIEW hourly_volume
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS hour,
    merchant_id,
    SUM(amount_vnd) as total_volume,
    COUNT(*) as tx_count
FROM payment_analytics
GROUP BY hour, merchant_id;
```

---

## 🔐 Compliance & Security

### 6. AML Screening: TRM Labs → Own System

**Decision**: ✅ **Phased Approach**

**Phase 1 (MVP v1.1)**: **TRM Labs API**
- **Why?** Faster to integrate than Chainalysis
- **Cost**: ~$500/month (vs $1000+ Chainalysis)
- **Coverage**: OFAC, UN sanctions, mixing services
- **API**: REST API for wallet screening

**Phase 2 (v2.0)**: **Hybrid** (TRM Labs + Internal Rules)
- Build internal risk scoring based on:
  - Transaction patterns (velocity, amounts)
  - Merchant reputation
  - Payer country risk
- Use TRM Labs only for sanctions lists

**Phase 3 (v3.0)**: **Own AML System**
- Full on-chain analysis (Solana/BSC transaction graph)
- Machine learning for fraud detection
- Reduce dependency on 3rd party APIs

**Implementation (MVP v1.1)**:
```go
// internal/service/compliance/aml.go
type AMLService struct {
    trmClient *trm.Client
}

func (s *AMLService) ScreenWallet(ctx context.Context, address string, chain string) (*AMLResult, error) {
    // Call TRM Labs API
    result, err := s.trmClient.ScreenAddress(ctx, address, chain)
    if err != nil {
        return nil, err
    }

    if result.RiskScore > 75 || result.IsSanctioned {
        // Flag payment for manual review
        return &AMLResult{
            IsSafe: false,
            Reason: result.Flags,
        }, nil
    }

    return &AMLResult{IsSafe: true}, nil
}
```

---

## 🌐 Real-Time Communication

### 7. WebSocket: gorilla/websocket

**Decision**: ✅ **gorilla/websocket** (Go standard)

**Why not SSE (Server-Sent Events)?**
- WebSocket: Bidirectional (can add Payer chat in v3.0 Escrow disputes)
- SSE: Unidirectional (server → client only)
- Both work for payment status updates, WebSocket more flexible

**Implementation**:
```go
// internal/api/websocket/payment.go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *PaymentWSHandler) HandleConnection(c *gin.Context) {
    paymentID := c.Param("id")

    conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)
    defer conn.Close()

    // Subscribe to Redis pub/sub
    pubsub := h.redis.Subscribe(ctx, "payment_events:"+paymentID)
    defer pubsub.Close()

    for msg := range pubsub.Channel() {
        conn.WriteJSON(map[string]interface{}{
            "event": msg.Payload,
            "timestamp": time.Now(),
        })
    }
}
```

**Scaling Strategy**:
- MVP: Single server handles 100 concurrent WebSocket connections
- v2.0: Multiple servers + Redis Pub/Sub (broadcast to all servers)
- v3.0: Consider NATS for distributed WebSocket (if > 10K concurrent)

---

## 📊 Change Data Capture (v2.0)

### 8. CDC Tool: Debezium

**Decision**: ✅ **Debezium** (PostgreSQL → NATS)

**Why Debezium over PostgreSQL Logical Replication?**
- ✅ Production-ready (used by Netflix, Uber)
- ✅ Change event format standardized (JSON)
- ✅ Built-in transformations and filtering
- ✅ Supports NATS connector (via Kafka Connect API)

**Architecture**:
```
PostgreSQL (Write-Ahead Log)
    ↓
Debezium Connector
    ↓
NATS JetStream
    ↓ (Multiple Consumers)
├── TimescaleDB Writer (analytics)
├── Elasticsearch (search) [optional v3.0]
└── Webhook Queue (real-time notifications)
```

**Configuration**:
```json
// Debezium PostgreSQL connector
{
  "name": "payment-gateway-cdc",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres",
    "database.dbname": "payment_gateway",
    "table.include.list": "public.payments,public.ledger_entries,public.payouts",
    "publication.name": "payment_gateway_publication",
    "plugin.name": "pgoutput"
  }
}
```

**Deployment** (v2.0):
- Docker Compose: Debezium + NATS in same VPS initially
- v3.0: Separate server for CDC infrastructure

---

## 🔌 Plugin & SDK Architecture

### 9. Multi-Language Plugin Strategy

**Decision**: ✅ **Each plugin in native language** (NOT shared Go SDK)

**Why?**
- ✅ Each platform has different conventions:
  - **Shopify**: Node.js/React (Remix framework)
  - **WooCommerce**: PHP (WordPress hooks)
  - **Haravan/Sapo**: JavaScript (Vietnamese platform SDKs)
- ✅ Easier merchant adoption (familiar tech stack)
- ❌ Shared Go SDK (via WASM/CGO) considered too complex

**Shared Logic: REST API Only**

**Architecture**:
```
Shopify Plugin (Node.js)  ─┐
WooCommerce (PHP)         ─┼──→  Payment Gateway REST API
Haravan Plugin (JS)       ─┘      (Single source of truth)
```

**Plugin Responsibilities**:
1. Render payment UI in merchant's store
2. Call API: `POST /api/v1/payments`
3. Handle webhook: `POST /merchant/webhook/payment-completed`
4. Update order status in platform

**API Design for Plugins** (v2.0):
```go
// POST /api/v1/plugins/shopify/checkout
// - Auto-creates merchant if not exists (API key in header)
// - Returns payment URL for redirect
// - Registers webhook automatically
```

---

## 🎯 Escrow Architecture

### 10. State Machine Extension

**Decision**: ✅ **Extend current payment state machine** (NOT separate service)

**Why?**
- ✅ Escrow is tightly coupled to payment lifecycle
- ✅ Needs ACID transactions with ledger (same DB)
- ✅ Monolithic architecture principle

**Extended State Machine**:
```
Current (v1.0):
CREATED → PENDING → CONFIRMING → COMPLETED
                              ↓
                          EXPIRED/FAILED

Extended (v3.0 Escrow):
CREATED → PENDING → CONFIRMING → ESCROW_HELD
                              ↓
                          AWAITING_RELEASE
                              ↓
                    ┌─────────┴─────────┐
                    ↓                   ↓
              COMPLETED            DISPUTED
                                        ↓
                                   RESOLVED
```

**Ledger Accounting** (escrow):
```go
// When payment confirmed → ESCROW_HELD
DEBIT:  crypto_pool (+100 USDT)
CREDIT: escrow_liability_pool (+2,300,000 VND)

// When payer clicks "Release Funds" → COMPLETED
DEBIT:  escrow_liability_pool (+2,300,000 VND)
CREDIT: merchant_available_balance (+2,277,000 VND)
CREDIT: fee_revenue (+23,000 VND)
```

**Implementation**:
```go
// internal/model/payment.go
type PaymentStatus string

const (
    PaymentStatusCreated       PaymentStatus = "created"
    PaymentStatusPending       PaymentStatus = "pending"
    PaymentStatusConfirming    PaymentStatus = "confirming"
    PaymentStatusEscrowHeld    PaymentStatus = "escrow_held"     // NEW
    PaymentStatusAwaitingRelease PaymentStatus = "awaiting_release" // NEW
    PaymentStatusCompleted     PaymentStatus = "completed"
    PaymentStatusDisputed      PaymentStatus = "disputed"       // NEW v3.1
    PaymentStatusExpired       PaymentStatus = "expired"
    PaymentStatusFailed        PaymentStatus = "failed"
)

// Escrow-specific fields
type Payment struct {
    // ... existing fields
    IsEscrow         bool       `json:"is_escrow"`
    EscrowReleasedAt *time.Time `json:"escrow_released_at,omitempty"`
    EscrowReleasedBy uuid.UUID  `json:"escrow_released_by,omitempty"` // Payer ID
}
```

---

## 🚀 Deployment Stack

**MVP v1.1 - v2.0**: Single VPS (Monolithic)
```yaml
# docker-compose.yml
services:
  app:
    image: payment-gateway:latest
    # Single Go binary (api + listener + worker)

  postgres:
    image: postgres:15-alpine
    # Partitioned audit_logs

  redis:
    image: redis:7-alpine
    # Cache + Pub/Sub

  nginx:
    image: nginx:alpine
    # Reverse proxy + SSL
```

**v2.0 (with Analytics)**:
```yaml
services:
  # ... existing services

  timescaledb:
    image: timescale/timescaledb:latest-pg15
    # Separate analytics DB

  nats:
    image: nats:alpine
    # Message queue for CDC

  debezium:
    image: debezium/connect:latest
    # CDC connector
```

**v3.0 (Microservices - if needed)**:
```yaml
# Extract services when monolith can't scale
services:
  api-gateway:
    # HTTP API only

  ledger-service:
    # Payment + Ledger (keep together for ACID)

  blockchain-listener:
    # Separate CPU-intensive service

  analytics-service:
    # Separate read workload
```

---

## 📦 Frontend Stack

### 11. Payer Experience Layer (MVP v1.1)

**Decision**: ✅ **Next.js 14 App Router**

**Tech Stack**:
```json
{
  "framework": "Next.js 14 (App Router)",
  "language": "TypeScript",
  "styling": "TailwindCSS + shadcn/ui",
  "state": "React Query (TanStack Query)",
  "websocket": "native WebSocket API",
  "qr": "qrcode.react"
}
```

**Deployment**:
- **MVP**: Same VPS as backend (static build, served by NGINX)
- **v2.0+**: Vercel or Cloudflare Pages (if traffic grows)

**Pages**:
```
/order/[payment_id]         # Payment status page (public)
/order/[payment_id]/success # Success page
/dashboard                  # Merchant dashboard (protected)
/admin                      # Admin panel (protected)
```

---

## 🧪 Analytics Engine

### 12. Analytics Service: Go (v2.0)

**Decision**: ✅ **Build in Go** (same language as monolith)

**Why Go over Python/Node.js?**
- ✅ Team already knows Go (no context switching)
- ✅ Share models/types with main app (import `internal/model`)
- ✅ Better performance for real-time aggregations
- ❌ Python considered for ML (defer to v3.0 Fraud Detection)

**Architecture**:
```
internal/service/analytics/
├── consumer.go        # NATS consumer (reads CDC events)
├── aggregator.go      # Real-time aggregations
├── insights.go        # "Giờ vàng", payer analysis
└── writer.go          # Write to TimescaleDB
```

**Example Insight**: "Giờ vàng" (Peak Hours Analysis)
```go
// internal/service/analytics/insights.go
type PeakHoursInsight struct {
    MerchantID uuid.UUID
    PeakHours  []int // [14, 15, 20, 21] (2PM, 3PM, 8PM, 9PM)
    AvgVolume  decimal.Decimal
    Recommendation string // "Schedule promotions at 2-3PM for +25% volume"
}

func (s *AnalyticsService) GetPeakHours(merchantID uuid.UUID) (*PeakHoursInsight, error) {
    // Query TimescaleDB continuous aggregate
    rows := s.timescaleDB.Query(`
        SELECT
            EXTRACT(HOUR FROM hour) as hour,
            AVG(total_volume) as avg_volume
        FROM hourly_volume
        WHERE merchant_id = $1
        GROUP BY hour
        ORDER BY avg_volume DESC
        LIMIT 4
    `, merchantID)
    // ...
}
```

---

## 📋 Summary: Technology Stack by Phase

### MVP v1.1 (Compliance + Payer Layer)
```
Backend:  Golang (Gin + GORM + solana-go)
Database: PostgreSQL 15 (partitioned audit logs)
Cache:    Redis 7 (cache + pub/sub)
Storage:  AWS S3 + Glacier (KYC + archives)
AML:      TRM Labs API
Frontend: Next.js 14 + TypeScript + TailwindCSS
WebSocket: gorilla/websocket + Redis Pub/Sub
Deploy:   Docker Compose (single VPS)
```

### v2.0 (SDKs + Analytics)
```
MVP v1.1 +
Message Queue: NATS JetStream
CDC:           Debezium (PostgreSQL → NATS)
Analytics DB:  TimescaleDB (separate instance)
Analytics:     Go service (NATS consumer → TimescaleDB writer)
Plugins:       Shopify (Node.js), WooCommerce (PHP), Haravan (JS)
```

### v3.0 (Escrow + Scale)
```
v2.0 +
State Machine: Extended (ESCROW_HELD, DISPUTED)
Ledger:        Escrow liability accounts
Payer UI:      "Release Funds" button
Optional:      Extract Blockchain Listener to separate service
Optional:      ML Fraud Detection (Python/Go)
```

---

## 🎯 Key Architectural Principles

1. ✅ **Monolithic first**, microservices when needed (>$1M volume)
2. ✅ **Modular boundaries** (internal packages as "proto-services")
3. ✅ **Single source of truth** (PostgreSQL for OLTP, TimescaleDB for OLAP)
4. ✅ **Event-driven ready** (Redis Pub/Sub → NATS migration path)
5. ✅ **Horizontal scaling** (stateless app servers behind load balancer)
6. ✅ **Observability** (structured logging, metrics, distributed tracing in v2.0)
7. ✅ **Cost-conscious** (use managed services only when necessary)

---

## ⚖️ Trade-offs Accepted

| Decision | Trade-off | Mitigation |
|----------|-----------|------------|
| **Monolithic** | Harder to scale individual components | Modular design, extract later |
| **TimescaleDB over ClickHouse** | Lower write throughput | Good enough for <100K events/day |
| **NATS over Kafka** | Less mature ecosystem | Simpler ops, perfect for our scale |
| **TRM Labs (paid)** | Ongoing cost (~$500/mo) | Build own system in v3.0 |
| **Redis Pub/Sub** | No message persistence | Migrate to NATS in v2.0 for CDC |

---

**Status**: ✅ Ready for Implementation
**Next Step**: Start MVP v1.1 development with this stack
**Review Date**: After v2.0 launch (re-evaluate microservices need)
