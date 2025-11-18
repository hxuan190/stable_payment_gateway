# MVP v1.1 Task Breakdown - Implementation Plan

**Created**: 2025-11-18
**Status**: Ready for Implementation
**Baseline**: TDD v1.0 codebase analysis completed

---

## 📊 Codebase Analysis Summary

### ✅ What EXISTS (v1.0 Baseline - Ready to Use)

**Backend Core:**
- ✅ **Models**: `Payment`, `Merchant`, `Ledger`, `Audit`, `Payout`, `Balance`, `BlockchainTransaction`
- ✅ **Services**: `PaymentService`, `MerchantService`, `LedgerService`, `PayoutService`, `NotificationService`, `ExchangeRateService`
- ✅ **Repositories**: All repositories implemented with interfaces
- ✅ **API Handlers**: `PaymentHandler`, `MerchantHandler`, `PayoutHandler`, `AdminHandler`, `HealthHandler`
- ✅ **Middleware**: Auth (API key), AdminAuth (JWT), RateLimit, Logging, Validator
- ✅ **Blockchain**: Solana listener (full implementation), Solana client, wallet management
- ✅ **QR Code**: Generator service (internal/pkg/qrcode)
- ✅ **Database**: All migrations (001-009) for core tables

**Database Tables:**
- ✅ `merchants` - KYC status, API keys, webhooks
- ✅ `payments` - Full payment lifecycle
- ✅ `payouts` - Withdrawal requests
- ✅ `ledger_entries` - Double-entry accounting
- ✅ `merchant_balances` - Computed balances
- ✅ `audit_logs` - Audit trail (⚠️ **NOT partitioned** - needs refactoring)
- ✅ `blockchain_transactions` - On-chain tx tracking
- ✅ `wallet_balance_snapshots` - Hot wallet monitoring

**Infrastructure:**
- ✅ `go.mod` with all core dependencies
- ✅ Docker Compose configuration
- ✅ Makefile for common tasks
- ✅ Four main services: `cmd/api`, `cmd/listener`, `cmd/worker`, `cmd/admin`

---

### ❌ What's MISSING (MVP v1.1 Requirements)

**Epic 1: Compliance Engine**
1. ❌ Travel Rule data model + table + repository
2. ❌ KYC Tier system (migration to add columns to `merchants` table)
3. ❌ KYC documents table + repository
4. ❌ AML screening service (TRM Labs integration)
5. ❌ Compliance service (orchestrator)
6. ❌ S3/Glacier storage service
7. ❌ Monthly volume tracking (migration + cron job)
8. ❌ Audit logs partitioning (migration to partition existing table)
9. ❌ Compliance API endpoints (admin)

**Epic 2: Payer Experience Layer**
1. ❌ Entire frontend (Next.js project - not started)
2. ❌ Payment status public API endpoint
3. ❌ WebSocket handler + route
4. ❌ Redis Pub/Sub integration for real-time events
5. ❌ Payment status page (`/order/{id}`)
6. ❌ Success page (`/order/{id}/success`)
7. ❌ QR code display component
8. ❌ Real-time status updates (WebSocket client)

---

## 🏗️ EPIC 1: Compliance Engine

### Phase 1.1: Data Models & Database Schema

#### ✅ Task 1.1.1: Create Travel Rule Data Model
**Status**: ❌ Not Started
**Files to Create**:
```
internal/model/travel_rule.go
```

**Implementation**:
```go
package model

type TravelRuleData struct {
    ID                  string
    PaymentID           string // FK to payments
    PayerFullName       string
    PayerWalletAddress  string
    PayerCountry        string // ISO 3166-1 alpha-2
    PayerIDDocument     sql.NullString
    MerchantFullName    string
    MerchantCountry     string
    TransactionAmount   decimal.Decimal
    TransactionCurrency string
    CreatedAt           time.Time
}
```

**Definition of Done**:
- [ ] Model created with all required fields
- [ ] Validation tags added
- [ ] Helper methods implemented
- [ ] Unit tests written

---

