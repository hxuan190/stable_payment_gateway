# Requirements: v3.0 Pillar 3 - Giải pháp Ký quỹ (Escrow Services)

**Phase**: v3.0 Quarter 3-4
**Timeline**: 12 weeks (sau khi v2.0 đã stable 6+ tháng)
**Status**: 🟡 Strategic Moat - Requires Regulatory Approval

---

## 🎯 Mục tiêu Phase

**Trụ cột 3** là chiến lược **Thống lĩnh thị trường** (Domination) bằng cách tạo **"con hào"** (moat) mà đối thủ không thể dễ dàng sao chép.

### Strategic Value Shift
- **FROM**: Bán "thanh toán" (commodity - ai cũng làm được)
- **TO**: Bán "niềm tin" (trust - asset vô giá)

### Target Market
**Freelancer & Dịch vụ số** Việt Nam nhận thanh toán từ khách hàng quốc tế:
- Graphic designers, developers, content creators
- Digital agencies, consulting firms
- Dropshipping, e-commerce sellers

### Core Problem Solved
"Làm sao tôi đảm bảo khách hàng quốc tế sẽ trả tiền $5,000 cho tôi sau khi tôi giao sản phẩm/dịch vụ?"

### Competitive Moat
Dịch vụ Ký quỹ đòi hỏi:
1. **Technical Complexity**: Ledger Service bút toán kép + State Machine processor đáng tin cậy (đã có từ TDD v1.0)
2. **Legal Complexity**: Điều hướng hành lang pháp lý chưa từng có (Sandbox Đà Nẵng)
3. **Trust**: Reputation takes years to build

→ Đối thủ như Basal Pay (focus du lịch) không thể pivot nhanh sang escrow.

---

## ⚠️ CRITICAL: Legal Prerequisite

**Điều kiện tiên quyết TUYỆT ĐỐI**:
- [ ] ✅ v2.0 (SDKs + SaaS) đã hoạt động ổn định 6+ tháng
- [ ] ✅ Đã xây dựng lòng tin với cơ quan quản lý Sandbox Đà Nẵng
- [ ] ✅ Compliance Engine có track record tốt (0 vi phạm)
- [ ] ✅ Đã gửi báo cáo định kỳ cho Sở KH&CN Đà Nẵng
- [ ] ✅ Nhận phê duyệt MỞ RỘNG Sandbox sang "Dịch vụ Ký quỹ"

**Lập luận pháp lý**:
- ❌ Không xin phép SBV (Nghị định 101): Escrow không trong danh sách dịch vụ TTTT được phép
- ✅ Định vị: "Dịch vụ công nghệ hỗ trợ tin cậy" (technology-enabled trust service) gắn liền thanh toán xuyên biên giới
- ✅ Nằm trong phạm vi Nghị quyết 222: Thử nghiệm mô hình FinTech mới tại Đà Nẵng

---

## 📦 Epic 1: Escrow Payment Flow

### Feature 1.1: Escrow Invoice Creation

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 1.1.1**: Create Escrow Invoice
```
As a: Merchant (Freelancer)
I want to: Tạo "Escrow Invoice" thay vì payment thông thường
So that: Tiền sẽ được giữ an toàn cho đến khi tôi giao hàng/dịch vụ
```

**Acceptance Criteria**:
- [ ] API endpoint mới: `POST /api/v1/escrow/invoices`
- [ ] Request body:
  ```json
  {
    "merchant_id": "uuid",
    "amount_vnd": 115000000, // ~$5,000 USD
    "currency": "USDT",
    "chain": "solana",
    "invoice_type": "escrow", // NEW
    "description": "Website design project for ABC Corp",
    "payer_email": "client@abccorp.com", // Required for escrow
    "payer_name": "John Doe", // Required for escrow
    "milestone_description": "Complete homepage + 3 landing pages", // Optional
    "expected_delivery_date": "2025-12-01"
  }
  ```
- [ ] Response:
  ```json
  {
    "id": "esc_invoice_uuid",
    "status": "created",
    "payment_url": "pay.gateway.com/escrow/esc_invoice_uuid",
    "expires_at": "2025-11-20T10:00:00Z" // 48h expiry (longer than normal)
  }
  ```

