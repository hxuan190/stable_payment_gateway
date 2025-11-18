# Requirements: v2.0 Pillar 2 - Dịch vụ Giá trị Gia tăng (SaaS & Insights)

**Phase**: v2.0 Quarter 1-2 (parallel với Pillar 1)
**Timeline**: 8-10 weeks
**Status**: 🟢 Retention & Revenue Strategy

---

## 🎯 Mục tiêu Phase

**Trụ cột 2** giải quyết bài toán **Giữ chân khách hàng** (Retention) và tạo **dòng doanh thu ổn định** (MRR - Monthly Recurring Revenue).

### Strategic Transformation
- **FROM**: "Chi phí" (Cost Center) - Merchant chỉ trả phí giao dịch → dễ chuyển sang đối thủ rẻ hơn
- **TO**: "Cộng sự" (Partner) - Cung cấp insights giúp Merchant tăng doanh thu → khó rời bỏ

### Value Proposition
"Chúng tôi không chỉ LẤY tiền của bạn (phí), chúng tôi giúp bạn KIẾM được nhiều tiền hơn."

### Revenue Model
- **Free Tier**: Basic analytics (luôn miễn phí)
- **Pro Tier**: Advanced insights - $29/month
- **Enterprise**: Custom reports + API access - $99/month

---

## 📦 Epic 1: CDC Architecture (Change Data Capture)

### 🎯 Technical Context

**Problem**: Nếu chạy analytics queries trực tiếp trên Ledger database (TDD 3.1):
- ❌ OLAP queries nặng → ảnh hưởng OLTP (giao dịch) performance
- ❌ Không scale được khi merchant base tăng
- ❌ Rủi ro: Analytics down → kéo theo payment processing down

**Solution**: **Tách biệt hoàn toàn** OLTP (giao dịch) vs OLAP (phân tích) bằng CDC.

---

### Feature 1.1: Debezium CDC Integration

**Priority**: 🔴 P0 (Foundation)

#### User Stories

**Story 1.1.1**: Setup Debezium Connector for PostgreSQL
```
As a: System Architect
I want to: Deploy Debezium để capture changes từ Ledger database
So that: Stream data sang Data Warehouse mà không ảnh hưởng production DB
```

**Acceptance Criteria**:
- [ ] Debezium PostgreSQL connector deployed (Docker hoặc Kubernetes)
- [ ] Monitor PostgreSQL Write-Ahead Log (WAL) của `ledger_entries` table
- [ ] Capture events:
  - `INSERT` (mỗi bút toán mới)
  - NO UPDATE/DELETE (ledger is immutable)
- [ ] Publish events vào Kafka topic: `ledger.public.ledger_entries`

**Technical Setup**:
```yaml
# docker-compose.yml (Debezium service)
services:
  debezium:
    image: debezium/connect:2.5
    ports:
      - "8083:8083"
    environment:
      - BOOTSTRAP_SERVERS=kafka:9092
      - GROUP_ID=1
      - CONFIG_STORAGE_TOPIC=my_connect_configs
      - OFFSET_STORAGE_TOPIC=my_connect_offsets
    depends_on:
      - kafka
      - postgres

  # Debezium Connector Config (POST to Debezium API)
  # POST http://localhost:8083/connectors
  {
    "name": "ledger-connector",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "postgres",
      "database.port": "5432",
      "database.user": "debezium_user",
      "database.password": "secret",
      "database.dbname": "payment_gateway",
      "database.server.name": "ledger",
      "table.include.list": "public.ledger_entries",
      "plugin.name": "pgoutput"
    }
  }
```

**Impact**: Near-zero impact trên Ledger database (chỉ đọc WAL file).

---

### Feature 1.2: Kafka Message Queue Setup

**Priority**: 🔴 P0 (Foundation)

#### User Stories

**Story 1.2.1**: Deploy Kafka Cluster
```
As a: DevOps Engineer
I want to: Setup Kafka cluster để stream ledger events
So that: Analytics Service có thể consume real-time data
```