#### ✅ Task 1.1.2: Create Travel Rule Migration
**Status**: ❌ Not Started
**Files to Create**:
```
migrations/010_create_travel_rule_data_table.up.sql
migrations/010_create_travel_rule_data_table.down.sql
```

**SQL Schema**:
```sql
CREATE TABLE travel_rule_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    payer_full_name VARCHAR(255) NOT NULL,
    payer_wallet_address VARCHAR(255) NOT NULL,
    payer_country CHAR(2) NOT NULL,
    payer_id_document VARCHAR(255),
    merchant_full_name VARCHAR(255) NOT NULL,
    merchant_country CHAR(2) NOT NULL,
    transaction_amount DECIMAL(20,8) NOT NULL,
    transaction_currency VARCHAR(10) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_travel_rule_payment_id ON travel_rule_data(payment_id);
CREATE INDEX idx_travel_rule_created_at ON travel_rule_data(created_at);
CREATE INDEX idx_travel_rule_country ON travel_rule_data(payer_country);
```

**Definition of Done**:
- [ ] Up migration created
- [ ] Down migration created
- [ ] Tested locally (up + down)
- [ ] Indexes created for common queries

---

#### ✅ Task 1.1.3: Add KYC Tier Columns to Merchants Table
**Status**: ❌ Not Started (⚠️ **REFACTOR existing table**)
**Files to Create**:
```
migrations/011_add_kyc_tier_to_merchants.up.sql
migrations/011_add_kyc_tier_to_merchants.down.sql
```

**SQL Changes**:
```sql
-- Up migration
ALTER TABLE merchants
ADD COLUMN kyc_tier VARCHAR(10) NOT NULL DEFAULT 'tier1'
    CHECK (kyc_tier IN ('tier1', 'tier2', 'tier3')),
ADD COLUMN monthly_limit_usd DECIMAL(20,2) NOT NULL DEFAULT 5000.00,
ADD COLUMN total_volume_this_month_usd DECIMAL(20,2) NOT NULL DEFAULT 0,
ADD COLUMN volume_last_reset_at TIMESTAMP NOT NULL DEFAULT NOW();

CREATE INDEX idx_merchants_kyc_tier ON merchants(kyc_tier);

-- Down migration
ALTER TABLE merchants
DROP COLUMN kyc_tier,
DROP COLUMN monthly_limit_usd,
DROP COLUMN total_volume_this_month_usd,
DROP COLUMN volume_last_reset_at;
```

**Model Updates Needed**:
- ⚠️ Update `internal/model/merchant.go` to add new fields

**Definition of Done**:
- [ ] Migration created and tested
- [ ] Model updated with new fields
- [ ] Helper methods added (`IsWithinMonthlyLimit`, `GetRemainingLimit`)
- [ ] Unit tests updated

---

#### ✅ Task 1.1.4: Create KYC Documents Table
**Status**: ❌ Not Started
**Files to Create**:
```
migrations/012_create_kyc_documents_table.up.sql
migrations/012_create_kyc_documents_table.down.sql
internal/model/kyc_document.go
```

**Schema**:
```sql
CREATE TABLE kyc_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    document_type VARCHAR(50) NOT NULL, -- business_registration, tax_certificate, etc
    file_url TEXT NOT NULL,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    reviewed_by UUID,
    reviewed_at TIMESTAMP,
    reviewer_notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kyc_documents_merchant ON kyc_documents(merchant_id);
CREATE INDEX idx_kyc_documents_status ON kyc_documents(status);
```

**Definition of Done**:
- [ ] Table created with proper constraints
- [ ] Model created
- [ ] Repository interface defined

---

#### ✅ Task 1.1.5: Partition Existing Audit Logs Table
**Status**: ❌ Not Started (⚠️ **CRITICAL REFACTOR**)
**Priority**: 🔴 P0 (Required for 5-year retention compliance)
**Files to Create**:
```
migrations/013_partition_audit_logs.up.sql
migrations/013_partition_audit_logs.down.sql
```

