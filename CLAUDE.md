# CLAUDE.md - AI Assistant Guide

**Project**: Stablecoin Payment Gateway - PRD v2.2
**Last Updated**: 2025-11-19
**Status**: Documentation Phase (Pre-Implementation)

---

## 🎯 Project Overview

This is a **stablecoin payment gateway** designed for Vietnam's tourism market (starting in Da Nang). The core value proposition is:

> **"Merchant creates QR → User scans → Crypto payment → OTC converts → Merchant receives VND"**

### Business Context
- **Market Opportunity**: Tether + Da Nang partnership (Nov 2025) creates regulatory sandbox
- **Target Market**: Tourism merchants (hotels, restaurants, tourist services) in Da Nang
- **Revenue Model**: 1% transaction fees + payout fees + OTC spread
- **PRD v2.2 Timeline**: 8-10 weeks (phased implementation)
- **Target Metrics (Month 1)**: 5 pilot merchants, 1B+ VND volume, 99% uptime, KYC recognition >95%, Notification delivery >95%

### PRD v2.2 Key Features
- ✅ **Smart Identity Mapping**: One-time KYC, automatic wallet recognition
- ✅ **Omni-channel Notifications**: Speaker/TTS, Telegram, Zalo, Email, Webhook
- ✅ **Custodial Treasury**: Multi-sig cold wallet + auto-sweeping (every 6 hours)
- ✅ **Infinite Data Retention**: S3 Glacier archival + transaction hashing
- ✅ **Advanced Off-ramp**: On-demand, Scheduled, Threshold-based withdrawals

### Key Stakeholders
- **Merchants**: Accept crypto payments, receive VND settlements
- **End Users**: Tourists paying with USDT/USDC on Solana or BSC
- **Product Owner**: Managing business strategy and compliance
- **Ops Team**: Manual KYC review, payout approvals, OTC settlements

---

## 📁 Repository Structure

### Current State
This repository is currently in the **documentation/planning phase**. No code has been implemented yet. The repository contains comprehensive planning documents:

```
stable_payment_gateway/
├── README.md                      # Project overview and quick summary
├── PRD_v2.2.md                   # 🆕 Product Requirements Document v2.2
├── PRD_v2.2_ROADMAP.md           # 🆕 10-Week Implementation Roadmap (detailed)
├── ARCHITECTURE.md                # Technical architecture, system design, database schema
├── TECH_STACK_GOLANG.md          # Golang implementation guide (recommended stack)
├── REQUIREMENTS.md                # Functional/non-functional requirements (phased)
├── AML_ENGINE.md                 # In-house AML compliance engine
├── GETTING_STARTED.md            # Dev team onboarding and setup guide
├── STAKEHOLDER_ANALYSIS.md       # Business model and stakeholder perspectives
├── TOURISM_USE_CASES.md          # Specific use cases for Da Nang tourism
│
├── 🆕 PRD v2.2 - New Modules
│   ├── IDENTITY_MAPPING.md       # Smart Wallet→User Identity (one-time KYC)
│   ├── NOTIFICATION_CENTER.md    # Omni-channel notifications
│   ├── DATA_RETENTION.md         # Infinite storage with S3 Glacier
│   └── OFF_RAMP_STRATEGIES.md    # Flexible withdrawal modes
│
└── CLAUDE.md                     # This file - AI assistant guide
```

### Planned Structure (Post-Implementation)
Once implementation begins, the codebase will follow this structure:

```
stable_payment_gateway/
├── cmd/                          # Main applications
│   ├── api/                      # REST API server
│   ├── listener/                 # Blockchain listener service
│   ├── worker/                   # Background jobs (payouts, etc.)
│   └── admin/                    # Admin API
│
├── internal/                     # Private application code
│   ├── api/                      # HTTP handlers, middleware, routes
│   ├── blockchain/               # Multi-chain blockchain integration
│   │   ├── solana/              # Solana listener, wallet, transactions
│   │   ├── bsc/                 # BSC listener, wallet, ERC20
│   │   └── types/               # Common blockchain types
│   ├── service/                 # Business logic layer
│   ├── repository/              # Data access layer
│   ├── model/                   # Domain models
│   ├── config/                  # Configuration management
│   └── pkg/                     # Shared utilities
│
├── web/                         # Frontend applications
│   ├── dashboard/               # Merchant dashboard (Next.js)
│   ├── payment/                 # Public payment page
│   └── admin/                   # Admin panel
│
├── migrations/                  # Database migrations (SQL)
├── scripts/                     # Deployment and utility scripts
├── docker/                      # Docker configurations
└── docs/                        # Additional documentation
```

