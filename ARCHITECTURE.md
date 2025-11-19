# Technical Architecture - Stablecoin Payment Gateway MVP

## System Overview

```
┌───────────────────────────────────────────────────────────────────────┐
│                          EXTERNAL ACTORS                              │
├────────────────┬──────────────────┬──────────────────┬───────────────┤
│   End Users    │    Merchants     │   Blockchain     │  OTC Partner  │
│   (Payers)     │    (Business)    │   (Solana)       │  (Liquidity)  │
└────────┬───────┴────────┬─────────┴─────────┬────────┴──────┬────────┘
         │                │                   │               │
         │                │                   │               │
┌────────▼────────────────▼───────────────────▼───────────────▼────────┐
│                         API GATEWAY LAYER                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │   Public     │  │   Merchant   │  │   Internal   │               │
│  │   Payment    │  │     API      │  │   Admin API  │               │
│  │     API      │  │              │  │              │               │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘               │
│         │                 │                  │                        │
│         │                 │                  │                        │
│  ┌──────┴─────────────────┴──────────────────┴───────┐               │
│  │         Authentication & Authorization            │               │
│  │         - API Keys (Merchants)                    │               │
│  │         - JWT (Admin)                              │               │
│  │         - Rate Limiting                            │               │
│  └────────────────────────┬───────────────────────────┘               │
└────────────────────────────┼──────────────────────────────────────────┘
                             │
┌────────────────────────────▼──────────────────────────────────────────┐
│                      APPLICATION LAYER                                │
│                                                                        │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │
│  │   Payment       │  │   Merchant      │  │   Payout        │      │
│  │   Service       │  │   Service       │  │   Service       │      │
│  │                 │  │                 │  │                 │      │
│  │ - Create        │  │ - Register      │  │ - Request       │      │
│  │ - Validate      │  │ - KYC           │  │ - Approve       │      │
│  │ - Status        │  │ - Balance       │  │ - Execute       │      │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘      │
│           │                    │                     │               │
│  ┌────────┴────────────────────┴─────────────────────┴────────┐     │
│  │                  Ledger Service                             │     │
│  │  - Double-entry accounting                                  │     │
│  │  - Balance management                                       │     │
│  │  - Transaction log                                          │     │
│  └──────────────────────────────────────────────────────────────┘     │
│                                                                       │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │               AML Engine (Anti-Money Laundering)              │   │
│  │  - Risk scoring        - Sanctions screening                 │   │
│  │  - Txn monitoring      - Wallet screening                    │   │
│  │  - Alert management    - Regulatory reporting                │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │               Notification Service                            │   │
│  │  - Webhook dispatcher                                         │   │
│  │  - Email notifications                                        │   │
│  │  - Retry logic                                                │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
                             │
┌────────────────────────────▼──────────────────────────────────────────┐
│                    BLOCKCHAIN LAYER                                   │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────┐     │
│  │              Blockchain Listener Service                     │     │
│  │                                                              │     │
│  │  ┌───────────────┐     ┌──────────────┐                    │     │
│  │  │ Solana        │     │ Transaction  │                    │     │
│  │  │ Listener      │────▶│ Validator    │                    │     │
│  │  │               │     │              │                    │     │
│  │  │ - Monitor     │     │ - Verify     │                    │     │
│  │  │ - Confirm     │     │ - Parse memo │                    │     │
│  │  │ - Retry       │     │ - Amount     │                    │     │
│  │  └───────────────┘     └──────────────┘                    │     │
│  │                                                              │     │
│  └──────────────────────────────┬───────────────────────────────┘     │
│                                 │                                     │
│  ┌──────────────────────────────▼───────────────────────────────┐    │
│  │              Wallet Service                                   │    │
│  │                                                               │    │
│  │  - Hot wallet (receive payments)                             │    │
│  │  - Transaction signing                                       │    │
│  │  - Balance monitoring                                        │    │
│  │  - Transfer to cold wallet                                   │    │
│  └───────────────────────────────────────────────────────────────┘    │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
                             │
┌────────────────────────────▼──────────────────────────────────────────┐
│                         DATA LAYER                                    │
│                                                                        │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │
│  │   PostgreSQL     │  │     Redis        │  │   File Storage   │   │
│  │                  │  │                  │  │                  │   │
│  │ - Merchants      │  │ - Rate limit     │  │ - KYC docs       │   │
│  │ - Payments       │  │ - Session        │  │ - Audit files    │   │
│  │ - Payouts        │  │ - Cache          │  │ - SAR reports    │   │
│  │ - Ledger         │  │ - Wallet cache   │  │ (S3/MinIO)       │   │
│  │ - Audit logs     │  │                  │  │                  │   │
│  │ - AML tables     │  │                  │  │                  │   │
│  │   • Risk scores  │  │                  │  │                  │   │
│  │   • Alerts       │  │                  │  │                  │   │
│  │   • Cases        │  │                  │  │                  │   │
│  │   • Sanctions    │  │                  │  │                  │   │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘   │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Core Services Detail

### 1. Payment Service

**Responsibilities**
- Create payment requests
- Generate payment links/QR codes
- Track payment status
- Handle payment lifecycle

**Key Operations**
```typescript
interface PaymentService {
  createPayment(merchantId: string, params: CreatePaymentParams): Promise<Payment>
  getPaymentStatus(paymentId: string): Promise<PaymentStatus>
  confirmPayment(paymentId: string, txHash: string): Promise<void>
  expirePayment(paymentId: string): Promise<void>
}