**Strategy**:
Since audit_logs table already exists with data, we need a **safe migration**:

```sql
-- Step 1: Rename existing table
ALTER TABLE audit_logs RENAME TO audit_logs_old;

-- Step 2: Create partitioned table
CREATE TABLE audit_logs (
    -- Same schema as before
    id UUID NOT NULL,
    actor_type VARCHAR(50) NOT NULL,
    -- ... all existing columns
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Step 3: Create partitions for each year
CREATE TABLE audit_logs_2025 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

CREATE TABLE audit_logs_2026 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');

-- ... create partitions 2027-2030

-- Step 4: Copy data from old table
INSERT INTO audit_logs SELECT * FROM audit_logs_old;

-- Step 5: Drop old table (after verification)
DROP TABLE audit_logs_old;

-- Step 6: Recreate all indexes
-- ... recreate indexes from 006_create_audit_logs_table.up.sql
```

**Definition of Done**:
- [ ] Migration written and reviewed
- [ ] Tested on staging with sample data
- [ ] Rollback plan documented
- [ ] Partitions created for 5 years (2025-2030)

---

### Phase 1.2: Services & Business Logic

#### ✅ Task 1.2.1: Create TRM Labs AML Service
**Status**: ❌ Not Started
**Files to Create**:
```
internal/service/aml.go
internal/service/aml_test.go
internal/pkg/trmlabs/client.go
internal/pkg/trmlabs/client_test.go
```

**Service Interface**:
```go
package service

type AMLService interface {
    ScreenWalletAddress(ctx context.Context, address string, chain string) (*AMLResult, error)
    RecordScreeningResult(ctx context.Context, paymentID string, result *AMLResult) error
}

type AMLResult struct {
    WalletAddress string
    Chain         string
    RiskScore     int       // 0-100
    IsSanctioned  bool
    Flags         []string  // ["mixer", "sanctions", etc]
    ScreenedAt    time.Time
}
```

**TRM Labs Client**:
```go
package trmlabs

type Client struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

func (c *Client) ScreenAddress(ctx context.Context, address string, chain string) (*ScreeningResponse, error) {
    // Call TRM Labs API
    // POST https://api.trmlabs.com/public/v1/screening/addresses
}
```

**Definition of Done**:
- [ ] TRM Labs client implemented
- [ ] AML service created
- [ ] Integration tests with TRM Labs sandbox
- [ ] Mock implementation for unit tests
- [ ] Repository for storing screening results

---

#### ✅ Task 1.2.2: Create Compliance Service (Orchestrator)
**Status**: ❌ Not Started
**Files to Create**:
```
internal/service/compliance.go
internal/service/compliance_test.go
internal/repository/travel_rule.go
internal/repository/kyc_document.go
```

**Service Interface**:
```go
type ComplianceService struct {
    amlService          AMLService
    travelRuleRepo      TravelRuleRepository
    kycDocumentRepo     KYCDocumentRepository
    merchantRepo        MerchantRepository
}

func (s *ComplianceService) ValidatePaymentCompliance(ctx context.Context, payment *model.Payment) error
func (s *ComplianceService) CheckMonthlyLimit(ctx context.Context, merchantID string, amountUSD decimal.Decimal) error
func (s *ComplianceService) StoreTravelRuleData(ctx context.Context, data *model.TravelRuleData) error
func (s *ComplianceService) UpgradeKYCTier(ctx context.Context, merchantID string, tier string) error
```

**Definition of Done**:
- [ ] Compliance service implemented
- [ ] Travel rule repository created
- [ ] KYC document repository created
- [ ] Unit tests with mocked dependencies
- [ ] Integration tests

---

#### ✅ Task 1.2.3: Create S3 Storage Service
**Status**: ❌ Not Started
**Files to Create**:
```
internal/pkg/storage/s3.go
internal/pkg/storage/s3_test.go
internal/pkg/storage/interface.go
```