---

## 🏗️ Architecture Overview

### System Components

1. **API Gateway Layer**
   - Public Payment API (for payment creation/status)
   - Merchant API (balance, payouts, transactions)
   - Internal Admin API (KYC, payout approvals)
   - Authentication: API keys (merchants), JWT (admin)

2. **Application Layer**
   - Payment Service (create, validate, confirm payments)
   - Merchant Service (registration, KYC, balance management)
   - Payout Service (requests, approvals, execution)
   - Ledger Service (double-entry accounting)
   - Notification Service (webhooks, emails)
   - 🆕 **Identity Mapping Service** (wallet→user KYC recognition)
   - 🆕 **Notification Dispatcher** (multi-channel plugin architecture)
   - 🆕 **Data Archival Service** (S3 Glacier + transaction hashing)
   - 🆕 **Treasury Service** (custodial wallet sweeping)

3. **Blockchain Layer**
   - Multi-chain listeners (Solana + BSC initially)
   - Wallet Service (hot wallet management, transaction signing)
   - Transaction Validator (verify amount, parse memo)

4. **Data Layer**
   - PostgreSQL (merchants, payments, payouts, ledger, audit logs)
   - Redis (rate limiting, caching, session management)
   - File Storage (KYC documents - S3/MinIO)

### Multi-Chain Support

**Supported Chains (PRD v2.2)**:
- 🆕 **TRON (Priority HIGH)**: USDT (TRC20) - **Cheapest fees (~$1)**, massive adoption in Asia
- **Solana**: USDT, USDC (SPL) - **Fastest finality (~13s)**, low fees
- **BNB Chain (BSC)**: USDT, BUSD (BEP20) - Popular in SEA

**Chain Selection Philosophy**:
- **TRON for cost** (tourism sector needs low fees)
- Solana for speed and developer experience
- BSC for Asia-Pacific market penetration
- Ethereum support planned for Phase 2

---

## 💻 Tech Stack

### Backend (Golang - Recommended)

**Why Golang?**
- High performance for blockchain listeners
- Excellent concurrency (goroutines for multi-chain monitoring)
- Strong typing (critical for financial calculations)
- Single binary deployment
- Robust blockchain libraries

**Core Dependencies**:
```
- Go 1.21+
- Gin or Fiber (HTTP framework)
- GORM or sqlx (database ORM)
- PostgreSQL 15
- Redis 7
- github.com/gagliardetto/solana-go (Solana)
- github.com/ethereum/go-ethereum (BSC/Ethereum)
- 🆕 github.com/fbsobreira/gotron-sdk (TRON)
- github.com/shopspring/decimal (money calculations)
- github.com/hibiken/asynq (background jobs)
- 🆕 github.com/aws/aws-sdk-go (S3 Glacier archival)
- 🆕 Sumsub SDK (KYC + Face Liveness)
- 🆕 Telegram Bot API (notifications)
- 🆕 Zalo API (ZNS notifications)
```

**Alternative**: Node.js + TypeScript stack is documented in MVP_ROADMAP.md

### Frontend
- Next.js 14 (App Router)
- TypeScript
- TailwindCSS
- shadcn/ui components
- React Query (data fetching)

### Infrastructure
- Docker + Docker Compose
- PostgreSQL 15 (ACID compliance for ledger)
- Redis 7 (cache, rate limiting, job queues)
- NGINX (reverse proxy)

---

## 🗄️ Database Schema

### Core Tables

**merchants**
- Stores merchant information, KYC status, API keys, webhook configuration
- Key fields: `id`, `email`, `business_name`, `kyc_status`, `api_key`, `webhook_url`

**payments**
- Tracks all payment requests and their lifecycle
- Key fields: `id`, `merchant_id`, `amount_vnd`, `amount_crypto`, `tx_hash`, `status`
- States: `created` → `pending` → `confirming` → `completed` | `expired` | `failed`

**payouts**
- Records merchant withdrawal requests
- Key fields: `id`, `merchant_id`, `amount_vnd`, `bank_account_number`, `status`
- States: `requested` → `approved` → `processing` → `completed` | `rejected`

**ledger_entries**
- Double-entry accounting system
- Tracks all financial movements (payments, payouts, fees)
- Key fields: `debit_account`, `credit_account`, `amount`, `currency`, `reference_id`