**Acceptance Criteria**:
- [ ] Kafka cluster: 3 brokers (high availability)
- [ ] Topic: `ledger-events` (from Debezium)
  - Partitions: 6 (for scalability)
  - Replication factor: 2
  - Retention: 7 days (cho reprocessing nếu cần)
- [ ] Kafka Connect: Debezium connector deployed
- [ ] Monitoring: Kafka lag, throughput metrics

**Message Format** (from Debezium):
```json
{
  "before": null,
  "after": {
    "id": "uuid-123",
    "debit_account": "merchant_A_receivable",
    "credit_account": "hot_wallet_usdt_liability",
    "amount": "100.000000",
    "currency": "USDT",
    "reference_type": "payment",
    "reference_id": "payment-uuid-456",
    "created_at": 1700000000000
  },
  "source": {
    "db": "payment_gateway",
    "table": "ledger_entries"
  },
  "op": "c", // create
  "ts_ms": 1700000000123
}
```

---

### Feature 1.3: Analytics Service (ETL)

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 1.3.1**: Build Analytics Service Consumer
```
As a: Analytics Service
I want to: Consume `ledger-events` topic và transform data
So that: Load vào Data Warehouse cho phân tích
```

**Acceptance Criteria**:
- [ ] Service: `internal/services/analytics/consumer.go`
- [ ] Consume Kafka topic: `ledger-events`
- [ ] Transform logic:
  - Parse ledger entries
  - Aggregate by merchant_id, date, hour
  - Calculate metrics:
    - `total_revenue_vnd` (sum of all credit to merchant accounts)
    - `total_volume_crypto` (sum of all debits to hot wallet)
    - `transaction_count`
    - `average_transaction_size`
    - `hourly_revenue` (for "Giờ vàng" analysis)
- [ ] Load vào Data Warehouse tables:
  - `analytics.merchant_daily_stats`
  - `analytics.merchant_hourly_stats`
  - `analytics.payer_behavior`

**Go Implementation**:
```go
// internal/services/analytics/consumer.go
package analytics

import (
    "github.com/segmentio/kafka-go"
    "encoding/json"
)

type LedgerEventConsumer struct {
    reader     *kafka.Reader
    warehouse  *DataWarehouse
}

func (c *LedgerEventConsumer) Start() {
    for {
        msg, err := c.reader.ReadMessage(context.Background())
        if err != nil {
            log.Error("Failed to read message", err)
            continue
        }

        var event DebeziumEvent
        json.Unmarshal(msg.Value, &event)

        // Transform & aggregate
        stats := c.transformLedgerEntry(event.After)

        // Load to warehouse
        c.warehouse.UpsertDailyStats(stats)
    }
}

func (c *LedgerEventConsumer) transformLedgerEntry(entry LedgerEntry) MerchantDailyStats {
    // Extract merchant_id from account name (e.g., "merchant_A_receivable" → "A")
    merchantID := extractMerchantID(entry.CreditAccount)

    // Aggregate logic
    return MerchantDailyStats{
        MerchantID:   merchantID,
        Date:         entry.CreatedAt.Truncate(24 * time.Hour),
        TotalRevenue: entry.Amount,
        TxCount:      1,
    }
}
```

---

## 📦 Epic 2: Data Warehouse Setup

### Feature 2.1: Choose Data Warehouse Technology

**Priority**: 🔴 P0 (Foundation)

#### Options Analysis

**Option A: ClickHouse** (Recommended)
- ✅ Extremely fast for OLAP queries (100x faster than PostgreSQL)
- ✅ Columnar storage → perfect for analytics
- ✅ Open-source, self-hosted
- ❌ Learning curve (SQL dialect hơi khác)

**Option B: TimescaleDB**
- ✅ PostgreSQL extension → dễ học
- ✅ Time-series data (perfect cho metrics)
- ❌ Slower than ClickHouse cho large aggregations

