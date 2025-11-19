# PRD v2.2 Implementation Roadmap

**Project**: Stablecoin Payment Gateway - PRD v2.2
**Timeline**: 10 Weeks (Phased Implementation)
**Start Date**: TBD
**Target Launch**: Week 10
**Last Updated**: 2025-11-19

---

## 📋 Executive Summary

This roadmap details the 10-week implementation plan for PRD v2.2, transforming the stablecoin payment gateway from concept to production-ready system with advanced features:

### Implementation Approach
- **Phased Rollout**: 4 phases over 10 weeks
- **Agile Methodology**: 2-week sprints with daily standups
- **Testnet First**: All blockchain features tested on testnet before mainnet
- **Parallel Development**: Backend, frontend, and infrastructure teams work in parallel

### Key Milestones
- **Week 4**: Core Payment System + Identity Mapping (Phase 1 Complete)
- **Week 6**: Notification Center + Treasury (Phase 2 Complete)
- **Week 8**: Data Retention + Off-ramp (Phase 3 Complete)
- **Week 10**: Production Launch (Phase 4 Complete)

### Team Structure (Recommended)
- **Backend Team**: 2-3 Golang developers
- **Frontend Team**: 1-2 React/Next.js developers
- **Blockchain Team**: 1 specialist (can overlap with backend)
- **DevOps**: 1 engineer
- **QA**: 1 tester (manual + automated)
- **Product Owner**: 1 (part-time, owns PRD)

---

## 🎯 Success Criteria

### Technical Targets
- ✅ Payment success rate > 98%
- ✅ KYC recognition rate > 95%
- ✅ Notification delivery > 95%
- ✅ System uptime > 99%
- ✅ Treasury sweep success > 99%

### Business Targets
- ✅ 5 pilot merchants onboarded
- ✅ 1B+ VND transaction volume (Month 1)
- ✅ All compliance requirements met

---

## 📅 Phase Overview

### Phase 1: Core Foundation (Weeks 1-4)
**Goal**: Functional payment gateway with identity mapping

**Deliverables**:
- ✅ Multi-chain payment processing (TRON, Solana, BSC)
- ✅ Smart identity mapping with KYC
- ✅ Merchant dashboard
- ✅ Admin panel (basic)
- ✅ Double-entry ledger

**Team Focus**:
- Backend: Payment APIs, blockchain listeners, ledger
- Frontend: Merchant dashboard, payment pages
- Blockchain: Multi-chain integration, wallet management
- DevOps: Infrastructure setup, CI/CD

---

### Phase 2: Notifications & Treasury (Weeks 5-6)
**Goal**: Omni-channel notifications + custodial treasury

**Deliverables**:
- ✅ Multi-channel notification system (5 channels)
- ✅ Custodial hot + cold wallet architecture
- ✅ Automated treasury sweeping (6-hour intervals)
- ✅ Multi-sig wallet support

**Team Focus**:
- Backend: Notification dispatcher, treasury service
- Frontend: Notification preferences UI
- Blockchain: Multi-sig implementation, sweeping logic
- DevOps: AWS integration, monitoring

---

### Phase 3: Data & Off-ramp (Weeks 7-8)
**Goal**: Infinite data retention + flexible payouts

**Deliverables**:
- ✅ S3 Glacier archival system
- ✅ Transaction hashing + Merkle trees
- ✅ Scheduled withdrawals
- ✅ Threshold-based withdrawals
- ✅ Advanced payout management

**Team Focus**:
- Backend: Archival service, payout scheduler
- Frontend: Withdrawal configuration UI
- DevOps: S3 setup, backup verification
- QA: Data integrity testing

---

### Phase 4: Testing & Launch (Weeks 9-10)
**Goal**: Production-ready system with pilot merchants

**Deliverables**:
- ✅ End-to-end testing completed
- ✅ Security audit passed
- ✅ Performance testing (100+ concurrent payments)
- ✅ Pilot merchant onboarding
- ✅ Production deployment

**Team Focus**:
- All teams: Bug fixing, optimization
- QA: Comprehensive testing
- Product: Merchant onboarding, training
- DevOps: Production deployment, monitoring

---

# 🗓️ Week-by-Week Breakdown

## PHASE 1: CORE FOUNDATION

---

## Week 1: Infrastructure & Database Foundation

### Goals
- ✅ Development environment ready
- ✅ Database schema deployed
- ✅ CI/CD pipeline working
- ✅ First API endpoint live

### Daily Tasks

#### **Monday: Project Setup**
**Backend Team**:
- [ ] Initialize Go project structure (`cmd/`, `internal/`, `migrations/`)
- [ ] Set up Go modules, install core dependencies (Gin, GORM, decimal, etc.)
- [ ] Configure `.env` file with testnet RPC endpoints
- [ ] Create `config/` package for environment variable loading

**Frontend Team**:
- [ ] Initialize Next.js 14 project with TypeScript
- [ ] Set up TailwindCSS + shadcn/ui
- [ ] Configure project structure (`app/`, `components/`, `lib/`)
- [ ] Create layout components (header, sidebar, footer)

**DevOps Team**:
- [ ] Provision VPS/cloud server (4 CPU, 8GB RAM)
- [ ] Install Docker + Docker Compose
- [ ] Set up PostgreSQL 15 container
- [ ] Set up Redis 7 container
- [ ] Configure firewall rules

**Deliverable**: Project repositories initialized, development environment ready

---

#### **Tuesday: Database Schema**
**Backend Team**:
- [ ] Write migration `001_create_merchants.up.sql`
  - merchants table (id, email, business_name, kyc_status, api_key, webhook_url, created_at, updated_at)
- [ ] Write migration `002_create_payments.up.sql`
  - payments table (id, merchant_id, amount_vnd, amount_crypto, crypto_currency, chain, status, tx_hash, expires_at, created_at)
- [ ] Write migration `003_create_payouts.up.sql`
  - payouts table (id, merchant_id, amount_vnd, bank_account_number, bank_name, status, approved_at, created_at)
- [ ] Write migration `004_create_ledger_entries.up.sql`
  - ledger_entries table (id, debit_account, credit_account, amount, currency, reference_type, reference_id, created_at)
- [ ] Run migrations on local PostgreSQL
- [ ] Test rollback (`.down.sql` files)

**QA Team**:
- [ ] Review schema for completeness
- [ ] Test migration rollback
- [ ] Verify indexes and foreign keys

**Deliverable**: Core database schema created and tested

---

