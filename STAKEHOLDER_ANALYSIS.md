# Stakeholder Analysis - Stablecoin Payment Gateway

**Context**:
- Tether + Da Nang partnership (Nov 2025)
- Da Nang sandbox: Resolution 222/2025/QH15 (International Financial Center)
- Target: Digital economy = 35-40% GRDP by 2030
- Focus: blockchain payment systems, RWA tokenization

---

## 🛍️ MERCHANT Perspective

### Pain Points (Current State)

**International Payments**
- Cross-border payment takes 3-5 days
- Bank fees: 2-5% per transaction
- Currency exchange spread: 1-3%
- Total cost: **4-8% + slow**

**Crypto-Savvy Customers**
- Many customers have crypto but can't spend it
- Losing sales to competitors who accept crypto
- Manual crypto → VND process is risky (fraud, price volatility)

**Cash Flow**
- Tourism businesses in Da Nang need instant settlement
- Hotel/restaurant can't wait 3 days for international card payments
- High-value items (jewelry, electronics) need fast confirmation

### What Merchants Want

**Must Have (Non-negotiable)**
1. **Legal & Safe**: Must comply with Vietnamese law
   - "I can't risk my business license"
   - "Need proper receipts for tax audit"

2. **Simple Integration**:
   - "I'm not a tech company, just want QR code"
   - "5 minutes setup, not 5 days"

3. **Predictable Costs**:
   - "Show me exactly how much I pay"
   - "No surprise fees or exchange rate games"

4. **Fast Settlement**:
   - "I need VND in my bank within 24 hours"
   - "Can't manage 2 wallets and exchanges myself"

**Nice to Have**
- Dashboard to see all transactions
- Export to Excel for accounting
- Mobile app to check on the go
- Multi-currency display (show USD, VND, crypto)

### Merchant Personas

**Persona 1: Tourism Business Owner (Hotel/Restaurant)**
- Location: Da Nang beachfront
- Revenue: 500M - 2B VND/month
- Customer: 60% international tourists
- Pain: High card fees (3-4%), chargebacks, slow settlement
- **Willingness to pay**: 1-2% (cheaper than Visa/Mastercard)

**Persona 2: E-commerce Seller (Electronics/Fashion)**
- Platform: Shopee, Lazada, own website
- Revenue: 1B - 5B VND/month
- Customer: Tech-savvy, crypto holders
- Pain: Payment gateway fees (2-3%), account freezing risk
- **Willingness to pay**: 0.5-1.5% (competitive with current gateways)

**Persona 3: Freelancer / Service Provider**
- Service: Design, dev, consulting
- Revenue: 50M - 300M VND/month
- Customer: Global clients
- Pain: PayPal fees (4-5%), withdrawal limits, account freezing
- **Willingness to pay**: 2-3% (cheaper than PayPal)

**Persona 4: Luxury Goods (Jewelry, Watches)**
- Location: High-end mall
- Transaction: 50M - 500M VND each
- Customer: Crypto whales, expats
- Pain: Large cash transactions (suspicious), bank limits
- **Willingness to pay**: 0.5-1% (volume based)

### Key Decision Factors

1. **Trust**: "Is this company legit? Will my money be safe?"
2. **Support**: "Can I call someone if there's a problem?"
3. **Speed**: "How fast do I get my VND?"
4. **Cost**: "Total cost must be < current solution"
5. **Legal**: "Can I show this to tax authorities?"

---

## 👤 END USER Perspective (Payer)

### User Personas

**Persona 1: Crypto Holder (Vietnamese)**
- Age: 25-35
- Profile: Tech worker, trader, early adopter
- Holdings: $5k - $50k in crypto (mostly USDT, USDC)
- Pain: "I have crypto but can't spend it easily"
- Motivation: "Use crypto for daily life, not just hold"

**Persona 2: International Tourist/Expat**
- Age: 30-50
- Profile: Digital nomad, tourist, expat
- Holdings: $1k - $20k in crypto
- Pain: "Don't want to carry cash, ATM fees are high"
- Motivation: "Convenient, secure, familiar (use crypto at home)"

**Persona 3: Remittance Sender**
- Age: 25-45
- Profile: Vietnamese working abroad, sending money home
- Holdings: $500 - $5k per month to send
- Pain: "Western Union/bank fees are 5-10%"
- Motivation: "Save money on fees, faster delivery"

### What Users Want