**Database Schema**:
```sql
-- Extend payments table
ALTER TABLE payments
ADD COLUMN invoice_type VARCHAR(20) DEFAULT 'normal'
    CHECK (invoice_type IN ('normal', 'escrow')),
ADD COLUMN payer_email VARCHAR(255),
ADD COLUMN payer_name VARCHAR(255),
ADD COLUMN milestone_description TEXT,
ADD COLUMN expected_delivery_date DATE;

CREATE INDEX idx_payments_invoice_type ON payments(invoice_type);
```

---

### Feature 1.2: Escrow Payment Page (Payer Layer Extension)

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 1.2.1**: Escrow Payment Page with Terms
```
As a: Payer (Client from US/EU)
I want to: Thấy rõ ràng điều khoản escrow trước khi thanh toán
So that: Hiểu tiền của tôi sẽ an toàn và tôi có quyền kiểm soát
```

**Acceptance Criteria**:
- [ ] URL: `pay.gateway.com/escrow/{invoice_id}`
- [ ] UI khác với payment page thông thường:
  - **Badge**: "🔒 Escrow Protection"
  - **Terms Section**:
    - "Your payment will be held securely in escrow"
    - "Funds will only be released to merchant after you approve"
    - "You have X days to inspect the delivery and approve/dispute"
  - **Escrow Details**:
    - Merchant name
    - Project description
    - Expected delivery date
    - Escrow fee: 2.5% (higher than normal 1% payment fee)
  - **Checkbox**: "I agree to Escrow Terms & Conditions" (required)
- [ ] After payment:
  - Status: `ESCROW_HELD` (not `completed`)
  - Message: "Payment received and held in escrow. Merchant will be notified to start work."
  - Email to Payer: "Your $5,000 USDT is now held securely. Release funds when satisfied."
  - Email to Merchant: "Client paid $5,000 into escrow. Start work. Funds will be released upon client approval."

---

### Feature 1.3: Ledger Integration (Double-Entry for Escrow)

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 1.3.1**: Escrow Accounting Flow
```
As a: Ledger Service (TDD 3.1)
I want to: Ghi nhận bút toán escrow theo chuẩn kế toán bút toán kép
So that: Đảm bảo tiền được theo dõi chính xác và có thể audit
```

**Ledger Flow**:

**Bước 1: Payer gửi tiền vào Escrow**
```
DEBIT: hot_wallet_usdt_asset (+$5,000 USDT received)
CREDIT: escrow_liability_esc_invoice_123 (+$5,000 held for this invoice)
```

**Bước 2: Payer nhấn "Release Funds"**
```
DEBIT: escrow_liability_esc_invoice_123 (-$5,000 released)
CREDIT: merchant_A_payable (+$4,875 = $5,000 - 2.5% fee)
CREDIT: system_revenue_escrow_fee (+$125 = 2.5% of $5,000)
```

**Bước 3: (Optional) Payer disputes → Refund**
```
DEBIT: escrow_liability_esc_invoice_123 (-$5,000)
CREDIT: hot_wallet_usdt_asset (-$5,000 refunded to Payer)
DEBIT: system_revenue_dispute_fee (+$50 dispute handling fee, charged to merchant)
CREDIT: merchant_A_payable (-$50)
```

**Ledger Service API Call**:
```go
// internal/service/escrow_service.go
func (s *EscrowService) ReleaseFunds(invoiceID uuid.UUID, releasedBy string) error {
    invoice := s.repo.GetInvoice(invoiceID)

    // Validate: only Payer can release
    if releasedBy != invoice.PayerEmail {
        return ErrUnauthorized
    }

    // Calculate amounts
    escrowFeePercent := decimal.NewFromFloat(0.025) // 2.5%
    escrowFee := invoice.AmountCrypto.Mul(escrowFeePercent)
    merchantReceives := invoice.AmountCrypto.Sub(escrowFee)

    // Call Ledger Service (TDD 3.1)
    err := s.ledger.RecordTransaction(ledger.Transaction{
        Entries: []ledger.Entry{
            {Account: fmt.Sprintf("escrow_liability_%s", invoiceID), Type: "DEBIT", Amount: invoice.AmountCrypto},
            {Account: fmt.Sprintf("merchant_%s_payable", invoice.MerchantID), Type: "CREDIT", Amount: merchantReceives},
            {Account: "system_revenue_escrow_fee", Type: "CREDIT", Amount: escrowFee},
        },
        ReferenceType: "escrow_release",
        ReferenceID:   invoiceID,
    })

    if err != nil {
        return err
    }

    // Update invoice status
    invoice.Status = "released"
    invoice.ReleasedAt = time.Now()
    s.repo.Update(invoice)

    // Trigger notifications
    s.notification.Send("escrow.released", invoice)

    return nil
}
```

