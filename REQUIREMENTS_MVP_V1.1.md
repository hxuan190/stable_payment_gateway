# Requirements: MVP v1.1 - Nền tảng Tuân thủ

**Phase**: MVP v1.1 (Pre-Sandbox Application)
**Timeline**: Ưu tiên cao nhất - Trước khi nộp hồ sơ Sandbox Đà Nẵng
**Status**: 🔴 CRITICAL - Bắt buộc để được cấp phép

---

## 🎯 Mục tiêu Phase

MVP v1.1 là phiên bản **có khả năng được cấp phép** (licensable) của TDD v1.0. Nó bổ sung hai thành phần bắt buộc mà Basal Pay đã chứng minh là tiêu chuẩn tối thiểu của Sandbox Đà Nẵng:

1. **Compliance Engine** (nâng cấp từ AML Engine)
2. **Payer Experience Layer** (đưa vào MVP từ v2.0)

---

## 📦 Epic 1: Compliance Engine (Nâng cấp AML Engine)

### 🎯 Business Context
- **Vấn đề**: TDD v1.0 chỉ có "AML Engine" cơ bản (TDD 3.4)
- **Tiêu chuẩn**: Basal Pay đã thiết lập chuẩn cao hơn: **FATF Travel Rule** + KYC 3 tiers + lưu trữ 5 năm
- **Giải pháp**: Nâng cấp thành "Compliance Engine" đầy đủ

---

### Feature 1.1: FATF Travel Rule Integration

**Priority**: 🔴 P0 (Blocker)

#### User Stories

**Story 1.1.1**: Collect Travel Rule Data (Payer Information)
```
As a: System
I want to: Collect và lưu trữ thông tin Payer theo chuẩn FATF
So that: Tuân thủ yêu cầu "Travel Rule" khi giao dịch > $1,000 USD
```

**Acceptance Criteria**:
- [ ] Khi tạo payment với amount > $1,000 USD:
  - System bắt buộc thu thập:
    - Payer full name
    - Payer wallet address (originating address)
    - Payer country of residence
    - Optional: Payer identification document number
- [ ] Lưu vào bảng `travel_rule_data` (liên kết với `payments.id`)
- [ ] Validation: không được tạo payment nếu thiếu dữ liệu bắt buộc

**Technical Implementation**:
```go
// internal/model/travel_rule.go
type TravelRuleData struct {
    ID                  uuid.UUID
    PaymentID           uuid.UUID // FK to payments
    PayerFullName       string    `validate:"required"`
    PayerWalletAddress  string    `validate:"required,crypto_address"`
    PayerCountry        string    `validate:"required,iso3166"`
    PayerIDDocument     string    // Optional
    MerchantFullName    string    // From merchants table
    MerchantCountry     string    // From merchants table
    TransactionAmount   decimal.Decimal
    TransactionCurrency string
    CreatedAt           time.Time
}
```

**Database Migration**:
```sql
-- migrations/XXX_create_travel_rule_data.up.sql
CREATE TABLE travel_rule_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    payer_full_name VARCHAR(255) NOT NULL,
    payer_wallet_address VARCHAR(255) NOT NULL,
    payer_country CHAR(2) NOT NULL, -- ISO 3166
    payer_id_document VARCHAR(255),
    merchant_full_name VARCHAR(255) NOT NULL,
    merchant_country CHAR(2) NOT NULL,
    transaction_amount DECIMAL(20,8) NOT NULL,
    transaction_currency VARCHAR(10) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    INDEX idx_payment_id (payment_id),
    INDEX idx_created_at (created_at)
);
```

**API Changes**:
```go
// POST /api/v1/payments
type CreatePaymentRequest struct {
    MerchantID  uuid.UUID `json:"merchant_id"`
    AmountVND   decimal.Decimal `json:"amount_vnd"`
    Currency    string `json:"currency"` // "USDT", "USDC"
    Chain       string `json:"chain"` // "solana", "bsc"

    // NEW: Travel Rule data (required if amount_usd > 1000)
    TravelRule  *TravelRuleRequest `json:"travel_rule,omitempty"`
}

type TravelRuleRequest struct {
    PayerFullName      string `json:"payer_full_name" validate:"required"`
    PayerWalletAddress string `json:"payer_wallet_address" validate:"required"`
    PayerCountry       string `json:"payer_country" validate:"required,iso3166"`
    PayerIDDocument    string `json:"payer_id_document,omitempty"`
}
```

