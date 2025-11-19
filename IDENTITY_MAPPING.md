# Smart Identity Mapping System

**Project**: Stablecoin Payment Gateway - Identity Management Module
**Last Updated**: 2025-11-19
**Status**: Design Phase (PRD v2.2)

---

## 🎯 Overview

The **Smart Identity Mapping System** creates a persistent link between wallet addresses and user identities, enabling **one-time KYC with lifetime recognition**.

### Problem Statement

Traditional payment systems require KYC for every transaction or merchant. In crypto, wallets are pseudonymous - we need to map wallet addresses to real identities **once**, then recognize returning users instantly.

### Solution

```
First-time User:  Wallet Address → KYC Required → Create Mapping → Allow Payment
Returning User:   Wallet Address → Found in DB → Skip KYC → Allow Payment
```

### Benefits

- ✅ **Frictionless UX**: Returning users skip KYC (payment in <30 seconds)
- ✅ **Compliance Maintained**: KYC still done, just once
- ✅ **Fast Recognition**: Redis cache (<10ms lookup)
- ✅ **Multi-merchant**: One KYC works across all merchants
- ✅ **Multi-wallet**: Users can link multiple wallets to one identity

---

## 🏗️ Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                 IDENTITY MAPPING SYSTEM                      │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐       ┌─────────────────────────┐    │
│  │  Wallet          │       │  KYC Session            │    │
│  │  Detection       │◄──────┤  Manager                │    │
│  │                  │       │                         │    │
│  │  - Parse address │       │  - Initiate session     │    │
│  │  - Check cache   │       │  - Track progress       │    │
│  │  - Query DB      │       │  - Store documents      │    │
│  └────────┬─────────┘       └───────────┬─────────────┘    │
│           │                             │                   │
│           ▼                             ▼                   │
│  ┌──────────────────┐       ┌─────────────────────────┐    │
│  │  Redis Cache     │       │  KYC Provider           │    │
│  │                  │       │  Integration            │    │
│  │  wallet:chain:   │       │                         │    │
│  │  addr → user_id  │       │  - Sumsub API           │    │
│  │                  │       │  - Onfido API           │    │
│  │  TTL: 7 days     │       │  - Face liveness        │    │
│  └──────────────────┘       │  - ID verification      │    │
│                              └───────────┬─────────────┘    │
│                                          │                   │
│  ┌──────────────────────────────────────▼─────────────┐    │
│  │         Identity Mapping Database                   │    │
│  │                                                      │    │
│  │  wallet_identity_mappings:                          │    │
│  │    - wallet_address → user_id                       │    │
│  │    - encrypted identity info                        │    │
│  │    - KYC verification status                        │    │
│  │                                                      │    │
│  │  users:                                             │    │
│  │    - full_name, dob, nationality                   │    │
│  │    - encrypted PII                                  │    │
│  └──────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## 🗄️ Database Schema

### Table: `users`

Stores encrypted user identity information (PII).

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identity (encrypted at rest)
    full_name VARCHAR(255) NOT NULL,
    date_of_birth DATE,
    nationality VARCHAR(10), -- ISO 3166-1 alpha-2 code
    gender VARCHAR(10),

    -- Contact
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50),

    -- Identity documents
    id_type VARCHAR(20), -- 'passport', 'national_id', 'drivers_license'
    id_number_encrypted TEXT, -- PGP encrypted
    id_country VARCHAR(10),
    id_expiry_date DATE,

    -- Risk profile
    risk_level VARCHAR(20) DEFAULT 'low', -- low, medium, high
    pep_status BOOLEAN DEFAULT FALSE, -- Politically Exposed Person

    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_risk ON users(risk_level);
```

### Table: `wallet_identity_mappings`

Links wallet addresses to user identities.

```sql
CREATE TABLE wallet_identity_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Wallet info
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL, -- 'solana', 'bsc', 'tron', 'polygon', 'ethereum'

    -- User reference
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- KYC status
    kyc_status VARCHAR(20) DEFAULT 'verified', -- pending, verified, rejected, expired
    kyc_verified_at TIMESTAMP,
    kyc_provider VARCHAR(50), -- 'sumsub', 'onfido', 'manual'
    kyc_provider_session_id VARCHAR(255),

    -- Usage stats
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    payment_count INTEGER DEFAULT 0,
    total_volume_usd DECIMAL(20, 2) DEFAULT 0,

    -- Metadata
    source VARCHAR(50), -- 'payment', 'registration'
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(wallet_address, blockchain)
);