#### **Wednesday: Merchant & Payment Models**
**Backend Team**:
- [ ] Create `internal/model/merchant.go` (Merchant struct with GORM tags)
- [ ] Create `internal/model/payment.go` (Payment struct, PaymentStatus enum)
- [ ] Create `internal/model/payout.go` (Payout struct, PayoutStatus enum)
- [ ] Create `internal/model/ledger_entry.go` (LedgerEntry struct)
- [ ] Create `internal/repository/merchant_repository.go` (CRUD operations)
- [ ] Create `internal/repository/payment_repository.go` (Create, GetByID, UpdateStatus)
- [ ] Write unit tests for repositories (use testify)

**Frontend Team**:
- [ ] Create TypeScript types for Merchant, Payment, Payout
- [ ] Set up API client with axios
- [ ] Create reusable data fetching hooks (useQuery from react-query)

**Deliverable**: Data models and repository layer complete

---

#### **Thursday: API Framework & Authentication**
**Backend Team**:
- [ ] Set up Gin HTTP server in `cmd/api/main.go`
- [ ] Create middleware: `internal/api/middleware/auth.go` (API key validation)
- [ ] Create middleware: `internal/api/middleware/cors.go`
- [ ] Create middleware: `internal/api/middleware/rate_limit.go` (100 req/min)
- [ ] Create `internal/api/handler/health.go` (GET /health endpoint)
- [ ] Create `internal/service/merchant_service.go` (business logic)
- [ ] Implement merchant registration (POST /api/v1/merchants)
- [ ] Write integration tests for registration endpoint

**Frontend Team**:
- [ ] Create login page for admin panel
- [ ] Implement JWT token storage (localStorage)
- [ ] Create auth context provider
- [ ] Implement protected route wrapper

**Deliverable**: API server running with authentication

---

#### **Friday: CI/CD & Testing**
**Backend Team**:
- [ ] Write Dockerfile (multi-stage build)
- [ ] Create `docker-compose.yml` (api, postgres, redis)
- [ ] Write unit tests for merchant service
- [ ] Achieve >80% code coverage

**DevOps Team**:
- [ ] Set up GitHub Actions workflow (`.github/workflows/ci.yml`)
- [ ] Configure automatic testing on PR
- [ ] Set up Docker image build + push to registry
- [ ] Deploy to staging environment

**QA Team**:
- [ ] Test merchant registration flow end-to-end
- [ ] Verify API key generation
- [ ] Test rate limiting (send 101 requests, expect 429)

**Deliverable**: CI/CD pipeline live, first deployment to staging

**Week 1 Retrospective**:
- Review what went well / what didn't
- Adjust Week 2 plan if needed
- Team morale check

---

## Week 2: Multi-Chain Blockchain Integration

### Goals
- ✅ TRON listener working
- ✅ Solana listener working
- ✅ BSC listener working
- ✅ Payment creation API complete

### Daily Tasks