**Must Have**
1. **Simple & Fast**:
   - "Scan QR → send crypto → done"
   - "No account signup, no KYC for me (small amounts)"
   - "Payment confirmed in < 30 seconds"

2. **Clear Instructions**:
   - "Show me exact amount, address, memo"
   - "No mistakes → I don't want to lose money"

3. **Trustworthy**:
   - "Is merchant real? Will I get my product?"
   - "What if I send wrong amount?"

4. **Flexible**:
   - "Let me pay with any wallet (Phantom, MetaMask, Binance)"
   - "Support multiple stablecoins (USDT, USDC)"

**Nice to Have**
- Payment history ("Where did I spend my crypto?")
- Refund support
- Loyalty rewards
- Multi-language (EN, VN)

### User Journey (Happy Path)

```
1. Merchant shows QR code / payment link
   ↓
2. User scans with phone → payment page opens
   ↓
3. Payment page shows:
   - Amount: 2,300,000 VND = 100 USDT
   - Address: 8xK7...jP2m
   - Memo/Reference: PAY-12345
   - Instructions: "Send exactly 100 USDT on Solana"
   ↓
4. User opens wallet app (Phantom, Trust Wallet)
   ↓
5. User sends crypto
   ↓
6. Payment page auto-refreshes → "Payment received! Confirming..."
   ↓
7. After 10-15 seconds → "Payment confirmed ✓"
   ↓
8. Merchant receives notification → delivers product/service
```

### User Concerns

**Security**
- "What if I send to wrong address?"
- "What if website is fake?"
- "Can merchant see my wallet balance?" (privacy)

**Price**
- "Exchange rate fair or not?"
- "Any hidden fees?"

**Support**
- "Who do I contact if payment stuck?"
- "Can I get refund?"

---

## 🎯 PRODUCT OWNER Perspective

### Business Model Analysis

**Revenue Streams**
1. **Transaction fees**: 1% of payment volume
   - Conservative: 1B VND/month volume → 10M VND revenue
   - Target: 100B VND/month volume → 1B VND revenue

2. **Payout fees**: 50,000 VND per payout
   - Average: 100 payouts/month → 5M VND revenue

3. **Premium features** (Phase 2+):
   - API access: 5M VND/month
   - White-label: 20M VND/month
   - Priority support: 2M VND/month

4. **OTC spread** (hidden revenue):
   - Keep 0.3-0.5% from OTC conversion
   - 100B volume → 300-500M VND

5. **Yield/Staking** (Phase 3):
   - Stake idle stablecoins → 5-8% APY
   - 10B VND pool → 500M-800M VND/year

**Total Revenue Projection (Year 1)**
- Month 1-3 (Pilot): 5-10M VND/month
- Month 4-6 (Growth): 50-100M VND/month
- Month 7-12 (Scale): 200-500M VND/month
- **Year 1 Total**: ~2-3B VND revenue

### Cost Structure

**Fixed Costs (Monthly)**
- Engineering team (3-5 people): 100-150M VND
- Ops team (2 people): 40-60M VND
- Legal/Compliance: 20-30M VND
- Infrastructure (servers, tools): 20M VND
- Marketing: 30-50M VND
- **Total Fixed**: 210-310M VND/month

**Variable Costs**
- OTC conversion spread: ~0.3% (paid to OTC partner)
- Bank transfer fees: ~0.1%
- Customer support: scales with volume

**Break-even**
- Need ~50-100B VND volume/month to break even
- Target: 5-10 merchants × 10-20B VND/month each

### Market Opportunity

**TAM (Total Addressable Market) - Vietnam**
- E-commerce GMV: ~500T VND/year ($20B)
- Tourism revenue: ~800T VND/year ($32B)
- Freelance/gig economy: ~100T VND/year ($4B)
- **Total TAM**: ~1,400T VND/year

**SAM (Serviceable Available Market)**
- Da Nang + nearby cities: ~10% of TAM = 140T VND/year
- Crypto-friendly merchants: ~5% = 7T VND/year

**SOM (Serviceable Obtainable Market - Year 1)**
- Capture 0.1-0.5% of SAM = 7-35B VND/year
- **Realistic target**: 12-20B VND/year (Year 1)

### Competitive Analysis

**Current Alternatives**