interface CreatePaymentParams {
  amountVND: number
  orderId: string
  callbackUrl?: string
  metadata?: Record<string, any>
}
```

**State Machine**
```
CREATED → PENDING → CONFIRMING → COMPLETED
                             ↓
                         EXPIRED/FAILED
```

---

### 2. Blockchain Listener Service

**Responsibilities**
- Monitor blockchain for incoming transactions
- Confirm transaction finality
- Extract payment metadata (memo/reference)
- Trigger payment confirmation

**Architecture**
```typescript
class SolanaListener {
  private connection: Connection
  private wallet: PublicKey

  async start() {
    // Subscribe to wallet transactions
    this.connection.onAccountChange(
      this.wallet,
      this.handleTransaction.bind(this)
    )
  }

  async handleTransaction(accountInfo: AccountInfo) {
    // 1. Parse transaction
    // 2. Extract memo (payment_id)
    // 3. Verify amount
    // 4. Wait for confirmation (commitment: 'finalized')
    // 5. Update payment status
    // 6. Trigger webhook
  }
}
```

**Confirmation Levels**
- Solana: Wait for `finalized` commitment (~32 blocks, ~13 seconds)
- Future Ethereum: Wait for 12 confirmations (~3 minutes)

---

### 3. Ledger Service

**Responsibilities**
- Double-entry accounting
- Balance tracking
- Transaction history
- Audit trail

**Data Model**
```typescript
interface LedgerEntry {
  id: string
  timestamp: Date
  type: 'PAYMENT_RECEIVED' | 'PAYOUT_REQUESTED' | 'PAYOUT_COMPLETED' | 'FEE_CHARGED'
  merchantId: string

  // Double entry
  debitAccount: string   // e.g., 'crypto_pool', 'merchant_balance'
  creditAccount: string  // e.g., 'vnd_pool', 'merchant_balance'

  amount: number
  currency: 'VND' | 'USDT' | 'USDC'

  reference: string      // payment_id or payout_id
  metadata: any
}

interface MerchantBalance {
  merchantId: string
  availableVND: number   // Can withdraw
  pendingVND: number     // Not yet confirmed
  totalReceived: number
  totalPaidOut: number
}
```

**Example Flows**

1. **Payment Received**
```typescript
// Crypto received
DEBIT:  crypto_pool (+100 USDT)
CREDIT: liability_to_merchant (+2,300,000 VND)

// After OTC conversion
DEBIT:  liability_to_merchant (+2,300,000 VND)
CREDIT: vnd_pool (+2,300,000 VND)
CREDIT: merchant_available_balance (+2,277,000 VND)
CREDIT: fee_revenue (+23,000 VND)  // 1% fee
```

2. **Payout**
```typescript
DEBIT:  merchant_available_balance (+1,000,000 VND)
DEBIT:  merchant_available_balance (+50,000 VND)  // fee
CREDIT: vnd_pool (+1,000,000 VND)
CREDIT: fee_revenue (+50,000 VND)
```

---

### 4. Wallet Service

**Hot Wallet (Receives Payments)**
```typescript
interface HotWallet {
  chain: 'solana' | 'ethereum'
  address: string
  privateKey: string  // Encrypted in vault

  // Operations
  getBalance(): Promise<number>
  signTransaction(tx: Transaction): Promise<SignedTransaction>
  transferToColdWallet(amount: number): Promise<string>
}
```

**Security Measures**
- Private keys stored in environment variables (MVP) → HashiCorp Vault (Phase 2)
- Hot wallet keeps minimum balance (~$10k worth)
- Auto-sweep to cold wallet when balance > threshold
- Multi-sig for cold wallet (Phase 2)

---

### 5. Notification Service

**Webhook System**
```typescript
interface WebhookPayload {
  event: 'payment.completed' | 'payment.failed' | 'payout.completed'
  timestamp: string
  data: {
    paymentId?: string
    payoutId?: string
    merchantId: string
    amount: number
    status: string
  }
  signature: string  // HMAC-SHA256
}

