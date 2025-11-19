# Stablecoin Payment Gateway - MVP

> **"Merchant tạo QR → User scan → Crypto → OTC convert → Merchant nhận VND"**

## 🎯 Vision

Build a legal, compliant stablecoin payment gateway for Vietnam (starting with Da Nang) that allows merchants to accept crypto payments and receive VND settlements.

**Market Opportunity**: Tether + Da Nang partnership (Nov 2025) creates regulatory sandbox for blockchain payment systems. Perfect timing to launch.

---

## 📚 Documentation

### Core Documents

| Document | Description |
|----------|-------------|
| **[PRD_v2.2.md](./PRD_v2.2.md)** | **🆕 Product Requirements Document v2.2** - Complete requirements with new features |
| **[REQUIREMENTS.md](./REQUIREMENTS.md)** | Full functional/non-functional requirements, phased by MVP → Phase 2 → Phase 3 |
| **[ARCHITECTURE.md](./ARCHITECTURE.md)** | Technical architecture, system design, database schema, API specs + PRD v2.2 updates |
| **[AML_ENGINE.md](./AML_ENGINE.md)** | **In-house AML compliance engine**: risk scoring, transaction monitoring, sanctions screening |
| **[STAKEHOLDER_ANALYSIS.md](./STAKEHOLDER_ANALYSIS.md)** | Merchant, User, Product Owner perspectives + business model |
| **[MVP_ROADMAP.md](./MVP_ROADMAP.md)** | Week-by-week implementation plan (4-6 weeks to launch) |
| **[TECH_STACK_GOLANG.md](./TECH_STACK_GOLANG.md)** | Golang implementation guide: frameworks, libraries, multi-chain architecture |
| **[TOURISM_USE_CASES.md](./TOURISM_USE_CASES.md)** | Detailed use cases for hotels, restaurants, tourist services in Da Nang |
| **[GETTING_STARTED.md](./GETTING_STARTED.md)** | Step-by-step guide for dev team to start implementation |
| **[CLAUDE.md](./CLAUDE.md)** | Comprehensive guide for AI assistants working on this project |

### PRD v2.2 - New Modules (Design Phase)

| Document | Description |
|----------|-------------|
| **[IDENTITY_MAPPING.md](./IDENTITY_MAPPING.md)** | **🆕 Smart Wallet→User Identity** - One-time KYC with Redis caching |
| **[NOTIFICATION_CENTER.md](./NOTIFICATION_CENTER.md)** | **🆕 Omni-channel Notifications** - Speaker/TTS, Telegram, Zalo, Email, Webhook |
| **[DATA_RETENTION.md](./DATA_RETENTION.md)** | **🆕 Infinite Storage** - S3 Glacier archival + transaction hashing |
| **[OFF_RAMP_STRATEGIES.md](./OFF_RAMP_STRATEGIES.md)** | **🆕 Flexible Withdrawals** - On-demand, Scheduled, Threshold-based |

---

## 🚀 Quick Summary

### PRD v2.2 Scope (8-10 weeks)

**Core Features (MVP)**
- ✅ Merchant creates payment → QR code generated
- ✅ User scans QR → sends crypto (USDT on **TRON/Solana/BSC**)
- ✅ System detects payment → confirms → updates merchant balance
- ✅ Merchant requests payout → manual approval → VND bank transfer
- ✅ **In-house AML Engine**: transaction monitoring, risk scoring, sanctions screening
- ✅ KYC compliance (manual review)
- ✅ Audit logging for all transactions

**🆕 PRD v2.2 Enhancements**
- ✅ **Smart Identity Mapping**: One-time KYC, returning users auto-recognized
- ✅ **Omni-channel Notifications**: Speaker/TTS, Telegram Bot, Zalo OA/ZNS, Email
- ✅ **Custodial Treasury**: Multi-sig cold wallet + auto-sweeping (every 6 hours)
- ✅ **Flexible Off-ramp**: On-demand, Scheduled (weekly/monthly), Threshold-based
- ✅ **Infinite Data Retention**: S3 Glacier archival + transaction hashing (immutability)

**Tech Stack**
- Backend: **Golang** + Gin + GORM + PostgreSQL
- Frontend: Next.js + TailwindCSS
- Blockchain: **TRON + Solana + BSC** (multi-chain)
  - TRON: USDT (TRC20) - Cheapest fees (~$1)
  - Solana: USDT, USDC (SPL) - Fastest finality (~13s)
  - BSC: USDT, BUSD (BEP20) - Popular in SEA
- **New Integrations**:
  - Sumsub: KYC & Face Liveness ($0.50/check)
  - Telegram Bot API: Real-time notifications
  - Zalo OA/ZNS: Vietnam market leader ($0.01/msg)
  - Google Cloud TTS: Speaker notifications ($4/1M chars)
  - AWS S3 Glacier: Long-term storage ($4/TB/month)
