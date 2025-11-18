# Backend Task Breakdown - Stablecoin Payment Gateway MVP

**Tech Stack**: Golang + PostgreSQL + Redis + Solana
**Timeline**: 4-6 weeks
**Last Updated**: 2025-11-16

---

## 📋 Task Organization

Tasks are organized into 8 major categories:
1. Infrastructure & Project Setup
2. Database Layer
3. Core Services
4. Blockchain Integration
5. API Layer
6. Authentication & Security
7. Background Jobs & Workers
8. Testing & Deployment

---

## 1️⃣ INFRASTRUCTURE & PROJECT SETUP

### 1.1 Project Structure Setup
- [ ] **Task**: Initialize Go project with standard layout
  - [ ] Create directory structure (`cmd/`, `internal/`, `pkg/`)
  - [ ] Initialize `go.mod` with project dependencies
  - [ ] Set up `cmd/api/` for REST API server
  - [ ] Set up `cmd/listener/` for blockchain listener
  - [ ] Set up `cmd/worker/` for background jobs
  - [ ] Create `internal/` subdirectories (api, service, repository, model, config)
  - **Files**: Project root structure
  - **Dependencies**: Go 1.21+
  - **Estimated Time**: 2 hours

### 1.2 Configuration Management
- [ ] **Task**: Implement environment-based configuration
  - [ ] Create `internal/config/config.go` - config struct and loader
  - [ ] Support `.env` file loading (use `godotenv`)
  - [ ] Define config for: database, Redis, blockchain RPCs, API settings
  - [ ] Add config validation on startup
  - [ ] Document all environment variables in `.env.example`
  - **Files**: `internal/config/config.go`, `.env.example`
  - **Dependencies**: `github.com/joho/godotenv`
  - **Estimated Time**: 3 hours

### 1.3 Docker Development Environment
- [ ] **Task**: Create Docker Compose for local development
  - [ ] `docker-compose.yml` with PostgreSQL, Redis, MinIO
  - [ ] Create Dockerfile for API service
  - [ ] Create Dockerfile for listener service
  - [ ] Add health check scripts
  - [ ] Document local setup in `GETTING_STARTED.md`
  - **Files**: `docker-compose.yml`, `Dockerfile.api`, `Dockerfile.listener`
  - **Dependencies**: Docker, Docker Compose
  - **Estimated Time**: 4 hours

### 1.4 Logging & Monitoring Setup
- [ ] **Task**: Implement structured logging
  - [ ] Set up logger with `github.com/sirupsen/logrus` or `go.uber.org/zap`
  - [ ] Create logging middleware for HTTP requests
  - [ ] Implement log levels (debug, info, warn, error)
  - [ ] Add correlation IDs for request tracing
  - [ ] Configure log output format (JSON for production)
  - **Files**: `internal/pkg/logger/logger.go`
  - **Dependencies**: Logging library
  - **Estimated Time**: 3 hours

---

## 2️⃣ DATABASE LAYER

### 2.1 Database Schema & Migrations
- [ ] **Task**: Create initial database schema
  - [ ] Install migration tool (`golang-migrate/migrate`)
  - [ ] Create migration: `001_create_merchants_table.up.sql`
  - [ ] Create migration: `002_create_payments_table.up.sql`
  - [ ] Create migration: `003_create_payouts_table.up.sql`
  - [ ] Create migration: `004_create_ledger_entries_table.up.sql`
  - [ ] Create migration: `005_create_merchant_balances_table.up.sql`
  - [ ] Create migration: `006_create_audit_logs_table.up.sql`
  - [ ] Create migration: `007_create_blockchain_transactions_table.up.sql`
  - [ ] Create corresponding `.down.sql` files for rollback
  - [ ] Add indexes for performance (see ARCHITECTURE.md)
  - **Files**: `migrations/00X_*.sql`
  - **Dependencies**: `golang-migrate/migrate`
  - **Estimated Time**: 6 hours

### 2.2 Database Connection Pool
- [ ] **Task**: Implement database connection management
  - [ ] Create `internal/pkg/database/postgres.go`
  - [ ] Set up connection pool with `database/sql` + `lib/pq`
  - [ ] Configure pool settings (max connections, idle timeout)
  - [ ] Implement health check function
  - [ ] Add graceful shutdown handling
  - **Files**: `internal/pkg/database/postgres.go`
  - **Dependencies**: `github.com/lib/pq`
  - **Estimated Time**: 2 hours

### 2.3 Domain Models
- [ ] **Task**: Define Go structs for domain entities
  - [ ] `internal/model/merchant.go` - Merchant struct
  - [ ] `internal/model/payment.go` - Payment struct with status enum
  - [ ] `internal/model/payout.go` - Payout struct
  - [ ] `internal/model/ledger.go` - LedgerEntry struct
  - [ ] `internal/model/balance.go` - MerchantBalance struct
  - [ ] `internal/model/audit.go` - AuditLog struct
  - [ ] `internal/model/blockchain.go` - BlockchainTransaction struct
  - [ ] Add JSON tags for API serialization
  - [ ] Add validation tags using `github.com/go-playground/validator`
  - **Files**: `internal/model/*.go`
  - **Dependencies**: `github.com/go-playground/validator/v10`
  - **Estimated Time**: 4 hours