---

**Story 1.1.2**: Travel Rule Reporting API (for Regulators)
```
As a: Compliance Officer / Regulator
I want to: Truy xuất dữ liệu Travel Rule theo thời gian hoặc giao dịch
So that: Báo cáo định kỳ cho Sở KH&CN Đà Nẵng
```

**Acceptance Criteria**:
- [ ] API endpoint: `GET /admin/v1/compliance/travel-rule`
- [ ] Query parameters:
  - `start_date`, `end_date` (required)
  - `min_amount` (optional, default: 1000 USD)
  - `country` (optional filter)
- [ ] Response format: CSV export hoặc JSON
- [ ] Chỉ admin với role `compliance_officer` mới được truy cập

---

### Feature 1.2: Three-Tier Identification (KYC Levels)

**Priority**: 🔴 P0 (Blocker)

#### User Stories

**Story 1.2.1**: Implement KYC Tier System for Merchants
```
As a: System
I want to: Phân loại Merchant theo 3 tier KYC
So that: Tuân thủ yêu cầu AML và đặt transaction limits phù hợp
```

**KYC Tiers**:
| Tier | Requirements | Monthly Limit | Document Required |
|------|--------------|---------------|-------------------|
| **Tier 1** | Email + Phone | $5,000 USD | None |
| **Tier 2** | + Business Registration | $50,000 USD | Business License |
| **Tier 3** | + Full KYC | Unlimited | License + Tax ID + Bank Statement |

**Acceptance Criteria**:
- [ ] Bảng `merchants` thêm cột `kyc_tier` (enum: tier1, tier2, tier3)
- [ ] Bảng `merchants` thêm cột `monthly_limit_usd` (tự động set theo tier)
- [ ] Khi tạo payment, check:
  - `merchant.total_volume_this_month + payment.amount_usd <= merchant.monthly_limit_usd`
  - Nếu vượt → reject với error code `MONTHLY_LIMIT_EXCEEDED`
- [ ] Dashboard hiển thị:
  - Current tier
  - Monthly volume used / limit
  - "Upgrade to Tier X" button

**Database Migration**:
```sql
-- migrations/XXX_add_kyc_tier.up.sql
ALTER TABLE merchants
ADD COLUMN kyc_tier VARCHAR(10) NOT NULL DEFAULT 'tier1'
    CHECK (kyc_tier IN ('tier1', 'tier2', 'tier3')),
ADD COLUMN monthly_limit_usd DECIMAL(20,2) NOT NULL DEFAULT 5000.00,
ADD COLUMN total_volume_this_month_usd DECIMAL(20,2) NOT NULL DEFAULT 0,
ADD COLUMN volume_last_reset_at TIMESTAMP NOT NULL DEFAULT NOW();

CREATE INDEX idx_merchants_kyc_tier ON merchants(kyc_tier);
```

**Business Logic**:
```go
// internal/service/merchant_service.go
func (s *MerchantService) CheckMonthlyLimit(merchantID uuid.UUID, amountUSD decimal.Decimal) error {
    merchant, err := s.repo.GetByID(merchantID)
    if err != nil {
        return err
    }

    // Reset counter if new month
    if merchant.VolumeLastResetAt.Month() != time.Now().Month() {
        merchant.TotalVolumeThisMonthUSD = decimal.Zero
        merchant.VolumeLastResetAt = time.Now()
        s.repo.Update(merchant)
    }

    newTotal := merchant.TotalVolumeThisMonthUSD.Add(amountUSD)
    if newTotal.GreaterThan(merchant.MonthlyLimitUSD) {
        return ErrMonthlyLimitExceeded
    }

    return nil
}
```

---

**Story 1.2.2**: KYC Document Upload & Verification Workflow
```
As a: Merchant
I want to: Upload KYC documents để nâng cấp tier
So that: Tăng transaction limit
```