CREATE INDEX idx_wallet_mapping_address ON wallet_identity_mappings(wallet_address, blockchain);
CREATE INDEX idx_wallet_mapping_user ON wallet_identity_mappings(user_id);
CREATE INDEX idx_wallet_mapping_status ON wallet_identity_mappings(kyc_status);
CREATE INDEX idx_wallet_mapping_last_used ON wallet_identity_mappings(last_used_at);
```

### Table: `kyc_sessions`

Tracks KYC verification sessions and their progress.

```sql
CREATE TABLE kyc_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Target wallet
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL,

    -- Session status
    status VARCHAR(20) DEFAULT 'initiated',
    -- Possible values:
    --   'initiated': Session created, waiting for user input
    --   'documents_uploaded': User uploaded ID/selfie
    --   'verifying': Provider is verifying documents
    --   'completed': KYC approved, mapping created
    --   'rejected': KYC failed verification
    --   'expired': Session timed out (30 minutes)

    -- User input
    full_name VARCHAR(255),
    date_of_birth DATE,
    nationality VARCHAR(10),
    id_type VARCHAR(20),
    id_number VARCHAR(100),

    -- Document uploads (S3 URLs)
    id_front_url VARCHAR(500),
    id_back_url VARCHAR(500),
    selfie_url VARCHAR(500),
    liveness_video_url VARCHAR(500),

    -- Verification scores (0.00-1.00)
    liveness_score DECIMAL(3, 2),
    face_match_score DECIMAL(3, 2), -- Selfie vs ID photo
    id_verification_score DECIMAL(3, 2), -- OCR + database check
    overall_score DECIMAL(3, 2),

    -- Provider integration
    provider VARCHAR(50), -- 'sumsub', 'onfido', 'manual'
    provider_session_id VARCHAR(255),
    provider_applicant_id VARCHAR(255),
    provider_response JSONB, -- Full API response

    -- Rejection reason
    rejection_reason VARCHAR(255),
    rejection_details TEXT,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    documents_uploaded_at TIMESTAMP,
    verification_started_at TIMESTAMP,
    completed_at TIMESTAMP,
    expires_at TIMESTAMP, -- Created_at + 30 minutes

    UNIQUE(wallet_address, blockchain, status)
);

CREATE INDEX idx_kyc_session_wallet ON kyc_sessions(wallet_address, blockchain);
CREATE INDEX idx_kyc_session_status ON kyc_sessions(status);
CREATE INDEX idx_kyc_session_expires ON kyc_sessions(expires_at);
CREATE INDEX idx_kyc_session_provider ON kyc_sessions(provider, provider_session_id);
```

---

## 🔄 User Flows

### Flow 1: First-Time Payer (KYC Required)

```
┌────────────────────────────────────────────────────────────┐
│ 1. USER INITIATES PAYMENT                                  │
└────────────────────────────────────────────────────────────┘
   User scans QR code or clicks payment link
   Payment page loads in browser

┌────────────────────────────────────────────────────────────┐
│ 2. WALLET DETECTION                                         │
└────────────────────────────────────────────────────────────┘
   Frontend: Prompt user to connect wallet
   Options:
     - Phantom (Solana)
     - MetaMask (BSC/Polygon/Ethereum)
     - TronLink (TRON)

   User connects wallet → Get wallet address
   Example: "8xK7zY9Q2..." (Solana)

┌────────────────────────────────────────────────────────────┐
│ 3. CHECK IDENTITY MAPPING                                   │
└────────────────────────────────────────────────────────────┘
   API Call: GET /api/v1/wallet/solana/8xK7zY9Q2.../kyc-status

   Backend:
     1. Check Redis cache: wallet:solana:8xK7zY9Q2...
     2. If cache MISS → Query database
     3. Result: NOT FOUND (new wallet)