### 2.4 Repository Layer - Merchants
- [ ] **Task**: Implement merchant data access
  - [ ] Create `internal/repository/merchant.go`
  - [ ] Implement `Create(merchant *model.Merchant) error`
  - [ ] Implement `GetByID(id string) (*model.Merchant, error)`
  - [ ] Implement `GetByEmail(email string) (*model.Merchant, error)`
  - [ ] Implement `GetByAPIKey(apiKey string) (*model.Merchant, error)`
  - [ ] Implement `Update(merchant *model.Merchant) error`
  - [ ] Implement `UpdateKYCStatus(id string, status string) error`
  - [ ] Use prepared statements to prevent SQL injection
  - **Files**: `internal/repository/merchant.go`
  - **Estimated Time**: 4 hours

### 2.5 Repository Layer - Payments
- [ ] **Task**: Implement payment data access
  - [ ] Create `internal/repository/payment.go`
  - [ ] Implement `Create(payment *model.Payment) error`
  - [ ] Implement `GetByID(id string) (*model.Payment, error)`
  - [ ] Implement `GetByTxHash(txHash string) (*model.Payment, error)`
  - [ ] Implement `UpdateStatus(id, status string) error`
  - [ ] Implement `ListByMerchant(merchantID string, limit, offset int) ([]*model.Payment, error)`
  - [ ] Implement `GetExpiredPayments() ([]*model.Payment, error)`
  - [ ] Use `decimal.Decimal` for money amounts
  - **Files**: `internal/repository/payment.go`
  - **Dependencies**: `github.com/shopspring/decimal`
  - **Estimated Time**: 5 hours

### 2.6 Repository Layer - Payouts
- [ ] **Task**: Implement payout data access
  - [ ] Create `internal/repository/payout.go`
  - [ ] Implement `Create(payout *model.Payout) error`
  - [ ] Implement `GetByID(id string) (*model.Payout, error)`
  - [ ] Implement `ListByMerchant(merchantID string) ([]*model.Payout, error)`
  - [ ] Implement `ListPending() ([]*model.Payout, error)`
  - [ ] Implement `UpdateStatus(id, status string) error`
  - [ ] Implement `MarkCompleted(id, referenceNumber string) error`
  - **Files**: `internal/repository/payout.go`
  - **Estimated Time**: 3 hours

### 2.7 Repository Layer - Ledger
- [ ] **Task**: Implement ledger data access
  - [ ] Create `internal/repository/ledger.go`
  - [ ] Implement `CreateEntry(entry *model.LedgerEntry) error`
  - [ ] Implement `CreateEntries(entries []*model.LedgerEntry) error` (transaction)
  - [ ] Implement `GetByMerchant(merchantID string, limit, offset int) ([]*model.LedgerEntry, error)`
  - [ ] Implement `GetByReference(refType, refID string) ([]*model.LedgerEntry, error)`
  - [ ] Ensure atomic operations with database transactions
  - **Files**: `internal/repository/ledger.go`
  - **Estimated Time**: 4 hours

### 2.8 Repository Layer - Balances
- [ ] **Task**: Implement balance data access
  - [ ] Create `internal/repository/balance.go`
  - [ ] Implement `GetByMerchantID(merchantID string) (*model.MerchantBalance, error)`
  - [ ] Implement `UpdateBalance(merchantID string, balance *model.MerchantBalance) error`
  - [ ] Implement `IncrementAvailable(merchantID string, amount decimal.Decimal) error`
  - [ ] Implement `DecrementAvailable(merchantID string, amount decimal.Decimal) error`
  - [ ] Add row locking for concurrent updates (`SELECT ... FOR UPDATE`)
  - **Files**: `internal/repository/balance.go`
  - **Estimated Time**: 3 hours

### 2.9 Repository Layer - Audit Logs
- [ ] **Task**: Implement audit log data access
  - [ ] Create `internal/repository/audit.go`
  - [ ] Implement `Create(log *model.AuditLog) error`
  - [ ] Implement `List(filters map[string]interface{}) ([]*model.AuditLog, error)`
  - [ ] Ensure append-only (no update/delete methods)
  - **Files**: `internal/repository/audit.go`
  - **Estimated Time**: 2 hours

---

## 3️⃣ CORE SERVICES

### 3.1 Exchange Rate Service
- [ ] **Task**: Implement crypto/VND exchange rate fetching
  - [ ] Create `internal/service/exchange_rate.go`
  - [ ] Implement `GetUSDTToVND() (decimal.Decimal, error)`
  - [ ] Integrate with CoinGecko API or Binance API
  - [ ] Add caching with Redis (5-minute TTL)
  - [ ] Add fallback to secondary API if primary fails
  - [ ] Handle API rate limits
  - **Files**: `internal/service/exchange_rate.go`
  - **Dependencies**: HTTP client, Redis client
  - **Estimated Time**: 4 hours