**merchant_balances**
- Computed/materialized view of merchant balances
- Key fields: `available_vnd`, `pending_vnd`, `total_received_vnd`, `total_paid_out_vnd`

**audit_logs**
- Comprehensive audit trail for compliance
- Logs: payment actions, KYC approvals, payout approvals, admin actions
- Key fields: `actor_type`, `action`, `resource_type`, `resource_id`, `metadata`

**blockchain_transactions**
- Tracks blockchain transaction details
- Key fields: `chain`, `tx_hash`, `amount`, `confirmations`, `payment_id`, `status`

### 🆕 PRD v2.2 Tables

**wallet_identity_mappings**
- Maps wallet addresses to user identities for one-time KYC
- Key fields: `wallet_address`, `blockchain`, `user_id`, `kyc_status`, `payment_count`

**notification_logs**
- Tracks all notifications sent across all channels
- Key fields: `payment_id`, `channel`, `status`, `sent_at`, `delivered_at`, `error_message`

**payout_schedules**
- Merchant withdrawal configuration (scheduled + threshold-based)
- Key fields: `merchant_id`, `scheduled_enabled`, `scheduled_frequency`, `threshold_enabled`, `threshold_usdt`

**archived_records**
- Metadata for data archived to S3 Glacier
- Key fields: `original_id`, `table_name`, `archive_path`, `data_hash`, `archived_at`

**transaction_hashes**
- SHA-256 hash chain for transaction immutability
- Key fields: `table_name`, `record_id`, `data_hash`, `previous_hash`, `merkle_root`

**treasury_operations**
- Logs sweeping operations from hot wallets to cold storage
- Key fields: `from_wallet`, `to_wallet`, `amount`, `chain`, `tx_hash`, `operation_type`

---

## 🔄 Core Workflows

### 1. Payment Creation Flow
```
Merchant API Request
  → Validate merchant & amount
  → Get current exchange rate
  → Calculate crypto amount
  → Create payment record (status: created)
  → Generate QR code (wallet + amount + memo)
  → Return payment details to merchant
```

### 2. Payment Confirmation Flow
```
User sends crypto to wallet
  → Blockchain listener detects transaction
  → Extract memo (payment_id) and amount
  → Wait for finality (Solana: ~13s, BSC: ~3min)
  → Validate amount matches payment
  → Update payment status: pending → confirming → completed
  → Record ledger entry (crypto received → merchant balance)
  → Send webhook to merchant
  → Send email notification
```

### 3. Payout Flow (Manual for MVP)
```
Merchant requests payout
  → Validate balance sufficient
  → Create payout record (status: requested)
  → Admin reviews in admin panel
  → Admin approves/rejects
  → If approved: Ops team executes bank transfer
  → Ops team marks payout completed
  → Record ledger entry (deduct from merchant balance)
  → Send confirmation email
```

### 4. OTC Settlement Flow (Manual for MVP)
```
Daily: Check hot wallet balance
  → If balance > threshold
  → Ops team contacts OTC partner
  → Transfer crypto to OTC partner
  → Receive VND to business bank account
  → Update system VND pool in ledger
  → Record OTC transaction
```

### 🆕 5. Smart Identity Mapping Flow (PRD v2.2)
```
User scans payment QR
  → Extract wallet address from transaction
  → Check Redis cache: wallet→user mapping
  → If cached: retrieve user_id, skip KYC
  → If NOT cached: Check PostgreSQL wallet_identity_mappings
  → If found in DB: load user data, cache in Redis (7-day TTL)
  → If NEW wallet: Trigger KYC flow (Sumsub face liveness)
  → After KYC: Create wallet_identity_mapping record
  → Cache in Redis for 7 days
  → Process payment with user context
```

### 🆕 6. Omni-channel Notification Flow (PRD v2.2)
```
Payment confirmed event
  → Notification Dispatcher receives event
  → Load merchant notification preferences
  → For each enabled channel (Speaker, Telegram, Zalo, Email, Webhook):
    → Create notification job in Bull Queue (Redis)
    → Plugin-specific worker picks up job
    → Send notification via channel API
    → Log result in notification_logs table
    → If failed: retry with exponential backoff (3 attempts)
  → Track delivery success rate per channel
```

### 🆕 7. Treasury Sweeping Flow (PRD v2.2)
```
Every 6 hours (cron job)
  → For each hot wallet (TRON, Solana, BSC):
    → Check balance
    → If balance > $10,000:
      → Calculate sweep amount (leave $5k buffer)
      → Create multi-sig transaction (2-of-3)
      → Transfer to cold wallet
      → Log in treasury_operations table
      → Send alert to ops team
```