┌────────────────────────────────────────────────────────────┐
│ 4. SHOW KYC MODAL                                          │
└────────────────────────────────────────────────────────────┘
   Frontend displays:
     "Welcome! To comply with regulations, we need to verify your identity.
      This is a one-time process - next time you pay, we'll recognize you instantly!"

   Form fields:
     - Full Name: [____________]
     - Date of Birth: [____/____/____]
     - Nationality: [Dropdown]
     - ID Type: [Passport | National ID | Driver's License]
     - ID Number: [____________]

   [Continue to Upload Documents]

┌────────────────────────────────────────────────────────────┐
│ 5. INITIATE KYC SESSION                                    │
└────────────────────────────────────────────────────────────┘
   API Call: POST /api/v1/wallet/kyc/initiate
   Body: {
     wallet_address: "8xK7zY9Q2...",
     blockchain: "solana",
     full_name: "John Doe",
     date_of_birth: "1990-01-15",
     nationality: "US",
     id_type: "passport",
     id_number: "AB1234567"
   }

   Backend:
     1. Create kyc_sessions record (status: 'initiated')
     2. Generate session_id
     3. Set expires_at = NOW() + 30 minutes
     4. Return session_id to frontend

   Response: {
     session_id: "550e8400-...",
     expires_at: "2025-11-19T10:30:00Z",
     upload_url: "/api/v1/wallet/kyc/upload"
   }

┌────────────────────────────────────────────────────────────┐
│ 6. UPLOAD DOCUMENTS                                         │
└────────────────────────────────────────────────────────────┘
   Frontend:
     "Please upload photos of your ID and a selfie"

   User uploads:
     - ID Front: [Upload]
     - ID Back (optional): [Upload]
     - Selfie: [Upload]

   API Call: POST /api/v1/wallet/kyc/upload
   Content-Type: multipart/form-data
   Files:
     - session_id: "550e8400-..."
     - id_front: <file>
     - id_back: <file>
     - selfie: <file>

   Backend:
     1. Validate session exists and not expired
     2. Upload files to S3 (private bucket)
     3. Update kyc_sessions:
        - id_front_url = "s3://kyc-docs/550e8400.../id_front.jpg"
        - status = 'documents_uploaded'
     4. Trigger async verification job

┌────────────────────────────────────────────────────────────┐
│ 7. FACE LIVENESS CHECK                                     │
└────────────────────────────────────────────────────────────┘
   Frontend:
     "Now let's verify you're a real person"
     - Show video recording instructions
     - "Please blink twice and turn your head left, then right"

   User records 5-second video

   API Call: POST /api/v1/wallet/kyc/liveness
   Body: {
     session_id: "550e8400-...",
     video: "base64_encoded_video_data"
   }

   Backend:
     1. Upload video to S3
     2. Call liveness detection API (Sumsub or Onfido)
     3. Get liveness_score (0.00-1.00)
     4. Compare video face vs ID photo face
     5. Get face_match_score (0.00-1.00)

┌────────────────────────────────────────────────────────────┐
│ 8. VERIFICATION (Async)                                    │
└────────────────────────────────────────────────────────────┘
   Background Worker:
     1. Call KYC provider API (Sumsub/Onfido)
        - Send: ID images, selfie, liveness video
     2. Provider performs:
        - OCR on ID document
        - Validate ID authenticity (security features, fonts)
        - Check against government databases (if available)
        - Face matching (selfie vs ID photo)
        - Liveness detection
     3. Receive verification result:
        {
          status: "approved",
          liveness_score: 0.92,
          face_match_score: 0.89,
          id_verification_score: 0.95,
          overall_score: 0.92
        }

   If scores > threshold (0.85):
     - Proceed to next step
   Else:
     - Update kyc_sessions.status = 'rejected'
     - Notify user

┌────────────────────────────────────────────────────────────┐
│ 9. CREATE IDENTITY MAPPING                                 │
└────────────────────────────────────────────────────────────┘
   Backend (after KYC approved):
     1. Create user record:
        INSERT INTO users (full_name, dob, nationality, ...)
        VALUES ('John Doe', '1990-01-15', 'US', ...)
        RETURNING id

     2. Create wallet mapping:
        INSERT INTO wallet_identity_mappings
        (wallet_address, blockchain, user_id, kyc_status, ...)
        VALUES ('8xK7zY9Q2...', 'solana', <user_id>, 'verified', ...)

     3. Cache in Redis:
        SET wallet:solana:8xK7zY9Q2...
        VALUE {user_id: "<uuid>", full_name: "John Doe", kyc_status: "verified"}
        EX 604800  -- 7 days TTL

     4. Update kyc_sessions:
        status = 'completed'
        completed_at = NOW()

┌────────────────────────────────────────────────────────────┐
│ 10. ALLOW PAYMENT                                          │
└────────────────────────────────────────────────────────────┘
   Frontend:
     - Hide KYC modal
     - Show success message: "Identity verified! ✓"
     - Display payment info: "Send 10.50 USDT to 8xK7..."

   User completes payment via wallet app

   Total Time: ~3 minutes (with KYC)
```

### Flow 2: Returning Payer (KYC Skipped)

```
┌────────────────────────────────────────────────────────────┐
│ 1. USER INITIATES PAYMENT                                  │
└────────────────────────────────────────────────────────────┘
   User scans QR code (same wallet as before)

┌────────────────────────────────────────────────────────────┐
│ 2. WALLET DETECTION                                         │
└────────────────────────────────────────────────────────────┘
   User connects wallet: "8xK7zY9Q2..." (Solana)

┌────────────────────────────────────────────────────────────┐
│ 3. CHECK IDENTITY MAPPING (Cache Hit!)                     │
└────────────────────────────────────────────────────────────┘
   API Call: GET /api/v1/wallet/solana/8xK7zY9Q2.../kyc-status

   Backend:
     1. Check Redis cache: wallet:solana:8xK7zY9Q2...
     2. Cache HIT! (< 10ms)
     3. Result: {
          has_kyc: true,
          user: {
            id: "uuid",
            full_name: "John Doe",
            kyc_status: "verified"
          }
        }

┌────────────────────────────────────────────────────────────┐
│ 4. DISPLAY WELCOME MESSAGE                                 │
└────────────────────────────────────────────────────────────┘
   Frontend shows:
     "Welcome back, John Doe! ✓"
     "Your identity is verified. Proceeding to payment..."

   No KYC modal displayed!

┌────────────────────────────────────────────────────────────┐
│ 5. SHOW PAYMENT INFO                                       │
└────────────────────────────────────────────────────────────┘
   Display: "Send 10.50 USDT to 8xK7..."

   User completes payment

   Total Time: < 30 seconds (no KYC!)
```

### Flow 3: Link Additional Wallet (Same User)

```
User has wallet A (already KYC'd), wants to add wallet B

1. User connects wallet B
2. System detects: No KYC for wallet B
3. Show: "Is this wallet also yours? Link to existing identity?"
4. User clicks "Yes, link to my identity"
5. Backend:
   - Create new wallet_identity_mappings record
   - Same user_id as wallet A
   - No new KYC required (already verified)
6. Wallet B now recognized instantly
```

---

## 🔌 API Endpoints

### 1. Check Wallet KYC Status

Check if a wallet address has completed KYC.

**Endpoint**: `GET /api/v1/wallet/:blockchain/:address/kyc-status`

**Parameters**:
- `blockchain`: Chain name (`solana`, `bsc`, `tron`, `polygon`, `ethereum`)
- `address`: Wallet address (Base58 or hex)

**Response** (200 OK):
```json
{
  "has_kyc": true,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "John Doe",
    "kyc_status": "verified",
    "kyc_verified_at": "2025-11-01T10:30:00Z"
  },
  "wallet": {
    "address": "8xK7zY9Q2...",
    "blockchain": "solana",
    "first_seen_at": "2025-11-01T10:25:00Z",
    "last_used_at": "2025-11-19T15:30:00Z"
  },
  "stats": {
    "payment_count": 12,
    "total_volume_usd": 1250.50
  }
}
```

**Response** (404 Not Found - No KYC):
```json
{
  "has_kyc": false,
  "message": "This wallet has not completed KYC. Please verify your identity to continue.",
  "kyc_url": "/api/v1/wallet/kyc/initiate"
}
```

### 2. Initiate KYC Session

Start a new KYC verification session for a wallet.

**Endpoint**: `POST /api/v1/wallet/kyc/initiate`

**Request Body**:
```json
{
  "wallet_address": "8xK7zY9Q2pUjxVRd3MAGkWv4bXnRNMzw5xz...",
  "blockchain": "solana",
  "full_name": "John Doe",
  "date_of_birth": "1990-01-15",
  "nationality": "US",
  "id_type": "passport",
  "id_number": "AB1234567"
}
```

**Response** (201 Created):
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "initiated",
  "expires_at": "2025-11-19T10:30:00Z",
  "next_step": "upload_documents",
  "upload_url": "/api/v1/wallet/kyc/upload"
}
```

### 3. Upload KYC Documents

Upload ID photos and selfie for verification.

**Endpoint**: `POST /api/v1/wallet/kyc/upload`

**Content-Type**: `multipart/form-data`

**Form Fields**:
- `session_id`: UUID of KYC session
- `id_front`: File (image, max 10MB)
- `id_back`: File (image, optional, max 10MB)
- `selfie`: File (image, max 10MB)

**Response** (200 OK):
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "documents_uploaded",
  "next_step": "liveness_check",
  "liveness_url": "/api/v1/wallet/kyc/liveness"
}
```

### 4. Perform Face Liveness Check

Submit video for liveness detection and face matching.

**Endpoint**: `POST /api/v1/wallet/kyc/liveness`

**Request Body**:
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "video": "base64_encoded_video_data"
}
```