**Decision**: **ClickHouse** (optimized for speed, future-proof for scale)

---

### Feature 2.2: ClickHouse Schema Design

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 2.2.1**: Design Analytics Tables in ClickHouse
```
As a: Data Engineer
I want to: Thiết kế schema tối ưu cho analytics queries
So that: Dashboard queries trả về kết quả < 1 second
```

**ClickHouse Tables**:

```sql
-- Table 1: Merchant Daily Stats (aggregated)
CREATE TABLE analytics.merchant_daily_stats (
    merchant_id UUID,
    date Date,
    total_revenue_vnd Decimal(20, 2),
    total_volume_crypto Decimal(20, 8),
    transaction_count UInt32,
    average_transaction_vnd Decimal(20, 2),
    unique_payers UInt32
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (merchant_id, date);

-- Table 2: Hourly Stats (for "Giờ vàng" analysis)
CREATE TABLE analytics.merchant_hourly_stats (
    merchant_id UUID,
    hour DateTime,
    revenue_vnd Decimal(20, 2),
    tx_count UInt32
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (merchant_id, hour);

-- Table 3: Payer Behavior (for insights)
CREATE TABLE analytics.payer_behavior (
    payer_wallet_address String,
    merchant_id UUID,
    first_payment_at DateTime,
    last_payment_at DateTime,
    total_payments UInt32,
    total_amount_vnd Decimal(20, 2),
    avg_payment_vnd Decimal(20, 2)
)
ENGINE = ReplacingMergeTree()
ORDER BY (payer_wallet_address, merchant_id);

-- Table 4: Raw Ledger Events (for historical queries)
CREATE TABLE analytics.ledger_events_raw (
    id UUID,
    event_time DateTime,
    debit_account String,
    credit_account String,
    amount Decimal(20, 8),
    currency String,
    reference_type String,
    reference_id UUID,
    metadata String -- JSON
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (event_time, reference_id);
```

**Partitioning Strategy**:
- Partition by month (toYYYYMM) → query chỉ scan relevant partitions
- TTL: 24 months (sau đó archive sang S3)

---

## 📦 Epic 3: Analytics API Endpoints

### Feature 3.1: Dashboard Analytics APIs

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 3.1.1**: Revenue Overview API
```
As a: Merchant Dashboard
I want to: Gọi API để lấy revenue overview của merchant
So that: Hiển thị charts trong "Analytics" tab
```

**API Endpoint**:
```
GET /api/v1/analytics/revenue-overview?merchant_id={uuid}&period=30d
```

**Response**:
```json
{
  "period": "30d",
  "total_revenue_vnd": 150000000,
  "total_transactions": 245,
  "average_transaction_vnd": 612244,
  "growth_vs_prev_period": 12.5, // %
  "daily_breakdown": [
    {
      "date": "2025-11-01",
      "revenue_vnd": 5200000,
      "tx_count": 8
    },
    // ... 29 more days
  ]
}
```

**ClickHouse Query**:
```sql
SELECT
    date,
    SUM(total_revenue_vnd) AS revenue_vnd,
    SUM(transaction_count) AS tx_count
FROM analytics.merchant_daily_stats
WHERE merchant_id = '{merchant_id}'
  AND date >= today() - INTERVAL 30 DAY
GROUP BY date
ORDER BY date;
```

---

**Story 3.1.2**: "Giờ vàng" Analysis API
```
As a: Merchant
I want to: Biết giờ nào trong ngày có doanh thu cao nhất
So that: Optimize marketing campaigns
```

**API Endpoint**:
```
GET /api/v1/analytics/golden-hours?merchant_id={uuid}&period=30d
```

**Response**:
```json
{
  "golden_hours": [
    {
      "hour": 14, // 2 PM
      "avg_revenue_vnd": 850000,
      "avg_tx_count": 12,
      "percentage_of_daily": 18.5
    },
    {
      "hour": 20, // 8 PM
      "avg_revenue_vnd": 720000,
      "avg_tx_count": 10,
      "percentage_of_daily": 15.2
    }
  ],
  "recommendation": "Your peak hours are 2 PM and 8 PM. Consider running promotions during these times."
}
```