### 🆕 8. Data Archival Flow (PRD v2.2)
```
Monthly (1st of month, 2 AM UTC)
  → Find records > 12 months old (payments, payouts, ledger)
  → For each batch:
    → Compute SHA-256 hash for each record
    → Compress batch (gzip)
    → Upload to S3 Glacier
    → Create archived_records metadata entry
    → Mark original records as archived (keep IDs + hashes)
  → Daily: Compute Merkle root for day's hashes
  → Store Merkle root for integrity verification
```

---

## 🔐 Security Considerations

### MVP Security Requirements
- API authentication via API keys (merchants) and JWT (admin)
- Rate limiting: 100 requests/min per API key
- Webhook HMAC signature verification
- Database encryption at rest
- Audit logging for all critical operations
- Private keys stored in environment variables (→ Vault in Phase 2)
- Hot wallet maintains minimum balance (<$10k)

### 🆕 PRD v2.2 Security Enhancements
- **Multi-sig Cold Wallet**: 2-of-3 signature scheme for cold storage
- **Auto-sweeping**: Hot wallet → Cold wallet every 6 hours when balance > $10k
- **Face Liveness Detection**: Anti-spoofing for KYC (Sumsub)
- **Transaction Hash Chain**: SHA-256 hash chain for immutability
- **Daily Merkle Root**: Batch integrity verification
- **S3 Glacier Encryption**: AES-256 encryption for archived data
- **Redis Cache Security**: Encrypted wallet identity mappings with 7-day TTL

### Secure Coding Practices
- **NEVER use float64 for money calculations** - always use `decimal.Decimal`
- Always validate and sanitize user input
- Prevent SQL injection (use parameterized queries)
- Implement CSRF protection for web interfaces
- Use TLS 1.3 for all external communications
- Redact PII from logs

### Critical Security Checks
- Before committing: ensure no secrets in code or config files
- Validate all webhook signatures before processing
- Verify transaction amounts match exactly (no tolerance)
- Log all authentication failures and suspicious activities
- Implement transaction limits (start conservative: 10M VND/tx max)

---

## 📋 Development Workflows

### Git Workflow

**Branching Strategy**:
- `main`: Production-ready code
- Feature branches: `claude/feature-name-{session-id}`
- All development must happen on designated branches
- **CRITICAL**: Branch names must start with `claude/` and end with matching session ID

**Commit Guidelines**:
- Write clear, descriptive commit messages
- Focus on "why" rather than "what"
- Format: Use heredoc for multi-line messages
- Example: `git commit -m "$(cat <<'EOF'\nAdd payment confirmation logic\n\nImplements blockchain listener to detect and confirm payments...\nEOF\n)"`

**Git Operations**:
- Use `git push -u origin <branch-name>` for pushing
- For network failures: retry up to 4 times with exponential backoff (2s, 4s, 8s, 16s)
- Fetch specific branches: `git fetch origin <branch-name>`
- Avoid force push to main/master

### Code Review Process
- All PRs require review before merging
- Run tests before creating PR
- Ensure all migrations are reversible
- Document breaking API changes
- Follow Go/TypeScript style guides

### Testing Strategy

**Unit Tests**:
- Test business logic in services
- Mock external dependencies (blockchain RPC, database)
- Use `github.com/stretchr/testify` for assertions

**Integration Tests**:
- Test on testnet first (Solana devnet, BSC testnet)
- End-to-end payment flow testing
- Verify webhook delivery

**Testing Checklist Before Production**:
- [ ] All unit tests passing
- [ ] Integration tests on testnet successful
- [ ] Load testing completed (100+ concurrent payments)
- [ ] Security audit passed
- [ ] Database backup/restore tested

---

## 🚀 Deployment

### MVP Deployment (Single VPS)
- **Server**: 4 CPU cores, 8GB RAM, 100GB SSD
- **OS**: Ubuntu 22.04 LTS
- **Stack**: Docker + Docker Compose + NGINX + Let's Encrypt
- **Services**: API, Listener, PostgreSQL, Redis, Frontend

### Environment Configuration

**Required Environment Variables**:
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<secret>
DB_NAME=payment_gateway

# Redis
REDIS_HOST=localhost:6379
REDIS_PASSWORD=<secret>

# Blockchain (use testnet first!)
# 🆕 TRON (Priority HIGH)
TRON_RPC_URL=https://api.shasta.trongrid.io (testnet)
TRON_HOT_WALLET_PRIVATE_KEY=<secret>
TRON_HOT_WALLET_ADDRESS=<public>
TRON_COLD_WALLET_ADDRESS=<public>