**Response** (202 Accepted):
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "verifying",
  "message": "Your identity is being verified. This may take 1-2 minutes.",
  "polling_url": "/api/v1/wallet/kyc/session/550e8400-.../status"
}
```

### 5. Get KYC Session Status

Poll for verification results.

**Endpoint**: `GET /api/v1/wallet/kyc/session/:session_id/status`

**Response** (200 OK - Pending):
```json
{
  "session_id": "550e8400-...",
  "status": "verifying",
  "message": "Verification in progress. Please wait..."
}
```

**Response** (200 OK - Approved):
```json
{
  "session_id": "550e8400-...",
  "status": "completed",
  "result": "approved",
  "user_id": "7c9e6679-...",
  "scores": {
    "liveness": 0.92,
    "face_match": 0.89,
    "id_verification": 0.95,
    "overall": 0.92
  },
  "completed_at": "2025-11-19T10:28:30Z"
}
```

**Response** (200 OK - Rejected):
```json
{
  "session_id": "550e8400-...",
  "status": "rejected",
  "result": "rejected",
  "reason": "Face match score too low",
  "details": "The selfie photo did not match the ID photo. Please ensure good lighting and clear image.",
  "retry_allowed": true,
  "retry_url": "/api/v1/wallet/kyc/retry"
}
```

### 6. Link Additional Wallet

Link a new wallet to an existing verified user.

**Endpoint**: `POST /api/v1/wallet/link`

**Authentication**: Requires user JWT token

**Request Body**:
```json
{
  "wallet_address": "9yH8kL3Q...",
  "blockchain": "bsc",
  "signature": "0x..."  // Signed message to prove wallet ownership
}
```

**Response** (201 Created):
```json
{
  "message": "Wallet linked successfully",
  "wallet": {
    "address": "9yH8kL3Q...",
    "blockchain": "bsc",
    "user_id": "7c9e6679-..."
  }
}
```

---

## 🔐 Redis Caching Strategy

### Cache Key Format

```
wallet:{blockchain}:{address}
```

Examples:
- `wallet:solana:8xK7zY9Q2pUjxVRd3MAGkWv4bXnRNMzw5xz...`
- `wallet:bsc:0xAb1234567890abcdef...`
- `wallet:tron:TYs7zK2pLm9N3hQ...`

### Cache Value (JSON)

```json
{
  "user_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "full_name": "John Doe",
  "kyc_status": "verified",
  "kyc_verified_at": "2025-11-01T10:30:00Z",
  "last_used_at": "2025-11-19T15:30:00Z",
  "payment_count": 12,
  "risk_level": "low"
}
```

### Cache TTL

**7 days (604800 seconds)**

Rationale:
- Long enough to avoid repeated DB queries for active users
- Short enough to refresh risk level changes
- Balance between performance and data freshness

### Cache Warming

On application start, pre-load cache for:
- Top 1000 most active wallets (by payment_count)
- Wallets used in last 24 hours

```typescript
async function warmCache() {
  const activeWallets = await db.wallet_identity_mappings.findMany({
    where: {
      OR: [
        { payment_count: { gte: 10 } },
        { last_used_at: { gte: new Date(Date.now() - 24 * 3600 * 1000) } }
      ]
    },
    include: { user: true },
    take: 1000
  })

  for (const wallet of activeWallets) {
    const cacheKey = `wallet:${wallet.blockchain}:${wallet.wallet_address}`
    await redis.setex(cacheKey, 604800, JSON.stringify({
      user_id: wallet.user_id,
      full_name: wallet.user.full_name,
      kyc_status: wallet.kyc_status,
      // ... other fields
    }))
  }

  console.log(`Warmed cache with ${activeWallets.length} wallets`)
}
```

### Cache Invalidation

Invalidate cache when:
- KYC status changes (e.g., expired, rejected)
- User updates personal information
- Risk level changes
- User requests data deletion (GDPR)

```typescript
async function invalidateWalletCache(walletAddress: string, blockchain: string) {
  const cacheKey = `wallet:${blockchain}:${walletAddress}`
  await redis.del(cacheKey)
}
```

---

## 🔒 Security & Privacy

### PII Encryption

All Personally Identifiable Information (PII) must be encrypted at rest.

**Encryption Method**: AES-256-GCM

**Fields to Encrypt**:
- `users.id_number_encrypted`
- `users.email` (optional, for highly sensitive deployments)
- `users.phone`
- KYC document URLs (stored in private S3 bucket)

**Example** (Node.js):
```typescript
import crypto from 'crypto'