### 3.2 Payment Service - Core Logic
- [ ] **Task**: Implement payment business logic
  - [ ] Create `internal/service/payment.go`
  - [ ] Implement `CreatePayment(merchantID, amountVND, orderID, callbackURL)`
    - Validate merchant exists and is approved
    - Get current exchange rate
    - Calculate crypto amount
    - Generate payment ID
    - Set expiration (30 minutes)
    - Save to database
    - Create audit log entry
  - [ ] Implement `GetPaymentStatus(paymentID string) (*model.Payment, error)`
  - [ ] Implement `ValidatePayment(paymentID string) error` - check not expired
  - **Files**: `internal/service/payment.go`
  - **Dependencies**: Repository layer, exchange rate service
  - **Estimated Time**: 6 hours

### 3.3 Payment Service - Confirmation Logic
- [ ] **Task**: Implement payment confirmation workflow
  - [ ] Add `ConfirmPayment(paymentID, txHash string, actualAmount decimal.Decimal)` to payment service
    - Validate payment exists and is in pending state
    - Verify amount matches expected amount exactly
    - Update payment status: pending → confirming → completed
    - Record blockchain transaction details
    - Create ledger entries
    - Update merchant balance
    - Trigger webhook
    - Create audit log
  - [ ] Add `ExpirePayment(paymentID string) error`
  - [ ] Add `FailPayment(paymentID, reason string) error`
  - **Files**: `internal/service/payment.go`
  - **Dependencies**: Ledger service, notification service
  - **Estimated Time**: 6 hours

### 3.4 Ledger Service
- [ ] **Task**: Implement double-entry accounting logic
  - [ ] Create `internal/service/ledger.go`
  - [ ] Implement `RecordPaymentReceived(paymentID, merchantID, amountCrypto, amountVND)`
    - DEBIT: crypto_pool
    - CREDIT: merchant_pending_balance
  - [ ] Implement `RecordPaymentConfirmed(paymentID, merchantID, amountVND, feeVND)`
    - DEBIT: merchant_pending_balance
    - CREDIT: merchant_available_balance (amount - fee)
    - CREDIT: fee_revenue
  - [ ] Implement `RecordPayout(payoutID, merchantID, amount, fee)`
    - DEBIT: merchant_available_balance (amount + fee)
    - CREDIT: vnd_pool (amount)
    - CREDIT: fee_revenue (fee)
  - [ ] Ensure all operations are atomic (use database transactions)
  - [ ] Validate debit = credit for every entry
  - **Files**: `internal/service/ledger.go`
  - **Dependencies**: Repository layer
  - **Estimated Time**: 8 hours

### 3.5 Merchant Service
- [ ] **Task**: Implement merchant management logic
  - [ ] Create `internal/service/merchant.go`
  - [ ] Implement `RegisterMerchant(email, businessName, taxID, kycData)`
  - [ ] Implement `GetMerchantBalance(merchantID) (*model.MerchantBalance, error)`
  - [ ] Implement `ApproveKYC(merchantID, approvedBy string)` - generate API key
  - [ ] Implement `RejectKYC(merchantID, reason string)`
  - [ ] Implement `GenerateAPIKey() string` - secure random key
  - [ ] Implement `RotateAPIKey(merchantID string) (string, error)`
  - **Files**: `internal/service/merchant.go`
  - **Dependencies**: Repository layer, crypto/rand
  - **Estimated Time**: 5 hours

### 3.6 Payout Service
- [ ] **Task**: Implement payout request and processing logic
  - [ ] Create `internal/service/payout.go`
  - [ ] Implement `RequestPayout(merchantID, amount, bankInfo)`
    - Validate merchant has sufficient balance
    - Check minimum amount (1M VND)
    - Calculate fee
    - Lock merchant balance
    - Create payout record
    - Create audit log
  - [ ] Implement `ApprovePayout(payoutID, approvedBy string)` (admin only)
  - [ ] Implement `RejectPayout(payoutID, reason string)`
  - [ ] Implement `CompletePayout(payoutID, referenceNumber string)`
    - Record ledger entry
    - Update merchant balance
    - Send notification
  - **Files**: `internal/service/payout.go`
  - **Dependencies**: Repository layer, ledger service
  - **Estimated Time**: 6 hours

### 3.7 Notification Service - Webhooks
- [ ] **Task**: Implement webhook delivery system
  - [ ] Create `internal/service/notification.go`
  - [ ] Implement `SendWebhook(merchantID, event, payload)`
    - Get merchant webhook URL and secret
    - Create HMAC-SHA256 signature
    - POST to webhook URL with retry logic
    - Exponential backoff (1s, 2s, 4s, 8s, 16s)
    - Log all delivery attempts
  - [ ] Implement webhook event types: `payment.completed`, `payment.failed`, `payout.completed`
  - [ ] Add webhook verification helper for merchants
  - **Files**: `internal/service/notification.go`
  - **Dependencies**: HTTP client, crypto/hmac
  - **Estimated Time**: 5 hours

