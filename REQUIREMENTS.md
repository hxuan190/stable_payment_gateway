# Stablecoin Payment Gateway MVP - Requirements

**Vision**: "Merchant tạo QR → User scan → Crypto → OTC convert → Merchant nhận VND"

**Goal**: Build minimum viable product that can process real transactions legally, with architecture ready to scale.

---

## 🎯 MVP SCOPE (Phase 1 - 4-6 weeks)

### ✅ MUST HAVE (MVP Core)

#### 🔧 TECH

**Backend Core**
- [x] Multi-chain listener (priority: 1 chain first - Solana USDT recommended)
  - Detect incoming transactions
  - Confirm transaction finality
  - Validate payment_id + amount
- [x] Ledger system
  - Track: crypto_received → vnd_pending → vnd_balance → vnd_paid
  - Double-entry accounting
  - Merchant balance tracking
- [x] Payment API
  - Create payment request
  - Generate QR code / payment link
  - Query payment status
  - Webhook callback to merchant
- [x] Wallet management (simplified MVP)
  - 1 hot wallet per chain (receive payments)
  - Manual cold wallet transfer (ops process)
  - Transaction signing service
- [x] OTC integration
  - API to trigger crypto→VND conversion
  - Manual fallback process
  - Settlement tracking

**Database Schema**
- [x] Merchants (id, kyc_status, wallet_address, balance_vnd)
- [x] Payments (id, merchant_id, amount_crypto, amount_vnd, tx_hash, status)
- [x] Payouts (id, merchant_id, amount, bank_info, status, fee)
- [x] Ledger (transaction log for accounting)
- [x] Audit_logs (all critical operations)

**Security (Baseline)**
- [x] API authentication (API key per merchant)
- [x] Webhook signature verification
- [x] Rate limiting
- [x] Database encryption at rest
- [x] Secrets management (env vars / vault)

#### 📋 COMPLIANCE

**KYC/AML (Manual Process MVP)**
- [x] Merchant registration form
  - Business name, tax ID, address
  - Owner ID, phone, email
  - Business license upload
- [x] Manual KYC review (admin panel)
  - Approve/reject merchant
  - Store KYC documents (encrypted)
- [x] Audit logging
  - All payments: payment_id, merchant_id, tx_hash, amount, timestamp
  - All payouts: payout_id, merchant_id, bank_info, amount, timestamp
  - KYC actions: who approved, when

**Legal Compliance**
- [x] Terms of Service template
- [x] Privacy Policy template
- [x] Record retention policy (7 years)
- [x] Transaction limits (start conservative, e.g., 10M VND/transaction max)

#### 🏪 MERCHANT FEATURES

**Registration & Onboarding**
- [x] Signup form with KYC submission
- [x] Email verification
- [x] Pending approval status page

**Payment Operations**
- [x] Create payment request (amount_vnd, order_id, callback_url)
- [x] Get QR code (data: wallet_address, amount_crypto, payment_id)
- [x] View payment status (pending/confirmed/completed/failed)
- [x] Receive webhook notification (payment confirmed)

**Dashboard (Basic)**
- [x] Current VND balance
- [x] Payment history (last 30 days)
- [x] Payout history
- [x] Request payout (manual approval for MVP)

**Payout**
- [x] Request VND withdrawal (amount, bank account info)
- [x] Manual approval process (admin reviews, triggers bank transfer)
- [x] Email notification when payout completed
- [x] Simple fee: flat 1-2% (no instant vs batch for MVP)

#### 👤 END USER EXPERIENCE

**Payment Flow**
- [x] Scan QR code or click payment link
- [x] Show: amount in USDT, destination wallet address, payment_id (memo)
- [x] Instructions: "Send exactly X USDT to address Y with memo Z"
- [x] Payment status page: pending → confirmed → completed
- [x] Success page with confirmation

#### 🔄 OPERATIONS

**OTC Settlement**
- [x] Manual process:
  1. Check hot wallet balance daily
  2. Send crypto to OTC partner
  3. Receive VND to business bank account
  4. Update system VND pool
- [x] Internal admin tool to record OTC transactions