| Solution | Fee | Speed | Crypto Support | Pain Point |
|----------|-----|-------|----------------|------------|
| Bank wire | 2-5% | 3-5 days | No | Slow, expensive |
| Visa/MC | 3-4% | 1-3 days | No | High fee, chargebacks |
| PayPal | 4-5% | Instant | No | Account freezing |
| Binance Pay | 0% | Instant | Yes | Not VND, compliance risk |
| Manual crypto | Variable | Manual | Yes | Complex, risky |
| **Our Solution** | **1-2%** | **<24h** | **Yes** | **New, unproven** |

**Competitive Advantages**
1. ✅ **Legal compliance** (Da Nang sandbox)
2. ✅ **VND settlement** (not crypto balance)
3. ✅ **Simple UX** (QR code, no tech knowledge needed)
4. ✅ **Lower fees** than cards
5. ✅ **Faster** than bank wire

**Competitive Disadvantages**
1. ❌ **New/unknown brand** (no trust yet)
2. ❌ **Limited to Da Nang** initially (regulatory)
3. ❌ **Manual processes** (MVP = slower ops)
4. ❌ **Small merchant network** (chicken-egg problem)

### GTM Strategy (Go-to-Market)

**Phase 1: Pilot (Month 1-3)**
- Target: 5-10 friendly merchants in Da Nang
- Focus: Tourism (hotels, restaurants), luxury goods
- Approach: Direct sales, personal relationships
- Goal: Prove product works, collect feedback

**Phase 2: Early Adopters (Month 4-6)**
- Target: 20-50 merchants
- Focus: E-commerce, freelancers
- Approach: Content marketing, referral program
- Goal: 10B+ VND volume/month

**Phase 3: Growth (Month 7-12)**
- Target: 100-200 merchants
- Focus: Expand beyond Da Nang
- Approach: Partnerships, integrations (Shopify, WooCommerce)
- Goal: 50B+ VND volume/month

### Risk Analysis

**High Priority Risks**

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Regulatory change | Medium | Critical | Da Nang sandbox, legal advisors |
| OTC partner fails | Low | Critical | Multiple OTC backups |
| Security breach | Low | Critical | Audit, insurance, best practices |
| Low merchant adoption | High | High | Pilot program, feedback loop |
| User confusion (UX) | Medium | Medium | User testing, clear instructions |

**Medium Priority Risks**
- Competition from Binance/international players
- Crypto price volatility (USDT depeg risk)
- Bank account closure
- Technical bugs causing payment loss

### Success Metrics (KPIs)

**North Star Metric**
- **VND volume processed per month**

**Primary KPIs**
- Number of active merchants
- Payment success rate (%)
- Average payment confirmation time
- Customer acquisition cost (CAC)
- Merchant lifetime value (LTV)

**Secondary KPIs**
- NPS (Net Promoter Score)
- Support ticket resolution time
- API uptime (%)
- Payout processing time

**Targets (Month 6)**
- 30+ active merchants
- 15B+ VND volume/month
- 98%+ payment success rate
- <15 second confirmation time
- NPS > 50

### Product Roadmap Priorities

**What to Build First (MVP)**
1. ✅ Core payment flow (QR → crypto → VND)
2. ✅ Merchant dashboard (basic)
3. ✅ Manual KYC + payout
4. ✅ 1 chain only (Solana USDT)
5. ✅ Legal compliance (audit logs, T&C)

**What to Build Next (Phase 2)**
1. Automated payouts
2. Multi-chain support
3. API + SDKs
4. Merchant integrations (Shopify, WooCommerce)
5. Better analytics

**What to Build Later (Phase 3)**
1. Yield layer
2. Mobile apps
3. Refund/dispute system
4. White-label solution
5. International expansion

### Strategic Decisions

**Build vs Buy vs Partner**

| Component | Decision | Reasoning |
|-----------|----------|-----------|
| Blockchain listener | **Build** | Core competency, need control |
| KYC service | **Partner** (Phase 2) | Not our expertise, regulated |
| OTC conversion | **Partner** | Need liquidity, legal complexity |
| Payment page | **Build** | Core UX, differentiation |
| Merchant dashboard | **Build** | Core product |
| Bank integration | **Partner** | Regulated, need bank relationships |
| Accounting/ERP | **Integrate** (Phase 3) | Merchants already have tools |

**Key Strategic Questions**

1. **Geography**: Da Nang first or Vietnam-wide?
   - **Decision**: Da Nang first (sandbox advantage)
   - **Rationale**: Legal clarity, pilot program, expand later

2. **Chains**: Single-chain or multi-chain?
   - **Decision**: Single-chain (Solana) for MVP
   - **Rationale**: Simplicity, fast/cheap, expand later