class WebhookDispatcher {
  async send(merchantId: string, payload: WebhookPayload) {
    // 1. Sign payload with merchant secret
    // 2. POST to merchant callback URL
    // 3. Retry up to 5 times with exponential backoff
    // 4. Log all attempts
  }
}
```

**Email Notifications**
- Payment received (merchant)
- Payout approved (merchant)
- Daily settlement report (ops)
- Failed transaction alerts (ops)

---

### 6. AML Engine (Anti-Money Laundering)

**Responsibilities**
- Customer risk scoring and assessment
- Real-time transaction monitoring
- Sanctions screening (OFAC, UN, EU lists)
- Wallet risk analysis (crypto-specific)
- Alert generation and management
- Case management for investigations
- Regulatory reporting (SAR, threshold reports)

**Key Components**

```typescript
interface AMLEngine {
  // Customer risk assessment
  calculateCustomerRiskScore(merchant: Merchant): Promise<RiskScore>
  screenSanctions(name: string, idNumber: string): Promise<SanctionsHit>

  // Transaction monitoring
  monitorTransaction(payment: Payment): Promise<MonitoringResult>
  screenWallet(address: string, blockchain: string): Promise<WalletRisk>

  // Alert management
  createAlert(alert: Alert): Promise<void>
  resolveAlert(alertId: string, resolution: Resolution): Promise<void>

  // Reporting
  generateSAR(caseId: string): Promise<Report>
  generateThresholdReport(period: DateRange): Promise<Report>
}

interface RiskScore {
  merchantId: string
  riskLevel: 'low' | 'medium' | 'high' | 'prohibited'
  riskScore: number  // 0-100
  riskFactors: {
    businessType: number
    kycComplete: number
    transactionVolume: number
    geographicRisk: number
    historicalIssues: number
  }
  nextReviewDate: Date
}

interface MonitoringResult {
  shouldBlock: boolean
  riskScore: number
  triggeredRules: string[]
  walletRisk?: WalletRisk
  alerts: Alert[]
}

interface WalletRisk {
  address: string
  blockchain: string
  riskLevel: 'low' | 'medium' | 'high' | 'prohibited'
  riskScore: number
  isSanctioned: boolean
  riskFactors: {
    directMixer: boolean
    indirectMixer: boolean
    darknetExposure: boolean
    sanctionedInteraction: boolean
    walletAge: number
  }
}
```

**AML Rules (Examples)**

```typescript
// Threshold Rule - Vietnam Legal Requirement
{
  id: "THRESHOLD_VN_001",
  name: "Vietnam Legal Threshold",
  description: "Transactions ≥ 400M VND require reporting",
  category: "threshold",
  severity: "MEDIUM",
  conditions: [
    { field: "amount_vnd", operator: ">=", value: 400000000 }
  ],
  actions: [
    { type: "create_alert" },
    { type: "flag_for_reporting" }
  ]
}

// Structuring Detection
{
  id: "STRUCT_001",
  name: "Structuring Detection",
  description: "Multiple transactions just below threshold",
  category: "pattern",
  severity: "MEDIUM",
  conditions: [
    { field: "tx_count_24h", operator: ">=", value: 3 },
    { field: "amount_vnd", operator: "between", value: [8000000, 9500000] }
  ],
  actions: [
    { type: "create_alert" }
  ]
}