**Payout Process**
- [x] Manual bank transfer (admin reviews payout requests)
- [x] Update ledger after bank transfer confirmed
- [x] Email merchant with transaction reference

**Monitoring (Basic)**
- [x] Health check endpoint
- [x] Failed transaction alerts (email to ops)
- [x] Daily settlement report

---

## 🔶 PHASE 2 (After MVP Launch - 2-3 months)

### SHOULD HAVE

#### 🔧 TECH
- [ ] Multi-chain support (add Ethereum USDC, BSC BUSD)
- [ ] Automated OTC integration (API-based settlement)
- [ ] Cold wallet + multi-sig setup
- [ ] Automated payout batching (scheduled daily/weekly)
- [ ] Two-tier payout: instant (2%) vs batch (0.5%)
- [ ] Retry mechanism for failed webhooks
- [ ] Advanced monitoring (Prometheus/Grafana)

#### 📋 COMPLIANCE
- [ ] Automated KYC service (e.g., Sumsub, Onfido)
- [ ] Transaction monitoring rules (AML automation)
- [ ] Suspicious activity reporting workflow

#### 🏪 MERCHANT
- [ ] API documentation (public)
- [ ] Merchant API integration (RESTful + SDKs)
- [ ] Advanced dashboard (analytics, charts)
- [ ] Fee breakdown transparency
- [ ] Multi-currency support (USD, EUR display)

#### 👤 USER
- [ ] Mobile-optimized payment page
- [ ] Support multiple wallets (MetaMask, Phantom, Trust Wallet)
- [ ] Better UX for tx confirmation waiting

#### 🔄 OPS
- [ ] Automated OTC triggers (when balance > threshold)
- [ ] Automated payout processing
- [ ] Bank reconciliation automation

---

## ⭐ PHASE 3+ (Scale & Optimize - 6+ months)

### NICE TO HAVE

#### Advanced Features
- [ ] Yield/staking layer (stake idle stablecoins, earn yield → reduce fees)
- [ ] Refund/dispute handling system
- [ ] Partial payments
- [ ] Recurring payments / subscriptions
- [ ] Merchant loyalty program (volume discounts)
- [ ] White-label solution for partners

#### UX Enhancements
- [ ] Mobile app (merchant POS)
- [ ] Tablet POS integration
- [ ] NFC payment support
- [ ] Multi-language (English, Vietnamese)

#### Advanced Tech
- [ ] HSM integration for wallet security
- [ ] Multi-region deployment (latency reduction)
- [ ] Advanced fraud detection (ML-based)
- [ ] Blockchain reorg handling
- [ ] Gas fee optimization
- [ ] Custom token support

---

## 🎯 MVP Success Criteria

**Technical**
- [ ] Process 1 real payment successfully (testnet → mainnet)
- [ ] 99% uptime for 1 month
- [ ] <10 second payment detection
- [ ] <24 hour payout processing

**Business**
- [ ] 5 pilot merchants onboarded
- [ ] 100 transactions processed
- [ ] 1M+ VND in volume
- [ ] <1% error rate

**Compliance**
- [ ] All transactions properly logged
- [ ] KYC records stored securely
- [ ] Zero compliance violations

---

## 📊 MVP Technical Architecture

```
┌─────────────┐
│  End User   │
│ (Mobile/Web)│
└──────┬──────┘
       │ Scan QR / Click Link
       ▼
┌─────────────────────┐
│  Payment Page       │
│  (Next.js/React)    │
│  - Show QR          │
│  - Instructions     │
│  - Status tracking  │
└─────────────────────┘
       │
       │ Send crypto to wallet
       ▼
┌─────────────────────┐     ┌──────────────┐
│  Blockchain         │────▶│ Blockchain   │
│  (Solana/Ethereum)  │     │ Listener     │
└─────────────────────┘     │ (Node.js)    │
                            └──────┬───────┘
                                   │
                                   ▼
                            ┌─────────────┐
                            │   Gateway   │
                            │   Backend   │
                            │ (Node.js/   │
                            │  Express)   │
                            │             │
                            │ - Validate  │
                            │ - Update    │
                            │   Ledger    │
                            │ - Webhook   │
                            └─────┬───────┘
                                  │
                    ┌─────────────┼─────────────┐
                    │             │             │
                    ▼             ▼             ▼
              ┌──────────┐  ┌─────────┐  ┌──────────┐
              │PostgreSQL│  │  OTC    │  │ Merchant │
              │ Database │  │ Partner │  │Dashboard │
              └──────────┘  │  API    │  │(Next.js) │
                            └─────────┘  └──────────┘
```