**Acceptance Criteria**:
- [ ] API endpoint: `POST /api/v1/merchants/kyc/upload`
- [ ] File storage: S3/MinIO (encrypted at rest)
- [ ] Supported formats: PDF, JPG, PNG (max 10MB)
- [ ] Document types:
  - Business Registration Certificate (`business_registration`)
  - Tax ID Certificate (`tax_certificate`)
  - Bank Statement (`bank_statement`)
  - Director ID Card (`director_id`)
- [ ] Bảng `kyc_documents`:
  - `merchant_id`, `document_type`, `file_url`, `status` (pending/approved/rejected), `uploaded_at`, `reviewed_at`, `reviewer_notes`
- [ ] Admin panel: Review queue
  - Approve → auto upgrade merchant tier
  - Reject → send email with reason

---

### Feature 1.3: AML Screening Integration (Chainalysis)

**Priority**: 🟡 P1 (High, but can be mocked initially)

#### User Stories

**Story 1.3.1**: Screen Wallet Addresses against Sanctions Lists
```
As a: System
I want to: Kiểm tra wallet address của Payer với Chainalysis
So that: Từ chối giao dịch từ địa chỉ trong blacklist (OFAC, UN sanctions)
```

**Acceptance Criteria**:
- [ ] Integration với Chainalysis API hoặc alternative (TRM Labs, Elliptic)
- [ ] Khi nhận transaction on-chain:
  - Extract `from_address` (originating wallet)
  - Call screening API
  - If risk score > threshold hoặc flagged:
    - Mark payment as `status = 'flagged_aml'`
    - Send alert to compliance officer
    - DO NOT complete payment automatically
- [ ] Bảng `aml_screening_results`:
  - `payment_id`, `wallet_address`, `risk_score`, `flags`, `screened_at`

**Technical Notes**:
- **MVP**: Có thể mock với whitelist/blacklist đơn giản
- **Production**: Tích hợp Chainalysis API (paid service ~$500-1000/month)

```go
// internal/service/compliance_service.go
type AMLScreeningResult struct {
    WalletAddress string
    RiskScore     int // 0-100
    Flags         []string // ["sanctions", "mixer", "darknet"]
    IsSafe        bool
}

func (s *ComplianceService) ScreenWalletAddress(address string, chain string) (*AMLScreeningResult, error) {
    // Call Chainalysis API
    // Or use mock implementation for MVP
}
```

---

### Feature 1.4: 5-Year Transaction Record Storage

**Priority**: 🔴 P0 (Blocker)

#### User Stories

**Story 1.4.1**: Implement Immutable Audit Log
```
As a: System
I want to: Lưu trữ tất cả transaction records trong 5 năm
So that: Tuân thủ yêu cầu lưu trữ dữ liệu của cơ quan quản lý
```

**Acceptance Criteria**:
- [ ] Bảng `audit_logs` phải:
  - Immutable (không có UPDATE, chỉ INSERT)
  - Partition by year (để dễ archive)
  - Lưu JSON đầy đủ của payment, merchant, travel rule data
- [ ] Retention policy:
  - Active database: 2 years
  - Cold storage (S3/Glacier): 3-5 years
  - Auto-archive job chạy monthly
- [ ] Query API cho regulator:
  - `GET /admin/v1/compliance/audit-logs`
  - Filter: date range, merchant_id, payment_id, event_type

**Database Schema**:
```sql
-- migrations/XXX_enhance_audit_logs.up.sql
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    event_time TIMESTAMP NOT NULL DEFAULT NOW(),
    event_type VARCHAR(50) NOT NULL, -- 'payment_created', 'payment_confirmed', 'kyc_approved', etc
    actor_type VARCHAR(50), -- 'system', 'merchant', 'admin', 'payer'
    actor_id UUID,
    resource_type VARCHAR(50) NOT NULL, -- 'payment', 'merchant', 'payout'
    resource_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'create', 'update', 'approve', 'reject'
    metadata JSONB NOT NULL, -- Full snapshot of resource
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create partitions for each year
CREATE TABLE audit_logs_2025 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

CREATE TABLE audit_logs_2026 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');

CREATE INDEX idx_audit_logs_event_time ON audit_logs(event_time);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
```

---

## 📦 Epic 2: Payer Experience Layer (TDD 5.1)