**Interface**:
```go
type StorageService interface {
    UploadKYCDocument(ctx context.Context, merchantID string, docType string, file io.Reader) (string, error)
    DownloadKYCDocument(ctx context.Context, fileURL string) (io.ReadCloser, error)
    ArchiveAuditLogs(ctx context.Context, year int, data []byte) error
    DeleteFile(ctx context.Context, fileURL string) error
}
```

**S3 Implementation**:
```go
type S3Storage struct {
    client     *s3.Client
    bucket     string
    region     string
    encryption string // "AES256" or "aws:kms"
}

// Lifecycle policy for Glacier transition
// - KYC docs: Standard (90 days) → Glacier
// - Audit archives: Direct to Glacier
```

**Definition of Done**:
- [ ] S3 client configured
- [ ] Upload/download methods implemented
- [ ] Encryption at rest enabled
- [ ] Lifecycle policies documented
- [ ] Integration tests with MinIO (local S3)

---

### Phase 1.3: API Endpoints & Handlers

#### ✅ Task 1.3.1: Update Payment API to Support Travel Rule
**Status**: ❌ Not Started (⚠️ **REFACTOR existing handler**)
**Files to Modify**:
```
internal/api/handler/payment.go  (REFACTOR)
internal/api/dto/payment.go      (ADD fields)
```

**Changes to DTO**:
```go
// Add to CreatePaymentRequest
type CreatePaymentRequest struct {
    // ... existing fields
    TravelRule  *TravelRuleRequest `json:"travel_rule,omitempty"`
}

type TravelRuleRequest struct {
    PayerFullName      string `json:"payer_full_name" validate:"required"`
    PayerWalletAddress string `json:"payer_wallet_address" validate:"required"`
    PayerCountry       string `json:"payer_country" validate:"required,iso3166"`
    PayerIDDocument    string `json:"payer_id_document,omitempty"`
}
```

**Handler Changes**:
```go
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
    // ... existing code

    // NEW: Check if Travel Rule required (amount > $1000)
    amountUSD := convertVNDToUSD(req.AmountVND, exchangeRate)
    if amountUSD.GreaterThan(decimal.NewFromInt(1000)) {
        if req.TravelRule == nil {
            c.JSON(400, dto.ErrorResponse("TRAVEL_RULE_REQUIRED", "Travel Rule data required for transactions > $1000"))
            return
        }
    }

    // NEW: Store Travel Rule data
    if req.TravelRule != nil {
        err := h.complianceService.StoreTravelRuleData(ctx, travelRuleData)
        // handle error
    }
}
```

**Definition of Done**:
- [ ] DTO updated
- [ ] Handler refactored
- [ ] Validation logic added
- [ ] Unit tests updated
- [ ] API documentation updated

---

#### ✅ Task 1.3.2: Create KYC Document Upload API
**Status**: ❌ Not Started
**Files to Create**:
```
internal/api/handler/kyc.go
internal/api/handler/kyc_test.go
internal/api/dto/kyc.go
```

**Endpoints**:
```go
// POST /api/v1/merchants/kyc/documents
func (h *KYCHandler) UploadDocument(c *gin.Context)

// GET /api/v1/merchants/kyc/documents
func (h *KYCHandler) ListDocuments(c *gin.Context)

// DELETE /api/v1/merchants/kyc/documents/:id
func (h *KYCHandler) DeleteDocument(c *gin.Context)
```

**Implementation**:
```go
func (h *KYCHandler) UploadDocument(c *gin.Context) {
    file, _ := c.FormFile("file")
    docType := c.PostForm("document_type")

    // Validate file size (max 10MB)
    // Validate file type (PDF, JPG, PNG)

    // Upload to S3
    fileURL, err := h.storageService.UploadKYCDocument(ctx, merchantID, docType, file)

    // Save to database
    doc := &model.KYCDocument{
        MerchantID: merchantID,
        DocumentType: docType,
        FileURL: fileURL,
        Status: "pending",
    }
    err = h.kycDocumentRepo.Create(doc)
}
```

**Definition of Done**:
- [ ] Handler created
- [ ] File upload validation (size, type)
- [ ] S3 integration working
- [ ] Database record created
- [ ] Unit tests + integration tests