#### **Monday: TRON Integration**
**Blockchain Team**:
- [ ] Install `github.com/fbsobreira/gotron-sdk`
- [ ] Create `internal/blockchain/tron/client.go` (TronGrid RPC client)
- [ ] Create `internal/blockchain/tron/listener.go` (monitor wallet for USDT TRC20)
- [ ] Implement transaction parsing (extract amount, memo, sender)
- [ ] Test on TRON Shasta testnet (https://api.shasta.trongrid.io)
- [ ] Create hot wallet on testnet, fund with test TRX

**Backend Team**:
- [ ] Create migration `005_create_blockchain_transactions.up.sql`
- [ ] Create `internal/model/blockchain_transaction.go`
- [ ] Create `internal/repository/blockchain_tx_repository.go`

**Deliverable**: TRON listener detects test transactions

---

#### **Tuesday: Solana Integration**
**Blockchain Team**:
- [ ] Install `github.com/gagliardetto/solana-go`
- [ ] Create `internal/blockchain/solana/client.go` (Solana RPC client)
- [ ] Create `internal/blockchain/solana/listener.go` (monitor wallet for USDT/USDC SPL)
- [ ] Implement transaction parsing (parse memo field)
- [ ] Test on Solana devnet (https://api.devnet.solana.com)
- [ ] Create hot wallet on devnet, request airdrop

**Backend Team**:
- [ ] Add Solana-specific fields to blockchain_transactions table
- [ ] Test transaction finality detection (`finalized` commitment level)

**Deliverable**: Solana listener detects test transactions

---

#### **Wednesday: BSC Integration**
**Blockchain Team**:
- [ ] Install `github.com/ethereum/go-ethereum`
- [ ] Create `internal/blockchain/bsc/client.go` (BSC RPC client)
- [ ] Create `internal/blockchain/bsc/listener.go` (monitor wallet for USDT BEP20)
- [ ] Implement ERC20 transfer event parsing
- [ ] Test on BSC testnet (https://data-seed-prebsc-1-s1.binance.org:8545)
- [ ] Create hot wallet on testnet, get test BNB from faucet

**Backend Team**:
- [ ] Add BSC-specific fields to blockchain_transactions table
- [ ] Implement chain-agnostic transaction validator

**Deliverable**: BSC listener detects test transactions

---

#### **Thursday: Payment Creation API**
**Backend Team**:
- [ ] Create `internal/service/payment_service.go`
- [ ] Implement CreatePayment() logic:
  - Validate merchant API key
  - Get current exchange rate (CoinGecko API)
  - Calculate crypto amount (VND → USDT/USDC)
  - Generate unique payment_id
  - Create payment record (status: `created`)
  - Return payment details + QR code data
- [ ] Create API endpoint: POST /api/v1/payments
- [ ] Write integration tests

**Frontend Team**:
- [ ] Create payment creation form in merchant dashboard
- [ ] Display QR code using `qrcode.react`
- [ ] Show payment details (amount, crypto address, memo, expiry)

**Deliverable**: Merchants can create payment requests

---

#### **Friday: Payment Confirmation Flow**
**Backend Team**:
- [ ] Create `internal/service/payment_confirmation_service.go`
- [ ] Implement payment matching logic:
  - Match blockchain tx to payment by memo (payment_id)
  - Validate amount matches exactly (use decimal.Decimal)
  - Check payment not expired
  - Update payment status: `pending` → `confirming` → `completed`
- [ ] Integrate blockchain listeners with confirmation service
- [ ] Create ledger entries when payment confirmed
- [ ] Implement webhook delivery (HMAC signature)

**QA Team**:
- [ ] Test end-to-end payment flow on TRON testnet
- [ ] Test end-to-end payment flow on Solana devnet
- [ ] Test end-to-end payment flow on BSC testnet
- [ ] Test payment expiry (create payment, wait 31 minutes, send crypto, expect failure)

**Deliverable**: Complete payment flow working on all 3 chains

**Week 2 Retrospective**

---

## Week 3: Smart Identity Mapping & KYC

### Goals
- ✅ Sumsub KYC integration working
- ✅ Wallet→user identity mapping functional
- ✅ Redis caching implemented
- ✅ Face liveness detection tested

### Daily Tasks

#### **Monday: Database & Models for Identity Mapping**
**Backend Team**:
- [ ] Create migration `006_create_users.up.sql`
  - users table (id, full_name, date_of_birth, nationality, email, phone, kyc_status, sumsub_applicant_id, created_at)
- [ ] Create migration `007_create_wallet_identity_mappings.up.sql`
  - wallet_identity_mappings table (id, wallet_address, blockchain, user_id, kyc_status, kyc_verified_at, payment_count, total_volume_usd, first_seen_at, last_seen_at)
  - Add unique constraint on (wallet_address, blockchain)
- [ ] Create `internal/model/user.go`
- [ ] Create `internal/model/wallet_identity_mapping.go`
- [ ] Create repositories for both tables

**Deliverable**: Identity mapping database schema ready

---

#### **Tuesday: Sumsub KYC Integration**
**Backend Team**:
- [ ] Sign up for Sumsub account (https://sumsub.com/)
- [ ] Get API credentials (APP_TOKEN, SECRET_KEY)
- [ ] Install Sumsub SDK or create HTTP client
- [ ] Create `internal/service/kyc_service.go`
- [ ] Implement CreateApplicant() - create Sumsub applicant
- [ ] Implement GetApplicantStatus() - check KYC status
- [ ] Implement GetAccessToken() - generate SDK token for frontend
- [ ] Test on Sumsub sandbox

**Frontend Team**:
- [ ] Install `@sumsub/websdk` npm package
- [ ] Create KYC flow component
- [ ] Implement Sumsub SDK initialization
- [ ] Test face liveness capture on mobile device

**Deliverable**: KYC flow functional with face liveness

---

#### **Wednesday: Wallet Identity Recognition**
**Backend Team**:
- [ ] Create `internal/service/identity_mapping_service.go`
- [ ] Implement CheckWalletIdentity(walletAddress, blockchain):
  - Check if wallet exists in wallet_identity_mappings
  - If exists and kyc_status = "verified", return user_id
  - If not exists, return null (trigger KYC)
- [ ] Implement CreateWalletMapping(walletAddress, blockchain, userId)
- [ ] Update payment confirmation flow to call identity service
- [ ] Write unit tests for identity matching logic

**Deliverable**: Wallet recognition logic complete

---

#### **Thursday: Redis Caching**
**Backend Team**:
- [ ] Install Redis client: `github.com/go-redis/redis/v8`
- [ ] Create `internal/cache/redis_cache.go`
- [ ] Implement cache functions:
  - SetWalletMapping(walletAddress, blockchain, userData, ttl=7days)
  - GetWalletMapping(walletAddress, blockchain)
  - InvalidateWalletMapping(walletAddress, blockchain)
- [ ] Update identity_mapping_service to use cache-aside pattern:
  - Check Redis first
  - On miss, query PostgreSQL
  - Store result in Redis with 7-day TTL
- [ ] Hash wallet addresses before caching (SHA-256)
- [ ] Write unit tests with Redis mock

**DevOps Team**:
- [ ] Configure Redis persistence (RDB + AOF)
- [ ] Set up Redis monitoring

**Deliverable**: Redis caching working with 7-day TTL

---

#### **Friday: End-to-End Identity Mapping Test**
**Backend Team**:
- [ ] Implement webhook from Sumsub when KYC approved
- [ ] Create API endpoint: POST /api/v1/webhooks/sumsub
- [ ] Verify webhook signature
- [ ] Update user kyc_status when approved
- [ ] Create wallet_identity_mapping record

**QA Team**:
- [ ] Test complete flow:
  1. New user scans payment QR
  2. System detects new wallet, triggers KYC
  3. User completes face liveness check
  4. KYC approved, wallet mapped to user
  5. Same user pays again with same wallet
  6. System recognizes user from cache (no KYC)
- [ ] Measure cache hit rate (target >90%)
- [ ] Test cache expiration (7 days)

**Deliverable**: Smart identity mapping fully functional

**Week 3 Retrospective**

---

## Week 4: Merchant Dashboard & Ledger System

### Goals
- ✅ Merchant dashboard complete
- ✅ Double-entry ledger working
- ✅ Balance calculations accurate
- ✅ Phase 1 deliverable ready

### Daily Tasks

#### **Monday: Double-Entry Ledger Implementation**
**Backend Team**:
- [ ] Create `internal/service/ledger_service.go`
- [ ] Implement CreateEntry(debit, credit, amount, currency, reference):
  - ALWAYS create TWO ledger entries (debit + credit)
  - Validate debit != credit
  - Use decimal.Decimal for amount
  - Record reference (payment_id, payout_id, etc.)
- [ ] Define account types:
  - `merchant:{merchant_id}:balance_vnd` (merchant VND balance)
  - `system:revenue:transaction_fees` (1% fee)
  - `system:pool:crypto_received` (crypto received)
  - `system:pool:vnd_available` (VND pool for payouts)
- [ ] Implement ledger entries for payment:
  - Debit: `system:pool:crypto_received` (100 USDT)
  - Credit: `merchant:{id}:balance_vnd` (2,300,000 VND)
  - Debit: `merchant:{id}:balance_vnd` (23,000 VND)
  - Credit: `system:revenue:transaction_fees` (23,000 VND)

**Deliverable**: Ledger service with double-entry logic

---

#### **Tuesday: Balance Calculations**
**Backend Team**:
- [ ] Create migration `008_create_merchant_balances.up.sql`
  - merchant_balances table (merchant_id, available_vnd, pending_vnd, total_received_vnd, total_paid_out_vnd, last_updated_at)
- [ ] Create `internal/service/balance_service.go`
- [ ] Implement CalculateBalance(merchantId):
  - Sum all ledger credits to `merchant:{id}:balance_vnd`
  - Subtract all ledger debits from `merchant:{id}:balance_vnd`
  - Verify balance matches merchant_balances table (reconciliation)
- [ ] Create API endpoint: GET /api/v1/merchants/me/balance
- [ ] Write tests to ensure ledger always balances (sum of all debits = sum of all credits)

**Deliverable**: Accurate balance calculations

---

#### **Wednesday: Merchant Dashboard - Transactions Page**
**Frontend Team**:
- [ ] Create transactions list page (`app/dashboard/transactions/page.tsx`)
- [ ] Display table with columns:
  - Payment ID, Amount VND, Amount Crypto, Chain, Status, Tx Hash, Created At
- [ ] Implement pagination (10 per page)
- [ ] Add filters: date range, status, chain
- [ ] Add search by payment ID or tx hash
- [ ] Display transaction details modal (click row to open)

**Backend Team**:
- [ ] Create API endpoint: GET /api/v1/merchants/me/payments
- [ ] Support query params: page, limit, status, chain, start_date, end_date, search

**Deliverable**: Transaction history page functional

---

#### **Thursday: Merchant Dashboard - Balance & Analytics**
**Frontend Team**:
- [ ] Create dashboard homepage (`app/dashboard/page.tsx`)
- [ ] Display KPI cards:
  - Available Balance (VND)
  - Pending Balance (VND)
  - Total Received (This Month)
  - Transaction Count (This Month)
- [ ] Add chart: Daily transaction volume (last 30 days) using recharts
- [ ] Add payment success rate chart
- [ ] Show recent transactions (last 5)

**Backend Team**:
- [ ] Create API endpoint: GET /api/v1/merchants/me/analytics
- [ ] Return daily volume, success rate, top chains used

**Deliverable**: Dashboard homepage with analytics

---

#### **Friday: Phase 1 Testing & Demo**
**QA Team**:
- [ ] Run full regression test suite
- [ ] Test payment flow on all 3 chains (TRON, Solana, BSC)
- [ ] Test identity mapping (new user + returning user)
- [ ] Verify ledger balance reconciliation
- [ ] Test merchant dashboard functionality
- [ ] Performance test: 50 concurrent payment creations

**All Teams**:
- [ ] Fix critical bugs found in testing
- [ ] Code review and refactoring
- [ ] Update documentation

**Product Owner**:
- [ ] Demo to stakeholders
- [ ] Gather feedback
- [ ] Prepare for Phase 2 kickoff

**Deliverable**: Phase 1 complete - Core payment gateway functional

**Week 4 Retrospective**

---

## PHASE 2: NOTIFICATIONS & TREASURY

---

## Week 5: Omni-Channel Notification System

### Goals
- ✅ Notification dispatcher architecture implemented
- ✅ 3+ notification channels working
- ✅ Retry logic functional
- ✅ Delivery tracking enabled

### Daily Tasks

#### **Monday: Notification Architecture & Database**
**Backend Team**:
- [ ] Create migration `009_create_notification_logs.up.sql`
  - notification_logs table (id, payment_id, merchant_id, channel, status, sent_at, delivered_at, error_message, retry_count, created_at)
- [ ] Create migration `010_create_merchant_notification_preferences.up.sql`
  - merchant_notification_preferences table (merchant_id, channel, enabled, config JSONB)
- [ ] Create `internal/model/notification_log.go`
- [ ] Create `internal/notification/types.go` (NotificationMessage, NotificationPlugin interface)
- [ ] Define NotificationPlugin interface:
  ```go
  type NotificationPlugin interface {
    Name() string
    Send(message NotificationMessage) error
    ValidateConfig(config map[string]interface{}) error
  }
  ```

**Deliverable**: Notification database schema + architecture

---

#### **Tuesday: Email Notification Plugin**
**Backend Team**:
- [ ] Sign up for SendGrid account
- [ ] Create `internal/notification/plugins/email/email_plugin.go`
- [ ] Implement NotificationPlugin interface
- [ ] Create email templates (payment_confirmed.html, payment_failed.html)
- [ ] Implement Send() method using SendGrid API
- [ ] Test email delivery

**Frontend Team**:
- [ ] Create notification preferences page in merchant dashboard
- [ ] Add toggle for email notifications
- [ ] Add input field for notification email address

**Deliverable**: Email notifications working

---

#### **Wednesday: Telegram Bot Plugin**
**Backend Team**:
- [ ] Create Telegram bot via @BotFather
- [ ] Get bot token
- [ ] Create `internal/notification/plugins/telegram/telegram_plugin.go`
- [ ] Implement NotificationPlugin interface
- [ ] Implement Send() method using Telegram Bot API
- [ ] Handle rate limiting (30 messages/second)
- [ ] Create bot command: /start (link Telegram to merchant account)
- [ ] Create bot command: /unlink

**Frontend Team**:
- [ ] Add Telegram toggle in notification preferences
- [ ] Display QR code for Telegram bot /start link
- [ ] Show connected Telegram chat ID

**Deliverable**: Telegram notifications working

---

#### **Thursday: Webhook Plugin & Notification Dispatcher**
**Backend Team**:
- [ ] Create `internal/notification/plugins/webhook/webhook_plugin.go`
- [ ] Implement HMAC signature generation for webhooks
- [ ] Implement retry logic with exponential backoff (1min, 5min, 15min)
- [ ] Create `internal/notification/dispatcher.go`
- [ ] Implement plugin registration system
- [ ] Implement NotifyPaymentConfirmed() method:
  - Load merchant notification preferences
  - For each enabled channel, send notification
  - Log result in notification_logs table
  - Queue retry job if failed (max 3 attempts)
- [ ] Install Bull queue library for job management
- [ ] Create Redis-backed job queue for retries

**Deliverable**: Notification dispatcher with retry logic

---

#### **Friday: Zalo ZNS Integration (Optional) + Testing**
**Backend Team**:
- [ ] Research Zalo Official Account (OA) + Zalo Notification Service (ZNS)
- [ ] Sign up for Zalo OA (if feasible)
- [ ] Create `internal/notification/plugins/zalo/zalo_plugin.go`
- [ ] Implement Send() method
- [ ] Test ZNS message delivery (note: free tier = 100 messages/day)

**QA Team**:
- [ ] Test email notifications (payment confirmed, payment expired)
- [ ] Test Telegram notifications
- [ ] Test webhook delivery with HMAC signature validation
- [ ] Test retry logic (disable merchant webhook URL, verify 3 retries)
- [ ] Measure notification delivery rate (target >95%)

**Deliverable**: Multi-channel notifications functional

**Week 5 Retrospective**

---

## Week 6: Custodial Treasury & Auto-Sweeping

### Goals
- ✅ Hot + cold wallet architecture implemented
- ✅ Multi-sig cold wallet configured
- ✅ Auto-sweeping working (6-hour intervals)
- ✅ Treasury monitoring dashboard

### Daily Tasks

#### **Monday: Treasury Database & Models**
**Backend Team**:
- [ ] Create migration `011_create_treasury_wallets.up.sql`
  - treasury_wallets table (id, wallet_type, blockchain, address, balance_crypto, balance_usd, last_swept_at, created_at)
  - wallet_type: 'hot' or 'cold'
- [ ] Create migration `012_create_treasury_operations.up.sql`
  - treasury_operations table (id, operation_type, from_wallet_id, to_wallet_id, amount, chain, tx_hash, status, initiated_by, created_at, completed_at)
  - operation_type: 'sweep', 'manual_transfer', 'otc_settlement'
- [ ] Create `internal/model/treasury_wallet.go`
- [ ] Create `internal/model/treasury_operation.go`

**Deliverable**: Treasury database schema

---

#### **Tuesday: Multi-Sig Cold Wallet Setup**
**Blockchain Team**:
- [ ] Research multi-sig wallet solutions:
  - TRON: TronLink multi-sig or Gnosis Safe equivalent
  - Solana: Squads Protocol or Goki multi-sig
  - BSC: Gnosis Safe
- [ ] Create 2-of-3 multi-sig cold wallet for each chain
- [ ] Generate 3 signing keys, distribute to team (2 required to sign)
- [ ] Test multi-sig transaction on testnet:
  - Create transaction
  - Sign with key 1
  - Sign with key 2
  - Broadcast transaction
- [ ] Document cold wallet addresses in treasury_wallets table

**DevOps Team**:
- [ ] Set up secure key storage (AWS Secrets Manager or HashiCorp Vault)
- [ ] Configure access policies (only authorized team members can access)

**Deliverable**: Multi-sig cold wallets configured

---

#### **Wednesday: Hot Wallet Sweeping Logic**
**Backend Team**:
- [ ] Create `internal/service/treasury_service.go`
- [ ] Implement CheckHotWalletBalance(chain):
  - Query hot wallet balance via RPC
  - Convert to USD equivalent
  - Return balance
- [ ] Implement ShouldSweep(chain) logic:
  - If balance > $10,000, return true
  - Calculate sweep amount (balance - $5,000 buffer)
- [ ] Implement CreateSweepTransaction(chain, amount):
  - Create unsigned transaction from hot → cold wallet
  - If chain supports multi-sig, create multi-sig tx
  - Return transaction for signing
- [ ] Implement ExecuteSweep(chain, signedTx):
  - Broadcast transaction to blockchain
  - Log in treasury_operations table
  - Update treasury_wallets balances

**Deliverable**: Sweeping logic implemented

---

#### **Thursday: Automated Sweeping Cron Job**
**Backend Team**:
- [ ] Install cron library: `github.com/robfig/cron/v3`
- [ ] Create `internal/worker/treasury_sweeper.go`
- [ ] Implement sweeping cron job (runs every 6 hours):
  - For each chain (TRON, Solana, BSC):
    - Check hot wallet balance
    - If balance > $10k threshold:
      - Calculate sweep amount
      - Create multi-sig transaction
      - Send alert to ops team (Telegram/Email)
      - Wait for manual approval (for now)
      - Execute sweep after approval
- [ ] Create API endpoint: POST /admin/treasury/approve-sweep/:operation_id
- [ ] Test sweeping on testnet (fund hot wallet with $11k equivalent, trigger sweep)

**DevOps Team**:
- [ ] Configure cron job to start on server boot
- [ ] Set up monitoring alerts for failed sweeps

**Deliverable**: Auto-sweeping functional

---

#### **Friday: Treasury Monitoring Dashboard**
**Frontend Team**:
- [ ] Create admin panel treasury page (`app/admin/treasury/page.tsx`)
- [ ] Display hot wallet balances (TRON, Solana, BSC) in real-time
- [ ] Display cold wallet balances
- [ ] Show pending sweep operations (awaiting approval)
- [ ] Add "Approve Sweep" button for pending operations
- [ ] Display treasury operation history (last 50 operations)

**Backend Team**:
- [ ] Create API endpoint: GET /admin/treasury/wallets
- [ ] Create API endpoint: GET /admin/treasury/operations
- [ ] Create API endpoint: GET /admin/treasury/pending-sweeps

**QA Team**:
- [ ] Test complete treasury flow:
  1. Fund hot wallet with test crypto
  2. Wait for balance > $10k
  3. Verify sweep operation created
  4. Admin approves sweep
  5. Verify crypto transferred to cold wallet
  6. Check treasury_operations log

**Deliverable**: Treasury monitoring dashboard live

**Week 6 Retrospective**

**Phase 2 Complete**: Notifications + Treasury functional

---

## PHASE 3: DATA RETENTION & OFF-RAMP

---

## Week 7: Infinite Data Retention & Transaction Hashing

### Goals
- ✅ S3 Glacier archival working
- ✅ SHA-256 hash chain implemented
- ✅ Merkle tree verification functional
- ✅ Data restore process tested

### Daily Tasks

#### **Monday: S3 Glacier Setup & Database Schema**
**DevOps Team**:
- [ ] Create AWS account (if not exists)
- [ ] Create S3 bucket: `payment-gateway-archives`
- [ ] Configure bucket lifecycle policy (move to Glacier after 0 days)
- [ ] Set up IAM user with S3 access permissions
- [ ] Generate AWS access keys

**Backend Team**:
- [ ] Create migration `013_create_archived_records.up.sql`
  - archived_records table (id, original_id, table_name, archive_path, data_hash, archived_at, restored_at)
- [ ] Create migration `014_create_transaction_hashes.up.sql`
  - transaction_hashes table (id, table_name, record_id, data_hash, previous_hash, merkle_root, created_at)
- [ ] Install AWS SDK: `github.com/aws/aws-sdk-go`
- [ ] Create `internal/archival/s3_client.go` (S3 upload/download wrapper)

**Deliverable**: S3 Glacier configured + database schema

---

#### **Tuesday: Transaction Hashing Implementation**
**Backend Team**:
- [ ] Create `internal/archival/hash_service.go`
- [ ] Implement ComputeHash(tableName, recordId, data):
  - Serialize record to JSON (canonical ordering)
  - Compute SHA-256 hash
  - Store in transaction_hashes table
  - Link to previous hash (hash chain)
- [ ] Update payment creation flow to compute hash after insert
- [ ] Update payout creation flow to compute hash
- [ ] Update ledger entry creation to compute hash
- [ ] Write unit tests for hash computation

**Deliverable**: Transaction hashing working

---

#### **Wednesday: Merkle Tree Computation**
**Backend Team**:
- [ ] Create `internal/archival/merkle_tree.go`
- [ ] Implement BuildMerkleTree(hashes []string):
  - Take array of transaction hashes
  - Build binary Merkle tree
  - Return root hash
- [ ] Implement daily cron job (runs at 2 AM UTC):
  - Get all transaction hashes created yesterday
  - Build Merkle tree
  - Store root hash in transaction_hashes table
- [ ] Write unit tests for Merkle tree construction

**Deliverable**: Daily Merkle root computation

---

#### **Thursday: Monthly Archival Job**
**Backend Team**:
- [ ] Create `internal/worker/archival_worker.go`
- [ ] Implement monthly archival cron job (1st of month, 2 AM UTC):
  - Find records > 12 months old (payments, payouts, ledger_entries)
  - For each batch of 1000 records:
    - Compute SHA-256 hash for each record
    - Compress batch using gzip
    - Upload to S3 Glacier
    - Create archived_records entry
    - Mark original records as archived (add `archived` boolean column)
- [ ] Test archival on staging with sample data

**DevOps Team**:
- [ ] Monitor S3 Glacier costs (estimate: $4/TB/month)
- [ ] Set up CloudWatch alerts for archival job failures

**Deliverable**: Monthly archival job functional

---

#### **Friday: Data Restore Process**
**Backend Team**:
- [ ] Create `internal/archival/restore_service.go`
- [ ] Implement RestoreArchive(archiveId):
  - Initiate Glacier restore request (Expedited tier = 1-5 hours)
  - Poll restore status every 30 minutes
  - When ready, download archive from S3
  - Decompress gzip
  - Verify SHA-256 hash matches archived_records.data_hash
  - Return restored data
- [ ] Create API endpoint: POST /admin/archives/restore
- [ ] Test restore process on staging

**QA Team**:
- [ ] Test complete archival flow:
  1. Create old test data (simulate 12+ months old)
  2. Run archival job manually
  3. Verify data uploaded to S3 Glacier
  4. Verify archived_records entry created
  5. Initiate restore
  6. Verify restored data matches original
  7. Verify hash integrity

**Deliverable**: Data restore process tested

**Week 7 Retrospective**

---

## Week 8: Advanced Off-Ramp Strategies

### Goals
- ✅ Scheduled withdrawals working
- ✅ Threshold-based withdrawals working
- ✅ Payout scheduler functional
- ✅ Merchant withdrawal configuration UI complete

### Daily Tasks

#### **Monday: Payout Scheduling Database**
**Backend Team**:
- [ ] Create migration `015_create_payout_schedules.up.sql`
  - payout_schedules table (merchant_id UNIQUE, scheduled_enabled, scheduled_frequency, scheduled_day_of_week, scheduled_day_of_month, scheduled_time, scheduled_withdraw_percentage, threshold_enabled, threshold_usdt, threshold_withdraw_percentage, last_triggered_at)
- [ ] Create `internal/model/payout_schedule.go`
- [ ] Create `internal/repository/payout_schedule_repository.go`

**Deliverable**: Payout scheduling schema

---

#### **Tuesday: Scheduled Withdrawal Logic**
**Backend Team**:
- [ ] Create `internal/service/payout_scheduler_service.go`
- [ ] Implement CheckScheduledPayouts() logic:
  - Query payout_schedules where scheduled_enabled = true
  - For weekly schedules:
    - Check if today is scheduled_day_of_week AND scheduled_time has passed
  - For monthly schedules:
    - Check if today is scheduled_day_of_month AND scheduled_time has passed
  - If match, create payout request:
    - Calculate amount = available_balance * scheduled_withdraw_percentage / 100
    - Create payout record (status: `scheduled`)
- [ ] Implement daily cron job (runs every hour):
  - Call CheckScheduledPayouts()
  - Create pending payouts for matched schedules

**Deliverable**: Scheduled withdrawal logic

---

#### **Wednesday: Threshold-Based Withdrawal Logic**
**Backend Team**:
- [ ] Implement CheckThresholdPayouts() logic:
  - Query payout_schedules where threshold_enabled = true
  - For each merchant:
    - Get current available_balance (in VND)
    - Convert threshold_usdt to VND (using exchange rate)
    - If available_balance >= threshold_vnd:
      - Calculate amount = available_balance * threshold_withdraw_percentage / 100
      - Create payout request (status: `threshold_triggered`)
- [ ] Add threshold checking to payment confirmation flow:
  - After payment confirmed and balance updated
  - Call CheckThresholdPayouts() for that merchant

**Deliverable**: Threshold-based withdrawal logic

---

#### **Thursday: Merchant Payout Configuration UI**
**Frontend Team**:
- [ ] Create payout settings page (`app/dashboard/settings/payouts/page.tsx`)
- [ ] Add "On-Demand Withdrawals" section (always enabled)
- [ ] Add "Scheduled Withdrawals" section:
  - Toggle to enable/disable
  - Dropdown: frequency (weekly/monthly)
  - For weekly: day of week selector
  - For monthly: day of month input (1-31)
  - Time picker (HH:mm)
  - Slider: withdrawal percentage (10-100%)
- [ ] Add "Threshold-Based Withdrawals" section:
  - Toggle to enable/disable
  - Input: threshold amount (USDT)
  - Slider: withdrawal percentage (10-100%)
- [ ] Add "Preview" showing next scheduled payout date/time

**Backend Team**:
- [ ] Create API endpoint: GET /api/v1/merchants/me/payout-schedule
- [ ] Create API endpoint: PUT /api/v1/merchants/me/payout-schedule
- [ ] Validate inputs (frequency, day_of_week, day_of_month, percentage)

**Deliverable**: Payout configuration UI complete

---

#### **Friday: End-to-End Payout Testing**
**QA Team**:
- [ ] Test scheduled withdrawals:
  - Configure weekly schedule (every Monday at 10 AM, 80% withdrawal)
  - Simulate time passage (change system clock or wait)
  - Verify payout created automatically
- [ ] Test threshold-based withdrawals:
  - Configure threshold ($1000 USDT, 90% withdrawal)
  - Make payments until balance exceeds threshold
  - Verify payout auto-created
- [ ] Test manual on-demand withdrawals still work
- [ ] Test disabling scheduled/threshold withdrawals
- [ ] Verify merchant can see payout history

**Backend Team**:
- [ ] Fix bugs found in testing
- [ ] Add logging for all payout scheduler actions

**Deliverable**: Advanced off-ramp functional

**Week 8 Retrospective**

**Phase 3 Complete**: Data retention + off-ramp strategies ready

---

## PHASE 4: TESTING & LAUNCH

---

## Week 9: Comprehensive Testing & Security Audit

### Goals
- ✅ All unit tests passing (>80% coverage)
- ✅ Integration tests complete
- ✅ Performance testing done (100+ concurrent payments)
- ✅ Security audit completed
- ✅ All critical bugs fixed

### Daily Tasks

#### **Monday: Unit Test Coverage Push**
**Backend Team**:
- [ ] Run test coverage report: `go test -cover ./...`
- [ ] Identify uncovered code paths
- [ ] Write missing unit tests for:
  - Payment service
  - Ledger service
  - Identity mapping service
  - Notification dispatcher
  - Treasury service
  - Archival service
- [ ] Target: >80% code coverage
- [ ] Configure GitHub Actions to fail PR if coverage drops below 80%

**Frontend Team**:
- [ ] Write unit tests for React components using Jest + React Testing Library
- [ ] Test critical flows (payment creation, dashboard, settings)
- [ ] Target: >70% frontend coverage

**Deliverable**: >80% backend test coverage achieved

---

#### **Tuesday: Integration Testing**
**QA Team**:
- [ ] Write integration test suite (use Postman or custom Go tests):
  - Merchant registration flow
  - Payment creation + confirmation (all 3 chains)
  - Identity mapping (new user + returning user)
  - Notification delivery (email, Telegram, webhook)
  - Payout request + approval
  - Treasury sweeping
  - Data archival + restore
- [ ] Run integration tests on staging environment
- [ ] Document test results

**Backend Team**:
- [ ] Fix integration test failures
- [ ] Add database seeding scripts for test data

**Deliverable**: Integration test suite passing

---

#### **Wednesday: Performance Testing**
**QA Team**:
- [ ] Set up load testing tool (k6 or Artillery)
- [ ] Write load test scripts:
  - Scenario 1: 100 concurrent payment creations
  - Scenario 2: 50 concurrent payment confirmations
  - Scenario 3: 1000 requests/minute to merchant dashboard
- [ ] Run load tests on staging
- [ ] Measure:
  - Average response time (target <500ms for APIs)
  - P95 latency (target <1s)
  - Error rate (target <1%)
  - Database connection pool usage
- [ ] Generate performance report

**Backend Team**:
- [ ] Optimize slow API endpoints (add indexes, caching)
- [ ] Add database connection pooling if not configured
- [ ] Optimize blockchain listener performance

**DevOps Team**:
- [ ] Monitor server resources during load test (CPU, memory, disk I/O)
- [ ] Scale resources if needed

**Deliverable**: Performance benchmarks documented

---

#### **Thursday: Security Audit**
**All Teams**:
- [ ] Run security scanning tools:
  - Backend: `gosec` for Go code
  - Frontend: `npm audit` for dependencies
  - Infrastructure: Nmap for open ports
- [ ] Manual security review:
  - Check for SQL injection vulnerabilities (all queries parameterized?)
  - Check for XSS vulnerabilities (all user input sanitized?)
  - Review authentication logic (API keys secure? JWT expiry set?)
  - Review webhook signature validation
  - Check for exposed secrets (no .env in git?)
  - Review rate limiting (prevents DDoS?)
- [ ] Penetration testing:
  - Attempt to bypass authentication
  - Attempt to manipulate payment amounts
  - Attempt to trigger race conditions
  - Test CORS configuration
- [ ] Document findings in security audit report

**Backend Team**:
- [ ] Fix high/critical security issues immediately
- [ ] Plan fixes for medium/low issues

**Deliverable**: Security audit report + fixes deployed

---

#### **Friday: Bug Bash & Final Fixes**
**All Teams**:
- [ ] Organize bug bash session (entire team tests for 4 hours)
- [ ] Test all features end-to-end
- [ ] Document bugs in issue tracker (priority: Critical, High, Medium, Low)
- [ ] Triage bugs:
  - Critical/High: Must fix before launch
  - Medium: Fix if time permits
  - Low: Move to post-launch backlog
- [ ] Fix critical and high priority bugs

**Product Owner**:
- [ ] Review all features against PRD v2.2
- [ ] Sign off on deliverables

**Deliverable**: All critical bugs fixed, launch-ready code

**Week 9 Retrospective**

---

## Week 10: Production Deployment & Pilot Launch

### Goals
- ✅ Production environment ready
- ✅ Mainnet wallets configured
- ✅ 5 pilot merchants onboarded
- ✅ Production monitoring live
- ✅ Launch successful

### Daily Tasks

#### **Monday: Production Environment Setup**
**DevOps Team**:
- [ ] Provision production server (8 CPU, 16GB RAM, 200GB SSD)
- [ ] Install Docker + Docker Compose
- [ ] Set up PostgreSQL 15 with replication
- [ ] Set up Redis 7 with persistence
- [ ] Configure SSL certificates (Let's Encrypt)
- [ ] Set up NGINX reverse proxy
- [ ] Configure firewall (allow only ports 80, 443, 22)
- [ ] Set up backup automation (daily PostgreSQL backups to S3)
- [ ] Configure monitoring (Prometheus + Grafana or Datadog)
- [ ] Set up log aggregation (ELK stack or CloudWatch)

**Backend Team**:
- [ ] Create production .env file (use mainnet RPC endpoints)
- [ ] Generate production API keys
- [ ] Configure production database (run migrations)
- [ ] Deploy backend to production
- [ ] Verify health endpoint: https://api.payment-gateway.com/health

**Deliverable**: Production environment live

---

#### **Tuesday: Mainnet Wallet Configuration**
**Blockchain Team**:
- [ ] Create mainnet hot wallets (TRON, Solana, BSC)
- [ ] Create mainnet multi-sig cold wallets (2-of-3)
- [ ] Fund hot wallets with small initial amount:
  - TRON: $1000 USDT + 100 TRX (gas)
  - Solana: $1000 USDC + 10 SOL (gas)
  - BSC: $1000 USDT + 1 BNB (gas)
- [ ] Update treasury_wallets table with mainnet addresses
- [ ] Test blockchain listeners on mainnet (send test transactions)
- [ ] Verify payment confirmation works on mainnet

**Security Team**:
- [ ] Store hot wallet private keys in AWS Secrets Manager
- [ ] Distribute cold wallet signing keys to authorized team members
- [ ] Document key management procedures

**Deliverable**: Mainnet wallets operational

---

#### **Wednesday: Pilot Merchant Onboarding**
**Product Owner**:
- [ ] Onboard 5 pilot merchants:
  1. Hotel in Da Nang
  2. Restaurant in Da Nang
  3. Tourist tour operator
  4. Souvenir shop
  5. Coffee shop
- [ ] For each merchant:
  - Complete merchant registration
  - Submit KYC documents
  - Approve KYC in admin panel
  - Generate API key
  - Configure webhook URL (if applicable)
  - Set up bank account for payouts
  - Configure notification preferences
  - Train merchant on dashboard usage

**Frontend Team**:
- [ ] Create onboarding documentation (PDF + video)
- [ ] Create merchant FAQ page

**Deliverable**: 5 merchants onboarded and trained

---

#### **Thursday: Production Monitoring & Alerting**
**DevOps Team**:
- [ ] Set up monitoring dashboards:
  - System metrics (CPU, memory, disk, network)
  - Application metrics (API response times, error rates)
  - Blockchain metrics (listener lag, transaction confirmation times)
  - Business metrics (payment volume, success rate, revenue)
- [ ] Configure alerts:
  - Critical: API error rate >5% (alert via Telegram + email)
  - Critical: Payment success rate <95%
  - Warning: Database connection pool >80% used
  - Warning: Hot wallet balance <$500
  - Info: Daily payment volume report
- [ ] Set up on-call rotation
- [ ] Document incident response procedures

**Backend Team**:
- [ ] Add application metrics (using Prometheus client)
- [ ] Add custom metrics for business KPIs

**Deliverable**: Production monitoring live

---

#### **Friday: Launch Day 🚀**
**Morning (09:00-12:00)**:
- [ ] Final production smoke test:
  - Create test payment on all 3 chains
  - Verify payment confirmation
  - Verify notifications sent
  - Verify ledger entries created
  - Test payout request
- [ ] Announcement:
  - Update website with "Live" banner
  - Post on social media
  - Send email to pilot merchants
  - Notify stakeholders

**Afternoon (13:00-18:00)**:
- [ ] Monitor production systems closely
- [ ] Support pilot merchants (respond to questions)
- [ ] Track first real payments
- [ ] Celebrate first successful payment 🎉

**Evening (18:00-20:00)**:
- [ ] Review launch day metrics:
  - Total payment volume
  - Payment success rate
  - Average confirmation time
  - Notification delivery rate
  - Any errors or issues
- [ ] Document lessons learned

**Deliverable**: Production launch successful!

**Week 10 Retrospective**

---

## Post-Launch Support (Week 11+)

### Immediate Priorities
1. **Monitor KPIs daily** (first 2 weeks)
   - Payment success rate
   - System uptime
   - Merchant satisfaction
2. **Fix production bugs** as they arise
3. **Gather merchant feedback** weekly
4. **Optimize performance** based on real usage patterns

### Month 1 Goals
- 1B+ VND transaction volume
- Payment success rate >98%
- KYC recognition rate >95%
- Notification delivery >95%
- Zero security incidents
- NPS >30

---

## 📊 Risk Management

### High-Risk Items

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Blockchain RPC downtime** | Critical - payments fail | Use multiple RPC providers (primary + fallback), implement circuit breaker |
| **Hot wallet compromise** | Critical - loss of funds | Store keys in Secrets Manager, implement sweeping (max $10k exposure), monitor for suspicious transactions |
| **KYC provider (Sumsub) downtime** | High - new users can't pay | Implement queue system, fallback to manual KYC |
| **Database failure** | Critical - system unusable | Set up PostgreSQL replication, automated backups, test restore procedures |
| **Exchange rate API failure** | High - can't create payments | Use multiple exchange rate sources (CoinGecko + fallback), cache last known rates |
| **S3 Glacier restore delay** | Medium - delayed data access | Document SLA (1-5 hours Expedited), set expectations with stakeholders |
| **Multi-sig key loss** | Critical - can't access cold wallet | Implement key recovery procedures, backup keys in secure locations |
| **Team member unavailable** | Medium - delayed development | Cross-train team members, document critical procedures |

### Medium-Risk Items

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Notification delivery failure** | Medium - merchant misses alerts | Implement retry logic (3 attempts), support multiple channels |
| **Payment confirmation delay** | Medium - poor UX | Display pending status clearly, set realistic expectations (Solana 13s, BSC 3min) |
| **Ledger imbalance** | High - accounting errors | Daily reconciliation job, automated balance checks, alerts on discrepancies |
| **Performance degradation** | Medium - slow UX | Implement caching, optimize database queries, scale horizontally if needed |

---

## 📈 Success Metrics Tracking

### Daily Metrics (Monitor in real-time)
- Payment volume (VND)
- Payment count
- Payment success rate
- Average confirmation time
- Error rate (by type)
- System uptime

### Weekly Metrics
- New merchants onboarded
- Active merchants (made ≥1 payment)
- KYC completion rate
- Notification delivery rate (by channel)
- Treasury sweep success rate
- Average merchant balance

### Monthly Metrics
- Total transaction volume
- Revenue (transaction fees + payout fees)
- NPS score
- Merchant retention rate
- Multi-chain adoption breakdown

---

## 🎓 Team Responsibilities

### Backend Team
- **Week 1-2**: Database, APIs, blockchain integration
- **Week 3**: Identity mapping, KYC integration
- **Week 4**: Ledger system, merchant APIs
- **Week 5**: Notification system
- **Week 6**: Treasury service
- **Week 7**: Data archival
- **Week 8**: Payout scheduler
- **Week 9-10**: Testing, deployment

### Frontend Team
- **Week 1-2**: Dashboard layout, authentication
- **Week 3**: KYC flow UI
- **Week 4**: Transaction pages, analytics
- **Week 5**: Notification preferences
- **Week 6**: Treasury monitoring
- **Week 7-8**: Payout settings
- **Week 9-10**: Bug fixes, polish

### Blockchain Team
- **Week 1-2**: Multi-chain listener implementation
- **Week 3**: Wallet integration for KYC
- **Week 4**: Transaction validation
- **Week 6**: Multi-sig wallets, sweeping
- **Week 7**: Hash verification
- **Week 9-10**: Mainnet deployment

### DevOps Team
- **Week 1**: Infrastructure setup
- **Week 2-8**: CI/CD, monitoring, AWS services
- **Week 9**: Performance testing, security audit
- **Week 10**: Production deployment

### QA Team
- **Week 1-8**: Feature testing as developed
- **Week 9**: Comprehensive testing, security audit
- **Week 10**: Production smoke testing

---

## 📝 Documentation Deliverables

Throughout the 10 weeks, maintain:
- ✅ API documentation (Swagger/OpenAPI)
- ✅ Database schema documentation
- ✅ Deployment runbooks
- ✅ Incident response procedures
- ✅ Merchant onboarding guide
- ✅ Admin panel user guide
- ✅ Code comments and README files

---

## 🎉 Launch Checklist

### Week 10 Pre-Launch Checklist

**Technical**:
- [ ] All unit tests passing (>80% coverage)
- [ ] All integration tests passing
- [ ] Performance benchmarks met
- [ ] Security audit completed, critical issues fixed
- [ ] Production environment configured
- [ ] Mainnet wallets funded
- [ ] SSL certificates installed
- [ ] Monitoring and alerting configured
- [ ] Backup automation tested
- [ ] Disaster recovery plan documented

**Business**:
- [ ] 5 pilot merchants onboarded
- [ ] Merchant training completed
- [ ] KYC documents approved
- [ ] Bank accounts configured for payouts
- [ ] Legal compliance verified
- [ ] OTC partner identified and contacted
- [ ] Support procedures documented

**Communication**:
- [ ] Launch announcement prepared
- [ ] Marketing materials ready
- [ ] FAQ page published
- [ ] Support email/chat configured

---

**Good luck with the implementation! 🚀**

**Questions or need adjustments to the roadmap?** Consult the Product Owner or update this document as the project evolves.