- Infrastructure: Docker + NGINX + Redis + Bull Queue

**Target Metrics (Month 1)**
- 5 pilot merchants
- 1B+ VND volume
- 10M+ VND revenue
- 99% uptime
- **🆕 KYC Recognition Rate > 95%**
- **🆕 Notification Delivery > 95%**

---

## 💰 Business Model

### Revenue Streams
1. **Transaction fees**: 1% of payment volume
2. **Payout fees**: 50,000 VND per withdrawal
3. **OTC spread**: 0.3-0.5% (hidden revenue)
4. *(Future) Yield/staking*: 5-8% APY on idle stablecoins

### Competitive Advantage
- ✅ Legal compliance (Da Nang sandbox)
- ✅ VND settlement (not crypto balance)
- ✅ Lower fees than cards (1% vs 3-4%)
- ✅ Faster than bank wire (<24h vs 3-5 days)

---

## 🎯 Target Market

### Primary: Tourism in Da Nang (Hotels & Restaurants)
- **Hotels**: Room deposits, check-in payments, spa/minibar extras
- **Restaurants**: Dine-in bills, takeaway orders, tourist group payments
- **Tourist Services**: Water sports, spa, motorbike rental, day tours
- **Multi-stablecoin**: USDT/USDC (Solana), USDT/BUSD (BSC) for flexibility

### Market Size
- Vietnam e-commerce: 500T VND/year
- Da Nang tourism: ~80T VND/year
- **Target (Year 1)**: 12-20B VND volume

---

## 📊 Implementation Roadmap

### Week 1-2: Foundation
- Project setup (monorepo, database, auth)
- Basic API structure
- Development environment

### Week 3-4: Core Payment Flow
- Payment creation API
- Blockchain listener (Solana)
- Ledger system
- Payment confirmation

### Week 5: Merchant Features
- Merchant dashboard
- KYC form + approval
- Payout request system

### Week 6: Launch Preparation
- Admin panel
- Security audit
- Testing (testnet → mainnet)
- Documentation
- Pilot merchant onboarding

**→ Full detailed roadmap: [MVP_ROADMAP.md](./MVP_ROADMAP.md)**

---

## 🏛️ Legal & Compliance

### Vietnam Regulatory Context
- **Da Nang Sandbox**: Resolution 222/2025/QH15 (International Financial Center)
- **Tether Partnership**: Nov 2025 - blockchain governance & payment systems
- **Compliance**: KYC/AML, audit logs, T&C, privacy policy

### Risk Mitigation
- Partner with licensed OTC desk
- Clear T&C: "We are NOT a financial institution"
- Manual KYC review (no automated approval for MVP)
- Conservative transaction limits
- Legal advisor on retainer

---

## 🔐 Security

### MVP Security Measures
- API authentication (API keys for merchants, JWT for admin)
- Rate limiting (100 req/min)
- Webhook HMAC signature verification
- Database encryption at rest
- Audit logging for all operations
- Private keys in environment vault
- Hot wallet with minimum balance (<$10k)

### Phase 2 Enhancements
- Multi-sig cold wallet
- HSM for key management
- Automated fraud detection
- Advanced monitoring (Prometheus/Grafana)

---

## 🛡️ AML Compliance (In-House Engine)

### Why Build In-House?
- **Cost Savings**: 3rd party AML services: $500-5,000/month → In-house: development cost only
- **Customization**: Vietnam-specific rules, tourism industry patterns, crypto-specific monitoring
- **Data Control**: Keep sensitive customer data in-house, comply with local regulations

### AML Engine Capabilities

**Customer Risk Scoring**
- Risk levels: Low, Medium, High, Prohibited
- Factors: Business type, KYC completeness, transaction volume, geographic risk, PEP status
- Periodic reviews based on risk level (quarterly/annually)

**Transaction Monitoring**
- Threshold monitoring (Vietnam: 400M VND legal requirement)
- Structuring detection (multiple transactions below threshold)
- Velocity anomalies (sudden spikes in volume/frequency)
- Rapid cash-out detection (payment → immediate withdrawal)
- Behavioral analysis (deviation from merchant baseline)

**Sanctions Screening**
- OFAC SDN List (US Treasury)
- UN Consolidated Sanctions List
- EU Sanctions Map
- Vietnam government blacklists
- Crypto wallet sanctions (OFAC sanctioned addresses)

**Wallet Risk Analysis (Crypto-Specific)**
- Mixing service detection (Tornado Cash, etc.)
- Darknet market exposure
- Sanctioned wallet addresses
- Transaction history analysis (3-hop depth)

**Alert Management**
- Severity levels: LOW, MEDIUM, HIGH, CRITICAL
- Workflow: Created → Assigned → Under Review → Resolved
- Auto-prioritization for compliance team
- False positive tracking and rule tuning