---

### Feature 1.4: Transaction Processor Integration (State Machine)

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 1.4.1**: Extend State Machine for Escrow
```
As a: Transaction Processor (TDD 3.3)
I want to: Hỗ trợ luồng escrow với trạng thái mới
So that: Xử lý escrow payments đúng quy trình
```

**State Machine Mở rộng**:

Normal Payment States:
```
CREATED → PENDING → CONFIRMING → COMMITTED → PAYOUT_PENDING → COMPLETED
```

Escrow Payment States:
```
CREATED → PENDING → CONFIRMING → COMMITTED → ESCROW_HELD → [Wait for Payer action]
  → ESCROW_RELEASED → PAYOUT_PENDING → COMPLETED
  → ESCROW_DISPUTED → MANUAL_REVIEW → REFUNDED / RELEASED
```

**State Machine Code**:
```go
// internal/processor/state_machine.go
func (sm *StateMachine) ProcessEvent(payment *Payment, event Event) error {
    if payment.InvoiceType == "escrow" {
        return sm.processEscrowEvent(payment, event)
    }
    return sm.processNormalEvent(payment, event)
}

func (sm *StateMachine) processEscrowEvent(payment *Payment, event Event) error {
    switch payment.Status {
    case "COMMITTED":
        // Instead of PAYOUT_PENDING, go to ESCROW_HELD
        payment.Status = "ESCROW_HELD"
        payment.EscrowHeldAt = time.Now()
        payment.EscrowExpiresAt = time.Now().Add(30 * 24 * time.Hour) // 30 days to release

    case "ESCROW_HELD":
        if event.Type == "RELEASE_FUNDS" {
            payment.Status = "ESCROW_RELEASED"
            payment.ReleasedAt = time.Now()
            // Trigger ledger transaction (see Feature 1.3)
        } else if event.Type == "DISPUTE" {
            payment.Status = "ESCROW_DISPUTED"
            payment.DisputedAt = time.Now()
            // Notify admin for manual review
        }

    case "ESCROW_RELEASED":
        payment.Status = "PAYOUT_PENDING"
        // Continue normal payout flow

    default:
        return ErrInvalidStateTransition
    }

    return sm.repo.UpdatePayment(payment)
}
```

---

## 📦 Epic 2: Payer Controls (Release / Dispute)

### Feature 2.1: Release Funds UI

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 2.1.1**: Payer Release Funds Button
```
As a: Payer
I want to: Nhấn nút "Release Funds" sau khi nhận được sản phẩm/dịch vụ hài lòng
So that: Merchant nhận tiền
```

**Acceptance Criteria**:
- [ ] Payment Status Page (TDD 5.1) mở rộng cho escrow:
  - URL: `pay.gateway.com/escrow/{invoice_id}`
  - Sau khi thanh toán (status = ESCROW_HELD):
    - Message: "✅ Payment held in escrow ($5,000 USDT)"
    - Status: "Waiting for merchant to deliver"
    - Countdown: "You have 28 days left to release or dispute"
    - Merchant info: "Freelancer: John Nguyen (john@example.com)"
    - Project description
    - **Button**: "Release Funds" (primary, green)
    - **Button**: "Report Issue / Dispute" (secondary, red)
- [ ] Click "Release Funds":
  - Confirmation modal: "Are you sure? This action cannot be undone."
  - If confirmed:
    - Call API: `POST /api/v1/escrow/invoices/{id}/release`
    - Require Payer authentication:
      - Email OTP (send code to `payer_email`)
      - Or wallet signature (sign message with same wallet that paid)