const ENCRYPTION_KEY = process.env.ENCRYPTION_KEY // 32 bytes
const ALGORITHM = 'aes-256-gcm'

function encrypt(text: string): string {
  const iv = crypto.randomBytes(16)
  const cipher = crypto.createCipheriv(ALGORITHM, Buffer.from(ENCRYPTION_KEY, 'hex'), iv)

  let encrypted = cipher.update(text, 'utf8', 'hex')
  encrypted += cipher.final('hex')

  const authTag = cipher.getAuthTag()

  return iv.toString('hex') + ':' + authTag.toString('hex') + ':' + encrypted
}

function decrypt(encrypted: string): string {
  const parts = encrypted.split(':')
  const iv = Buffer.from(parts[0], 'hex')
  const authTag = Buffer.from(parts[1], 'hex')
  const encryptedText = parts[2]

  const decipher = crypto.createDecipheriv(ALGORITHM, Buffer.from(ENCRYPTION_KEY, 'hex'), iv)
  decipher.setAuthTag(authTag)

  let decrypted = decipher.update(encryptedText, 'hex', 'utf8')
  decrypted += decipher.final('utf8')

  return decrypted
}
```

### Access Control

Who can access identity data:

| Role | Read User Info | Read ID Numbers | Update KYC | Delete User |
|------|----------------|-----------------|------------|-------------|
| **System** | ✅ (for payment flow) | ❌ | ❌ | ❌ |
| **Compliance Officer** | ✅ | ✅ (with audit log) | ✅ | ❌ |
| **Admin** | ✅ | ❌ | ❌ | ❌ |
| **Merchant** | ❌ | ❌ | ❌ | ❌ |
| **User (self)** | ✅ | ✅ | ✅ (re-verify) | ✅ (GDPR) |

### Audit Logging

Log all access to sensitive identity data:

```typescript
await auditLog.create({
  actor_type: 'compliance_officer',
  actor_id: 'admin_user_uuid',
  action: 'view_user_identity',
  resource_type: 'user',
  resource_id: 'user_uuid',
  metadata: {
    reason: 'Regulatory inquiry #12345',
    fields_accessed: ['full_name', 'id_number', 'kyc_documents']
  },
  ip_address: '192.168.1.100'
})
```

### Data Retention

**GDPR Right to be Forgotten**:

When user requests deletion:
1. Check if user has pending payments/payouts → Cannot delete
2. If clear:
   - Anonymize PII (replace with "DELETED_USER_...")
   - Keep wallet mappings but remove link to user
   - Delete KYC documents from S3
   - Add tombstone record: `deleted_users` table

---

## 📊 Monitoring & Analytics

### KPIs to Track

| Metric | Target | Measurement |
|--------|--------|-------------|
| **KYC Completion Rate** | > 90% | Completed / Initiated |
| **KYC Approval Rate** | > 85% | Approved / Completed |
| **Returning User Recognition** | > 95% | Cache hits / Total checks |
| **Average KYC Time** | < 3 minutes | Time from initiate to approved |
| **Cache Hit Rate** | > 90% | Redis hits / Total wallet checks |
| **False Rejection Rate** | < 5% | User disputes / Total rejections |

### Dashboard Queries

**Daily KYC Stats**:
```sql
SELECT
    DATE(created_at) as date,
    COUNT(*) as total_sessions,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
    COUNT(CASE WHEN status = 'rejected' THEN 1 END) as rejected,
    AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 60) as avg_completion_minutes