### 3.8 Notification Service - Email
- [ ] **Task**: Implement email notifications
  - [ ] Add email templates (HTML + plain text)
  - [ ] Implement `SendPaymentConfirmation(merchantID, paymentID)`
  - [ ] Implement `SendPayoutApproved(merchantID, payoutID)`
  - [ ] Implement `SendPayoutCompleted(merchantID, payoutID)`
  - [ ] Implement `SendDailySettlementReport(date)` for ops
  - [ ] Integrate with SendGrid or AWS SES
  - **Files**: `internal/service/notification.go`, `templates/email/*.html`
  - **Dependencies**: Email service SDK
  - **Estimated Time**: 4 hours

### 3.9 Audit Service
- [ ] **Task**: Implement audit logging wrapper
  - [ ] Create `internal/service/audit.go`
  - [ ] Implement `LogAction(actorType, actorID, action, resourceType, resourceID, metadata)`
  - [ ] Add helper methods for common actions:
    - `LogPaymentCreated`, `LogPaymentCompleted`
    - `LogKYCApproved`, `LogKYCRejected`
    - `LogPayoutRequested`, `LogPayoutApproved`
  - [ ] Automatically capture IP address and timestamp
  - **Files**: `internal/service/audit.go`
  - **Dependencies**: Repository layer
  - **Estimated Time**: 3 hours

---

## 4️⃣ BLOCKCHAIN INTEGRATION

### 4.1 Solana Wallet Management
- [ ] **Task**: Implement Solana wallet operations
  - [ ] Create `internal/blockchain/solana/wallet.go`
  - [ ] Implement `LoadWallet(privateKey string) (*Wallet, error)`
  - [ ] Implement `GetAddress() string`
  - [ ] Implement `GetBalance(tokenMint string) (decimal.Decimal, error)`
  - [ ] Implement `SignTransaction(tx *solana.Transaction) error`
  - [ ] Add support for USDT and USDC SPL tokens
  - [ ] Store private key securely (environment variable for MVP)
  - **Files**: `internal/blockchain/solana/wallet.go`
  - **Dependencies**: `github.com/gagliardetto/solana-go`
  - **Estimated Time**: 5 hours

### 4.2 Solana RPC Client
- [ ] **Task**: Set up Solana blockchain connection
  - [ ] Create `internal/blockchain/solana/client.go`
  - [ ] Implement connection to Solana RPC (Helius/QuickNode)
  - [ ] Implement `GetTransaction(signature string) (*Transaction, error)`
  - [ ] Implement `GetConfirmations(signature string) (int, error)`
  - [ ] Add health check function
  - [ ] Add retry logic for RPC failures
  - **Files**: `internal/blockchain/solana/client.go`
  - **Dependencies**: `github.com/gagliardetto/solana-go`
  - **Estimated Time**: 4 hours

### 4.3 Solana Transaction Listener
- [ ] **Task**: Implement real-time transaction monitoring
  - [ ] Create `internal/blockchain/solana/listener.go`
  - [ ] Implement `Start()` - subscribe to wallet transactions
  - [ ] Implement `handleTransaction(signature string)`
    - Fetch transaction details from RPC
    - Parse memo field for payment_id
    - Extract amount transferred
    - Verify destination address matches our wallet
    - Wait for `finalized` commitment
    - Call payment service to confirm payment
  - [ ] Add error handling and retry logic
  - [ ] Add graceful shutdown
  - **Files**: `internal/blockchain/solana/listener.go`
  - **Dependencies**: Payment service
  - **Estimated Time**: 8 hours

### 4.4 Solana Transaction Parser
- [ ] **Task**: Implement SPL token transaction parsing
  - [ ] Create `internal/blockchain/solana/parser.go`
  - [ ] Implement `ParseSPLTokenTransfer(tx *Transaction) (*Transfer, error)`
  - [ ] Extract: from address, to address, amount, token mint
  - [ ] Implement `ExtractMemo(tx *Transaction) (string, error)`
  - [ ] Handle different SPL token decimals (USDT: 6, USDC: 6)
  - [ ] Add validation for transaction structure
  - **Files**: `internal/blockchain/solana/parser.go`
  - **Dependencies**: Solana SDK
  - **Estimated Time**: 6 hours

### 4.5 Blockchain Transaction Repository
- [ ] **Task**: Track blockchain transactions in database
  - [ ] Create `internal/repository/blockchain_tx.go`
  - [ ] Implement `Create(tx *model.BlockchainTransaction) error`
  - [ ] Implement `GetByTxHash(txHash string) (*model.BlockchainTransaction, error)`
  - [ ] Implement `UpdateConfirmations(txHash string, confirmations int) error`
  - [ ] Implement `GetPendingTransactions() ([]*model.BlockchainTransaction, error)`
  - **Files**: `internal/repository/blockchain_tx.go`
  - **Estimated Time**: 3 hours