// Rapid Cash-Out
{
  id: "RAPID_001",
  name: "Rapid Cash-Out",
  description: "Payout within 1 hour of payment",
  category: "pattern",
  severity: "HIGH",
  conditions: [
    { field: "time_since_payment_minutes", operator: "<=", value: 60 },
    { field: "payout_percentage", operator: ">=", value: 80 }
  ],
  actions: [
    { type: "create_alert" },
    { type: "require_enhanced_review" }
  ]
}
```

**Integration Points**

1. **Merchant Registration** (KYC Phase)
   - Sanctions screening of business owner
   - Initial risk score calculation
   - Block registration if sanctions hit or prohibited risk

2. **Payment Confirmation**
   - Wallet address screening (check against OFAC sanctioned addresses)
   - Transaction monitoring (run all enabled rules)
   - Generate alerts if rules triggered
   - Block payment if critical alert or prohibited wallet

3. **Payout Request**
   - Check merchant risk level
   - Review recent alerts
   - Require enhanced review for high-risk merchants

**Compliance Standards**
- FATF 40 Recommendations
- Vietnam Law on Anti-Money Laundering (2022)
- FATF Travel Rule for crypto (≥ $1,000 USD)
- Threshold reporting: 400M VND (~$16,000 USD)

**Data Sources**
- OFAC SDN List (sanctions)
- UN Consolidated Sanctions List
- EU Sanctions Map
- Chainalysis Sanctions Oracle (crypto wallet screening)
- TRM Labs (wallet risk analysis)

**Reference**: See [AML_ENGINE.md](./AML_ENGINE.md) for comprehensive documentation.

---

## Database Schema

### Core Tables

```sql
-- Merchants
CREATE TABLE merchants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  business_name VARCHAR(255) NOT NULL,
  tax_id VARCHAR(50),

  kyc_status VARCHAR(50) DEFAULT 'pending',  -- pending, approved, rejected
  kyc_data JSONB,  -- Encrypted KYC documents references

  api_key VARCHAR(255) UNIQUE,
  webhook_url VARCHAR(500),
  webhook_secret VARCHAR(255),

  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Payments
CREATE TABLE payments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id UUID REFERENCES merchants(id),

  amount_vnd DECIMAL(15, 2) NOT NULL,
  amount_crypto DECIMAL(20, 8),
  crypto_currency VARCHAR(10),  -- USDT, USDC

  order_id VARCHAR(255),  -- Merchant's internal order ID

  wallet_address VARCHAR(255),  -- Our receiving wallet
  tx_hash VARCHAR(255) UNIQUE,

  status VARCHAR(50) DEFAULT 'created',  -- created, pending, confirming, completed, expired, failed

  callback_url VARCHAR(500),
  metadata JSONB,

  expires_at TIMESTAMP,
  confirmed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_merchant ON payments(merchant_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_tx_hash ON payments(tx_hash);

-- Payouts
CREATE TABLE payouts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id UUID REFERENCES merchants(id),

  amount_vnd DECIMAL(15, 2) NOT NULL,
  fee_vnd DECIMAL(15, 2) DEFAULT 50000,

  bank_name VARCHAR(255),
  bank_account_number VARCHAR(50),
  bank_account_name VARCHAR(255),

  status VARCHAR(50) DEFAULT 'requested',  -- requested, approved, processing, completed, rejected

  approved_by UUID,  -- Admin user ID
  approved_at TIMESTAMP,
  completed_at TIMESTAMP,

  reference_number VARCHAR(255),  -- Bank transaction reference

  created_at TIMESTAMP DEFAULT NOW()
);

-- Ledger (Double-entry accounting)
CREATE TABLE ledger_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  timestamp TIMESTAMP DEFAULT NOW(),

  entry_type VARCHAR(50) NOT NULL,

  debit_account VARCHAR(100),
  credit_account VARCHAR(100),

  amount DECIMAL(20, 8) NOT NULL,
  currency VARCHAR(10) NOT NULL,

  merchant_id UUID REFERENCES merchants(id),
  reference_id UUID,  -- payment_id or payout_id
  reference_type VARCHAR(50),  -- 'payment', 'payout'

  metadata JSONB
);

CREATE INDEX idx_ledger_merchant ON ledger_entries(merchant_id);
CREATE INDEX idx_ledger_timestamp ON ledger_entries(timestamp);

-- Merchant Balances (Computed view or materialized view)
CREATE TABLE merchant_balances (
  merchant_id UUID PRIMARY KEY REFERENCES merchants(id),

  available_vnd DECIMAL(15, 2) DEFAULT 0,
  pending_vnd DECIMAL(15, 2) DEFAULT 0,

  total_received_vnd DECIMAL(15, 2) DEFAULT 0,
  total_paid_out_vnd DECIMAL(15, 2) DEFAULT 0,

  updated_at TIMESTAMP DEFAULT NOW()
);

-- Audit Logs
CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  timestamp TIMESTAMP DEFAULT NOW(),

  actor_type VARCHAR(50),  -- 'system', 'admin', 'merchant'
  actor_id VARCHAR(255),

  action VARCHAR(100),  -- 'payment.created', 'kyc.approved', etc.
  resource_type VARCHAR(50),
  resource_id UUID,

  metadata JSONB,
  ip_address INET
);

CREATE INDEX idx_audit_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_resource ON audit_logs(resource_type, resource_id);