FROM kyc_sessions
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date DESC;
```

**Cache Performance**:
```typescript
const stats = await redis.info('stats')
// Parse: keyspace_hits, keyspace_misses
const hitRate = keyspace_hits / (keyspace_hits + keyspace_misses) * 100
console.log(`Cache hit rate: ${hitRate.toFixed(2)}%`)
```

**Top Returning Users** (by payment count):
```sql
SELECT
    u.full_name,
    wim.wallet_address,
    wim.blockchain,
    wim.payment_count,
    wim.total_volume_usd,
    wim.last_used_at
FROM wallet_identity_mappings wim
JOIN users u ON wim.user_id = u.id
WHERE wim.payment_count > 5
ORDER BY wim.payment_count DESC
LIMIT 100;
```

---

## 🚀 Implementation Guide

### Phase 1: Database Setup (Day 1-2)

```bash
# Create migration
npx prisma migrate dev --name add_identity_mapping

# Or raw SQL
psql -U postgres -d payment_gateway -f migrations/001_identity_mapping.sql
```

### Phase 2: KYC Provider Integration (Day 3-5)

**Option A: Sumsub** (Recommended for MVP)

```typescript
import axios from 'axios'

class SumsubKYCProvider {
  private apiUrl = 'https://api.sumsub.com'
  private appToken = process.env.SUMSUB_APP_TOKEN
  private secretKey = process.env.SUMSUB_SECRET_KEY