**Regulatory Reporting**
- Suspicious Activity Reports (SAR)
- Threshold reports (>400M VND transactions)
- Monthly/quarterly compliance summaries
- Audit trail for regulatory inquiries

### Compliance Standards
- ✅ FATF 40 Recommendations
- ✅ Vietnam Law on Anti-Money Laundering (2022)
- ✅ FATF Travel Rule for crypto (≥ $1,000 USD)
- ✅ Threshold reporting: 400M VND (~$16,000 USD)

**→ Full details: [AML_ENGINE.md](./AML_ENGINE.md)**

---

## 📈 Success Criteria

### Technical
- [ ] Payment success rate > 98%
- [ ] Average confirmation time < 20 seconds
- [ ] System uptime > 99%
- [ ] Zero security incidents

### Business
- [ ] 5+ pilot merchants onboarded
- [ ] 100+ transactions processed
- [ ] 1B+ VND volume (Month 1)
- [ ] NPS > 30

### Compliance
- [ ] All transactions properly logged
- [ ] KYC records stored securely
- [ ] Zero compliance violations

---

## 🧑‍💼 Team Requirements

| Role | Headcount | Key Responsibilities |
|------|-----------|---------------------|
| Tech Lead | 1 | Architecture, blockchain, security |
| Full-stack Engineers | 2-3 | Backend, frontend, API |
| DevOps | 0.5 | Infrastructure, deployment |
| Ops Manager | 1 | KYC, payouts, merchant support |
| Legal Advisor | 0.5 | Compliance, contracts |

**Total**: 5-6 people

---

## 💵 Budget (MVP)

| Category | Cost (VND) |
|----------|-----------|
| Infrastructure (servers, tools) | 10M |
| Legal & compliance | 10M |
| OTC partner setup | 5M |
| Contingency (20%) | 5M |
| **Total** | **30M VND** |

*Salaries not included*

---

## 🎬 Next Steps

### For Product Owner / Founder
1. ✅ Review all documentation
2. ✅ Validate business model & pricing
3. ⏭️ Secure legal advisor (Vietnam fintech lawyer)
4. ⏭️ Line up OTC partners (2-3 options)
5. ⏭️ Identify 3-5 pilot merchants
6. ⏭️ Secure funding (if needed)

### For Tech Team
1. ✅ Review architecture & tech stack
2. ⏭️ Set up GitHub repo + project management
3. ⏭️ Start Week 1 tasks (see MVP_ROADMAP.md)
4. ⏭️ Daily standups, weekly reviews

### For Legal/Compliance
1. ⏭️ Register business entity
2. ⏭️ Draft T&C, Privacy Policy, Merchant Agreement
3. ⏭️ Apply for Da Nang sandbox (if required)
4. ⏭️ Set up bank account

### For Ops
1. ⏭️ Document KYC process
2. ⏭️ Document payout process
3. ⏭️ Set up support channels (email, phone)
4. ⏭️ Create merchant onboarding checklist

---

## 📞 Key Questions to Answer Before Build

- [ ] **Legal**: Do we have lawyer approval for Da Nang operations?
- [ ] **OTC**: Which OTC partner(s) will we use? Contract signed?
- [ ] **Banking**: Which bank for VND settlements? Account ready?
- [ ] **Merchants**: Who are our 3-5 pilot merchants?
- [ ] **Funding**: Do we have 6 months runway (salaries + ops)?
- [ ] **Team**: Do we have committed team for 6 weeks?

---

## 🌟 Why This Will Work

### Market Timing
- ✅ Tether + Da Nang partnership = regulatory green light
- ✅ Vietnam crypto adoption growing (top 10 globally)
- ✅ Tourism recovery post-COVID = demand for payment solutions

### Product-Market Fit
- ✅ Real pain point: merchants losing sales from crypto holders
- ✅ Clear value prop: lower fees + faster settlement
- ✅ Simple UX: QR code (merchants already understand)

### Competitive Moat
- ✅ Legal compliance = barrier to entry
- ✅ First-mover in Da Nang sandbox
- ✅ Network effects (more merchants → more users)

### Execution Risk: Low
- ✅ Proven tech stack
- ✅ Manual ops for MVP (de-risk automation)
- ✅ Small pilot (5 merchants) before scale
- ✅ Clear 6-week roadmap

---

## 📄 License & Legal

- Code: MIT License (TBD)
- Product: Requires merchant agreement, T&C
- Data: GDPR/Vietnam privacy law compliant

---

## 📧 Contact

- **Project Owner**: [TBD]
- **Tech Lead**: [TBD]
- **Legal Advisor**: [TBD]

---

**Built for Vietnam's blockchain future 🇻🇳**

*Last updated: 2025-11-19*