**Release API**:
```go
// POST /api/v1/escrow/invoices/{id}/release
type ReleaseRequest struct {
    OTP          string `json:"otp"` // Email OTP code
    Signature    string `json:"signature,omitempty"` // Wallet signature (alternative)
}

func (h *EscrowHandler) ReleaseFunds(c *gin.Context) {
    invoiceID := c.Param("id")
    var req ReleaseRequest
    c.BindJSON(&req)

    // Verify OTP or Signature
    invoice := h.service.GetInvoice(invoiceID)
    if req.OTP != "" {
        if !h.verifyOTP(invoice.PayerEmail, req.OTP) {
            c.JSON(403, gin.H{"error": "Invalid OTP"})
            return
        }
    }

    // Release funds
    err := h.service.ReleaseFunds(invoiceID, invoice.PayerEmail)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"message": "Funds released successfully"})
}
```

---

### Feature 2.2: Dispute Mechanism

**Priority**: 🟡 P1 (High, but can be manual MVP)

#### User Stories

**Story 2.2.1**: Payer Initiates Dispute
```
As a: Payer
I want to: Report issue nếu merchant không giao hàng hoặc chất lượng kém
So that: Được hỗ trợ giải quyết tranh chấp
```

**Acceptance Criteria**:
- [ ] Click "Report Issue":
  - Form:
    - Issue type: Dropdown (Not delivered / Poor quality / Scam / Other)
    - Description: Textarea (required, min 50 chars)
    - Evidence: File upload (screenshots, emails) - optional
  - Submit → Call API: `POST /api/v1/escrow/invoices/{id}/dispute`
- [ ] Backend:
  - Update status: `ESCROW_HELD` → `ESCROW_DISPUTED`
  - Send alert to admin panel: "New dispute case: Invoice #{id}"
  - Email to Merchant: "Client has disputed the payment. Please respond within 7 days."
  - Email to Payer: "We've received your dispute. Our team will review within 48 hours."

**Manual Review Process (MVP)**:
- Admin logs in to Admin Panel
- Views dispute details (description, evidence)
- Contacts both parties via email
- Makes decision:
  - **Full refund to Payer**: `POST /admin/v1/escrow/{id}/refund`
  - **Partial refund**: `POST /admin/v1/escrow/{id}/partial-refund` (e.g., 50% to each)
  - **Release to Merchant**: `POST /admin/v1/escrow/{id}/admin-release`

**Future (v3.1)**: Automated dispute resolution based on evidence + ML scoring.

---

## 📦 Epic 3: Merchant Experience (Escrow Dashboard)

### Feature 3.1: Escrow Invoices Management

**Priority**: 🔴 P0 (Core)

#### User Stories

**Story 3.1.1**: View Escrow Invoices in Dashboard
```
As a: Merchant
I want to: Xem danh sách escrow invoices và trạng thái
So that: Theo dõi các khoản thanh toán đang giữ
```

**Acceptance Criteria**:
- [ ] Merchant Dashboard (TDD 4.3) thêm tab "Escrow"
- [ ] Table columns:
  - Invoice ID
  - Payer name / email
  - Amount (crypto + VND)
  - Status (Created / Held / Released / Disputed / Refunded)
  - Created date
  - Expected delivery date
  - Days remaining (for release)
  - Actions (View details)
- [ ] Filter by status
- [ ] Metrics cards:
  - Total held in escrow: $25,000
  - Pending release: $15,000
  - Released this month: $80,000
  - Dispute rate: 2.5%

---

### Feature 3.2: Escrow Notifications

**Priority**: 🟡 P1 (High)

#### User Stories

**Story 3.2.1**: Notify Merchant of Escrow Events
```
As a: Merchant
I want to: Nhận email notification khi escrow có sự kiện
So that: Không bỏ lỡ việc khách hàng đã release funds
```

**Email Templates**:
1. **Escrow Payment Received**: "Client paid $5,000 into escrow for Project ABC. Start work now!"
2. **Funds Released**: "🎉 Client released $4,875 to your account (after 2.5% escrow fee). Payout available."
3. **Escrow Expiring Soon**: "Reminder: Escrow for Invoice #123 expires in 3 days. Contact client to release funds."
4. **Dispute Filed**: "⚠️ Client disputed Invoice #123. Please respond to our team within 7 days."

---

## 📦 Epic 4: Pricing & Revenue Model

### Feature 4.1: Escrow Fee Structure

**Priority**: 🔴 P0 (Core)

#### Pricing Strategy

**Escrow Fees**:
- **Normal Payment**: 1% transaction fee (baseline)
- **Escrow Payment**: 2.5% escrow fee (higher because of added service + liability)