### 🎯 Business Context
- **Vấn đề**: TDD v1.0 loại bỏ Payer Layer khỏi MVP, xem nó là v2.0
- **Hậu quả**: UX tệ, rủi ro mất tiền cho Payer, thua kém đối thủ, **không thể làm Escrow**
- **Giải pháp**: Đưa Payer Layer vào MVP v1.1 (bắt buộc)

---

### Feature 2.1: Payment Status Page

**Priority**: 🔴 P0 (Blocker)

#### User Stories

**Story 2.1.1**: Public Payment Status Page
```
As a: Payer (End User)
I want to: Truy cập URL thanh toán để xem trạng thái giao dịch
So that: Biết payment của mình đã được xác nhận chưa
```

**Acceptance Criteria**:
- [ ] URL format: `https://pay.gateway.com/order/{payment_id}`
- [ ] Public access (không cần login)
- [ ] Hiển thị:
  - Payment status (created/pending/confirming/completed/expired/failed)
  - Amount to pay (crypto + VND equivalent)
  - QR code (wallet address + amount + memo)
  - Countdown timer (30 minutes before expiry)
  - Transaction hash (khi đã detect on-chain)
  - Confirmations count (Solana: finalized, BSC: 12/12 blocks)
- [ ] Real-time updates qua WebSocket hoặc Server-Sent Events
- [ ] Mobile responsive

**Tech Stack**:
- Frontend: Next.js 14 App Router
- Styling: TailwindCSS + shadcn/ui
- Real-time: WebSocket hoặc SSE
- QR Code: `qrcode.react` library

**API Endpoint**:
```go
// GET /api/v1/payments/{payment_id}/status
type PaymentStatusResponse struct {
    ID              uuid.UUID       `json:"id"`
    Status          string          `json:"status"`
    AmountCrypto    decimal.Decimal `json:"amount_crypto"`
    AmountVND       decimal.Decimal `json:"amount_vnd"`
    Currency        string          `json:"currency"` // "USDT", "USDC"
    Chain           string          `json:"chain"`
    WalletAddress   string          `json:"wallet_address"`
    PaymentMemo     string          `json:"payment_memo"` // For tx memo/reference
    QRCodeData      string          `json:"qr_code_data"` // Formatted string for QR
    TxHash          string          `json:"tx_hash,omitempty"`
    Confirmations   int             `json:"confirmations"`
    ExpiresAt       time.Time       `json:"expires_at"`
    CreatedAt       time.Time       `json:"created_at"`
}
```

---

**Story 2.1.2**: Real-Time Status Updates (WebSocket)
```
As a: Payer
I want to: Nhận thông báo real-time khi payment được xác nhận
So that: Không phải refresh trang liên tục
```

**Acceptance Criteria**:
- [ ] WebSocket endpoint: `wss://pay.gateway.com/ws/payment/{payment_id}`
- [ ] Events:
  - `payment.pending` (transaction detected on-chain)
  - `payment.confirming` (waiting for finality)
  - `payment.completed` (finalized)
  - `payment.expired` (30 min timeout)
  - `payment.failed` (amount mismatch, etc)
- [ ] Auto-reconnect on disconnect
- [ ] Heartbeat/ping every 30s

**WebSocket Implementation**:
```go
// internal/api/websocket/payment_status.go
func (h *PaymentStatusHandler) HandleWebSocket(c *gin.Context) {
    paymentID := c.Param("payment_id")

    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    // Subscribe to Redis pub/sub channel: payment_events:{payment_id}
    pubsub := h.redis.Subscribe(fmt.Sprintf("payment_events:%s", paymentID))
    defer pubsub.Close()

    for {
        msg := <-pubsub.Channel()
        conn.WriteJSON(msg.Payload)
    }
}
```

---

### Feature 2.2: QR Code Generation & Payment Instructions

**Priority**: 🔴 P0 (Blocker)

#### User Stories

**Story 2.2.1**: Generate Chain-Specific QR Codes
```
As a: Payer
I want to: Scan QR code bằng ví crypto của mình
So that: Tự động điền wallet address, amount, và memo
```