**ClickHouse Query**:
```sql
SELECT
    toHour(hour) AS hour_of_day,
    AVG(revenue_vnd) AS avg_revenue,
    AVG(tx_count) AS avg_tx_count
FROM analytics.merchant_hourly_stats
WHERE merchant_id = '{merchant_id}'
  AND hour >= now() - INTERVAL 30 DAY
GROUP BY hour_of_day
ORDER BY avg_revenue DESC
LIMIT 5;
```

---

**Story 3.1.3**: Payer Behavior Insights API
```
As a: Merchant
I want to: Phân tích hành vi của payers
So that: Hiểu customer retention và repeat purchase rate
```

**API Endpoint**:
```
GET /api/v1/analytics/payer-insights?merchant_id={uuid}
```

**Response**:
```json
{
  "total_unique_payers": 128,
  "repeat_payers": 42, // Paid 2+ times
  "repeat_rate": 32.8, // %
  "top_payers": [
    {
      "wallet_address": "0xabc...",
      "total_payments": 8,
      "total_amount_vnd": 12500000,
      "avg_amount_vnd": 1562500,
      "first_payment": "2025-10-01",
      "last_payment": "2025-11-15"
    }
  ],
  "avg_customer_lifetime_value_vnd": 950000
}
```

---

### Feature 3.2: Cash Flow Forecasting API

**Priority**: 🟡 P1 (High value, but can be v2.1)

#### User Stories

**Story 3.2.1**: Predict Next 30 Days Revenue
```
As a: Merchant
I want to: Xem forecast doanh thu 30 ngày tới
So that: Lập kế hoạch tài chính (cash flow, payout scheduling)
```

**API Endpoint**:
```
GET /api/v1/analytics/forecast?merchant_id={uuid}&horizon=30d
```

**Response**:
```json
{
  "forecast_period": "2025-11-18 to 2025-12-18",
  "predicted_revenue_vnd": 18500000,
  "confidence_interval": {
    "low": 15000000,
    "high": 22000000
  },
  "daily_predictions": [
    {
      "date": "2025-11-19",
      "predicted_revenue_vnd": 620000
    }
    // ... 29 more days
  ],
  "model": "linear_regression", // Or ARIMA, Prophet
  "accuracy_last_month": 87.5 // %
}
```

**ML Model** (Simple MVP):
- Use **Facebook Prophet** library (Python)
- Train on historical `merchant_daily_stats` data
- Expose via gRPC or REST API

---

## 📦 Epic 4: Dashboard UI Enhancements

### Feature 4.1: Analytics Tab in Merchant Dashboard

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 4.1.1**: Analytics Dashboard Page
```
As a: Merchant
I want to: Truy cập tab "Analytics" trong Dashboard
So that: Xem insights về doanh thu và khách hàng
```

**UI Components** (Next.js + shadcn/ui):
- **Revenue Chart**: Line chart (last 30 days)
- **Golden Hours**: Bar chart (revenue by hour of day)
- **Payer Insights**: Table (top payers, repeat rate)
- **Cash Flow Forecast**: Line chart với confidence interval

**Tech Stack**:
- Charts: Recharts hoặc Chart.js
- Data fetching: React Query
- Real-time updates: WebSocket (optional, for live metrics)

---

### Feature 4.2: Export Reports (CSV/PDF)

**Priority**: 🟡 P1 (High)

#### User Stories

**Story 4.2.1**: Export Analytics Report
```
As a: Merchant
I want to: Export analytics report dưới dạng CSV hoặc PDF
So that: Chia sẻ với accountant hoặc đối tác
```

**API Endpoint**:
```
GET /api/v1/analytics/export?merchant_id={uuid}&format=csv&period=30d
```