**Justification**:
- Value provided: Insurance against non-payment (worth much more than 1.5% extra)
- Comparable services:
  - Upwork: 5-20% fee (plus payment processing)
  - Fiverr: 5.5% buyer fee + 20% seller fee
  - Escrow.com: 3.25% (minimum $25)

**Our positioning**: "Lower than Upwork, higher protection than direct payment"

**Revenue Projection**:
- If 30% of payments use escrow
- Average escrow amount: $3,000
- 100 transactions/month
- Revenue: 100 * $3,000 * 2.5% = $7,500/month (escrow fees alone)

---

### Feature 4.2: Escrow Fee Configuration

**Priority**: 🟡 P1 (High)

#### User Stories

**Story 4.2.1**: Admin Can Configure Escrow Fee
```
As a: Admin
I want to: Điều chỉnh escrow fee percentage
So that: Test different pricing strategies
```

**Database Schema**:
```sql
CREATE TABLE system_config (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    updated_by UUID -- admin user_id
);

INSERT INTO system_config (key, value) VALUES
('escrow_fee_percent', '{"value": 0.025, "currency": "all"}'),
('escrow_min_fee_usd', '{"value": 10}');
```

**API**:
```
GET  /admin/v1/config/escrow-fees
PUT  /admin/v1/config/escrow-fees
```

---

## 📦 Epic 5: Risk Management & Compliance

### Feature 5.1: Escrow Fund Reconciliation

**Priority**: 🔴 P0 (Critical for audit)

#### User Stories

**Story 5.1.1**: Daily Escrow Balance Reconciliation
```
As a: Finance Team
I want to: Reconcile escrow liabilities với hot wallet holdings
So that: Đảm bảo không thiếu hụt tiền và audit trail clean
```

**Acceptance Criteria**:
- [ ] Daily cron job:
  - Query Ledger: Sum of all `escrow_liability_*` accounts
  - Query Blockchain: Actual hot wallet balance
  - Compare:
    - `Total Escrow Liabilities + Other Liabilities + System Revenue <= Hot Wallet Balance`
  - If mismatch → alert admin
- [ ] Dashboard widget: "Escrow Liabilities vs Wallet Holdings"

**Query**:
```sql
-- Total escrow liabilities (should match blockchain holdings)
SELECT SUM(balance)
FROM ledger_accounts
WHERE account_name LIKE 'escrow_liability_%';
```

---

### Feature 5.2: Escrow Expiration Policy

**Priority**: 🟡 P1 (High)

#### User Stories

**Story 5.2.1**: Auto-Release After Expiration
```
As a: System
I want to: Tự động release funds nếu Payer không hành động sau 30 ngày
So that: Merchant không bị giữ tiền vô thời hạn
```

**Policy**:
- Default escrow period: 30 days
- 7 days before expiry: Email reminder to Payer ("Release funds or dispute within 7 days")
- After 30 days: Auto-release funds to Merchant (with notification to both parties)

**Cron Job**:
```go
// internal/worker/escrow_expiration_job.go
func (j *EscrowExpirationJob) Run() {
    expiredInvoices := j.repo.FindEscrowsExpiringSoon(time.Now())

    for _, invoice := range expiredInvoices {
        if invoice.Status == "ESCROW_HELD" {
            // Auto-release
            j.escrowService.ReleaseFunds(invoice.ID, "system_auto_release")
            j.notification.Send("escrow.auto_released", invoice)
        }
    }
}
```

---

## 📦 Epic 6: Plugin Integration (Shopify Escrow)

### Feature 6.1: Escrow Option in Shopify Plugin

**Priority**: 🟡 P1 (High - unlock full value proposition)

#### User Stories

**Story 6.1.1**: Merchant Enables Escrow for Products
```
As a: Shopify Merchant (Freelancer selling services)
I want to: Enable "Escrow Protection" cho certain products
So that: Buyers cảm thấy an toàn khi mua dịch vụ high-value
```

**Acceptance Criteria**:
- [ ] Shopify plugin settings:
  - Checkbox: "Enable Escrow for this product" (product-level setting)
  - Or: "Enable Escrow for orders > $X" (global setting)
- [ ] Khi customer checkout:
  - If product has escrow enabled → create `invoice_type: escrow` payment
  - Badge on checkout page: "🔒 Protected by Escrow"
