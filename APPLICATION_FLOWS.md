# APPLICATION FLOWS - Comprehensive System Flow Documentation

**Project**: Stablecoin Payment Gateway MVP
**Last Updated**: 2025-11-18
**Purpose**: Detailed flow documentation for new engineers

---

## 📖 Table of Contents

1. [System Overview](#system-overview)
2. [Core Application Flows](#core-application-flows)
   - [Merchant Onboarding Flow](#1-merchant-onboarding-flow)
   - [Payment Creation Flow](#2-payment-creation-flow)
   - [Payment Confirmation Flow](#3-payment-confirmation-flow)
   - [Payout Request Flow](#4-payout-request-flow)
   - [OTC Settlement Flow](#5-otc-settlement-flow)
3. [Supporting Flows](#supporting-flows)
   - [Webhook Delivery Flow](#6-webhook-delivery-flow)
   - [Exchange Rate Update Flow](#7-exchange-rate-update-flow)
   - [Balance Reconciliation Flow](#8-balance-reconciliation-flow)
4. [Error Handling Flows](#error-handling-flows)
5. [Edge Cases & Special Scenarios](#edge-cases--special-scenarios)
6. [State Machine Diagrams](#state-machine-diagrams)
7. [Database Transaction Patterns](#database-transaction-patterns)

---

## System Overview

### High-Level Architecture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Tourist   │         │   Merchant   │         │  Admin/Ops  │
│  (End User) │         │  Dashboard   │         │    Panel    │
└──────┬──────┘         └──────┬───────┘         └──────┬──────┘
       │                       │                         │
       │ Scans QR              │ API Request             │ Manual Review
       │                       │                         │
       ▼                       ▼                         ▼
┌────────────────────────────────────────────────────────────────┐
│                     API Gateway Layer                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ Payment API  │  │ Merchant API │  │  Admin API   │       │
│  │ (Public)     │  │ (Protected)  │  │ (Protected)  │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────────────────────────────────────────────┘
       │                       │                         │
       ▼                       ▼                         ▼
┌────────────────────────────────────────────────────────────────┐
│                   Application Services Layer                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Payment    │  │   Merchant   │  │    Payout    │       │
│  │   Service    │  │   Service    │  │   Service    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │    Ledger    │  │ Notification │  │  Blockchain  │       │
│  │   Service    │  │   Service    │  │   Service    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────────────────────────────────────────────┘
       │                       │                         │
       ▼                       ▼                         ▼
┌────────────────────────────────────────────────────────────────┐
│                      Data Layer                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │  PostgreSQL  │  │    Redis     │  │  S3/MinIO    │       │
│  │  (Primary)   │  │   (Cache)    │  │   (Files)    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────────────────────────────────────────────┘
       │
       │ Listens for transactions
       ▼
┌────────────────────────────────────────────────────────────────┐
│                   Blockchain Layer                             │
│  ┌──────────────┐  ┌──────────────┐                          │
│  │    Solana    │  │   BSC/BNB    │                          │
│  │   Listener   │  │   Listener   │                          │
│  └──────────────┘  └──────────────┘                          │
└────────────────────────────────────────────────────────────────┘
       │                       │
       ▼                       ▼
┌────────────────────────────────────────────────────────────────┐
│               External Blockchain Networks                     │
│         Solana Mainnet       BSC Mainnet                       │
└────────────────────────────────────────────────────────────────┘
```

### Data Flow Overview

```
1. Merchant creates payment request
2. System generates QR code with payment details
3. Tourist scans QR and sends crypto
4. Blockchain listener detects transaction
5. System confirms payment and credits merchant balance
6. Merchant requests payout
7. Admin approves payout
8. Ops team executes bank transfer
9. System records payout in ledger
```

---

## Core Application Flows

### 1. Merchant Onboarding Flow

**Overview**: Process for new merchants to register, complete KYC, and start accepting payments.

#### Flow Diagram

```
┌──────────────┐
│ Merchant     │
│ Registration │
│ (Web Form)   │
└──────┬───────┘
       │
       ▼
┌────────────────────────────────────────────┐
│ Step 1: Basic Information Submission      │
│ - Business name                            │
│ - Email                                    │
│ - Phone number                             │
│ - Business type (hotel/restaurant/tour)    │
│ - Business address                         │
└──────┬─────────────────────────────────────┘
       │
       ▼
┌────────────────────────────────────────────┐
│ Step 2: System Processing                 │
│ - Validate email format                    │
│ - Check for duplicate email                │
│ - Generate merchant_id (UUID)              │
│ - Generate API key (secure random)         │
│ - Set initial status: kyc_pending          │
│ - Create database record                   │
└──────┬─────────────────────────────────────┘
       │
       ▼
┌────────────────────────────────────────────┐
│ Step 3: Email Verification                │
│ - Send verification email                  │
│ - Merchant clicks link                     │
│ - Update email_verified: true              │
└──────┬─────────────────────────────────────┘
       │
       ▼
┌────────────────────────────────────────────┐
│ Step 4: KYC Document Upload                │
│ - Business registration certificate        │
│ - Tax ID number                            │
│ - Owner ID card/passport                   │
│ - Bank account details                     │
│ - Business license (if applicable)         │
│ - Upload to S3/MinIO                       │
│ - Store file references in database        │
└──────┬─────────────────────────────────────┘
       │
       ▼
┌────────────────────────────────────────────┐
│ Step 5: Manual KYC Review (Admin)         │
│ - Admin reviews documents                  │
│ - Verify business legitimacy               │
│ - Check against sanctions list             │
│ - Decision: Approve or Reject              │
└──────┬─────────────────────────────────────┘
       │
       ├─── Rejected ──────────────────────────┐
       │                                        │
       │                                        ▼
       │                              ┌─────────────────┐
       │                              │ Send rejection  │
       │                              │ email with      │
       │                              │ reason          │
       │                              └─────────────────┘
       │
       └─── Approved ──────────────────────────┐
                                                │
                                                ▼
┌────────────────────────────────────────────────────────┐
│ Step 6: Account Activation                            │
│ - Update kyc_status: approved                         │
│ - Set status: active                                   │
│ - Initialize merchant_balance record                   │
│   - available_vnd: 0                                   │
│   - pending_vnd: 0                                     │
│ - Create audit log entry                               │
│ - Send welcome email with:                             │
│   - API key                                            │
│   - API documentation link                             │
│   - Dashboard access link                              │
└────────────────────────────────────────────────────────┘
       │
       ▼
┌──────────────────┐
│ Merchant Active  │
│ Ready to accept  │
│ payments         │
└──────────────────┘
```

#### Database State Changes

**merchants table**:
```sql
-- Step 1-2: Initial creation
INSERT INTO merchants (
  id,                    -- UUID generated
  email,                 -- From form
  business_name,         -- From form
  phone,                 -- From form
  business_type,         -- From form
  business_address,      -- From form
  kyc_status,            -- 'pending'
  status,                -- 'inactive'
  api_key,               -- Generated secure random
  email_verified,        -- false
  created_at,            -- NOW()
  updated_at             -- NOW()
) VALUES (...);

-- Step 3: Email verified
UPDATE merchants
SET email_verified = true, updated_at = NOW()
WHERE id = ?;

-- Step 5: KYC approved
UPDATE merchants
SET
  kyc_status = 'approved',
  status = 'active',
  kyc_approved_at = NOW(),
  kyc_approved_by = ?,  -- Admin user ID
  updated_at = NOW()
WHERE id = ?;
```

**merchant_kyc_documents table**:
```sql
INSERT INTO merchant_kyc_documents (
  id,
  merchant_id,
  document_type,        -- 'business_registration', 'tax_id', 'owner_id', 'bank_account'
  file_path,            -- S3/MinIO URL
  file_name,
  file_size,
  uploaded_at
) VALUES (...);
```

**merchant_balances table**:
```sql
-- Step 6: Initialize balance
INSERT INTO merchant_balances (
  merchant_id,
  available_vnd,        -- 0
  pending_vnd,          -- 0
  total_received_vnd,   -- 0
  total_paid_out_vnd,   -- 0
  last_updated_at
) VALUES (?, 0, 0, 0, 0, NOW());
```

**audit_logs table**:
```sql
-- Track all major steps
INSERT INTO audit_logs (
  actor_type,           -- 'system', 'merchant', 'admin'
  actor_id,
  action,               -- 'merchant_registered', 'kyc_submitted', 'kyc_approved'
  resource_type,        -- 'merchant'
  resource_id,          -- merchant_id
  metadata,             -- JSON with additional details
  created_at
) VALUES (...);
```

#### API Endpoints

```
POST /api/v1/merchants/register
Body: {
  "email": "hotel@example.com",
  "business_name": "Sunrise Hotel Da Nang",
  "phone": "+84901234567",
  "business_type": "hotel",
  "business_address": "123 Bach Dang, Da Nang"
}
Response: {
  "data": {
    "merchant_id": "uuid",
    "email": "hotel@example.com",
    "status": "email_verification_pending",
    "message": "Verification email sent"
  }
}

POST /api/v1/merchants/verify-email
Body: {
  "token": "verification_token"
}
Response: {
  "data": {
    "email_verified": true,
    "next_step": "kyc_document_upload"
  }
}

POST /api/v1/merchants/kyc/upload
Headers: { "X-API-Key": "merchant_api_key" }
Body: FormData with files
Response: {
  "data": {
    "documents_uploaded": 4,
    "kyc_status": "pending_review",
    "message": "Documents submitted for review"
  }
}
```

---

### 2. Payment Creation Flow

**Overview**: Merchant creates a payment request, system generates QR code for tourist to scan.

#### Flow Diagram

```
┌──────────────────┐
│ Merchant System  │
│ (POS/Website)    │
└────────┬─────────┘
         │
         │ API Request
         ▼
┌─────────────────────────────────────────────────┐
│ POST /api/v1/payments                          │
│ Headers: { "X-API-Key": "merchant_api_key" }   │
│ Body: {                                         │
│   "amount_vnd": 2300000,                        │
│   "description": "Hotel booking #12345",        │
│   "customer_email": "tourist@example.com",      │
│   "callback_url": "https://merchant.com/cb"     │
│ }                                               │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 1: Request Validation                     │
│ - Verify API key exists and valid              │
│ - Check merchant status is 'active'            │
│ - Validate amount > 0                          │
│ - Validate amount <= max_transaction_limit     │
│ - Check rate limiting (100 req/min)            │
└────────┬────────────────────────────────────────┘
         │
         │ ✓ Valid
         ▼
┌─────────────────────────────────────────────────┐
│ Step 2: Exchange Rate Lookup                   │
│ - Get current USDT/VND rate from cache         │
│ - If cache miss:                                │
│   - Fetch from exchange rate API               │
│   - Cache for 60 seconds                        │
│ - Example: 1 USDT = 23,000 VND                 │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 3: Calculate Crypto Amount                │
│ - amount_vnd = 2,300,000                        │
│ - exchange_rate = 23,000                        │
│ - amount_crypto = amount_vnd / exchange_rate    │
│ - amount_crypto = 100.00 USDT                   │
│ - Use decimal.Decimal (NOT float64!)           │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 4: Select Payment Chain & Token           │
│ - Default: Solana USDT (fastest, cheapest)     │
│ - Alternative: BSC USDT (if specified)          │
│ - Get hot wallet address for selected chain    │
│ - Solana: "ABC123...XYZ789"                     │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 5: Create Payment Record                  │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ - Generate payment_id (UUID)                    │
│ - Set status: 'created'                         │
│ - Calculate expiry: NOW() + 30 minutes          │
│ - Store all payment details                     │
│                                                 │
│ INSERT INTO payments (                          │
│   id, merchant_id, amount_vnd, amount_crypto,   │
│   crypto_currency, blockchain, wallet_address,  │
│   exchange_rate, status, expires_at,            │
│   description, customer_email, callback_url     │
│ ) VALUES (...);                                 │
│                                                 │
│ - Create audit log                              │
│ COMMIT;                                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 6: Generate QR Code                       │
│ - Create payment URL format:                    │
│   solana:{wallet}?                              │
│     amount={amount_crypto}&                     │
│     spl-token={token_mint_address}&             │
│     memo={payment_id}&                          │
│     label=StablecoinGateway                     │
│                                                 │
│ - Generate QR code image (base64 or URL)       │
│ - Store QR code reference                       │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 7: Return Response to Merchant            │
│ {                                               │
│   "data": {                                     │
│     "payment_id": "uuid",                       │
│     "amount_vnd": 2300000,                      │
│     "amount_crypto": 100.00,                    │
│     "crypto_currency": "USDT",                  │
│     "blockchain": "solana",                     │
│     "wallet_address": "ABC123...XYZ789",        │
│     "qr_code_url": "https://...",               │
│     "qr_code_base64": "data:image/png;...",     │
│     "payment_url": "solana:...",                │
│     "status": "created",                        │
│     "expires_at": "2025-11-18T11:30:00Z",       │
│     "status_url": "/api/v1/payments/{id}"       │
│   }                                             │
│ }                                               │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌──────────────────────┐
│ Merchant displays    │
│ QR code to tourist   │
└──────────────────────┘
```

#### Database State Changes

**payments table**:
```sql
INSERT INTO payments (
  id,                     -- UUID (payment_id)
  merchant_id,            -- From API key
  amount_vnd,             -- 2,300,000
  amount_crypto,          -- 100.00 (DECIMAL, not float!)
  crypto_currency,        -- 'USDT'
  blockchain,             -- 'solana'
  wallet_address,         -- Hot wallet address
  exchange_rate,          -- 23,000.00
  status,                 -- 'created'
  description,            -- 'Hotel booking #12345'
  customer_email,         -- 'tourist@example.com'
  callback_url,           -- 'https://merchant.com/cb'
  qr_code_url,            -- Generated QR code URL
  payment_url,            -- Solana payment URL
  expires_at,             -- NOW() + 30 minutes
  created_at,             -- NOW()
  updated_at              -- NOW()
) VALUES (...);
```

#### Error Scenarios

```
Error 1: Invalid API Key
├─ Response: 401 Unauthorized
└─ Body: { "error": { "code": "INVALID_API_KEY", "message": "..." } }

Error 2: Merchant Not Active
├─ Response: 403 Forbidden
└─ Body: { "error": { "code": "MERCHANT_INACTIVE", "message": "..." } }

Error 3: Amount Too Large
├─ Response: 400 Bad Request
└─ Body: { "error": { "code": "AMOUNT_EXCEEDS_LIMIT", "message": "Maximum: 10,000,000 VND" } }

Error 4: Rate Limit Exceeded
├─ Response: 429 Too Many Requests
└─ Body: { "error": { "code": "RATE_LIMIT_EXCEEDED", "message": "..." } }

Error 5: Exchange Rate API Down
├─ Fallback: Use cached rate (if < 5 minutes old)
├─ If no cache: Return 503 Service Unavailable
└─ Body: { "error": { "code": "EXCHANGE_RATE_UNAVAILABLE", "message": "..." } }
```

---

### 3. Payment Confirmation Flow

**Overview**: Tourist sends crypto, blockchain listener detects transaction, system confirms payment.

This is the **most critical flow** in the system. It must be bulletproof.

#### Flow Diagram

```
┌──────────────────┐
│ Tourist scans QR │
│ Opens wallet app │
└────────┬─────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Tourist's Wallet App                            │
│ - Pre-fills recipient address                   │
│ - Pre-fills amount (100 USDT)                   │
│ - Pre-fills memo (payment_id)                   │
│ - Tourist reviews and confirms                  │
│ - Wallet signs and broadcasts transaction       │
└────────┬────────────────────────────────────────┘
         │
         │ Transaction broadcast
         ▼
┌─────────────────────────────────────────────────┐
│ Solana/BSC Blockchain Network                   │
│ - Transaction enters mempool                    │
│ - Validators process transaction                │
│ - Transaction included in block                 │
│ - Block finalized                               │
└────────┬────────────────────────────────────────┘
         │
         │ Blockchain Listener polling
         ▼
┌─────────────────────────────────────────────────┐
│ Step 1: Transaction Detection                  │
│                                                 │
│ Blockchain Listener Service:                    │
│ - Polls RPC endpoint every 2 seconds            │
│ - Solana: getSignaturesForAddress()             │
│ - BSC: eth_getLogs() for Transfer events        │
│ - Filters for our hot wallet address            │
│ - Detects new incoming transaction              │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 2: Transaction Parsing                    │
│                                                 │
│ Extract transaction details:                    │
│ - tx_hash: "0xabc123..."                        │
│ - from_address: "Tourist's wallet"              │
│ - to_address: "Our hot wallet"                  │
│ - amount: 100.00 USDT                           │
│ - memo/reference: "payment_id"                  │
│ - timestamp: "2025-11-18T10:15:30Z"             │
│ - confirmations: 1 (initial)                    │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 3: Payment Lookup                         │
│                                                 │
│ SELECT * FROM payments                          │
│ WHERE id = ? -- payment_id from memo            │
│   AND status IN ('created', 'pending');         │
│                                                 │
│ Validation checks:                              │
│ ✓ Payment exists                                │
│ ✓ Payment not expired                           │
│ ✓ Payment not already completed                 │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 4: Amount Validation                      │
│                                                 │
│ Compare amounts (CRITICAL!):                    │
│ - Expected: payment.amount_crypto               │
│ - Received: transaction.amount                  │
│                                                 │
│ Validation:                                     │
│ - MUST match exactly (no tolerance!)            │
│ - Use decimal comparison, not float             │
│                                                 │
│ if received_amount != expected_amount {         │
│   // Log mismatch                               │
│   // Create alert                               │
│   // Manual review required                     │
│   return ERROR                                  │
│ }                                               │
└────────┬────────────────────────────────────────┘
         │
         │ ✓ Amount matches
         ▼
┌─────────────────────────────────────────────────┐
│ Step 5: Update Payment Status to 'pending'     │
│                                                 │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ UPDATE payments                                 │
│ SET                                             │
│   status = 'pending',                           │
│   detected_at = NOW(),                          │
│   updated_at = NOW()                            │
│ WHERE id = ?                                    │
│   AND status = 'created'; -- Prevent races      │
│                                                 │
│ -- Create blockchain transaction record         │
│ INSERT INTO blockchain_transactions (           │
│   id, payment_id, blockchain, tx_hash,          │
│   from_address, amount, confirmations,          │
│   status, detected_at                           │
│ ) VALUES (...);                                 │
│                                                 │
│ COMMIT;                                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 6: Wait for Finality                      │
│                                                 │
│ Solana:                                         │
│ - Poll getTransaction() with 'finalized'        │
│ - Wait ~13 seconds for finality                 │
│ - confirmationStatus === 'finalized'            │
│                                                 │
│ BSC:                                            │
│ - Wait for 15 block confirmations               │
│ - ~45 seconds (3 sec/block)                     │
│ - Monitor block height                          │
│                                                 │
│ Update confirmations in real-time:              │
│ UPDATE blockchain_transactions                  │
│ SET confirmations = ?, updated_at = NOW()       │
│ WHERE id = ?;                                   │
└────────┬────────────────────────────────────────┘
         │
         │ Finality reached
         ▼
┌─────────────────────────────────────────────────┐
│ Step 7: Confirm Payment (CRITICAL TRANSACTION) │
│                                                 │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ -- 1. Update payment status                     │
│ UPDATE payments                                 │
│ SET                                             │
│   status = 'completed',                         │
│   confirmed_at = NOW(),                         │
│   tx_hash = ?,                                  │
│   updated_at = NOW()                            │
│ WHERE id = ?                                    │
│   AND status = 'pending'; -- Race protection    │
│                                                 │
│ -- 2. Update blockchain transaction             │
│ UPDATE blockchain_transactions                  │
│ SET status = 'confirmed'                        │
│ WHERE payment_id = ?;                           │
│                                                 │
│ -- 3. Create ledger entries (double-entry)      │
│ -- Debit: External (crypto received)            │
│ -- Credit: Merchant balance (VND equivalent)    │
│                                                 │
│ INSERT INTO ledger_entries (                    │
│   entry_type, debit_account, credit_account,    │
│   amount, currency, reference_type,             │
│   reference_id, created_at                      │
│ ) VALUES (                                      │
│   'payment_received',                           │
│   'external:crypto',                            │
│   'merchant:{merchant_id}',                     │
│   2300000, -- VND amount                        │
│   'VND',                                        │
│   'payment',                                    │
│   payment_id,                                   │
│   NOW()                                         │
│ );                                              │
│                                                 │
│ -- 4. Update merchant balance                   │
│ UPDATE merchant_balances                        │
│ SET                                             │
│   available_vnd = available_vnd + 2300000,      │
│   total_received_vnd = total_received_vnd + 2300000,│
│   last_updated_at = NOW()                       │
│ WHERE merchant_id = ?;                          │
│                                                 │
│ -- 5. Create audit log                          │
│ INSERT INTO audit_logs (                        │
│   actor_type, action, resource_type,            │
│   resource_id, metadata, created_at             │
│ ) VALUES (                                      │
│   'system',                                     │
│   'payment_confirmed',                          │
│   'payment',                                    │
│   payment_id,                                   │
│   '{"tx_hash": "...", "amount": 100}',          │
│   NOW()                                         │
│ );                                              │
│                                                 │
│ COMMIT;                                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 8: Trigger Notifications                  │
│                                                 │
│ Execute asynchronously (queue):                 │
│                                                 │
│ 1. Send webhook to merchant                     │
│    POST {callback_url}                          │
│    Body: {                                      │
│      "event": "payment.completed",              │
│      "payment_id": "...",                       │
│      "amount_vnd": 2300000,                     │
│      "tx_hash": "...",                          │
│      "confirmed_at": "..."                      │
│    }                                            │
│    Headers: {                                   │
│      "X-Webhook-Signature": "HMAC-SHA256"       │
│    }                                            │
│                                                 │
│ 2. Send email to customer                       │
│    Subject: "Payment Confirmed"                 │
│    Body: Receipt with details                   │
│                                                 │
│ 3. Send email to merchant                       │
│    Subject: "New Payment Received"              │
│    Body: Payment details                        │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌──────────────────────┐
│ Payment Complete ✓   │
│ Merchant can see     │
│ updated balance      │
└──────────────────────┘
```

#### Critical Edge Cases

```
Edge Case 1: Duplicate Transaction Detection
├─ Scenario: Listener processes same tx twice (due to restart/bug)
├─ Protection: Check if tx_hash already exists before processing
└─ Query: SELECT id FROM blockchain_transactions WHERE tx_hash = ?

Edge Case 2: Partial Amount Received
├─ Scenario: Tourist sends 99.99 USDT instead of 100.00
├─ Action: Do NOT confirm payment
├─ Create alert for manual review
└─ Merchant can refund or request additional payment

Edge Case 3: Payment Expired But TX Arrives
├─ Scenario: Tourist scans QR at 10:29, sends at 10:31 (expired at 10:30)
├─ Action: Do NOT confirm payment automatically
├─ Create record for manual review
└─ Can be manually approved by ops team

Edge Case 4: Wrong Memo/Missing Memo
├─ Scenario: Tourist edits memo or wallet doesn't support memo
├─ Action: Transaction detected but can't match to payment
├─ Store in unmatched_transactions table
└─ Manual reconciliation by ops team

Edge Case 5: Blockchain Reorganization
├─ Scenario: Block gets reorged (more common on BSC)
├─ Protection: Wait for finality (Solana) / 15 confirmations (BSC)
├─ Monitor for tx reversal
└─ Alert if confirmed tx disappears

Edge Case 6: Overpayment
├─ Scenario: Tourist sends 105 USDT instead of 100
├─ Action: Accept payment, credit full VND equivalent
├─ Merchant keeps the extra (or can refund manually)
└─ Log the discrepancy

Edge Case 7: Multiple Payments Same ID
├─ Scenario: Tourist accidentally sends twice
├─ Protection: Check payment.status before confirming
├─ First tx: completed
└─ Second tx: Goes to unmatched_transactions for refund
```

#### Concurrency & Race Conditions

```
Race Condition 1: Multiple Listeners
├─ Scenario: Listener restarts while processing
├─ Protection: Use WHERE status = 'created' in UPDATE
├─ Only first update succeeds
└─ Second update affects 0 rows (check affected_rows)

Race Condition 2: Simultaneous Confirmations
├─ Scenario: Two listener instances process same tx
├─ Protection: Database transaction isolation
├─ Use SELECT FOR UPDATE when reading payment
└─ Second transaction waits or errors

Race Condition 3: Balance Updates
├─ Scenario: Multiple payments confirm simultaneously
├─ Protection: Use atomic operations
├─ UPDATE merchant_balances SET available = available + ?
└─ NOT: SELECT balance -> calculate -> UPDATE balance = ?
```

---

### 4. Payout Request Flow

**Overview**: Merchant requests to withdraw VND from their balance to bank account.

#### Flow Diagram

```
┌──────────────────┐
│ Merchant         │
│ Dashboard        │
└────────┬─────────┘
         │
         │ Request payout
         ▼
┌─────────────────────────────────────────────────┐
│ POST /api/v1/payouts                           │
│ Headers: { "X-API-Key": "merchant_api_key" }   │
│ Body: {                                         │
│   "amount_vnd": 2000000,                        │
│   "bank_name": "Vietcombank",                   │
│   "bank_account_number": "1234567890",          │
│   "bank_account_name": "SUNRISE HOTEL"          │
│ }                                               │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 1: Request Validation                     │
│                                                 │
│ - Verify merchant API key                      │
│ - Check merchant status is 'active'            │
│ - Validate amount > 0                          │
│ - Validate bank details format                 │
│ - Check minimum payout: 500,000 VND            │
│ - Verify bank account matches KYC              │
└────────┬────────────────────────────────────────┘
         │
         │ ✓ Valid
         ▼
┌─────────────────────────────────────────────────┐
│ Step 2: Balance Check                          │
│                                                 │
│ SELECT available_vnd                            │
│ FROM merchant_balances                          │
│ WHERE merchant_id = ?;                          │
│                                                 │
│ Calculate:                                      │
│ - Requested: 2,000,000 VND                      │
│ - Fee (1%): 20,000 VND                          │
│ - Total needed: 2,020,000 VND                   │
│ - Available: 2,300,000 VND                      │
│                                                 │
│ ✓ Sufficient balance                            │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 3: Create Payout Record                   │
│                                                 │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ -- Generate payout_id                           │
│ -- Reserve balance (prevent double-spending)    │
│                                                 │
│ INSERT INTO payouts (                           │
│   id,                                           │
│   merchant_id,                                  │
│   amount_vnd,           -- 2,000,000            │
│   fee_vnd,              -- 20,000               │
│   total_vnd,            -- 2,020,000            │
│   bank_name,                                    │
│   bank_account_number,                          │
│   bank_account_name,                            │
│   status,               -- 'requested'          │
│   requested_at,         -- NOW()                │
│   created_at,                                   │
│   updated_at                                    │
│ ) VALUES (...);                                 │
│                                                 │
│ -- Reserve balance (move to pending)            │
│ UPDATE merchant_balances                        │
│ SET                                             │
│   available_vnd = available_vnd - 2020000,      │
│   pending_vnd = pending_vnd + 2020000,          │
│   last_updated_at = NOW()                       │
│ WHERE merchant_id = ?                           │
│   AND available_vnd >= 2020000; -- Safety check │
│                                                 │
│ IF (affected_rows = 0) THEN                     │
│   ROLLBACK;                                     │
│   RETURN ERROR "Insufficient balance";          │
│ END IF;                                         │
│                                                 │
│ -- Create audit log                             │
│ INSERT INTO audit_logs (...);                   │
│                                                 │
│ COMMIT;                                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 4: Return Response                        │
│ {                                               │
│   "data": {                                     │
│     "payout_id": "uuid",                        │
│     "amount_vnd": 2000000,                      │
│     "fee_vnd": 20000,                           │
│     "total_vnd": 2020000,                       │
│     "status": "requested",                      │
│     "estimated_completion": "1-2 business days",│
│     "message": "Payout request submitted"       │
│   }                                             │
│ }                                               │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 5: Admin Review (Manual for MVP)          │
│                                                 │
│ Admin logs into admin panel                     │
│ - Views pending payout requests                 │
│ - Reviews merchant history                      │
│ - Checks for fraud indicators                   │
│ - Verifies bank details                         │
│                                                 │
│ Decision: Approve or Reject                     │
└────────┬────────────────────────────────────────┘
         │
         ├─── REJECTED ─────────────────────┐
         │                                   │
         │                                   ▼
         │                 ┌──────────────────────────────┐
         │                 │ Step 6a: Rejection Process   │
         │                 │                              │
         │                 │ BEGIN TRANSACTION;           │
         │                 │                              │
         │                 │ UPDATE payouts               │
         │                 │ SET                          │
         │                 │   status = 'rejected',       │
         │                 │   rejection_reason = ?,      │
         │                 │   reviewed_by = ?,           │
         │                 │   reviewed_at = NOW()        │
         │                 │ WHERE id = ?;                │
         │                 │                              │
         │                 │ -- Return balance to merchant│
         │                 │ UPDATE merchant_balances     │
         │                 │ SET                          │
         │                 │   available_vnd = available_vnd + 2020000,│
         │                 │   pending_vnd = pending_vnd - 2020000│
         │                 │ WHERE merchant_id = ?;       │
         │                 │                              │
         │                 │ -- Audit log                 │
         │                 │ INSERT INTO audit_logs (...);│
         │                 │                              │
         │                 │ COMMIT;                      │
         │                 │                              │
         │                 │ -- Send rejection email      │
         │                 └──────────────────────────────┘
         │
         └─── APPROVED ─────────────────────┐
                                             │
                                             ▼
┌─────────────────────────────────────────────────┐
│ Step 6b: Approval Process                      │
│                                                 │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ UPDATE payouts                                  │
│ SET                                             │
│   status = 'approved',                          │
│   approved_by = ?,     -- Admin user ID         │
│   approved_at = NOW(),                          │
│   updated_at = NOW()                            │
│ WHERE id = ?;                                   │
│                                                 │
│ -- Audit log                                    │
│ INSERT INTO audit_logs (                        │
│   actor_type, actor_id, action,                 │
│   resource_type, resource_id, metadata          │
│ ) VALUES (                                      │
│   'admin', ?, 'payout_approved',                │
│   'payout', payout_id, '{}'                     │
│ );                                              │
│                                                 │
│ COMMIT;                                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 7: Ops Team Executes Bank Transfer        │
│                                                 │
│ Ops team:                                       │
│ 1. Logs into business bank account              │
│ 2. Creates bank transfer:                       │
│    - To: Merchant bank account                  │
│    - Amount: 2,000,000 VND                      │
│    - Reference: Payout ID                       │
│ 3. Confirms transfer                            │
│ 4. Takes screenshot of receipt                  │
│                                                 │
│ Update payout status:                           │
│ UPDATE payouts                                  │
│ SET                                             │
│   status = 'processing',                        │
│   processing_started_at = NOW()                 │
│ WHERE id = ?;                                   │
└────────┬────────────────────────────────────────┘
         │
         │ Bank transfer completed (T+0 to T+1)
         ▼
┌─────────────────────────────────────────────────┐
│ Step 8: Mark Payout as Completed               │
│                                                 │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ -- Update payout status                         │
│ UPDATE payouts                                  │
│ SET                                             │
│   status = 'completed',                         │
│   completed_at = NOW(),                         │
│   bank_reference = ?,   -- Bank tx reference    │
│   updated_at = NOW()                            │
│ WHERE id = ?;                                   │
│                                                 │
│ -- Create ledger entries (double-entry)         │
│ INSERT INTO ledger_entries (                    │
│   entry_type, debit_account, credit_account,    │
│   amount, currency, reference_type,             │
│   reference_id, created_at                      │
│ ) VALUES (                                      │
│   'payout_completed',                           │
│   'merchant:{merchant_id}',                     │
│   'external:bank',                              │
│   2000000,                                      │
│   'VND',                                        │
│   'payout',                                     │
│   payout_id,                                    │
│   NOW()                                         │
│ );                                              │
│                                                 │
│ -- Record fee                                   │
│ INSERT INTO ledger_entries (                    │
│   entry_type, debit_account, credit_account,    │
│   amount, currency, reference_type,             │
│   reference_id, created_at                      │
│ ) VALUES (                                      │
│   'payout_fee',                                 │
│   'merchant:{merchant_id}',                     │
│   'revenue:payout_fees',                        │
│   20000,                                        │
│   'VND',                                        │
│   'payout',                                     │
│   payout_id,                                    │
│   NOW()                                         │
│ );                                              │
│                                                 │
│ -- Update merchant balance (remove from pending)│
│ UPDATE merchant_balances                        │
│ SET                                             │
│   pending_vnd = pending_vnd - 2020000,          │
│   total_paid_out_vnd = total_paid_out_vnd + 2000000,│
│   last_updated_at = NOW()                       │
│ WHERE merchant_id = ?;                          │
│                                                 │
│ -- Audit log                                    │
│ INSERT INTO audit_logs (...);                   │
│                                                 │
│ COMMIT;                                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 9: Send Completion Notification           │
│                                                 │
│ Send email to merchant:                         │
│ - Subject: "Payout Completed"                   │
│ - Body:                                         │
│   - Amount: 2,000,000 VND                       │
│   - Bank account: Vietcombank ***7890           │
│   - Payout ID                                   │
│   - Bank reference                              │
│   - Completed at timestamp                      │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌──────────────────────┐
│ Payout Complete ✓    │
│ Merchant receives    │
│ VND in bank account  │
└──────────────────────┘
```

#### Payout States

```
State Machine:
requested → approved → processing → completed
    ↓
rejected (terminal state)
```

#### Security & Fraud Checks

```
Pre-Approval Checks:
1. Merchant account age > 7 days (for new merchants)
2. Total payments received > payout amount × 1.5
3. No chargebacks or disputes in last 30 days
4. Bank account matches KYC documents
5. Payout frequency (max 1 per day for new merchants)
6. Velocity check (max 10M VND per day initially)

Risk Scoring:
- Low risk: Auto-approve (future phase)
- Medium risk: Manual review
- High risk: Additional verification required

Fraud Indicators:
- Sudden large payout after small payments
- Bank account changed recently
- Multiple failed KYC attempts
- IP address from high-risk country
- Unusual payment patterns
```

---

### 5. OTC Settlement Flow

**Overview**: Convert accumulated crypto to VND via OTC partner.

#### Flow Diagram

```
┌──────────────────────────┐
│ Background Job (Daily)   │
│ Runs at 9 AM Vietnam time│
└────────┬─────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 1: Check Hot Wallet Balance               │
│                                                 │
│ Query blockchain:                               │
│ - Solana: getTokenAccountBalance()              │
│ - BSC: balanceOf() for USDT contract            │
│                                                 │
│ Example result:                                 │
│ - Solana USDT: 5,000 USDT                       │
│ - BSC USDT: 3,000 USDT                          │
│ - Total: 8,000 USDT                             │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 2: Check Threshold                        │
│                                                 │
│ if (total_balance < OTC_THRESHOLD) {            │
│   // Threshold: 50,000 USDT for MVP             │
│   log("Balance below threshold");               │
│   exit;                                         │
│ }                                               │
│                                                 │
│ ✓ Balance: 8,000 USDT > 5,000 threshold         │
│ Proceed to OTC settlement                       │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 3: Create OTC Settlement Request          │
│                                                 │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ INSERT INTO otc_settlements (                   │
│   id,                                           │
│   crypto_amount,        -- 8,000 USDT           │
│   crypto_currency,      -- 'USDT'               │
│   estimated_vnd,        -- 8000 × 23000         │
│   status,               -- 'pending'            │
│   requested_at,                                 │
│   created_at                                    │
│ ) VALUES (...);                                 │
│                                                 │
│ COMMIT;                                         │
│                                                 │
│ -- Send alert to ops team                       │
│ send_slack_notification(                        │
│   "OTC settlement required: 8,000 USDT"         │
│ );                                              │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 4: Ops Team Contacts OTC Partner          │
│                                                 │
│ Ops team:                                       │
│ 1. Reviews current VND needs                    │
│ 2. Contacts OTC partner (e.g., VAIX, Remitano)  │
│ 3. Gets quote:                                  │
│    - Amount: 8,000 USDT                         │
│    - Rate: 23,500 VND/USDT (premium included)   │
│    - Total VND: 188,000,000 VND                 │
│    - OTC fee: 0.5%                              │
│ 4. Confirms trade                               │
│                                                 │
│ Update record:                                  │
│ UPDATE otc_settlements                          │
│ SET                                             │
│   status = 'confirmed',                         │
│   otc_partner = 'VAIX',                         │
│   exchange_rate = 23500,                        │
│   vnd_amount = 188000000,                       │
│   otc_fee = 940000,                             │
│   confirmed_at = NOW()                          │
│ WHERE id = ?;                                   │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 5: Execute Crypto Transfer                │
│                                                 │
│ Ops team (or automated in future):              │
│                                                 │
│ 1. Generate transaction:                        │
│    - From: Hot wallet                           │
│    - To: OTC partner wallet                     │
│    - Amount: 8,000 USDT                         │
│                                                 │
│ 2. Sign transaction with hot wallet key         │
│                                                 │
│ 3. Broadcast to blockchain                      │
│                                                 │
│ 4. Record transaction hash                      │
│    UPDATE otc_settlements                       │
│    SET                                          │
│      crypto_tx_hash = ?,                        │
│      crypto_sent_at = NOW(),                    │
│      status = 'crypto_sent'                     │
│    WHERE id = ?;                                │
│                                                 │
│ 5. Wait for confirmation (similar to payment)   │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 6: Receive VND from OTC Partner           │
│                                                 │
│ OTC partner transfers VND:                      │
│ - To: Business bank account                     │
│ - Amount: 188,000,000 VND                       │
│ - Reference: OTC settlement ID                  │
│                                                 │
│ Ops team verifies receipt:                      │
│ 1. Check business bank account                  │
│ 2. Verify amount matches                        │
│ 3. Verify reference ID                          │
│                                                 │
│ Update record:                                  │
│ UPDATE otc_settlements                          │
│ SET                                             │
│   status = 'vnd_received',                      │
│   vnd_received_at = NOW(),                      │
│   bank_reference = ?                            │
│ WHERE id = ?;                                   │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 7: Complete Settlement & Update Ledger    │
│                                                 │
│ BEGIN TRANSACTION;                              │
│                                                 │
│ -- Mark settlement complete                     │
│ UPDATE otc_settlements                          │
│ SET                                             │
│   status = 'completed',                         │
│   completed_at = NOW()                          │
│ WHERE id = ?;                                   │
│                                                 │
│ -- Create ledger entries                        │
│ -- 1. Record crypto outflow                     │
│ INSERT INTO ledger_entries (                    │
│   entry_type, debit_account, credit_account,    │
│   amount, currency, reference_type,             │
│   reference_id, created_at                      │
│ ) VALUES (                                      │
│   'otc_crypto_sent',                            │
│   'asset:vnd_pool',                             │
│   'external:otc_partner',                       │
│   8000,                                         │
│   'USDT',                                       │
│   'otc_settlement',                             │
│   settlement_id,                                │
│   NOW()                                         │
│ );                                              │
│                                                 │
│ -- 2. Record VND inflow                         │
│ INSERT INTO ledger_entries (                    │
│   entry_type, debit_account, credit_account,    │
│   amount, currency, reference_type,             │
│   reference_id, created_at                      │
│ ) VALUES (                                      │
│   'otc_vnd_received',                           │
│   'external:otc_partner',                       │
│   'asset:vnd_pool',                             │
│   188000000,                                    │
│   'VND',                                        │
│   'otc_settlement',                             │
│   settlement_id,                                │
│   NOW()                                         │
│ );                                              │
│                                                 │
│ -- 3. Record OTC fee                            │
│ INSERT INTO ledger_entries (                    │
│   entry_type, debit_account, credit_account,    │
│   amount, currency, reference_type,             │
│   reference_id, created_at                      │
│ ) VALUES (                                      │
│   'otc_fee',                                    │
│   'asset:vnd_pool',                             │
│   'expense:otc_fees',                           │
│   940000,                                       │
│   'VND',                                        │
│   'otc_settlement',                             │
│   settlement_id,                                │
│   NOW()                                         │
│ );                                              │
│                                                 │
│ -- Audit log                                    │
│ INSERT INTO audit_logs (...);                   │
│                                                 │
│ COMMIT;                                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌──────────────────────┐
│ Settlement Complete  │
│ VND pool replenished │
└──────────────────────┘
```

#### OTC Settlement States

```
pending → confirmed → crypto_sent → vnd_received → completed
   ↓
cancelled (if needed)
```

---

## Supporting Flows

### 6. Webhook Delivery Flow

**Overview**: Notify merchant of payment events via webhook.

#### Flow Diagram

```
┌──────────────────────┐
│ Payment Event        │
│ (completed/failed)   │
└────────┬─────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 1: Prepare Webhook Payload                │
│                                                 │
│ payload = {                                     │
│   "event": "payment.completed",                 │
│   "payment_id": "uuid",                         │
│   "merchant_id": "uuid",                        │
│   "amount_vnd": 2300000,                        │
│   "amount_crypto": 100.00,                      │
│   "crypto_currency": "USDT",                    │
│   "blockchain": "solana",                       │
│   "tx_hash": "0xabc...",                        │
│   "status": "completed",                        │
│   "confirmed_at": "2025-11-18T10:15:30Z",       │
│   "timestamp": "2025-11-18T10:15:31Z"           │
│ };                                              │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 2: Generate HMAC Signature                │
│                                                 │
│ // Get merchant's webhook secret                │
│ secret = merchant.webhook_secret;               │
│                                                 │
│ // Generate signature                           │
│ message = JSON.stringify(payload);              │
│ signature = HMAC_SHA256(message, secret);       │
│                                                 │
│ // Encode as hex                                │
│ signature_hex = signature.toString('hex');      │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 3: Send HTTP Request                      │
│                                                 │
│ POST merchant.webhook_url                       │
│ Headers: {                                      │
│   "Content-Type": "application/json",           │
│   "X-Webhook-Signature": signature_hex,         │
│   "X-Webhook-Event": "payment.completed",       │
│   "X-Webhook-ID": unique_delivery_id,           │
│   "User-Agent": "StablecoinGateway/1.0"         │
│ }                                               │
│ Body: JSON.stringify(payload)                   │
│                                                 │
│ Timeout: 10 seconds                             │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 4: Handle Response                        │
│                                                 │
│ if (status_code === 200) {                      │
│   // Success                                    │
│   log_webhook_delivery(success);                │
│   return;                                       │
│ }                                               │
│                                                 │
│ // Failed - retry with exponential backoff      │
│ retry_attempts = [30s, 1m, 5m, 15m, 1h];        │
│                                                 │
│ for (attempt in retry_attempts) {               │
│   wait(attempt);                                │
│   result = send_webhook();                      │
│   if (result.success) {                         │
│     log_webhook_delivery(success, attempt);     │
│     return;                                     │
│   }                                             │
│ }                                               │
│                                                 │
│ // All retries failed                           │
│ log_webhook_delivery(failed);                   │
│ alert_ops_team("Webhook delivery failed");      │
└─────────────────────────────────────────────────┘
```

#### Webhook Delivery Record

```sql
CREATE TABLE webhook_deliveries (
  id UUID PRIMARY KEY,
  merchant_id UUID REFERENCES merchants(id),
  webhook_url TEXT,
  event_type VARCHAR(50),
  payload JSONB,
  signature VARCHAR(128),
  status VARCHAR(20), -- 'pending', 'delivered', 'failed'
  attempts INT DEFAULT 1,
  last_attempt_at TIMESTAMP,
  response_status_code INT,
  response_body TEXT,
  delivered_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);
```

---

### 7. Exchange Rate Update Flow

**Overview**: Fetch and cache current exchange rates.

#### Flow Diagram

```
┌──────────────────────────┐
│ Background Job           │
│ Runs every 60 seconds    │
└────────┬─────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 1: Fetch from Exchange Rate API           │
│                                                 │
│ Primary: CoinGecko API                          │
│ GET https://api.coingecko.com/api/v3/simple/price│
│ ?ids=tether,usd-coin                            │
│ &vs_currencies=vnd                              │
│                                                 │
│ Response:                                       │
│ {                                               │
│   "tether": { "vnd": 25300 },                   │
│   "usd-coin": { "vnd": 25280 }                  │
│ }                                               │
│                                                 │
│ Fallback: Binance API (if CoinGecko fails)      │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 2: Apply Spread/Margin                    │
│                                                 │
│ // Add 1.5% margin for volatility buffer        │
│ margin = 0.015;                                 │
│ usdt_vnd_rate = 25300 * (1 - margin);           │
│ usdt_vnd_rate = 24,920 VND (rounded)            │
│                                                 │
│ // This protects against rate movements         │
│ // between quote and settlement                 │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 3: Store in Redis Cache                   │
│                                                 │
│ SET exchange_rate:USDT:VND "24920"              │
│ EXPIRE exchange_rate:USDT:VND 120               │
│                                                 │
│ SET exchange_rate:USDC:VND "24900"              │
│ EXPIRE exchange_rate:USDC:VND 120               │
│                                                 │
│ // Also store timestamp                         │
│ SET exchange_rate:last_updated "2025-11-18T10:00:00Z"│
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 4: Store Historical Rate (Database)       │
│                                                 │
│ INSERT INTO exchange_rates (                    │
│   crypto_currency,                              │
│   fiat_currency,                                │
│   rate,                                         │
│   source,                                       │
│   fetched_at                                    │
│ ) VALUES (                                      │
│   'USDT', 'VND', 24920, 'coingecko', NOW()      │
│ );                                              │
│                                                 │
│ // Used for historical analysis and auditing    │
└─────────────────────────────────────────────────┘
```

#### Rate Usage in Payments

```
1. Payment creation:
   - GET exchange_rate:USDT:VND from Redis
   - If not found: Fetch from API immediately
   - Store rate in payment record (for audit trail)

2. Payment confirmation:
   - Use stored rate from payment record
   - Do NOT recalculate based on current rate
   - This ensures merchant gets quoted amount
```

---

### 8. Balance Reconciliation Flow

**Overview**: Daily verification that balances match ledger entries.

#### Flow Diagram

```
┌──────────────────────────┐
│ Background Job           │
│ Runs daily at midnight   │
└────────┬─────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 1: Calculate Ledger-Based Balances        │
│                                                 │
│ FOR each merchant:                              │
│                                                 │
│ -- Sum all credits (payments received)          │
│ SELECT SUM(amount) as total_credits             │
│ FROM ledger_entries                             │
│ WHERE credit_account = 'merchant:{id}'          │
│   AND currency = 'VND';                         │
│                                                 │
│ -- Sum all debits (payouts, fees)               │
│ SELECT SUM(amount) as total_debits              │
│ FROM ledger_entries                             │
│ WHERE debit_account = 'merchant:{id}'           │
│   AND currency = 'VND';                         │
│                                                 │
│ -- Calculate expected balance                   │
│ expected_balance = total_credits - total_debits;│
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 2: Compare with Merchant Balance Table    │
│                                                 │
│ SELECT available_vnd, pending_vnd               │
│ FROM merchant_balances                          │
│ WHERE merchant_id = ?;                          │
│                                                 │
│ actual_total = available_vnd + pending_vnd;     │
│                                                 │
│ discrepancy = expected_balance - actual_total;  │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Step 3: Handle Discrepancies                   │
│                                                 │
│ if (discrepancy === 0) {                        │
│   // Perfect match - log success                │
│   log_reconciliation(merchant_id, "success");   │
│   return;                                       │
│ }                                               │
│                                                 │
│ if (Math.abs(discrepancy) < 100) {              │
│   // Minor rounding difference - acceptable     │
│   log_reconciliation(merchant_id, "minor");     │
│   return;                                       │
│ }                                               │
│                                                 │
│ // Significant discrepancy - alert!             │
│ alert_ops_team({                                │
│   merchant_id: merchant_id,                     │
│   expected: expected_balance,                   │
│   actual: actual_total,                         │
│   discrepancy: discrepancy,                     │
│   severity: "HIGH"                              │
│ });                                             │
│                                                 │
│ // Create investigation ticket                  │
│ CREATE TABLE reconciliation_issues (            │
│   merchant_id, expected, actual,                │
│   discrepancy, status, created_at               │
│ );                                              │
└─────────────────────────────────────────────────┘
```

---

## Error Handling Flows

### Error Types & Recovery

```
1. Transient Errors (Retry)
   ├─ Network timeout
   ├─ RPC rate limit
   ├─ Temporary database connection loss
   └─ Strategy: Exponential backoff retry

2. Validation Errors (Reject)
   ├─ Invalid amount
   ├─ Insufficient balance
   ├─ Payment expired
   └─ Strategy: Return error to caller

3. Business Logic Errors (Alert & Manual)
   ├─ Amount mismatch
   ├─ Missing payment memo
   ├─ Duplicate transaction
   └─ Strategy: Queue for manual review

4. System Errors (Alert & Investigate)
   ├─ Database corruption
   ├─ Ledger imbalance
   ├─ Critical service down
   └─ Strategy: Alert on-call, investigate immediately
```

### Dead Letter Queue Pattern

```
┌─────────────┐
│   Payment   │
│   Event     │
└──────┬──────┘
       │
       ▼
┌──────────────┐
│  Processing  │──Success──► Done
└──────┬───────┘
       │
       │ Failure
       ▼
┌──────────────┐
│ Retry Queue  │──Success──► Done
└──────┬───────┘
       │
       │ Max retries exceeded
       ▼
┌──────────────────┐
│ Dead Letter      │
│ Queue (Manual)   │
└──────────────────┘
```

---

## State Machine Diagrams

### Payment Status State Machine

```
                    ┌─────────┐
                    │ created │
                    └────┬────┘
                         │
           User sends crypto│
                         │
                         ▼
                    ┌─────────┐
            ┌───────│ pending │
            │       └────┬────┘
            │            │
   Timeout  │            │ Finality reached
            │            │
            │            ▼
            │       ┌─────────────┐
            │       │ confirming  │
            │       └────┬────────┘
            │            │
            │            │ Amount matches
            │            │
            ▼            ▼
       ┌─────────┐  ┌───────────┐
       │ expired │  │ completed │ (terminal)
       └─────────┘  └───────────┘
                         ▲
                         │
            Amount       │
            mismatch     │
                         │
                    ┌────┴────┐
                    │ failed  │ (terminal)
                    └─────────┘
```

### Merchant Status State Machine

```
     ┌──────────┐
     │ inactive │
     └────┬─────┘
          │
          │ Registration complete
          │
          ▼
    ┌─────────────┐
    │ kyc_pending │
    └──────┬──────┘
           │
           ├─── KYC rejected ───► ┌──────────────┐
           │                      │ kyc_rejected │
           │                      └──────────────┘
           │
           │ KYC approved
           │
           ▼
      ┌────────┐
      │ active │ ◄──────────────┐
      └───┬────┘                │
          │                     │
          │ Violation           │ Review complete
          │                     │
          ▼                     │
     ┌──────────┐               │
     │suspended │───────────────┘
     └──────────┘
```

---

## Database Transaction Patterns

### Pattern 1: Payment Confirmation (ACID Critical)

```sql
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;

-- Lock payment record
SELECT * FROM payments
WHERE id = ?
FOR UPDATE;

-- Verify pre-conditions
IF payment.status != 'pending' THEN
  ROLLBACK;
  RAISE 'Invalid payment status';
END IF;

-- Update payment
UPDATE payments
SET status = 'completed', confirmed_at = NOW()
WHERE id = ?;

-- Create ledger entries (atomic)
INSERT INTO ledger_entries
  (debit_account, credit_account, amount, ...)
VALUES
  ('external:crypto', 'merchant:{id}', 2300000, ...);

-- Update balance (atomic increment)
UPDATE merchant_balances
SET available_vnd = available_vnd + 2300000
WHERE merchant_id = ?;

-- Verify ledger balance (invariant check)
SELECT verify_ledger_balance('merchant:{id}');

COMMIT;
```

### Pattern 2: Payout Request (Balance Reservation)

```sql
BEGIN TRANSACTION;

-- Lock balance record
SELECT available_vnd
FROM merchant_balances
WHERE merchant_id = ?
FOR UPDATE;

-- Check sufficient balance
IF available_vnd < payout_amount THEN
  ROLLBACK;
  RAISE 'Insufficient balance';
END IF;

-- Create payout record
INSERT INTO payouts (...) VALUES (...);

-- Reserve balance (atomic operation)
UPDATE merchant_balances
SET
  available_vnd = available_vnd - ?,
  pending_vnd = pending_vnd + ?
WHERE merchant_id = ?
  AND available_vnd >= ?; -- Double-check in WHERE

-- Verify affected rows
IF row_count = 0 THEN
  ROLLBACK;
  RAISE 'Concurrent modification detected';
END IF;

COMMIT;
```

### Pattern 3: Idempotent Payment Processing

```sql
-- Idempotency key: tx_hash
-- Prevents duplicate processing

BEGIN TRANSACTION;

-- Check if already processed
SELECT id FROM blockchain_transactions
WHERE tx_hash = ?;

IF found THEN
  COMMIT; -- Already processed, safe to return
  RETURN 'already_processed';
END IF;

-- Process transaction
INSERT INTO blockchain_transactions (...);
UPDATE payments SET status = 'completed' ...;
-- ... rest of confirmation logic

COMMIT;
RETURN 'processed';
```

---

## Key Takeaways for New Engineers

### Critical Success Factors

1. **Money Calculations**: ALWAYS use `decimal.Decimal`, NEVER `float64`
2. **Blockchain Finality**: Wait for full finality before confirming payments
3. **Idempotency**: All operations must be idempotent (safe to retry)
4. **Ledger Integrity**: Every VND movement must have double-entry ledger record
5. **Audit Trail**: Log all critical operations in audit_logs

### Common Pitfalls to Avoid

```
❌ Don't: Update balance with SELECT + UPDATE
✓ Do: Use atomic UPDATE balance = balance + ?

❌ Don't: Use float for money calculations
✓ Do: Use decimal.Decimal everywhere

❌ Don't: Confirm payment before finality
✓ Do: Wait for 'finalized' (Solana) or 15 confirmations (BSC)

❌ Don't: Process transactions without checking duplicates
✓ Do: Check tx_hash existence first

❌ Don't: Update payment status without WHERE status = 'expected'
✓ Do: Always include status in WHERE clause (optimistic locking)
```

### Testing Checklist

```
Before deploying any flow:

□ Test on testnet first (Solana devnet, BSC testnet)
□ Test with small amounts
□ Test timeout/expiry scenarios
□ Test concurrent requests (race conditions)
□ Test network failures (retry logic)
□ Test amount mismatches
□ Test missing memos
□ Verify ledger balance reconciliation
□ Check audit logs are created
□ Verify webhooks are delivered
□ Test rollback scenarios
```

---

## Next Steps

For implementation:
1. Start with **Merchant Onboarding Flow** (foundational)
2. Then **Payment Creation Flow** (core value)
3. Then **Payment Confirmation Flow** (most complex)
4. Then **Payout Flow** (completes the cycle)
5. Finally **OTC Settlement Flow** (operational)

Refer to `MVP_ROADMAP.md` for detailed week-by-week implementation plan.

---

**Document Version**: 1.0
**Last Updated**: 2025-11-18
**Maintained By**: Engineering Team
**Questions**: Reference this doc first, then consult team lead