SOLANA_RPC_URL=https://api.devnet.solana.com
SOLANA_HOT_WALLET_PRIVATE_KEY=<secret>
SOLANA_HOT_WALLET_ADDRESS=<public>
SOLANA_COLD_WALLET_ADDRESS=<public>

BSC_RPC_URL=https://data-seed-prebsc-1-s1.binance.org:8545
BSC_HOT_WALLET_PRIVATE_KEY=<secret>
BSC_HOT_WALLET_ADDRESS=<public>
BSC_COLD_WALLET_ADDRESS=<public>

# API
API_PORT=8080
JWT_SECRET=<secret>
API_RATE_LIMIT=100

# Exchange Rate
EXCHANGE_RATE_API=https://api.coingecko.com/api/v3

# 🆕 PRD v2.2 Integrations
# KYC & Face Liveness
SUMSUB_APP_TOKEN=<secret>
SUMSUB_SECRET_KEY=<secret>
SUMSUB_BASE_URL=https://api.sumsub.com

# Notifications
TELEGRAM_BOT_TOKEN=<secret>
ZALO_OA_ID=<public>
ZALO_OA_SECRET=<secret>
GOOGLE_TTS_API_KEY=<secret>
SENDGRID_API_KEY=<secret>

# Data Archival
AWS_ACCESS_KEY_ID=<secret>
AWS_SECRET_ACCESS_KEY=<secret>
AWS_S3_BUCKET=payment-gateway-archives
AWS_S3_REGION=ap-southeast-1

# Treasury
TREASURY_SWEEP_THRESHOLD_USD=10000
TREASURY_SWEEP_INTERVAL_HOURS=6
MULTISIG_REQUIRED_SIGNATURES=2

# Environment
ENV=development|staging|production
```

**⚠️ CRITICAL**: Always add `.env` to `.gitignore`

---

## 📐 Coding Conventions

### Golang Conventions

**Project Layout**:
- Follow [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- `cmd/` for main applications
- `internal/` for private application code
- `pkg/` for public libraries (if needed)

**Naming**:
- Package names: lowercase, single word
- Exported functions: PascalCase
- Private functions: camelCase
- Constants: PascalCase or SCREAMING_SNAKE_CASE

**Error Handling**:
```go
// Always check errors
payment, err := s.paymentRepo.GetByID(paymentID)
if err != nil {
    return nil, fmt.Errorf("failed to get payment: %w", err)
}

// Use custom error types for domain errors
if payment.Status != "pending" {
    return ErrInvalidPaymentStatus
}
```

**Money Calculations**:
```go
// NEVER use float64 for money
// ALWAYS use decimal.Decimal
import "github.com/shopspring/decimal"