---

#### ✅ Task 1.3.3: Create Admin Compliance Endpoints
**Status**: ❌ Not Started
**Files to Modify**:
```
internal/api/handler/admin.go (ADD methods)
internal/api/dto/admin.go     (ADD DTOs)
```

**New Endpoints**:
```go
// GET /admin/v1/compliance/travel-rule
func (h *AdminHandler) GetTravelRuleData(c *gin.Context)

// GET /admin/v1/compliance/kyc-documents/pending
func (h *AdminHandler) GetPendingKYCDocuments(c *gin.Context)

// POST /admin/v1/compliance/kyc-documents/:id/approve
func (h *AdminHandler) ApproveKYCDocument(c *gin.Context)

// POST /admin/v1/compliance/kyc-documents/:id/reject
func (h *AdminHandler) RejectKYCDocument(c *gin.Context)

// POST /admin/v1/compliance/merchants/:id/upgrade-tier
func (h *AdminHandler) UpgradeMerchantTier(c *gin.Context)
```

**Definition of Done**:
- [ ] All endpoints implemented
- [ ] Admin authorization enforced
- [ ] Audit logging added
- [ ] Unit tests written

---

## 🏗️ EPIC 2: Payer Experience Layer

### Phase 2.1: Backend - Real-Time API & WebSocket

#### ✅ Task 2.1.1: Create Public Payment Status API
**Status**: ❌ Not Started
**Files to Modify/Create**:
```
internal/api/handler/payment.go (ADD method)
internal/api/dto/payment.go     (ADD DTO)
internal/api/server.go          (ADD public route)
```

**New Endpoint**:
```go
// GET /api/v1/public/payments/:id/status
// No authentication required (public endpoint)
func (h *PaymentHandler) GetPublicPaymentStatus(c *gin.Context) {
    paymentID := c.Param("id")

    payment, err := h.paymentService.GetPaymentStatus(ctx, paymentID)
    if err != nil {
        c.JSON(404, dto.ErrorResponse("NOT_FOUND", "Payment not found"))
        return
    }

    // Return public-safe data (no merchant details, no sensitive info)
    response := dto.PaymentStatusResponse{
        ID:              payment.ID,
        Status:          string(payment.Status),
        AmountVND:       payment.AmountVND,
        AmountCrypto:    payment.AmountCrypto,
        Currency:        payment.Currency,
        Chain:           string(payment.Chain),
        WalletAddress:   payment.DestinationWallet,
        PaymentMemo:     payment.PaymentReference,
        TxHash:          payment.GetTxHash(),
        Confirmations:   payment.TxConfirmations.Int32,
        ExpiresAt:       payment.ExpiresAt,
        CreatedAt:       payment.CreatedAt,
    }

    c.JSON(200, dto.SuccessResponse(response))
}
```

**Definition of Done**:
- [ ] Endpoint created (no auth required)
- [ ] DTO created (public-safe fields only)
- [ ] Route added to server
- [ ] Unit tests written
- [ ] Rate limiting configured

---

#### ✅ Task 2.1.2: Create WebSocket Handler for Payment Status
**Status**: ❌ Not Started
**Files to Create**:
```
internal/api/websocket/payment.go
internal/api/websocket/payment_test.go
internal/api/server.go (ADD ws route)
```

**WebSocket Implementation**:
```go
package websocket

import (
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "github.com/redis/go-redis/v9"
)

type PaymentWSHandler struct {
    redis *redis.Client
    upgrader websocket.Upgrader
}

// GET /ws/payments/:id
func (h *PaymentWSHandler) HandlePaymentStatus(c *gin.Context) {
    paymentID := c.Param("id")

    // Upgrade HTTP to WebSocket
    conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    // Subscribe to Redis pub/sub for this payment
    pubsub := h.redis.Subscribe(c, "payment_events:"+paymentID)
    defer pubsub.Close()

    // Listen for events and forward to WebSocket client
    for {
        select {
        case msg := <-pubsub.Channel():
            conn.WriteJSON(msg.Payload)
        case <-c.Done():
            return
        }
    }
}
```