### 4.6 Wallet Balance Monitor
- [ ] **Task**: Implement hot wallet balance monitoring
  - [ ] Create `internal/blockchain/solana/monitor.go`
  - [ ] Implement periodic balance check (every 5 minutes)
  - [ ] Implement `CheckBalance() (decimal.Decimal, error)`
  - [ ] Send alert if balance < threshold (e.g., $1000)
  - [ ] Send alert if balance > max threshold (e.g., $10000)
  - [ ] Log balance to database for tracking
  - **Files**: `internal/blockchain/solana/monitor.go`
  - **Dependencies**: Notification service
  - **Estimated Time**: 3 hours

---

## 5️⃣ API LAYER

### 5.1 HTTP Server Setup
- [ ] **Task**: Initialize HTTP server with Gin framework
  - [ ] Create `internal/api/server.go`
  - [ ] Set up Gin router
  - [ ] Configure CORS middleware
  - [ ] Add request logging middleware
  - [ ] Add error handling middleware
  - [ ] Add panic recovery middleware
  - [ ] Configure graceful shutdown
  - **Files**: `internal/api/server.go`
  - **Dependencies**: `github.com/gin-gonic/gin`
  - **Estimated Time**: 4 hours

### 5.2 API Request/Response Models
- [ ] **Task**: Define API DTOs (Data Transfer Objects)
  - [ ] Create `internal/api/dto/payment.go`
    - CreatePaymentRequest, CreatePaymentResponse
    - GetPaymentResponse
  - [ ] Create `internal/api/dto/payout.go`
    - CreatePayoutRequest, CreatePayoutResponse
  - [ ] Create `internal/api/dto/merchant.go`
    - RegisterMerchantRequest, MerchantBalanceResponse
  - [ ] Create `internal/api/dto/common.go`
    - APIResponse wrapper, ErrorResponse
  - [ ] Add validation tags for all request DTOs
  - **Files**: `internal/api/dto/*.go`
  - **Estimated Time**: 3 hours

### 5.3 Payment API Handlers
- [ ] **Task**: Implement payment endpoints
  - [ ] Create `internal/api/handler/payment.go`
  - [ ] `POST /api/v1/payments` - CreatePayment
    - Validate request
    - Get merchant from context (set by auth middleware)
    - Call payment service
    - Generate QR code
    - Return payment details + QR code
  - [ ] `GET /api/v1/payments/:id` - GetPayment
    - Verify merchant owns this payment
    - Return payment status
  - [ ] Add input validation
  - [ ] Add proper error responses
  - **Files**: `internal/api/handler/payment.go`
  - **Dependencies**: Payment service, QR code library
  - **Estimated Time**: 5 hours

### 5.4 Merchant API Handlers
- [ ] **Task**: Implement merchant endpoints
  - [ ] Create `internal/api/handler/merchant.go`
  - [ ] `POST /api/v1/merchant/register` - RegisterMerchant
  - [ ] `GET /api/v1/merchant/balance` - GetBalance
  - [ ] `GET /api/v1/merchant/transactions` - GetTransactions (paginated)
  - [ ] Add pagination helpers
  - **Files**: `internal/api/handler/merchant.go`
  - **Dependencies**: Merchant service
  - **Estimated Time**: 4 hours

### 5.5 Payout API Handlers
- [ ] **Task**: Implement payout endpoints
  - [ ] Create `internal/api/handler/payout.go`
  - [ ] `POST /api/v1/merchant/payouts` - RequestPayout
    - Validate bank account details
    - Call payout service
  - [ ] `GET /api/v1/merchant/payouts` - ListPayouts
  - [ ] `GET /api/v1/merchant/payouts/:id` - GetPayoutStatus
  - **Files**: `internal/api/handler/payout.go`
  - **Dependencies**: Payout service
  - **Estimated Time**: 3 hours

### 5.6 Admin API Handlers
- [ ] **Task**: Implement admin endpoints
  - [ ] Create `internal/api/handler/admin.go`
  - [ ] `POST /api/admin/merchants/:id/kyc/approve` - ApproveKYC
  - [ ] `POST /api/admin/merchants/:id/kyc/reject` - RejectKYC
  - [ ] `POST /api/admin/payouts/:id/approve` - ApprovePayout
  - [ ] `POST /api/admin/payouts/:id/complete` - CompletePayout
  - [ ] `GET /api/admin/stats` - GetSystemStats
  - [ ] Ensure admin-only access (JWT middleware)
  - **Files**: `internal/api/handler/admin.go`
  - **Dependencies**: Merchant service, payout service
  - **Estimated Time**: 5 hours

### 5.7 Health Check Endpoints
- [ ] **Task**: Implement system health endpoints
  - [ ] Create `internal/api/handler/health.go`
  - [ ] `GET /health` - Basic health check (always returns 200)
  - [ ] `GET /api/v1/status` - Detailed status
    - Database connectivity
    - Redis connectivity
    - Blockchain RPC connectivity
    - Wallet balance
  - **Files**: `internal/api/handler/health.go`
  - **Estimated Time**: 2 hours

