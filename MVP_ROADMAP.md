# MVP Implementation Roadmap

**Timeline**: 4-6 weeks to working MVP
**Team**: 3-5 engineers + 1 ops + 1 legal advisor
**Budget**: ~20-30M VND (excluding salaries)

---

## Week 1: Foundation & Setup

### Infrastructure Setup (Days 1-2)
- [ ] **Project structure**
  - Monorepo setup (backend + frontend)
  - TypeScript + Node.js + Express
  - PostgreSQL + Prisma ORM
  - Next.js frontend
  - Docker Compose for local dev

- [ ] **Database schema**
  - Create all tables (merchants, payments, payouts, ledger, audit_logs)
  - Seed data for testing
  - Migration scripts

- [ ] **Development environment**
  - Git repo + branching strategy
  - CI/CD pipeline (GitHub Actions)
  - Staging environment

**Deliverable**: Team can run project locally, database schema ready

---

### Authentication & Basic API (Days 3-5)

- [ ] **Merchant authentication**
  - API key generation
  - API key validation middleware
  - Rate limiting (Redis)

- [ ] **Health check endpoints**
  - `/health` - Basic health check
  - `/api/v1/status` - System status

- [ ] **Admin authentication**
  - JWT-based login
  - Role-based access control

- [ ] **Audit logging**
  - Middleware to log all requests
  - Save to `audit_logs` table

**Deliverable**: Secure API foundation with auth

---

## Week 2: Core Payment Flow (Backend)

### Payment Creation API (Days 6-8)

- [ ] **POST /api/v1/payments**
  - Input validation (amount, merchant_id, order_id)
  - Get current USDT/VND exchange rate (API: Binance/CoinGecko)
  - Calculate crypto amount
  - Generate payment_id
  - Save to database
  - Return payment details

- [ ] **GET /api/v1/payments/:paymentId**
  - Return payment status
  - Include tx_hash if confirmed

- [ ] **Payment expiry logic**
  - Cron job to expire payments after 30 minutes
  - Update status: `created` → `expired`

**Deliverable**: Merchants can create payment requests via API

---

### Blockchain Listener (Days 9-12)

- [ ] **Solana wallet setup**
  - Generate wallet keypair
  - Store private key in env vault
  - Document wallet address

- [ ] **Solana transaction listener**
  - Connect to Solana RPC (Helius/QuickNode)
  - Subscribe to wallet transactions
  - Parse transaction details:
    - From address
    - Amount
    - Memo (payment_id)

- [ ] **Transaction confirmation**
  - Wait for `finalized` commitment
  - Update payment status: `pending` → `confirming` → `completed`
  - Update ledger

- [ ] **Error handling**
  - Retry failed confirmations
  - Alert on stuck transactions
  - Handle reorg (rare on Solana)

**Deliverable**: System automatically detects and confirms crypto payments

---

## Week 3: Ledger & Merchant Features

### Ledger System (Days 13-15)

- [ ] **Ledger service**
  - Double-entry accounting logic
  - Record payment received:
    ```
    DEBIT:  crypto_pool (+X USDT)
    CREDIT: merchant_pending_balance (+Y VND)
    ```
  - Record OTC conversion (manual entry for MVP):
    ```
    DEBIT:  crypto_pool (-X USDT)
    CREDIT: vnd_pool (+Y VND)
    DEBIT:  merchant_pending_balance (-Y VND)
    CREDIT: merchant_available_balance (+Y * 0.99 VND)
    CREDIT: fee_revenue (+Y * 0.01 VND)
    ```
  - Record payout:
    ```
    DEBIT:  merchant_available_balance (+Z VND)
    CREDIT: vnd_pool (+Z VND)
    ```

- [ ] **Balance calculation**
  - Real-time balance updates
  - Merchant balance query API
  - Admin view: total VND pool, crypto pool

**Deliverable**: Accurate accounting for all transactions

---

### Merchant Dashboard (Days 16-18)

- [ ] **Merchant registration page**
  - Form: email, business name, tax ID, owner info
  - File upload: business license, ID card
  - Email verification

- [ ] **Merchant login**
  - Email/password auth
  - Generate API key after KYC approval

- [ ] **Dashboard pages**
  - Overview: balance, recent payments
  - Payments list (paginated, filterable)
  - Payout request form
  - Transaction history

- [ ] **Payment creation UI**
  - Form: amount VND, order ID, callback URL
  - Generate QR code
  - Copy payment link

**Deliverable**: Merchants can manage their account via web dashboard

---

## Week 4: Payment Page & Webhooks

### Public Payment Page (Days 19-21)