**Definition of Done**:
- [ ] WebSocket handler created
- [ ] Redis Pub/Sub integrated
- [ ] Connection management (heartbeat, reconnect)
- [ ] Error handling
- [ ] Integration tests

---

#### ✅ Task 2.1.3: Integrate Redis Pub/Sub in Payment Service
**Status**: ❌ Not Started (⚠️ **REFACTOR existing service**)
**Files to Modify**:
```
internal/service/payment.go (REFACTOR ConfirmPayment method)
```

**Changes**:
```go
type PaymentService struct {
    // ... existing fields
    redis *redis.Client  // NEW
}

func (s *PaymentService) ConfirmPayment(ctx context.Context, paymentID string, txHash string) error {
    // ... existing confirmation logic

    // NEW: Publish event to Redis
    event := PaymentEvent{
        Type:      "payment.completed",
        PaymentID: paymentID,
        Status:    "completed",
        TxHash:    txHash,
        Timestamp: time.Now(),
    }

    err = s.redis.Publish(ctx, "payment_events:"+paymentID, event).Err()
    if err != nil {
        s.logger.Error("Failed to publish payment event", err)
        // Don't fail the request, just log
    }

    return nil
}
```

**Events to Publish**:
- `payment.pending` - Transaction detected on-chain
- `payment.confirming` - Waiting for finality
- `payment.completed` - Payment confirmed
- `payment.expired` - Payment expired
- `payment.failed` - Payment failed

**Definition of Done**:
- [ ] Redis client added to PaymentService
- [ ] Pub/Sub events published on status changes
- [ ] Event schema documented
- [ ] Integration tests

---

### Phase 2.2: Frontend - Next.js Payer Experience

#### ✅ Task 2.2.1: Initialize Next.js 14 Project
**Status**: ❌ Not Started
**Files/Dirs to Create**:
```
web/
├── payment-ui/
    ├── package.json
    ├── tsconfig.json
    ├── next.config.js
    ├── tailwind.config.js
    ├── src/
        ├── app/
        │   ├── layout.tsx
        │   ├── page.tsx
        │   └── order/
        │       └── [id]/
        │           ├── page.tsx
        │           └── success/
        │               └── page.tsx
        ├── components/
        │   ├── PaymentStatus.tsx
        │   ├── QRCode.tsx
        │   └── CountdownTimer.tsx
        └── lib/
            ├── api.ts
            └── websocket.ts
```

**Setup Commands**:
```bash
cd web
npx create-next-app@latest payment-ui --typescript --tailwind --app
cd payment-ui
npm install qrcode.react @tanstack/react-query
```

**Definition of Done**:
- [ ] Next.js 14 project initialized
- [ ] TypeScript configured
- [ ] TailwindCSS + shadcn/ui installed
- [ ] Project structure created
- [ ] Dev server runs successfully

---

#### ✅ Task 2.2.2: Build Payment Status Page
**Status**: ❌ Not Started
**Files to Create**:
```
web/payment-ui/src/app/order/[id]/page.tsx
web/payment-ui/src/components/PaymentStatus.tsx
web/payment-ui/src/components/QRCode.tsx
web/payment-ui/src/components/CountdownTimer.tsx
```

**Page Implementation**:
```tsx
// src/app/order/[id]/page.tsx
'use client'

import { useParams } from 'next/navigation'
import { usePaymentStatus } from '@/hooks/usePaymentStatus'
import { PaymentStatus } from '@/components/PaymentStatus'

export default function PaymentPage() {
    const { id } = useParams()
    const { payment, isLoading, error } = usePaymentStatus(id as string)

    if (isLoading) return <div>Loading...</div>
    if (error) return <div>Payment not found</div>

    return (
        <div className="container mx-auto p-4">
            <PaymentStatus payment={payment} />
        </div>
    )
}
```