### 5.8 API Routes Setup
- [ ] **Task**: Configure all API routes
  - [ ] Create `internal/api/routes/routes.go`
  - [ ] Set up public routes (no auth): `/health`, `/api/v1/status`
  - [ ] Set up merchant routes (API key auth): `/api/v1/payments`, `/api/v1/merchant/*`
  - [ ] Set up admin routes (JWT auth): `/api/admin/*`
  - [ ] Apply rate limiting middleware
  - [ ] Apply request validation middleware
  - **Files**: `internal/api/routes/routes.go`
  - **Estimated Time**: 3 hours

### 5.9 QR Code Generation
- [ ] **Task**: Implement QR code generation for payments
  - [ ] Create `internal/pkg/qrcode/generator.go`
  - [ ] Implement `GeneratePaymentQR(walletAddress, amount, memo) (string, error)`
  - [ ] Generate Solana Pay URL format
  - [ ] Return base64-encoded PNG image
  - [ ] Add size configuration
  - **Files**: `internal/pkg/qrcode/generator.go`
  - **Dependencies**: `github.com/skip2/go-qrcode`
  - **Estimated Time**: 3 hours

---

## 6️⃣ AUTHENTICATION & SECURITY

### 6.1 API Key Authentication Middleware
- [ ] **Task**: Implement API key validation
  - [ ] Create `internal/api/middleware/auth.go`
  - [ ] Implement `APIKeyAuth()` middleware
    - Extract API key from `Authorization: Bearer {key}` header
    - Look up merchant by API key
    - Validate merchant is approved
    - Set merchant in context
  - [ ] Add rate limiting per API key
  - [ ] Log authentication failures
  - **Files**: `internal/api/middleware/auth.go`
  - **Dependencies**: Merchant repository
  - **Estimated Time**: 4 hours

### 6.2 JWT Authentication for Admin
- [ ] **Task**: Implement JWT-based admin authentication
  - [ ] Create `internal/pkg/jwt/jwt.go`
  - [ ] Implement `GenerateToken(adminID, role string) (string, error)`
  - [ ] Implement `ValidateToken(token string) (*Claims, error)`
  - [ ] Add `JWTAuth()` middleware for admin routes
  - [ ] Add role-based access control
  - [ ] Set token expiration (24 hours)
  - **Files**: `internal/pkg/jwt/jwt.go`, `internal/api/middleware/admin_auth.go`
  - **Dependencies**: `github.com/golang-jwt/jwt/v5`
  - **Estimated Time**: 5 hours

### 6.3 Rate Limiting Middleware
- [ ] **Task**: Implement request rate limiting
  - [ ] Create `internal/api/middleware/rate_limit.go`
  - [ ] Use Redis for rate limiting (sliding window)
  - [ ] Implement 100 requests/minute per API key
  - [ ] Implement 1000 requests/minute per IP (global)
  - [ ] Return `429 Too Many Requests` when limit exceeded
  - [ ] Add rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining)
  - **Files**: `internal/api/middleware/rate_limit.go`
  - **Dependencies**: Redis client
  - **Estimated Time**: 4 hours

### 6.4 Input Validation
- [ ] **Task**: Implement request validation
  - [ ] Create validation middleware using `validator` library
  - [ ] Validate all money amounts (no negative, no zero, max limit)
  - [ ] Validate email formats, URLs, UUIDs
  - [ ] Sanitize inputs to prevent XSS
  - [ ] Return structured validation errors
  - **Files**: `internal/api/middleware/validator.go`
  - **Dependencies**: `github.com/go-playground/validator/v10`
  - **Estimated Time**: 3 hours

### 6.5 CORS Configuration
- [ ] **Task**: Configure Cross-Origin Resource Sharing
  - [ ] Create `internal/api/middleware/cors.go`
  - [ ] Allow merchant dashboard origins
  - [ ] Allow admin panel origins
  - [ ] Configure allowed methods (GET, POST, PUT, DELETE, OPTIONS)
  - [ ] Set credentials policy
  - **Files**: `internal/api/middleware/cors.go`
  - **Dependencies**: `github.com/gin-contrib/cors`
  - **Estimated Time**: 2 hours

### 6.6 Webhook Signature Verification
- [ ] **Task**: Implement webhook HMAC signature
  - [ ] Create `internal/pkg/webhook/signature.go`
  - [ ] Implement `GenerateSignature(payload, secret) string` using HMAC-SHA256
  - [ ] Implement `VerifySignature(payload, signature, secret) bool`
  - [ ] Add timestamp to prevent replay attacks
  - [ ] Document signature verification for merchants
  - **Files**: `internal/pkg/webhook/signature.go`
  - **Dependencies**: crypto/hmac
  - **Estimated Time**: 3 hours

### 6.7 Secrets Management
- [ ] **Task**: Implement secure secrets handling
  - [ ] Create `.env.example` with all required variables
  - [ ] Add validation for required environment variables on startup
  - [ ] Ensure private keys are never logged
  - [ ] Add `.env` to `.gitignore`
  - [ ] Document secrets rotation process
  - **Files**: `.env.example`, `internal/config/secrets.go`
  - **Estimated Time**: 2 hours

