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
│  └────────────────────────────┬─────────────────────────────────┘     │
│                               │                                      │
│  ┌────────────────────────────┴─────────────────────────────────┐   │
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
│  │ - Payouts        │  │ - Cache          │  │ (S3/MinIO)       │   │
│  │ - Ledger         │  │                  │  │                  │   │
│  │ - Audit logs     │  │                  │  │                  │   │
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