**CSV Format**:
```csv
Date,Revenue (VND),Transactions,Average (VND)
2025-11-01,5200000,8,650000
2025-11-02,6100000,10,610000
...
```

**PDF Generation**: Use library như `wkhtmltopdf` hoặc `puppeteer`

---

## 📦 Epic 5: Subscription Model (Pro/Enterprise Tiers)

### Feature 5.1: Tiered Pricing Implementation

**Priority**: 🟡 P1 (Revenue)

#### User Stories

**Story 5.1.1**: Define Analytics Tiers
```
As a: Product Team
I want to: Định nghĩa các tiers cho analytics features
So that: Merchant có thể upgrade để unlock advanced insights
```

**Pricing Tiers**:
| Feature | Free | Pro ($29/mo) | Enterprise ($99/mo) |
|---------|------|--------------|---------------------|
| Revenue Overview | ✅ Last 7 days | ✅ Unlimited | ✅ Unlimited |
| Golden Hours | ❌ | ✅ | ✅ |
| Payer Insights | ❌ | ✅ Basic | ✅ Advanced |
| Cash Flow Forecast | ❌ | ❌ | ✅ |
| Export Reports | ❌ | ✅ CSV | ✅ CSV + PDF |
| API Access | ❌ | ❌ | ✅ |
| Custom Dashboards | ❌ | ❌ | ✅ |

**Database Schema**:
```sql
ALTER TABLE merchants
ADD COLUMN analytics_tier VARCHAR(20) DEFAULT 'free'
    CHECK (analytics_tier IN ('free', 'pro', 'enterprise')),
ADD COLUMN analytics_subscription_id VARCHAR(255), -- Stripe subscription ID
ADD COLUMN analytics_subscription_expires_at TIMESTAMP;
```

---

**Story 5.1.2**: Stripe Integration for Subscriptions
```
As a: Merchant
I want to: Upgrade to Pro tier bằng cách thanh toán qua Stripe
So that: Unlock advanced analytics features
```

**Flow**:
1. Merchant clicks "Upgrade to Pro" trong Dashboard
2. Redirect to Stripe Checkout (hoặc embedded form)
3. After successful payment:
   - Stripe webhook → update `analytics_tier = 'pro'`
   - Enable advanced features
4. Recurring billing: Stripe auto-charges monthly

**API Endpoint**:
```
POST /api/v1/merchants/{id}/analytics/subscribe
{
  "tier": "pro", // or "enterprise"
  "payment_method": "stripe"
}
```

---

## 🧪 Testing Requirements

### Performance Tests
- [ ] ClickHouse query performance: Revenue Overview API < 500ms (for 1 year data)
- [ ] Golden Hours API < 300ms
- [ ] CDC lag: < 5 seconds (from Ledger insert → ClickHouse)

### Data Accuracy Tests
- [ ] Validate aggregations: Sum of `merchant_daily_stats` == sum of `ledger_entries`
- [ ] Test CDC failover: Stop Debezium → restart → no data loss

---

## 📊 Success Metrics

- [ ] **Merchant Engagement**: 60% merchants visit Analytics tab weekly
- [ ] **Upgrade Rate**: 20% merchants upgrade to Pro tier (within 3 months)
- [ ] **MRR Growth**: $5K+ MRR from analytics subscriptions (target month 6)
- [ ] **Switching Cost**: Merchants who use analytics are 3x less likely to churn

---

## ⚠️ Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| ClickHouse learning curve steep | 🐢 Delayed launch | Start with TimescaleDB (easier), migrate to ClickHouse later |
| CDC pipeline failures | 📊 Stale analytics | Implement monitoring + alerting, fallback to batch ETL |
| Merchants don't see value | 💰 Low upgrade rate | A/B test messaging, provide case studies showing ROI |

---

**Next Steps**: Launch Pillar 2 parallel với Pillar 1 → Merchants vừa onboard (SDKs) vừa thấy giá trị (Insights) → High retention → Ready for Pillar 3 (Escrow).