---

## 🚀 MVP Tech Stack Recommendation

**Backend**
- Node.js + TypeScript + Express (or NestJS)
- PostgreSQL (ledger, transactions)
- Redis (caching, rate limiting)
- Prisma ORM

**Blockchain Interaction**
- Solana: @solana/web3.js
- Ethereum: ethers.js / web3.js

**Frontend (Merchant Dashboard)**
- Next.js + TypeScript
- TailwindCSS
- shadcn/ui components

**Payment Page**
- Next.js (static generation)
- QR code library (qrcode.react)

**Infrastructure (MVP)**
- Docker + Docker Compose
- Single VPS (Digital Ocean / AWS EC2)
- Cloudflare (CDN + DDoS protection)
- GitHub Actions (CI/CD)

**Monitoring (MVP)**
- PM2 (process management)
- Winston (logging)
- Email alerts (via SendGrid/SES)

---

## 💰 Fee Structure (MVP)

**For Merchants**
- Payment processing: 1% (covers OTC spread + ops)
- Payout fee: Flat 50,000 VND per payout (manual processing)
- Minimum payout: 1,000,000 VND

**Revenue Model**
- Keep 0.3-0.5% spread from OTC
- Payout fees
- (Future: yield from staking)

---

## ⚖️ Legal Considerations (Vietnam-specific)

**Critical for MVP**
1. Business license for payment intermediary service
2. Partner with licensed OTC desk or exchange
3. Clear T&C: "We are NOT a financial institution"
4. Crypto is treated as digital asset, NOT currency in VN
5. All settlements in VND (fiat), not crypto
6. Keep detailed records for tax audit

**Recommended**
- Consult with fintech lawyer in Vietnam
- Register business as technology service provider
- Partner with local bank for VND settlements
- Consider pilot with small volume first

---

## 🎬 Implementation Order (MVP)

### Week 1-2: Foundation
1. Project setup (monorepo structure)
2. Database schema + Prisma setup
3. Basic API structure
4. Authentication system

### Week 3-4: Core Payment Flow
5. Blockchain listener (Solana USDT)
6. Payment creation API
7. QR code generation
8. Payment status tracking
9. Webhook system

### Week 5: Merchant Features
10. Merchant registration + KYC form
11. Admin panel (KYC approval)
12. Basic dashboard
13. Payout request system

### Week 6: Testing & Deployment
14. End-to-end testing (testnet)
15. Security audit (basic)
16. Deploy to production
17. Pilot merchant onboarding

---

## 📝 Notes

**Why Start with Solana USDT?**
- Fast finality (~400ms vs 12s Ethereum)
- Low tx fees (~$0.001 vs $5+ Ethereum)
- Good stablecoin liquidity in Asia
- Easy to add Ethereum later

**Why Manual OTC for MVP?**
- Automated OTC requires legal entity setup
- Manual process = faster to launch
- Can switch to API later without architecture change

**Why Manual Payout for MVP?**
- Banking API integration takes time
- Manual = full control + security
- Batch processing can be added later

**Scalability Built In**
- Ledger system supports any volume
- Blockchain listener can be parallelized
- Database schema supports multi-chain
- API design allows for automation later

---

## ✅ Decision Log

| Decision | Reasoning | Trade-off |
|----------|-----------|-----------|
| Single chain (Solana) first | Faster MVP, less complexity | Limited user wallet options |
| Manual OTC | Legal simplicity, faster launch | Operational overhead |
| Manual KYC | No vendor lock-in, lower cost | Slower onboarding |
| PostgreSQL | ACID compliance for ledger | Not as scalable as NoSQL |
| Flat fee structure | Simple to explain/implement | Less competitive pricing |
| No instant payout | Reduce risk, manual review | Worse merchant UX |