- [ ] **Payment page** (`/pay/:paymentId`)
  - Display payment details:
    - Amount: X VND = Y USDT
    - Wallet address
    - Payment reference/memo
    - QR code (includes: address + amount + memo)
  - Instructions:
    - "Open your crypto wallet (Phantom, Trust Wallet, etc.)"
    - "Scan QR code OR copy details below"
    - "Send exactly Y USDT on Solana network"
    - "Include payment reference in memo"
  - Real-time status updates (polling or websocket)
  - Payment states:
    - Waiting for payment...
    - Payment detected! Confirming...
    - Payment confirmed ✓

- [ ] **Mobile-optimized UI**
  - Responsive design
  - Large QR code
  - Copy-to-clipboard buttons

**Deliverable**: End users can easily pay via crypto

---

### Webhook System (Days 22-24)

- [ ] **Webhook dispatcher**
  - Trigger on payment status change
  - Sign payload with HMAC-SHA256
  - POST to merchant callback URL
  - Retry logic: exponential backoff, up to 5 attempts
  - Log all webhook attempts

- [ ] **Webhook events**
  - `payment.completed` - Payment confirmed
  - `payment.failed` - Payment failed/expired
  - `payout.completed` - Payout processed

- [ ] **Webhook testing**
  - Test endpoint for merchants
  - Webhook logs in dashboard

**Deliverable**: Merchants receive real-time notifications

---

## Week 5: Admin Panel & Payout

### Admin Panel (Days 25-27)

- [ ] **KYC review page**
  - List pending merchants
  - View KYC documents
  - Approve/reject with notes
  - Generate API key on approval

- [ ] **Payout approval page**
  - List pending payout requests
  - View merchant info, balance
  - Approve/reject
  - Record bank transfer details

- [ ] **System monitoring**
  - Total volume processed
  - Active merchants
  - Pending payouts
  - Hot wallet balance
  - Failed transactions

- [ ] **Manual OTC entry**
  - Form to record OTC conversions
  - Input: USDT sent, VND received
  - Update ledger

**Deliverable**: Ops team can manage merchants and payouts

---

### Payout System (Days 28-30)

- [ ] **Payout request API**
  - Validate merchant balance
  - Minimum amount: 1M VND
  - Deduct payout + fee from balance
  - Create payout record

- [ ] **Manual payout process (MVP)**
  1. Admin reviews payout request
  2. Admin approves → adds to daily batch
  3. Ops team executes bank transfer
  4. Ops team marks payout as completed
  5. System sends confirmation email to merchant

- [ ] **Email notifications**
  - Payout requested (to ops)
  - Payout approved (to merchant)
  - Payout completed (to merchant)

**Deliverable**: Merchants can withdraw VND to their bank

---

## Week 6: Testing, Security & Launch

### Security Audit (Days 31-33)

- [ ] **Code review**
  - Review authentication logic
  - Review payment validation
  - Review wallet key management
  - Review database queries (SQL injection check)

- [ ] **Penetration testing**
  - API endpoint testing
  - Rate limiting verification
  - CSRF/XSS checks

- [ ] **Secrets management**
  - Move all secrets to .env
  - Use environment variables
  - Document secret rotation process

- [ ] **Monitoring & alerts**
  - Set up email alerts for:
    - Failed transactions
    - Hot wallet low balance
    - Failed webhooks
    - System errors

**Deliverable**: Secure, production-ready system

---

### End-to-End Testing (Days 34-36)

- [ ] **Testnet testing**
  - Create test merchant
  - Create payment
  - Send testnet USDT
  - Verify confirmation
  - Request payout
  - Verify ledger

- [ ] **Load testing**
  - Test 100 concurrent payments
  - Test database performance
  - Test blockchain listener under load

- [ ] **User acceptance testing**
  - Real merchants test on testnet
  - Collect feedback
  - Fix critical issues

**Deliverable**: Tested, working system

---

### Documentation & Deployment (Days 37-40)

- [ ] **Documentation**
  - API documentation (Postman/Swagger)
  - Merchant onboarding guide
  - Integration guide
  - FAQ

- [ ] **Legal documents**
  - Terms of Service
  - Privacy Policy
  - Merchant Agreement
  - KYC requirements

- [ ] **Production deployment**
  - Set up VPS (DigitalOcean/AWS)
  - Configure NGINX + SSL
  - Deploy backend + frontend
  - Set up monitoring
  - Set up backups

- [ ] **Pilot merchant onboarding**
  - Onboard 3-5 pilot merchants
  - Walk through full process
  - Collect feedback

**Deliverable**: Live system with pilot merchants

---

## Post-Launch (Week 7+)

### Month 1: Stabilization

- [ ] Monitor system 24/7
- [ ] Fix bugs immediately
- [ ] Process first real payments
- [ ] Collect merchant feedback
- [ ] Optimize based on feedback

**Goal**: 5 merchants, 1B VND volume, 99% uptime

---

### Month 2-3: Optimization

- [ ] Improve UX based on feedback
- [ ] Add analytics to dashboard
- [ ] Optimize confirmation speed
- [ ] Automate ops processes
- [ ] Marketing to get more merchants