  async createApplicant(userData: {
    firstName: string
    lastName: string
    dob: string
  }) {
    const response = await axios.post(
      `${this.apiUrl}/resources/applicants`,
      {
        externalUserId: userData.walletAddress,
        fixedInfo: {
          firstName: userData.firstName,
          lastName: userData.lastName,
          dob: userData.dob
        }
      },
      { headers: this.getHeaders() }
    )

    return response.data.id // applicantId
  }

  async uploadDocument(applicantId: string, file: Buffer, docType: 'ID_FRONT' | 'ID_BACK' | 'SELFIE') {
    const formData = new FormData()
    formData.append('content', file, 'document.jpg')

    await axios.post(
      `${this.apiUrl}/resources/applicants/${applicantId}/info/idDoc`,
      formData,
      {
        headers: {
          ...this.getHeaders(),
          'Content-Type': 'multipart/form-data'
        },
        params: { idDocType: docType }
      }
    )
  }

  async getApplicantStatus(applicantId: string) {
    const response = await axios.get(
      `${this.apiUrl}/resources/applicants/${applicantId}/status`,
      { headers: this.getHeaders() }
    )

    return response.data
  }

  private getHeaders() {
    return {
      'X-App-Token': this.appToken,
      'X-App-Access-Sig': this.generateSignature(),
      'X-App-Access-Ts': Math.floor(Date.now() / 1000).toString()
    }
  }

  private generateSignature() {
    // HMAC signature logic (see Sumsub docs)
    // ...
  }
}
```

### Phase 3: API Implementation (Day 6-8)

```typescript
// identity-mapping.service.ts
export class IdentityMappingService {
  async checkWalletKYC(walletAddress: string, blockchain: string) {
    // 1. Check Redis cache
    const cacheKey = `wallet:${blockchain}:${walletAddress}`
    const cached = await redis.get(cacheKey)

    if (cached) {
      return { has_kyc: true, user: JSON.parse(cached) }
    }

    // 2. Query database
    const mapping = await db.wallet_identity_mappings.findUnique({
      where: {
        wallet_address_blockchain: { wallet_address: walletAddress, blockchain }
      },
      include: { user: true }
    })

    if (!mapping) {
      return { has_kyc: false }
    }

    // 3. Cache result
    await redis.setex(cacheKey, 604800, JSON.stringify({
      user_id: mapping.user_id,
      full_name: mapping.user.full_name,
      kyc_status: mapping.kyc_status
    }))

    return { has_kyc: true, user: mapping.user }
  }