---

## 7️⃣ BACKGROUND JOBS & WORKERS

### 7.1 Job Queue Setup
- [ ] **Task**: Set up asynchronous job processing
  - [ ] Create `internal/worker/queue.go`
  - [ ] Set up `asynq` for background jobs
  - [ ] Configure Redis as job broker
  - [ ] Define job types: `webhook_delivery`, `payment_expiry`, `balance_check`
  - [ ] Set up worker pools
  - **Files**: `internal/worker/queue.go`
  - **Dependencies**: `github.com/hibiken/asynq`
  - **Estimated Time**: 4 hours

### 7.2 Webhook Delivery Worker
- [ ] **Task**: Implement async webhook delivery
  - [ ] Create `internal/worker/webhook_worker.go`
  - [ ] Process `webhook_delivery` jobs
  - [ ] Retry failed webhooks (exponential backoff)
  - [ ] Mark as failed after 5 attempts
  - [ ] Log delivery status
  - **Files**: `internal/worker/webhook_worker.go`
  - **Dependencies**: Notification service
  - **Estimated Time**: 3 hours

### 7.3 Payment Expiry Worker
- [ ] **Task**: Implement payment expiration cron job
  - [ ] Create `internal/worker/payment_expiry_worker.go`
  - [ ] Run every 5 minutes
  - [ ] Fetch payments in `created` or `pending` state older than 30 minutes
  - [ ] Update status to `expired`
  - [ ] Create audit log entries
  - **Files**: `internal/worker/payment_expiry_worker.go`
  - **Dependencies**: Payment service
  - **Estimated Time**: 3 hours

### 7.4 Balance Monitor Worker
- [ ] **Task**: Implement hot wallet balance monitoring job
  - [ ] Create `internal/worker/balance_monitor_worker.go`
  - [ ] Run every 5 minutes
  - [ ] Check Solana wallet balance
  - [ ] Send alerts if balance out of range
  - [ ] Log balance history
  - **Files**: `internal/worker/balance_monitor_worker.go`
  - **Dependencies**: Blockchain wallet service
  - **Estimated Time**: 2 hours

### 7.5 Daily Settlement Report Worker
- [ ] **Task**: Generate daily reports for ops team
  - [ ] Create `internal/worker/settlement_report_worker.go`
  - [ ] Run daily at 8 AM
  - [ ] Generate report:
    - Total payments received
    - Total payouts processed
    - Current hot wallet balance
    - Pending payouts
  - [ ] Send email to ops team
  - **Files**: `internal/worker/settlement_report_worker.go`
  - **Dependencies**: Repository layer, notification service
  - **Estimated Time**: 4 hours

---

## 8️⃣ TESTING & DEPLOYMENT

### 8.1 Unit Tests - Repository Layer
- [ ] **Task**: Write unit tests for repositories
  - [ ] Test merchant repository CRUD operations
  - [ ] Test payment repository CRUD operations
  - [ ] Test ledger repository (especially transactions)
  - [ ] Use test database or mocks
  - [ ] Aim for >80% coverage
  - **Files**: `internal/repository/*_test.go`
  - **Dependencies**: `github.com/stretchr/testify`
  - **Estimated Time**: 8 hours

### 8.2 Unit Tests - Service Layer
- [ ] **Task**: Write unit tests for services
  - [ ] Test payment service business logic
  - [ ] Test ledger service (double-entry validation)
  - [ ] Test merchant service
  - [ ] Mock repository layer
  - [ ] Test error cases
  - **Files**: `internal/service/*_test.go`
  - **Dependencies**: `github.com/stretchr/testify/mock`
  - **Estimated Time**: 10 hours

### 8.3 Integration Tests - API
- [ ] **Task**: Write API integration tests
  - [ ] Test payment creation flow
  - [ ] Test payout request flow
  - [ ] Test authentication (API key, JWT)
  - [ ] Test rate limiting
  - [ ] Use test database
  - **Files**: `internal/api/integration_test.go`
  - **Estimated Time**: 8 hours

### 8.4 Integration Tests - Blockchain
- [ ] **Task**: Test blockchain integration on testnet
  - [ ] Set up Solana devnet wallet
  - [ ] Test transaction listening
  - [ ] Test payment confirmation flow
  - [ ] Verify memo parsing
  - [ ] Test finality waiting
  - **Files**: `internal/blockchain/solana/integration_test.go`
  - **Estimated Time**: 6 hours

### 8.5 End-to-End Test
- [ ] **Task**: Full payment flow test
  - [ ] Create payment via API
  - [ ] Send testnet USDT to wallet
  - [ ] Verify automatic confirmation
  - [ ] Check balance update
  - [ ] Verify webhook delivery
  - [ ] Request payout
  - [ ] Complete payout
  - [ ] Verify ledger accuracy
  - **Files**: `test/e2e/payment_flow_test.go`
  - **Estimated Time**: 6 hours