**Goal**: 20 merchants, 10B VND volume

---

### Month 4-6: Scale Preparation (Phase 2)

- [ ] Add Ethereum support
- [ ] Build public API
- [ ] Automated payouts
- [ ] Merchant integrations (Shopify, WooCommerce)

**Goal**: 50 merchants, 50B VND volume

---

## Resource Requirements

### Team

| Role | Headcount | Responsibility |
|------|-----------|----------------|
| Tech Lead | 1 | Architecture, blockchain, security |
| Full-stack Engineer | 2-3 | Backend, frontend, API |
| DevOps | 0.5 | Infrastructure, deployment |
| Ops Manager | 1 | KYC, payouts, merchant support |
| Legal Advisor | 0.5 | Compliance, T&C, contracts |

**Total**: 5-6 people

---

### Budget (MVP - 6 weeks)

| Item | Cost (VND) |
|------|------------|
| **Infrastructure** | |
| VPS (staging + production) | 5M |
| Domain + SSL | 1M |
| Solana RPC (Helius/QuickNode) | 2M |
| Email service (SendGrid) | 1M |
| File storage (S3/MinIO) | 1M |
| **Services** | |
| Legal consultation | 10M |
| OTC partner setup | 5M |
| **Contingency (20%)** | 5M |
| **Total** | **30M VND** |

*Note: Salaries not included (depends on team)*

---

## Success Metrics

### Technical KPIs

- [ ] Payment success rate: **> 98%**
- [ ] Average confirmation time: **< 20 seconds**
- [ ] System uptime: **> 99%**
- [ ] Webhook delivery rate: **> 95%**
- [ ] Zero security incidents

### Business KPIs

- [ ] Pilot merchants: **5+**
- [ ] Total volume (Month 1): **1B+ VND**
- [ ] Revenue (Month 1): **10M+ VND**
- [ ] Merchant satisfaction: **NPS > 30**
- [ ] Zero compliance violations

---

## Risk Mitigation

| Risk | Mitigation | Owner |
|------|------------|-------|
| Regulatory shutdown | Da Nang sandbox approval, legal advisor | Legal |
| Hot wallet hack | Multi-layer security, insurance, limited balance | Tech Lead |
| OTC partner failure | 2-3 backup OTC partners lined up | Ops |
| Low merchant adoption | Pilot program, personal sales, referral incentives | Business |
| Technical failure | Comprehensive testing, monitoring, backups | Tech Lead |
| Key person risk | Documentation, knowledge sharing | Everyone |

---

## Go / No-Go Checklist (Before Launch)

### Legal ✅
- [ ] Business registered
- [ ] T&C reviewed by lawyer
- [ ] Privacy policy compliant with Vietnam law
- [ ] OTC partner contract signed
- [ ] Bank account opened

### Technical ✅
- [ ] All core features working on testnet
- [ ] Security audit passed
- [ ] Load testing passed
- [ ] Monitoring & alerts configured
- [ ] Backups configured
- [ ] Hot wallet funded (small amount)

### Operational ✅
- [ ] Ops team trained
- [ ] Support process documented
- [ ] Payout process documented
- [ ] OTC settlement process documented
- [ ] Incident response plan ready

### Business ✅
- [ ] 3+ pilot merchants committed
- [ ] Pricing confirmed
- [ ] Marketing materials ready
- [ ] Onboarding process tested

---

## Next Steps (After This Document)

1. **Validate with stakeholders**
   - Review this roadmap with team
   - Get buy-in from legal, ops, business
   - Adjust timeline if needed

2. **Set up project**
   - Create GitHub repo
   - Set up project management (Jira/Linear)
   - Create detailed tasks from this roadmap

3. **Start Week 1**
   - Assign tasks to team
   - Daily standups
   - Weekly sprint reviews

4. **Iterate**
   - Adjust based on learnings
   - Don't be afraid to cut scope if needed
   - Focus on launching, not perfection

---

## Appendix: Tech Stack Decisions

### Backend
- **Node.js + TypeScript**: Team expertise, good ecosystem
- **Express**: Simple, proven, flexible
- **Prisma**: Type-safe ORM, great DX
- **PostgreSQL**: ACID compliance for ledger

### Frontend
- **Next.js**: SSR, great DX, production-ready
- **TailwindCSS**: Fast UI development
- **shadcn/ui**: Beautiful components

### Blockchain
- **@solana/web3.js**: Official Solana library
- **Helius/QuickNode**: Reliable RPC providers

### Infrastructure
- **Docker**: Consistent environments
- **NGINX**: Reverse proxy, SSL termination
- **PM2**: Process management
- **Redis**: Caching, rate limiting

### Monitoring
- **Winston**: Logging
- **Email alerts**: Simple, works for MVP
- *Phase 2: Prometheus + Grafana*