amountVND := decimal.NewFromFloat(2300000)
amountUSDT := decimal.NewFromFloat(100)
fee := amountVND.Mul(decimal.NewFromFloat(0.01)) // 1% fee
```

### API Design Conventions

**REST Endpoints**:
- Use versioning: `/api/v1/...`
- Use plural nouns: `/payments`, `/merchants`
- Use HTTP methods correctly: GET (read), POST (create), PUT (update), DELETE (delete)
- Return proper status codes: 200 (OK), 201 (Created), 400 (Bad Request), 401 (Unauthorized), 404 (Not Found), 500 (Server Error)

**Response Format**:
```json
{
  "data": { ... },
  "error": null,
  "timestamp": "2025-11-16T10:20:30Z"
}
```

**Error Response**:
```json
{
  "data": null,
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Merchant balance insufficient for payout"
  },
  "timestamp": "2025-11-16T10:20:30Z"
}
```

---

## 📚 Key Documentation References

### Required Reading (In Order)
1. **README.md** - Start here for project overview
2. **PRD_v2.2.md** - Complete product requirements (PRD v2.2)
3. **PRD_v2.2_ROADMAP.md** - 10-week implementation roadmap (detailed day-by-day plan)
4. **REQUIREMENTS.md** - Understand functional requirements
5. **ARCHITECTURE.md** - Deep dive into system design
6. **TECH_STACK_GOLANG.md** - Implementation details
7. **GETTING_STARTED.md** - Setup and development guide

### 🆕 PRD v2.2 Module Docs
- **IDENTITY_MAPPING.md** - Smart wallet→user identity (one-time KYC)
- **NOTIFICATION_CENTER.md** - Omni-channel notification architecture
- **DATA_RETENTION.md** - Infinite storage with S3 Glacier + transaction hashing
- **OFF_RAMP_STRATEGIES.md** - Flexible withdrawal modes (on-demand, scheduled, threshold)

### Domain-Specific Docs
- **AML_ENGINE.md** - In-house AML compliance engine
- **TOURISM_USE_CASES.md** - Understand target market and use cases
- **STAKEHOLDER_ANALYSIS.md** - Business model and stakeholder needs

---

## 🎯 PRD v2.2 Success Criteria

### Technical KPIs
- Payment success rate: **> 98%**
- Average confirmation time: **< 20 seconds**
- System uptime: **> 99%**
- Webhook delivery rate: **> 95%**
- 🆕 **KYC Recognition Rate**: **> 95%** (returning users auto-recognized)
- 🆕 **Notification Delivery Rate**: **> 95%** (across all channels)
- 🆕 **Treasury Sweep Success**: **> 99%** (automated sweeping operations)
- Zero security incidents

### Business KPIs
- Pilot merchants: **5+**
- Total volume (Month 1): **1B+ VND**
- Revenue (Month 1): **10M+ VND**
- NPS: **> 30**
- 🆕 **Multi-chain Adoption**: TRON > 50%, Solana > 30%, BSC > 20%

### Compliance Requirements
- All transactions properly logged in audit_logs
- KYC records stored securely and encrypted
- 🆕 **Infinite data retention** (S3 Glacier archival)
- 🆕 **Transaction immutability** (hash chain + Merkle root verification)
- Zero compliance violations

---

## ⚠️ Critical Implementation Notes

### Payment Flow
- Always wait for blockchain finality before confirming (Solana: `finalized` commitment)
- Payment memo/reference MUST match payment_id exactly
- Amount received MUST match expected amount exactly (no tolerance for errors)
- Expired payments (>30 minutes) should not be confirmable

### Ledger System
- MUST use double-entry accounting for all transactions
- Every debit must have a corresponding credit
- Ledger entries are immutable (never update, only create new entries)
- Balance calculations should always reconcile with ledger entries

### Security
- NEVER log private keys or sensitive data
- NEVER use default/example API keys in production
- ALWAYS validate webhook signatures before processing
- ALWAYS use HTTPS in production
- Hot wallet should trigger alerts when balance exceeds threshold

### Compliance
- All KYC documents must be encrypted at rest
- Audit logs must never be deleted (append-only)
- Transaction limits must be enforced (start: 10M VND/tx max)
- Payout requests must go through manual approval (MVP)

### 🆕 PRD v2.2 Critical Notes

**Identity Mapping**:
- ALWAYS check Redis cache first (wallet→user mapping)
- Cache TTL = 7 days (604,800 seconds)
- NEVER skip face liveness check for new wallets
- Hash wallet addresses before storing (privacy)

**Notification System**:
- MUST log all notification attempts in notification_logs
- Retry failed notifications max 3 times (exponential backoff: 1m, 5m, 15m)
- Rate limits: Zalo (100/day free tier), Telegram (30 msg/sec)
- Speaker notifications: ONLY for in-person merchant terminals

**Data Retention**:
- NEVER delete archived data from S3 Glacier
- ALWAYS compute SHA-256 hash before archiving
- Verify hash integrity before restoring from archive
- Daily Merkle root MUST be computed and stored

**Treasury Security**:
- Hot wallet sweeping: MUST use multi-sig 2-of-3
- NEVER sweep if balance < $5k buffer
- Log ALL sweeping operations in treasury_operations table
- Alert ops team for manual verification before sweep

---

## 🛠️ Common Development Tasks

### Adding a New Payment Chain

1. Create chain-specific listener in `internal/blockchain/<chain>/`
2. Implement `BlockchainListener` interface
3. Add chain-specific wallet management
4. Update `SupportedToken` configuration
5. Add integration tests on testnet
6. Update documentation

### Adding a New API Endpoint

1. Define handler in `internal/api/handler/`
2. Add route in `internal/api/routes/`
3. Implement service logic in `internal/service/`
4. Add repository methods in `internal/repository/`
5. Write unit tests
6. Update API documentation
7. Add audit logging

### Database Migration

1. Create migration files: `migrations/NNN_description.up.sql` and `.down.sql`
2. Test up migration
3. Test down migration (must be reversible)
4. Update model structs in `internal/model/`
5. Run migrations on staging before production

### 🆕 Adding a New Notification Channel (PRD v2.2)

1. Implement `NotificationPlugin` interface in `internal/notification/plugins/<channel>/`
2. Add channel-specific configuration (API keys, rate limits)
3. Implement send method with retry logic (3 attempts, exponential backoff)
4. Add integration tests with mock API
5. Update merchant dashboard to allow channel configuration
6. Add monitoring for delivery success rate
7. Update NOTIFICATION_CENTER.md documentation

### 🆕 Configuring Wallet Identity Caching (PRD v2.2)

1. Set up Redis cache with 7-day TTL
2. Implement cache-aside pattern: check cache → fallback to DB
3. Hash wallet addresses before caching (privacy)
4. Add cache invalidation logic (KYC status updates)
5. Monitor cache hit rate (target > 90%)
6. Test cache expiration and refresh logic

### 🆕 Setting Up S3 Glacier Archival (PRD v2.2)

1. Create S3 bucket with Glacier storage class
2. Implement monthly archival job (1st of month, 2 AM UTC)
3. Add SHA-256 hashing for each record before archival
4. Test archive creation, upload, and retrieval (Expedited tier)
5. Implement restore process (1-5 hours)
6. Add monitoring for archive success rate
7. Test integrity verification (hash matching)

---

## 🔍 Troubleshooting Guide

### Common Issues

**Issue**: Blockchain listener not detecting transactions
- Check RPC endpoint connectivity
- Verify wallet address configuration
- Check RPC rate limits (consider paid RPC like Helius, QuickNode)
- Review listener logs for errors

**Issue**: Payment amount mismatch
- Verify exchange rate API is working
- Check decimal precision (Solana: 6 decimals, BSC: 18 decimals)
- Ensure using `decimal.Decimal` not `float64`

**Issue**: Webhook delivery failures
- Verify merchant webhook URL is accessible
- Check HMAC signature generation
- Review retry logic and exponential backoff
- Check webhook logs in database

**Issue**: Database migration fails
- Check for conflicting schema changes
- Verify migration file syntax
- Test on local database first
- Consider breaking into smaller migrations

### 🆕 PRD v2.2 Common Issues

**Issue**: Wallet identity not recognized (cache miss rate high)
- Check Redis connection and TTL configuration
- Verify wallet address hashing consistency
- Review cache invalidation logic
- Check PostgreSQL wallet_identity_mappings table

**Issue**: Notification delivery failures
- Check channel-specific rate limits (Zalo: 100/day, Telegram: 30/sec)
- Verify API keys and credentials
- Review notification_logs for error patterns
- Test retry logic (3 attempts, exponential backoff)
- Check Bull Queue Redis connection

**Issue**: S3 Glacier restore timeout
- Verify using Expedited tier (1-5 hours, not Standard 3-5 hours)
- Check restore request status via AWS console
- Ensure sufficient time buffer (wait up to 6 hours)
- Test with smaller archive files first

**Issue**: Treasury sweeping failed
- Verify multi-sig 2-of-3 configuration
- Check hot wallet balance > $10k threshold
- Review treasury_operations logs for errors
- Ensure cold wallet addresses are correct
- Verify blockchain network connectivity (TRON, Solana, BSC)

**Issue**: KYC face liveness check failed
- Verify Sumsub API credentials
- Check image quality and lighting
- Review anti-spoofing detection logs
- Test with different devices/cameras
- Ensure user follows on-screen instructions

---

## 🌍 Vietnam-Specific Considerations

### Legal & Compliance
- Business must be registered as technology service provider (NOT financial institution)
- Partner with licensed OTC desk for crypto↔VND conversion
- Crypto treated as digital asset (NOT currency) in Vietnam
- All settlements must be in VND (fiat), not crypto
- Keep detailed records for tax audits (7-year retention)

### Da Nang Regulatory Sandbox
- Tether + Da Nang partnership (Nov 2025) enables blockchain payment pilot
- Resolution 222/2025/QH15 establishes International Financial Center framework
- Early-mover advantage in regulatory sandbox
- Must maintain compliance for continued operation

### Banking Integration
- Partner with Vietnamese bank for VND settlements
- Bank transfers typically T+1 (manual MVP acceptable)
- Consider Vietcombank, VietinBank, BIDV for merchant accounts

---

## 🔗 External Resources

### Blockchain Development
- [Solana Go SDK](https://github.com/gagliardetto/solana-go)
- [Go Ethereum Documentation](https://geth.ethereum.org/docs/developers/dapp-developer/native)
- [Solana RPC Methods](https://docs.solana.com/api/http)

### Golang Best Practices
- [Effective Go](https://go.dev/doc/effective_go)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Development Tools
- [GORM Documentation](https://gorm.io/docs/)
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [PostgreSQL 15 Documentation](https://www.postgresql.org/docs/15/)

---

## 📞 Support & Communication

### Development Communication
- **Daily Standups**: What did you do? What will you do? Any blockers?
- **Weekly Reviews**: Demo features, review code, plan next week
- **Code Reviews**: All PRs require approval, run tests before merging

### Documentation Updates
- Update CLAUDE.md when architecture changes
- Update API docs when endpoints change
- Update GETTING_STARTED.md when setup process changes
- Keep README.md accurate with current project status

---

## ✅ Pre-Implementation Checklist

Before starting implementation, ensure:

### Business Readiness
- [ ] Legal entity registered
- [ ] Bank account opened for VND settlements
- [ ] OTC partner identified (2-3 options)
- [ ] 3-5 pilot merchants lined up
- [ ] Legal advisor consulted on compliance

### Technical Readiness
- [ ] GitHub repository set up
- [ ] Team has access and permissions
- [ ] Development environment requirements documented
- [ ] Cloud/VPS account created
- [ ] Domain name purchased

### Team Alignment
- [ ] All team members read core documentation
- [ ] Tech stack approved (Golang vs Node.js decision made)
- [ ] Architecture reviewed and understood
- [ ] Timeline agreed (4-6 weeks realistic)
- [ ] Roles and responsibilities assigned

---

## 🎓 Learning Resources for New Contributors

### For Backend Engineers
1. Read ARCHITECTURE.md - understand system design
2. Review TECH_STACK_GOLANG.md - understand technology choices
3. Study ledger system design - critical for financial accuracy
4. Understand blockchain finality - Solana vs BSC differences

### For Frontend Engineers
1. Review merchant dashboard requirements in STAKEHOLDER_ANALYSIS.md
2. Study payment page UX in TOURISM_USE_CASES.md
3. Understand real-time status updates (polling vs WebSocket)
4. Review QR code generation requirements

### For DevOps Engineers
1. Review deployment architecture in ARCHITECTURE.md
2. Study Docker multi-stage builds in TECH_STACK_GOLANG.md
3. Understand monitoring requirements in MVP_ROADMAP.md
4. Plan backup and disaster recovery procedures

---

## 📝 Notes for AI Assistants

### When Working on This Project

**Always**:
- Read the relevant documentation before making changes
- Follow the established coding conventions
- Consider security implications of all changes
- Write tests for new functionality
- Update documentation when making architectural changes
- Use decimal.Decimal for all money calculations
- Log important operations for audit trail

**Never**:
- Use float64 for money calculations
- Commit secrets or API keys to git
- Skip input validation
- Make breaking changes without discussion
- Deploy to production without testing on testnet
- Modify ledger entries after creation

### Documentation Philosophy
- **README.md**: Quick overview for stakeholders
- **REQUIREMENTS.md**: What to build (functional requirements)
- **ARCHITECTURE.md**: How to build it (technical design)
- **TECH_STACK_GOLANG.md**: Implementation details
- **GETTING_STARTED.md**: Step-by-step setup for developers
- **CLAUDE.md**: Comprehensive guide for AI assistants

### Current Development Status
As of 2025-11-19, this project is in **PRD v2.2 documentation phase**. No code has been written yet. The documentation now includes:

✅ **Completed Documentation (PRD v2.2)**:
- PRD_v2.2.md - Complete product requirements
- IDENTITY_MAPPING.md - Smart wallet→user identity system
- NOTIFICATION_CENTER.md - Omni-channel notification architecture
- DATA_RETENTION.md - Infinite storage with S3 Glacier
- OFF_RAMP_STRATEGIES.md - Flexible withdrawal modes
- Updated ARCHITECTURE.md, REQUIREMENTS.md, README.md, CLAUDE.md

**Next Steps**:
1. Confirm tech stack (Golang recommended)
2. Set up project structure
3. Begin PRD v2.2 implementation (8-10 weeks phased rollout)
4. Phase 1 (Weeks 1-4): Core payment + Identity Mapping
5. Phase 2 (Weeks 5-6): Notification Center + Treasury
6. Phase 3 (Weeks 7-8): Data Retention + Off-ramp
7. Testing & Launch (Weeks 9-10)

---

**Last Updated**: 2025-11-19
**Maintained By**: Project Team
**Questions**: Refer to documentation first, then consult team lead