3. **Settlement**: Real-time or batch?
   - **Decision**: Batch (manual approval) for MVP
   - **Rationale**: Risk management, fraud prevention

4. **Target**: B2B (merchants) or B2C (users)?
   - **Decision**: B2B focus (merchants are customers)
   - **Rationale**: Users come automatically if merchants accept

5. **Pricing**: Fixed % or dynamic?
   - **Decision**: Fixed 1% for MVP
   - **Rationale**: Simple, predictable, competitive

---

## 🎯 MVP Prioritization Matrix

Using **RICE Framework** (Reach × Impact × Confidence / Effort)

| Feature | Reach | Impact | Confidence | Effort | RICE Score | Priority |
|---------|-------|--------|------------|--------|------------|----------|
| Payment creation + QR | 100% | 3 | 100% | 3 | **100** | ✅ P0 |
| Blockchain listener | 100% | 3 | 90% | 5 | **54** | ✅ P0 |
| Merchant dashboard | 80% | 2 | 100% | 4 | **40** | ✅ P0 |
| Manual payout | 100% | 2 | 100% | 2 | **100** | ✅ P0 |
| KYC form + approval | 100% | 3 | 100% | 3 | **100** | ✅ P0 |
| Webhook system | 60% | 2 | 80% | 3 | **32** | ✅ P0 |
| Multi-chain support | 40% | 2 | 70% | 7 | **8** | ❌ P1 |
| Automated payout | 80% | 2 | 80% | 5 | **26** | ⭐ P1 |
| API + SDK | 30% | 3 | 90% | 6 | **14** | ⭐ P1 |
| Refund system | 10% | 2 | 60% | 4 | **3** | ❌ P2 |
| Mobile app | 20% | 1 | 70% | 8 | **2** | ❌ P3 |
| Yield/staking | 5% | 2 | 50% | 8 | **0.6** | ❌ P3 |

---

## 💡 Key Insights & Recommendations

### From Merchant Perspective
- **Critical**: Trust & legal compliance > features
- **Keep it simple**: QR code, not complex integration
- **Predictable pricing**: Show all fees upfront
- **Fast settlement**: <24h is acceptable for MVP, instant is premium

### From User Perspective
- **Don't over-engineer**: Users don't care about tech, just want it to work
- **Clear instructions**: Reduce errors → reduce support costs
- **Mobile-first**: 90% of users will pay from phone
- **Multi-wallet support**: Don't force users to specific wallet

### From Product Perspective
- **Start narrow, go deep**: Da Nang only, 1 chain, simple features
- **Compliance is moat**: Legal advantage = competitive advantage
- **Network effects**: More merchants → more users → more merchants
- **Revenue model**: Transaction fees + OTC spread (don't rely on just one)

---

## 🚀 Final Recommendation

### MVP Scope (4-6 weeks)
Build **minimum** product that allows:
1. Merchant creates payment → QR code
2. User scans → sends crypto
3. System confirms → updates merchant balance
4. Merchant requests payout → manual approval → bank transfer

**Success = 5 merchants × 1B VND volume/month (6M VND revenue)**

### Phase 2 (2-3 months after MVP)
- Automated payouts
- Multi-chain (add Ethereum)
- API for integration
- Better UX (faster confirmation, nicer dashboard)

**Success = 30 merchants × 15B VND volume/month (150M VND revenue)**

### Phase 3 (6+ months)
- Yield layer (reduce fees, increase margin)
- White-label solution
- Expand beyond Da Nang
- International merchants

**Success = 100+ merchants × 50B+ VND volume/month (500M+ VND revenue)**

---

## ✅ Decision Framework for Feature Requests

When someone requests a feature, ask:

1. **Does it help MVP goal?** (Process first transaction legally)
   - Yes → Consider for MVP
   - No → Defer to Phase 2+

2. **Can we do it manually first?** (Reduce engineering effort)
   - Yes → Do manual for MVP, automate later
   - No → Must build

3. **Is it legally required?** (KYC, audit logs, etc.)
   - Yes → Must build
   - No → Nice to have

4. **Do > 50% of merchants need it?** (Validate demand)
   - Yes → High priority
   - No → Low priority

**Example: Refund feature**
- Helps MVP? No (rare case)
- Can do manually? Yes (ops can process manually)
- Legally required? No
- >50% need? No (< 5% of transactions)
- **Decision: Phase 2+, manual process for MVP**