-- Blockchain Transactions (Tracking)
CREATE TABLE blockchain_transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  chain VARCHAR(50) NOT NULL,  -- 'solana', 'ethereum'
  tx_hash VARCHAR(255) UNIQUE NOT NULL,

  from_address VARCHAR(255),
  to_address VARCHAR(255),

  amount DECIMAL(20, 8),
  currency VARCHAR(10),

  block_number BIGINT,
  confirmations INT,

  payment_id UUID REFERENCES payments(id),

  status VARCHAR(50),  -- 'pending', 'confirmed', 'failed'

  detected_at TIMESTAMP DEFAULT NOW(),
  confirmed_at TIMESTAMP
);

CREATE INDEX idx_blockchain_tx_hash ON blockchain_transactions(tx_hash);
CREATE INDEX idx_blockchain_payment ON blockchain_transactions(payment_id);
```

---

## API Endpoints

### Public Payment API

```typescript
// Create payment
POST /api/v1/payments
Authorization: Bearer {merchant_api_key}
{
  "amountVND": 2300000,
  "orderId": "ORDER-12345",
  "callbackUrl": "https://merchant.com/webhook/payment",
  "metadata": { "customerId": "USER-999" }
}

Response:
{
  "paymentId": "pay_xxxx",
  "amountVND": 2300000,
  "amountUSDT": 100,
  "exchangeRate": 23000,
  "walletAddress": "8xK7...",
  "expiresAt": "2025-11-15T10:30:00Z",
  "paymentUrl": "https://pay.gateway.com/pay_xxxx",
  "qrCode": "data:image/png;base64,..."
}

// Get payment status
GET /api/v1/payments/{paymentId}
Authorization: Bearer {merchant_api_key}

Response:
{
  "paymentId": "pay_xxxx",
  "status": "completed",
  "amountVND": 2300000,
  "txHash": "5j7k...",
  "confirmedAt": "2025-11-15T10:25:30Z"
}
```

### Merchant API

```typescript
// Get balance
GET /api/v1/merchant/balance
Authorization: Bearer {merchant_api_key}

Response:
{
  "availableVND": 5000000,
  "pendingVND": 1200000,
  "totalReceived": 50000000,
  "totalPaidOut": 44000000
}

// Request payout
POST /api/v1/merchant/payouts
Authorization: Bearer {merchant_api_key}
{
  "amountVND": 5000000,
  "bankName": "Vietcombank",
  "bankAccountNumber": "1234567890",
  "bankAccountName": "CONG TY ABC"
}

Response:
{
  "payoutId": "payout_xxxx",
  "amountVND": 5000000,
  "feeVND": 50000,
  "status": "requested",
  "estimatedCompletionTime": "24-48 hours"
}

// Get transaction history
GET /api/v1/merchant/transactions?limit=50&offset=0
Authorization: Bearer {merchant_api_key}