- [ ] After payment:
  - Shopify order status: `pending-escrow` (custom status)
  - Merchant fulfills order
  - Customer receives product
  - Customer clicks "Release Escrow" link (sent via email)
  - Order status → `completed`

**Marketing Angle**: "Shopify + Escrow = Ultimate protection for digital services & high-value products"

---

## 🧪 Testing Requirements

### Security Tests
- [ ] Test: Only Payer (verified by OTP/signature) can release funds
- [ ] Test: Cannot release funds twice (idempotency)
- [ ] Test: Ledger balance reconciliation after 100 escrow transactions

### Edge Cases
- [ ] Payer pays wrong amount → escrow should reject
- [ ] Payer disputes after 30 days → should fail (expired)
- [ ] Merchant tries to release own escrow → should fail (unauthorized)

### E2E Tests
- [ ] Full escrow flow:
  1. Merchant creates escrow invoice
  2. Payer pays crypto
  3. Funds held (ESCROW_HELD)
  4. Merchant delivers service
  5. Payer releases funds
  6. Merchant receives payout

---

## 📊 Success Metrics

- [ ] **Escrow Adoption**: 30% of payments use escrow (within 6 months of launch)
- [ ] **Escrow Volume**: $500K+ held in escrow monthly
- [ ] **Dispute Rate**: < 5% (well-managed escrow → low disputes)
- [ ] **Resolution Time**: 90% disputes resolved within 72 hours
- [ ] **NPS**: Freelancers rate escrow feature 9+/10

---

## 📚 Regulatory Preparation

### Legal Documentation Required
- [ ] Escrow Terms & Conditions (Vietnamese + English)
- [ ] Dispute Resolution Policy
- [ ] Refund Policy
- [ ] Privacy Policy update (escrow-specific data handling)
- [ ] Legal opinion letter: "Escrow service as technology-enabled trust service, not TTTT"

### Sandbox Expansion Proposal
- [ ] Document: "Đề xuất Mở rộng Phạm vi Sandbox: Dịch vụ Ký quỹ"
  - Background: v2.0 track record (6 months, 100+ merchants, 0 violations)
  - Market need: Freelancers need escrow protection
  - Alignment with Đà Nẵng goals: Attract global freelance economy
  - Risk mitigation: Compliance engine, audit logs, 5-year record storage
  - Request: 12-month trial for escrow services

---

## ⚠️ Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Sandbox expansion NOT approved | 🚫 Cannot launch escrow | Plan B: Launch escrow ONLY for transactions already approved (cross-border freelance) |
| High dispute rate (>10%) | 💰 Operational cost + reputation damage | Strict onboarding: Require Tier 2+ KYC for escrow merchants, educate users |
| Escrow fund theft/hack | 💀 Catastrophic | Cold wallet for >80% escrow funds, multi-sig, insurance policy |
| Payer never releases funds (griefing) | 😡 Merchant frustration | 30-day auto-release policy, clear communication upfront |

---

## 🚀 Launch Plan

### Phase 3.1: Private Beta (Month 1-2)
- Invite 10 trusted freelancer merchants
- Manual onboarding + education
- Collect feedback

### Phase 3.2: Public Launch (Month 3)
- Announce escrow feature on website, blog, social media
- Case studies: "How freelancer X got paid $10K safely with escrow"
- PR: "First crypto escrow service in Vietnam"

### Phase 3.3: Plugin Integration (Month 4-6)
- Add escrow option to Shopify plugin
- Marketing: "Shopify + Escrow = Trust for digital services"

---

**Final Note**: Trụ cột 3 (Escrow) is the **crown jewel** of the strategy. It transforms the platform from a commodity payment gateway into a **trusted partner** for Vietnam's digital economy. This is the "con hào" that will dominate the freelancer market for years to come.

**Prerequisites Checklist** (Trước khi bắt đầu v3.0):
- [ ] ✅ MVP v1.1 launched & stable
- [ ] ✅ v2.0 Pillar 1 (SDKs): 100+ installs
- [ ] ✅ v2.0 Pillar 2 (SaaS): 20% upgrade rate
- [ ] ✅ Compliance track record: 6+ months, 0 violations
- [ ] ✅ Sandbox expansion approved by UBND Đà Nẵng
- [ ] ✅ Legal docs prepared
- [ ] ✅ Security audit passed (especially hot wallet + ledger)

→ Only then: GO for Escrow! 🚀