**Acceptance Criteria**:
- [ ] QR code format:
  - **Solana**: `solana:{wallet_address}?amount={amount}&label={merchant_name}&message={payment_id}`
  - **BSC**: `ethereum:{wallet_address}?value={amount_wei}&data={payment_id_encoded}`
- [ ] Fallback: Copy button cho từng field (address, amount, memo)
- [ ] Visual: Hiển thị logo chain (Solana/BSC)
- [ ] Error handling: Invalid payment_id → 404 page

---

### Feature 2.3: Payment Confirmation Page

**Priority**: 🟡 P1 (High)

#### User Stories

**Story 2.3.1**: Success Page with Receipt
```
As a: Payer
I want to: Thấy trang xác nhận khi payment hoàn tất
So that: Có bằng chứng đã thanh toán
```

**Acceptance Criteria**:
- [ ] Redirect to: `/order/{payment_id}/success`
- [ ] Hiển thị:
  - ✅ "Payment Completed" message
  - Transaction hash (link to block explorer)
  - Amount paid (crypto + VND)
  - Merchant name
  - Payment ID (reference)
  - Download receipt button (PDF)
- [ ] Send email receipt to Payer (if email provided)

---

## 🧪 Testing Requirements

### Unit Tests
- [ ] Compliance Service: Travel Rule validation logic
- [ ] Merchant Service: Monthly limit calculation + tier upgrade
- [ ] AML Service: Wallet screening (with mocked API)

### Integration Tests
- [ ] API: Create payment với Travel Rule data → success
- [ ] API: Create payment > $1000 without Travel Rule → error
- [ ] API: Create payment vượt monthly limit → error
- [ ] WebSocket: Subscribe payment status → nhận events

### End-to-End Tests
- [ ] Full payment flow với Payer Layer:
  1. Merchant creates payment
  2. Payer opens payment URL
  3. Payer scans QR code
  4. System detects on-chain transaction
  5. WebSocket updates status → completed
  6. Success page displayed

---

## 📊 Success Metrics

- [ ] **Compliance Coverage**: 100% giao dịch > $1000 có Travel Rule data
- [ ] **KYC Conversion**: 80% merchants upgrade to Tier 2+ (để đạt volume)
- [ ] **AML False Positive Rate**: < 5% (không block quá nhiều giao dịch hợp lệ)
- [ ] **Payer UX**: Payment status page load time < 2s
- [ ] **WebSocket Reliability**: 99%+ uptime, < 5s delay cho status updates

---

## 🚀 Deployment Checklist

### Database
- [ ] Run migrations: `travel_rule_data`, `kyc_tier`, `audit_logs` partitions
- [ ] Setup read-replica cho audit log queries (không ảnh hưởng OLTP)

### Infrastructure
- [ ] S3/MinIO bucket cho KYC documents (encryption at rest enabled)
- [ ] Redis pub/sub cho WebSocket events
- [ ] SSL certificate cho `pay.gateway.com` subdomain

### Monitoring
- [ ] Alert: Monthly limit exceeded > 10 times/day → investigate merchant
- [ ] Alert: AML screening API down → switch to fallback/manual review
- [ ] Dashboard: Compliance metrics (Travel Rule coverage, KYC tier distribution)

---

## 📚 Documentation

- [ ] API docs: Swagger/OpenAPI spec cho Travel Rule endpoints
- [ ] Merchant guide: How to upgrade KYC tier
- [ ] Compliance manual: How to generate regulatory reports
- [ ] Runbook: Incident response nếu Chainalysis API down

---

## ⚠️ Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Chainalysis API đắt (~$1000/month) | 💰 High cost | MVP: Sử dụng mock hoặc cheaper alternative (TRM Labs) |
| KYC review queue nghẽn (manual) | 🐢 Slow merchant onboarding | Implement auto-approval cho Tier 1, semi-auto cho Tier 2 |
| Audit log table quá lớn | 💾 Storage cost | Partition + auto-archive sang S3 Glacier sau 2 năm |
| WebSocket scaling issues | ⚡ Performance | Use Redis pub/sub + multiple WS servers behind load balancer |

---

**Next Steps**: Khi MVP v1.1 hoàn thành → Ready to apply for Sandbox Đà Nẵng → Start building v2.0 Pillar 1 (SDKs)