**Components**:
```tsx
// PaymentStatus.tsx
- Display payment amount (VND + crypto)
- Show QR code for payment
- Real-time status updates via WebSocket
- Countdown timer (30 min expiry)
- Transaction hash + block explorer link

// QRCode.tsx
- Generate QR code for Solana payment
- Copy address/amount/memo buttons

// CountdownTimer.tsx
- 30-minute countdown
- Auto-redirect on expiry
```

**Definition of Done**:
- [ ] Page renders correctly
- [ ] QR code displays properly
- [ ] Countdown timer works
- [ ] Mobile responsive
- [ ] Error states handled

---

#### ✅ Task 2.2.3: Implement WebSocket Client
**Status**: ❌ Not Started
**Files to Create**:
```
web/payment-ui/src/hooks/usePaymentStatus.ts
web/payment-ui/src/lib/websocket.ts
```

**WebSocket Hook**:
```tsx
// usePaymentStatus.ts
import { useEffect, useState } from 'react'
import { createWebSocketConnection } from '@/lib/websocket'

export function usePaymentStatus(paymentId: string) {
    const [payment, setPayment] = useState(null)
    const [isLoading, setIsLoading] = useState(true)

    useEffect(() => {
        // 1. Fetch initial payment data (HTTP)
        fetchPaymentStatus(paymentId).then(setPayment)

        // 2. Connect to WebSocket for real-time updates
        const ws = createWebSocketConnection(paymentId)

        ws.on('payment.pending', (data) => {
            setPayment(prev => ({ ...prev, status: 'pending', ...data }))
        })

        ws.on('payment.completed', (data) => {
            setPayment(prev => ({ ...prev, status: 'completed', ...data }))
            // Redirect to success page
            router.push(`/order/${paymentId}/success`)
        })

        return () => ws.close()
    }, [paymentId])

    return { payment, isLoading }
}
```

**Definition of Done**:
- [ ] WebSocket connection established
- [ ] Real-time updates working
- [ ] Auto-reconnect on disconnect
- [ ] Events handled correctly
- [ ] Integration tested with backend

---

#### ✅ Task 2.2.4: Build Success Page
**Status**: ❌ Not Started
**Files to Create**:
```
web/payment-ui/src/app/order/[id]/success/page.tsx
web/payment-ui/src/components/SuccessReceipt.tsx
```

**Success Page**:
```tsx
export default function SuccessPage() {
    return (
        <div className="success-page">
            <CheckCircleIcon className="text-green-500" />
            <h1>Payment Completed!</h1>
            <SuccessReceipt payment={payment} />
            <Button>Download Receipt (PDF)</Button>
        </div>
    )
}
```

**Receipt Details**:
- Payment ID
- Amount paid (VND + crypto)
- Transaction hash (with block explorer link)
- Merchant name
- Timestamp

**Definition of Done**:
- [ ] Success page displays correctly
- [ ] Receipt shows all details
- [ ] PDF download works (optional for MVP)
- [ ] Mobile responsive

---

### Phase 2.3: Deployment & Integration

#### ✅ Task 2.3.1: Configure NGINX for Frontend
**Status**: ❌ Not Started
**Files to Modify**:
```
docker-compose.yml (ADD frontend service)
nginx.conf         (ADD routes for /order/*)
```

**Docker Compose**:
```yaml
services:
  frontend:
    build: ./web/payment-ui
    environment:
      - NEXT_PUBLIC_API_URL=http://api:8080
      - NEXT_PUBLIC_WS_URL=ws://api:8080
    depends_on:
      - api
```