Response:
{
  "transactions": [
    {
      "id": "pay_xxxx",
      "type": "payment",
      "amountVND": 2300000,
      "status": "completed",
      "createdAt": "2025-11-15T10:20:00Z"
    }
  ],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

### Internal Admin API

```typescript
// Approve KYC
POST /api/admin/merchants/{merchantId}/kyc/approve
Authorization: Bearer {admin_jwt}

// Approve payout
POST /api/admin/payouts/{payoutId}/approve
Authorization: Bearer {admin_jwt}

// System stats
GET /api/admin/stats
```

---

## Security Considerations

### API Security
- Rate limiting: 100 req/min per API key
- HMAC signature for webhooks
- TLS 1.3 only
- API key rotation support

### Data Security
- Encryption at rest (database)
- Encrypted KYC documents
- PII redacted in logs
- Private keys in vault

### Operational Security
- 2FA for admin access
- Audit logging for all critical operations
- Alerts for unusual activity
- Regular security audits

---

## Monitoring & Alerts

### Health Checks
- API endpoint: `GET /health`
- Database connectivity
- Blockchain RPC connectivity
- Redis connectivity

### Metrics to Track
- Payment success rate
- Average confirmation time
- Hot wallet balance
- Failed webhook deliveries
- API error rate

### Alerts
- Hot wallet balance < threshold
- Payment stuck in pending > 30 min
- Failed webhooks > 5 retries
- Unusual transaction patterns

---

## Deployment Architecture (MVP)

```
┌─────────────────────────────────────────────────────┐
│              Cloudflare (CDN + WAF)                 │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│          NGINX (Reverse Proxy + SSL)                │
└─────────────┬───────────────┬───────────────────────┘
              │               │
    ┌─────────▼─────┐   ┌─────▼─────────┐
    │  Frontend     │   │   Backend     │
    │  (Next.js)    │   │   (Node.js)   │
    │               │   │               │
    │  - Dashboard  │   │  - API        │
    │  - Payment    │   │  - Listener   │
    │    Page       │   │  - Worker     │
    └───────────────┘   └───────┬───────┘
                                │
                    ┌───────────┼────────────┐
                    │           │            │
            ┌───────▼───┐  ┌────▼────┐  ┌───▼──────┐
            │PostgreSQL │  │  Redis  │  │  MinIO   │
            └───────────┘  └─────────┘  └──────────┘
```

**Single VPS Specs (MVP)**
- 4 CPU cores
- 8 GB RAM
- 100 GB SSD
- Ubuntu 22.04 LTS

**Services**
- Docker + Docker Compose
- PM2 for process management
- PostgreSQL 15
- Redis 7
- Nginx

---

## Scalability Path

### Phase 2 (After MVP)
- Separate blockchain listener service
- Queue system (RabbitMQ/SQS)
- Multiple worker processes
- Read replicas for database

### Phase 3 (Scale)
- Kubernetes deployment
- Multi-region setup
- Dedicated blockchain nodes
- Microservices architecture

---

## 🆕 PRD v2.2 UPDATES (NEW MODULES)

**Last Updated**: 2025-11-19
**Status**: Design Phase

### New Modules Overview

PRD v2.2 introduces 5 major enhancements to the system:

| Module | Purpose | Priority | Doc Reference |
|--------|---------|----------|---------------|
| **Identity Mapping** | Wallet→User KYC (one-time) | 🔴 CRITICAL | [IDENTITY_MAPPING.md](./IDENTITY_MAPPING.md) |
| **Notification Center** | Omni-channel notifications | 🟡 HIGH | [NOTIFICATION_CENTER.md](./NOTIFICATION_CENTER.md) |
| **Data Retention** | Infinite storage (S3 Glacier) | 🔴 HIGH | [DATA_RETENTION.md](./DATA_RETENTION.md) |
| **Off-ramp Strategies** | Scheduled/threshold payouts | 🟢 MEDIUM | [OFF_RAMP_STRATEGIES.md](./OFF_RAMP_STRATEGIES.md) |
| **Custodial Treasury** | Multi-sig + sweeping | 🔴 CRITICAL | [PRD_v2.2.md](./PRD_v2.2.md) §2.2 |

---

### 1. Identity Mapping Service

**Purpose**: Link wallet addresses to user identities permanently

**Key Features**:
- First-time: KYC required (ID upload + face liveness)
- Returning: Auto-recognize wallet → Skip KYC
- Redis caching for <10ms lookup

**Database Tables**:
- `users` - Encrypted PII
- `wallet_identity_mappings` - Wallet ↔ User links
- `kyc_sessions` - Track KYC verification

**API Endpoints**:
```
GET /api/v1/wallet/:blockchain/:address/kyc-status
POST /api/v1/wallet/kyc/initiate
POST /api/v1/wallet/kyc/upload
POST /api/v1/wallet/kyc/liveness
```

**Integration Point**: Payment flow checks wallet KYC before allowing payment

---

### 2. Notification Center

**Purpose**: Multi-channel notifications to ensure merchants never miss payments

**Channels**:
1. **Speaker/TTS** (Priority 1): Audio alert at POS
2. **Telegram Bot** (Priority 2): Real-time push to boss
3. **Zalo OA/ZNS** (Priority 2): Vietnam market leader
4. **Email** (Priority 4): Invoice & statements
5. **Webhook** (Priority 3): POS/ERP integration

**Architecture**:
- Plugin-based design (easy to add channels)
- Redis Queue (Bull) for async delivery
- Retry logic with exponential backoff

**Database Table**:
- `notification_logs` - Track all notifications sent

**Delivery SLA**:
- Speaker: < 1 second
- Telegram/Zalo: < 5 seconds
- Email: < 30 seconds
- Webhook: < 10 seconds

---

### 3. Data Retention System

**Purpose**: Banking-grade infinite storage with immutability

**Storage Tiers**:
- **Hot Storage (0-12 months)**: PostgreSQL (fast queries)
- **Cold Storage (1+ years)**: S3 Glacier ($4/TB/month)

**Archival Process**:
- Monthly job moves old data to S3 Glacier
- Compressed JSON with gzip
- Keeps record IDs + hashes in DB for integrity

**Transaction Hashing**:
- SHA-256 hash per transaction
- Hash chain (each hash references previous)
- Daily Merkle tree for batch verification

**Database Tables**:
- `transaction_hashes` - Immutability tracking
- `archived_records` - S3 location metadata
- `merkle_roots` - Daily batch verification

**Restore Process**:
- S3 Glacier Expedited: 1-5 hours
- Automatic integrity verification via hash

---

### 4. Off-ramp Strategies

**Purpose**: Flexible VND withdrawal options for merchants

**Three Modes**:

**A. On-Demand** (Manual):
- Merchant clicks "Withdraw" → Enter amount → Ops approves

**B. Scheduled** (Auto):
- Weekly/Monthly schedule
- Example: Every Friday 4PM, withdraw 80% of balance

**C. Threshold-based** (Auto):
- Trigger when balance > threshold (e.g., 5,000 USDT)
- Auto-withdraw 90% when triggered

**Database Table**:
- `payout_schedules` - Store merchant withdrawal preferences

**Workers**:
- **PayoutScheduler**: Runs every minute, checks schedules
- **ThresholdMonitor**: Runs hourly, checks balance thresholds

---

### 5. Custodial Treasury (Enhanced)

**Purpose**: Secure multi-chain asset custody with automatic sweeping

**Hot Wallets** (per chain):
- TRON: Receive payments
- Solana: Receive payments
- BSC: Receive payments

**Cold Wallet**:
- Multi-sig 2-of-3 or MPC
- Main treasury storage

**Sweeping Mechanism**:
- Auto-sweep every 6 hours
- Threshold: Hot wallet > $10,000 USD
- Move excess to cold wallet
- Log all sweeps in `sweeping_logs` table

**Security**:
- Multi-sig requires 2 approvals
- Alerting when sweeping fails
- Manual backup process

---

### Updated Application Layer Diagram

```
┌────────────────────────────────────────────────────────────┐
│                   APPLICATION LAYER (PRD v2.2)             │
│                                                             │
│  ┌──────────────────┐  ┌──────────────────┐                │
│  │ Identity Mapping │  │ Payment Service  │                │
│  │ Service (NEW!)   │  │                  │                │
│  │ - Wallet→User    │  │ - Create payment │                │
│  │ - Face liveness  │  │ - Validate       │                │
│  │ - Redis cache    │  │ - Confirm        │                │
│  └──────────────────┘  └──────────────────┘                │
│                                                             │
│  ┌──────────────────┐  ┌──────────────────┐                │
│  │ Notification     │  │ Treasury Service │                │
│  │ Center (NEW!)    │  │ (ENHANCED)       │                │
│  │ - Speaker/TTS    │  │ - Custodial      │                │
│  │ - Telegram Bot   │  │ - Sweeping       │                │
│  │ - Zalo OA/ZNS    │  │ - Multi-sig      │                │
│  │ - Email/Webhook  │  └──────────────────┘                │
│  └──────────────────┘                                       │
│                        ┌──────────────────┐                 │
│  ┌──────────────────┐  │ Off-ramp Manager │                │
│  │ Data Archival    │  │ (NEW!)           │                │
│  │ Service (NEW!)   │  │ - On-demand      │                │
│  │ - Hot→Cold       │  │ - Scheduled      │                │
│  │ - S3 Glacier     │  │ - Threshold      │                │
│  │ - Hash verify    │  └──────────────────┘                │
│  └──────────────────┘                                       │
│                                                             │
│  ┌────────────────────────────────────────────────────┐    │
│  │             Ledger Service (ENHANCED)              │    │
│  │  - Double-entry accounting                         │    │
│  │  - Transaction hashing (NEW!)                      │    │
│  │  - Merkle tree verification (NEW!)                 │    │
│  └────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌────────────────────────────────────────────────────┐    │
│  │         AML Engine (Self-built) - EXISTING         │    │
│  │  - Customer risk scoring                           │    │
│  │  - Transaction monitoring                          │    │
│  │  - Sanctions screening                             │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

### Multi-Chain Expansion (PRD v2.2)

**Supported Chains**:

| Chain | Tokens | Priority | Reason |
|-------|--------|----------|--------|
| **TRON** | USDT (TRC20) | 🔴 HIGH | Cheapest fees (~$1), huge in Asia |
| **Solana** | USDT, USDC (SPL) | 🔴 HIGH | Fast finality (13s), low fees |
| **BSC** | USDT, BUSD (BEP20) | 🟡 MEDIUM | Popular in SEA |
| **Polygon** | USDT, USDC | 🟢 LOW | Growing ecosystem |
| **Ethereum** | USDT, USDC | ⚪ FUTURE | Expensive gas |

**Multi-Chain Listener Orchestrator**:
```
┌─────────────────────────────────────────────────────┐
│          Blockchain Listener Orchestrator           │
├─────────────────────────────────────────────────────┤
│  ┌──────┐  ┌────────┐  ┌─────┐  ┌────────┐        │
│  │ TRON │  │ Solana │  │ BSC │  │Polygon │        │
│  │Listen│  │Listener│  │List │  │Listener│        │
│  └───┬──┘  └───┬────┘  └──┬──┘  └───┬────┘        │
│      └─────────┴───────────┴─────────┘             │
│                    ↓                                │
│          Transaction Validator                      │
│          - Verify finality                          │
│          - AML wallet screening                     │
│                    ↓                                │
│          Treasury Service                           │
│          - Credit merchant balance                  │
│                    ↓                                │
│          Notification Dispatcher                    │
│          - Broadcast to all channels                │
└─────────────────────────────────────────────────────┘
```

---

### New Database Tables Summary

PRD v2.2 adds the following tables:

```sql
-- Identity Mapping
CREATE TABLE users (...);
CREATE TABLE wallet_identity_mappings (...);
CREATE TABLE kyc_sessions (...);

-- Notifications
CREATE TABLE notification_logs (...);

-- Off-ramp
CREATE TABLE payout_schedules (...);

-- Treasury
CREATE TABLE sweeping_logs (...);

-- Data Retention
CREATE TABLE archived_records (...);
CREATE TABLE transaction_hashes (...);
CREATE TABLE merkle_roots (...);
```

See detailed schemas in respective module documentation files.

---

### Updated Tech Stack

**New Dependencies**:

| Library | Purpose | Cost |
|---------|---------|------|
| **Sumsub SDK** | KYC & Face Liveness | $0.50/check |
| **node-telegram-bot-api** | Telegram integration | Free |
| **Zalo API** | Zalo OA/ZNS | $0.01/msg |
| **@sendgrid/mail** | Email delivery | $15/mo (40k emails) |
| **Google Cloud TTS** | Text-to-Speech | $4/1M chars |
| **aws-sdk (S3 Glacier)** | Long-term storage | $4/TB/month |
| **Bull** | Redis job queue | Free |

---

### Implementation Timeline (Revised)

**Original MVP**: 4-6 weeks
**PRD v2.2**: **8-10 weeks** (includes all new modules)

**Week-by-week breakdown**:
- Week 1-2: Foundation + Identity Mapping
- Week 3-4: Core Payment + Multi-chain
- Week 5: Notification Center
- Week 6: Treasury & Sweeping
- Week 7: Off-ramp + Data Retention
- Week 8: Admin Panel & Polish
- Week 9: Testing & Security Audit
- Week 10: Deployment & Pilot Launch

See [PRD_v2.2.md](./PRD_v2.2.md) Section 5 for full roadmap.

---

### Security Enhancements

**PRD v2.2 Additions**:

1. **PII Encryption**: All user identity data encrypted at rest (AES-256-GCM)
2. **Transaction Hashing**: SHA-256 hash chain for immutability
3. **Multi-sig Wallets**: Cold wallet requires 2-of-3 approvals
4. **Audit Logging**: All identity access logged with IP + reason
5. **Data Integrity**: Daily Merkle tree verification

---

### Compliance Standards

**New Requirements**:

- **GDPR Right to be Forgotten**: Anonymize user data on request
- **Vietnam Data Retention**: 7 years minimum (PRD v2.2: infinite)
- **KYC Re-verification**: Auto-prompt if KYC > 2 years old
- **Transaction Immutability**: Hash verification prevents tampering

---

### Monitoring & Alerts (PRD v2.2)

**New KPIs**:

| Metric | Target | Alert Threshold |
|--------|--------|----------------|
| KYC Recognition Rate | > 95% | < 90% |
| Notification Delivery | > 95% | < 90% |
| Speaker Latency | < 3 seconds | > 5 seconds |
| Sweeping Success Rate | 100% | < 100% |
| Data Integrity | 100% | < 100% |
| Cache Hit Rate | > 90% | < 80% |

---

### Reference Documents

For detailed implementation guides, see:

- [PRD_v2.2.md](./PRD_v2.2.md) - Complete product requirements
- [IDENTITY_MAPPING.md](./IDENTITY_MAPPING.md) - Wallet KYC system
- [NOTIFICATION_CENTER.md](./NOTIFICATION_CENTER.md) - Multi-channel notifications
- [DATA_RETENTION.md](./DATA_RETENTION.md) - Infinite storage architecture
- [OFF_RAMP_STRATEGIES.md](./OFF_RAMP_STRATEGIES.md) - Flexible withdrawals

---

**PRD v2.2 Status**: ✅ Design Complete
**Next Phase**: Implementation (Week 1-2 starting)
**Last Updated**: 2025-11-19