### 8.6 Load Testing
- [ ] **Task**: Test system under load
  - [ ] Use `k6` or `vegeta` for load testing
  - [ ] Test 100 concurrent payment creations
  - [ ] Test database performance
  - [ ] Test API response times
  - [ ] Identify bottlenecks
  - **Files**: `test/load/payment_load_test.js`
  - **Dependencies**: k6
  - **Estimated Time**: 4 hours

### 8.7 Security Audit
- [ ] **Task**: Review security implementation
  - [ ] Review authentication logic
  - [ ] Check for SQL injection vulnerabilities
  - [ ] Verify input sanitization
  - [ ] Check wallet key management
  - [ ] Verify HTTPS enforcement
  - [ ] Run `gosec` for security scanning
  - **Tools**: gosec, manual review
  - **Estimated Time**: 6 hours

### 8.8 Production Deployment
- [ ] **Task**: Deploy to production VPS
  - [ ] Provision VPS (4 CPU, 8GB RAM, 100GB SSD)
  - [ ] Install Docker, Docker Compose
  - [ ] Set up PostgreSQL, Redis
  - [ ] Configure NGINX with SSL (Let's Encrypt)
  - [ ] Deploy API service
  - [ ] Deploy blockchain listener service
  - [ ] Deploy worker service
  - [ ] Configure environment variables
  - [ ] Set up monitoring (PM2 or systemd)
  - [ ] Configure backups (PostgreSQL daily backup)
  - **Files**: `scripts/deploy.sh`, `docker-compose.prod.yml`
  - **Estimated Time**: 8 hours

### 8.9 Monitoring & Alerting Setup
- [ ] **Task**: Set up production monitoring
  - [ ] Configure email alerts for critical errors
  - [ ] Set up log aggregation
  - [ ] Configure health check monitoring (UptimeRobot or similar)
  - [ ] Set up database backup verification
  - [ ] Create runbook for common incidents
  - **Files**: `docs/RUNBOOK.md`
  - **Estimated Time**: 4 hours

---

## 📊 Task Summary

| Category | Task Count | Estimated Hours |
|----------|------------|-----------------|
| 1. Infrastructure & Setup | 4 | 12 hours |
| 2. Database Layer | 9 | 33 hours |
| 3. Core Services | 9 | 47 hours |
| 4. Blockchain Integration | 6 | 29 hours |
| 5. API Layer | 9 | 32 hours |
| 6. Authentication & Security | 7 | 23 hours |
| 7. Background Jobs & Workers | 5 | 16 hours |
| 8. Testing & Deployment | 9 | 60 hours |
| **TOTAL** | **58 tasks** | **252 hours** |

**Estimated Timeline**: 6-8 weeks (with 3-4 engineers)

---

## 🎯 Critical Path Tasks

These tasks are on the critical path and must be completed sequentially:

1. **Week 1**: Infrastructure → Database Schema → Core Models → Basic Repositories
2. **Week 2**: Payment Service → Blockchain Listener → Payment Confirmation
3. **Week 3**: Ledger Service → API Layer → Authentication
4. **Week 4**: Payout Service → Admin API → Webhooks
5. **Week 5**: Background Workers → Integration Tests
6. **Week 6**: End-to-End Tests → Security Audit → Deployment

---

## 📋 Task Dependencies

```
Database Schema (2.1)
  ↓
Models (2.3) + Connection Pool (2.2)
  ↓
Repositories (2.4-2.9)
  ↓
Services (3.1-3.9)
  ↓
API Handlers (5.1-5.7) + Blockchain (4.1-4.4)
  ↓
Workers (7.1-7.5)
  ↓
Testing (8.1-8.5)
  ↓
Deployment (8.8-8.9)
```

---

## 🚀 Getting Started

### For Backend Engineers

1. **Start with infrastructure tasks (1.1-1.4)** - set up project structure
2. **Move to database layer (2.1-2.9)** - establish data foundation
3. **Implement core services (3.1-3.9)** - build business logic
4. **Add blockchain integration (4.1-4.6)** - connect to Solana
5. **Build API layer (5.1-5.9)** - expose functionality
6. **Implement security (6.1-6.7)** - protect the system
7. **Add background jobs (7.1-7.5)** - async operations
8. **Test and deploy (8.1-8.9)** - ensure quality

### Team Allocation Suggestion

- **Engineer 1** (Backend Lead): Infrastructure + Core Services + Ledger
- **Engineer 2** (Blockchain): Blockchain Integration + Wallet + Listener
- **Engineer 3** (API): API Layer + Authentication + Security
- **Engineer 4** (DevOps/Testing): Workers + Testing + Deployment

---

## 📝 Notes

- All money calculations MUST use `decimal.Decimal`, NEVER `float64`
- All database operations involving money MUST be wrapped in transactions
- All critical operations MUST create audit log entries
- All API endpoints MUST have input validation
- All external API calls MUST have retry logic and error handling
- Private keys MUST NEVER be logged or committed to git

---

**Last Updated**: 2025-11-16
**Maintained By**: Backend Team