  async initiateKYC(data: InitiateKYCRequest) {
    // Create KYC session
    const session = await db.kyc_sessions.create({
      data: {
        wallet_address: data.wallet_address,
        blockchain: data.blockchain,
        full_name: data.full_name,
        date_of_birth: new Date(data.date_of_birth),
        nationality: data.nationality,
        id_type: data.id_type,
        id_number: data.id_number,
        status: 'initiated',
        expires_at: new Date(Date.now() + 30 * 60 * 1000) // 30 min
      }
    })

    return session
  }

  // ... other methods
}
```

### Phase 4: Frontend Integration (Day 9-10)

```typescript
// React component
import { useState } from 'react'

function PaymentPage() {
  const [walletAddress, setWalletAddress] = useState<string | null>(null)
  const [needsKYC, setNeedsKYC] = useState(false)
  const [userName, setUserName] = useState<string | null>(null)

  async function connectWallet() {
    // Connect wallet (Phantom for Solana)
    const resp = await window.solana.connect()
    const address = resp.publicKey.toString()
    setWalletAddress(address)

    // Check KYC status
    const kycStatus = await fetch(`/api/v1/wallet/solana/${address}/kyc-status`)
    const data = await kycStatus.json()

    if (data.has_kyc) {
      setUserName(data.user.full_name)
      setNeedsKYC(false)
      // Proceed to payment
    } else {
      setNeedsKYC(true)
      // Show KYC modal
    }
  }

  if (needsKYC) {
    return <KYCModal walletAddress={walletAddress} blockchain="solana" />
  }

  return (
    <div>
      <h1>Welcome back, {userName}!</h1>
      <button onClick={connectWallet}>Connect Wallet</button>
      {/* Payment info */}
    </div>
  )
}
```

---

## 📝 Testing Strategy

### Unit Tests

```typescript
describe('IdentityMappingService', () => {
  it('should return has_kyc=false for new wallet', async () => {
    const result = await service.checkWalletKYC('new_wallet_address', 'solana')
    expect(result.has_kyc).toBe(false)
  })

  it('should cache wallet→user mapping', async () => {
    await service.checkWalletKYC('8xK7zY9Q2...', 'solana')
    const cached = await redis.get('wallet:solana:8xK7zY9Q2...')
    expect(cached).toBeTruthy()
  })

  it('should invalidate cache on KYC status change', async () => {
    await service.updateKYCStatus('session_id', 'rejected')
    const cached = await redis.get('wallet:solana:8xK7zY9Q2...')
    expect(cached).toBeNull()
  })
})
```

### Integration Tests

```typescript
describe('KYC Flow E2E', () => {
  it('should complete full KYC flow', async () => {
    // 1. Initiate KYC
    const session = await request(app)
      .post('/api/v1/wallet/kyc/initiate')
      .send({
        wallet_address: 'test_wallet',
        blockchain: 'solana',
        full_name: 'Test User',
        // ... other fields
      })
    expect(session.status).toBe(201)

    // 2. Upload documents
    const upload = await request(app)
      .post('/api/v1/wallet/kyc/upload')
      .attach('id_front', './test/fixtures/id_front.jpg')
      .attach('selfie', './test/fixtures/selfie.jpg')
      .field('session_id', session.body.session_id)
    expect(upload.status).toBe(200)

    // 3. Check wallet KYC (should be approved after async verification)
    // Note: In test, mock KYC provider response
    await delay(2000) // Wait for async verification
    const kycStatus = await request(app)
      .get('/api/v1/wallet/solana/test_wallet/kyc-status')
    expect(kycStatus.body.has_kyc).toBe(true)
  })
})
```

---

## ✅ Success Criteria

Identity Mapping system is successful if:

- [x] **Recognition Rate > 95%**: Returning users recognized from cache
- [x] **KYC Completion < 3 min**: Average time from initiate to approved
- [x] **Cache Hit Rate > 90%**: Avoid repeated DB queries
- [x] **Zero PII Leaks**: All sensitive data encrypted at rest
- [x] **GDPR Compliant**: Users can request data deletion

---

**Document Status**: Design Phase
**Next Review**: 2025-12-01
**Owner**: Backend Team
**Last Updated**: 2025-11-19