**NGINX Config**:
```nginx
# Frontend routes
location /order/ {
    proxy_pass http://frontend:3000;
}

# API routes
location /api/ {
    proxy_pass http://api:8080;
}

# WebSocket upgrade
location /ws/ {
    proxy_pass http://api:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

**Definition of Done**:
- [ ] Frontend container builds successfully
- [ ] NGINX routes configured
- [ ] WebSocket proxying works
- [ ] HTTPS/SSL configured (Let's Encrypt)

---

## 📋 Implementation Order & Dependencies

### Week 1: Database & Compliance Backend

**Day 1-2**: Database Migrations
- ✅ Task 1.1.1: Travel Rule model
- ✅ Task 1.1.2: Travel Rule migration
- ✅ Task 1.1.3: KYC Tier migration
- ✅ Task 1.1.4: KYC documents migration
- ✅ Task 1.1.5: Partition audit logs (CRITICAL)

**Day 3-4**: Compliance Services
- ✅ Task 1.2.1: TRM Labs AML service
- ✅ Task 1.2.2: Compliance service
- ✅ Task 1.2.3: S3 storage service

**Day 5**: Compliance APIs
- ✅ Task 1.3.1: Update Payment API (Travel Rule)
- ✅ Task 1.3.2: KYC upload API
- ✅ Task 1.3.3: Admin compliance endpoints

---

### Week 2: Payer Experience Backend

**Day 1-2**: Real-Time Infrastructure
- ✅ Task 2.1.1: Public payment status API
- ✅ Task 2.1.2: WebSocket handler
- ✅ Task 2.1.3: Redis Pub/Sub integration

**Day 3**: Testing
- Integration tests for WebSocket
- End-to-end compliance flow testing

---

### Week 3: Payer Experience Frontend

**Day 1**: Setup
- ✅ Task 2.2.1: Initialize Next.js project

**Day 2-3**: Core Pages
- ✅ Task 2.2.2: Payment status page
- ✅ Task 2.2.3: WebSocket client

**Day 4**: Finish UI
- ✅ Task 2.2.4: Success page

**Day 5**: Deployment
- ✅ Task 2.3.1: NGINX + Docker integration

---

## ✅ Testing Checklist

### Unit Tests
- [ ] All new models have tests
- [ ] All new services have tests (with mocks)
- [ ] All new handlers have tests
- [ ] All new repositories have tests

### Integration Tests
- [ ] Travel Rule flow (create payment > $1000 → Travel Rule stored)
- [ ] KYC tier limits (create payment → check monthly limit)
- [ ] AML screening (wallet address → TRM Labs → flagged)
- [ ] WebSocket real-time updates
- [ ] S3 document upload/download

### End-to-End Tests
- [ ] Full payment flow with Payer Layer:
  1. Merchant creates payment
  2. Payer opens `/order/{id}`
  3. QR code displays
  4. Payer sends crypto (testnet)
  5. WebSocket updates status → completed
  6. Success page shows

---

## 🚀 Deployment Readiness

### Infrastructure
- [ ] S3 bucket created (with Glacier lifecycle)
- [ ] TRM Labs API key obtained (sandbox)
- [ ] Redis configured for Pub/Sub
- [ ] Database migrations tested on staging
- [ ] `pay.gateway.com` subdomain configured

### Monitoring
- [ ] Alerts for Travel Rule missing data
- [ ] Alerts for monthly limits exceeded
- [ ] WebSocket connection monitoring
- [ ] S3 upload failures

### Documentation
- [ ] API docs updated (Swagger/OpenAPI)
- [ ] Merchant guide: Travel Rule requirements
- [ ] Merchant guide: KYC tier upgrade
- [ ] Admin runbook: Compliance reports

---

## 📊 Metrics & Success Criteria

**Compliance Coverage**:
- [ ] 100% of payments > $1000 USD have Travel Rule data
- [ ] 0% false negatives on AML screening (no sanctioned wallets accepted)
- [ ] < 5% false positives on AML screening

**Payer UX**:
- [ ] Payment status page load time < 2s
- [ ] WebSocket latency < 500ms for status updates
- [ ] Mobile responsiveness score > 90 (Lighthouse)

**System Reliability**:
- [ ] WebSocket uptime > 99%
- [ ] S3 upload success rate > 99.9%
- [ ] Audit logs partitioning tested for 1M+ records

---

**Next Step**: Begin implementation with **Week 1, Day 1** (Database migrations)
**Priority**: Audit logs partitioning (Task 1.1.5) is CRITICAL for compliance
