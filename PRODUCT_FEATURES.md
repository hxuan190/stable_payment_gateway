# PRODUCT FEATURES - Core Product Capabilities

**Project**: Stablecoin Payment Gateway for Vietnam Tourism
**Version**: 1.0 (MVP)
**Last Updated**: 2025-11-18
**Product Stage**: Pre-Launch

---

## 📖 Table of Contents

1. [Product Overview](#product-overview)
2. [Target Users & Personas](#target-users--personas)
3. [Core Features](#core-features)
4. [User Journeys](#user-journeys)
5. [Feature Matrix (MVP vs Future)](#feature-matrix-mvp-vs-future)
6. [Competitive Advantages](#competitive-advantages)
7. [Success Metrics](#success-metrics)

---

## Product Overview

### What We Do

**Stablecoin Payment Gateway** enables Vietnamese tourism businesses to accept cryptocurrency payments from international tourists and receive Vietnamese Dong (VND) in their bank accounts.

### The Problem We Solve

**For Tourists:**
- Carrying cash is risky and inconvenient
- Currency exchange has poor rates and high fees
- Credit card acceptance is limited in Vietnam
- International transaction fees are expensive (3-5%)

**For Merchants:**
- Missing revenue from crypto-holding tourists
- Currency exchange costs eat into margins
- Payment processing fees are high
- Settlement takes days with traditional processors

### Our Solution

**Simple 3-Step Process:**

```
1. Merchant creates payment → QR code generated
2. Tourist scans QR → Pays with USDT/USDC
3. System converts crypto → Merchant receives VND
```

**Value Proposition:**
> "Accept crypto payments from tourists worldwide, receive VND in your bank account. No crypto knowledge required."

### Market Opportunity

- **Da Nang Partnership**: Tether + Da Nang city collaboration (Nov 2025)
- **Regulatory Sandbox**: Pilot program for blockchain payments
- **Tourism Volume**: 8.8M international visitors to Da Nang (2024)
- **Market Gap**: No existing crypto-to-VND payment solution for merchants

---

## Target Users & Personas

### Primary Persona 1: Hotel Manager (Merchant)

**Background:**
- Name: Minh, 38 years old
- Role: Manager of boutique hotel in Da Nang
- Tech Savvy: Medium (comfortable with POS, booking systems)
- Pain Points: High credit card fees, currency exchange hassles

**Goals:**
- Accept international payments easily
- Reduce payment processing costs
- Get VND in bank account quickly
- Provide modern payment options to guests

**Needs from Product:**
- Simple setup (no technical knowledge)
- Clear dashboard to track payments
- Reliable VND settlement
- Support when issues arise

**Quote:**
> "I don't need to understand crypto. I just want tourists to pay easily and get VND in my account."

---

### Primary Persona 2: International Tourist (End User)

**Background:**
- Name: Alex, 29 years old
- From: Europe/US/Asia
- Tech Savvy: High (owns crypto, uses mobile wallet)
- Pain Points: Carrying cash, poor exchange rates

**Goals:**
- Pay merchants directly with crypto
- Avoid currency exchange fees
- Use digital wallet instead of cash
- Fast, secure transactions

**Needs from Product:**
- Standard QR code payment (works with existing wallet)
- Clear payment amount in crypto
- Instant confirmation
- Receipt for records

**Quote:**
> "I hold USDT in my wallet. Why can't I just pay directly instead of exchanging to cash?"

---

### Secondary Persona 3: Tour Operator

**Background:**
- Name: Thanh, 42 years old
- Runs tour company with 15 guides
- Handles bookings, transportation, activities
- Deals with international customers daily

**Goals:**
- Accept online bookings with crypto payment
- Reduce payment gateway fees (currently 3-4%)
- Settle in VND for local expenses
- Avoid chargeback fraud

**Needs from Product:**
- API integration with booking system
- Webhook notifications
- Bulk payment management
- Detailed reporting

---

## Core Features

### Feature 1: QR Code Payment

**Description:**
Merchants generate a QR code for each payment. Tourists scan with their crypto wallet and pay instantly.

**User Flow:**
```
Merchant creates payment → System generates QR code →
Tourist scans QR → Wallet pre-fills details →
Tourist confirms → Payment received
```

**Key Capabilities:**
- **Instant QR Generation**: Create QR in < 1 second
- **Multi-Chain Support**: Solana (USDT, USDC), BSC (USDT)
- **Pre-Filled Amount**: Exact amount auto-populated in wallet
- **Payment Memo**: Auto-included reference ID for tracking
- **Expiry Timer**: 30-minute validity to prevent rate abuse

**Benefits:**
- ✅ No manual address entry (prevents errors)
- ✅ Works with any standard crypto wallet
- ✅ Mobile-friendly (scan from phone)
- ✅ No app download required for tourists

**Technical Details:**
- Standard Solana Pay protocol
- BIP-21 URI format for compatibility
- Base64 encoded QR image
- Responsive display (mobile/desktop)

**MVP Scope:**
- ✅ Single payment per QR
- ✅ Fixed amount (no partial payments)
- ✅ VND input, crypto output
- ❌ Recurring payments (future)
- ❌ Dynamic pricing (future)

---

### Feature 2: Real-Time Payment Tracking

**Description:**
Both merchants and tourists can track payment status in real-time as it progresses through confirmation.

**Status Flow:**
```
Created → Pending → Confirming → Completed
           ↓
        Expired (if no payment within 30 min)
```

**Key Capabilities:**
- **Live Status Updates**: Real-time blockchain monitoring
- **Confirmation Progress**: Show confirmations count
- **Time Estimates**: "~13 seconds remaining" for Solana
- **Success Notifications**: Email + webhook when complete
- **Payment Receipt**: Digital receipt with transaction hash

**Benefits:**
- ✅ Transparency for both parties
- ✅ Reduces anxiety during payment
- ✅ Clear confirmation when complete
- ✅ Audit trail for accounting

**User Interface:**

**Merchant View:**
```
Payment Status: ⏳ Confirming

Amount: 2,300,000 VND (100 USDT)
Blockchain: Solana
Confirmations: 12/13
Estimated: 5 seconds

[View Transaction] [Cancel]
```

**Tourist View:**
```
✅ Payment Sent!

Your payment of 100 USDT is being confirmed.
Expected confirmation: ~10 seconds

Transaction: abc123...xyz789
Status: Confirming (12/13 blocks)

[View Receipt]
```

**MVP Scope:**
- ✅ Basic status page
- ✅ Auto-refresh every 2 seconds
- ✅ Email confirmation
- ❌ SMS notifications (future)
- ❌ Push notifications (future)

---

### Feature 3: Merchant Dashboard

**Description:**
Web-based dashboard for merchants to manage payments, view balance, and request payouts.

**Key Sections:**

#### 3.1 Overview Tab
- **Today's Summary**: Total received, payment count, average amount
- **Balance Card**: Available VND, pending payouts
- **Recent Payments**: Last 10 transactions with status
- **Quick Actions**: Create payment, request payout

**Sample View:**
```
╔════════════════════════════════════════════╗
║  📊 Today's Performance                    ║
╠════════════════════════════════════════════╣
║  Total Received:      5,750,000 VND        ║
║  Payments:            3 transactions       ║
║  Average Amount:      1,916,667 VND        ║
╚════════════════════════════════════════════╝

╔════════════════════════════════════════════╗
║  💰 Balance                                ║
╠════════════════════════════════════════════╣
║  Available:           12,450,000 VND       ║
║  Pending Payout:      0 VND                ║
║                                            ║
║  [Request Payout] [View History]           ║
╚════════════════════════════════════════════╝
```

#### 3.2 Payments Tab
- **Payment List**: Filterable table of all payments
- **Search**: By amount, date, customer email, transaction hash
- **Filters**: Status, date range, blockchain
- **Export**: CSV download for accounting

**Columns:**
- Date/Time
- Amount (VND)
- Amount (Crypto)
- Customer Email
- Status
- Blockchain
- Actions (View Details, Download Receipt)

#### 3.3 Payouts Tab
- **Payout History**: All withdrawal requests
- **Status Tracking**: Requested → Approved → Processing → Completed
- **Bank Details**: Saved bank account information
- **Fee Breakdown**: Clear display of payout fees

#### 3.4 Settings Tab
- **Business Information**: Name, address, contact
- **Bank Account**: VND settlement account details
- **API Keys**: Generate and manage API keys
- **Webhooks**: Configure webhook URL and events
- **Notifications**: Email preferences

**Benefits:**
- ✅ Self-service merchant operations
- ✅ Real-time visibility into finances
- ✅ Easy accounting reconciliation
- ✅ No manual processes for basic operations

**MVP Scope:**
- ✅ Core dashboard with overview
- ✅ Payment list and filtering
- ✅ Payout request interface
- ✅ Basic settings
- ❌ Advanced analytics (future)
- ❌ Multi-user accounts (future)

---

### Feature 4: Automatic Currency Conversion

**Description:**
Merchants quote prices in VND. System automatically calculates crypto amount at current exchange rates.

**How It Works:**

**Step 1: Merchant Input**
```
Enter amount in VND: 2,300,000 ₫
```

**Step 2: System Calculation**
```
Fetching exchange rate...
1 USDT = 23,000 VND

Crypto amount: 100.00 USDT
(Rate valid for 30 minutes)
```

**Step 3: QR Generation**
```
QR code generated with:
- Amount: 100.00 USDT
- Network: Solana
- Wallet: [hot wallet address]
- Memo: [payment_id]
```

**Key Capabilities:**
- **Real-Time Rates**: Updated every 60 seconds from CoinGecko/Binance
- **Rate Lock**: Exchange rate fixed at payment creation
- **Volatility Buffer**: 1.5% margin to protect against rate movement
- **Multi-Token Support**: USDT, USDC on multiple chains
- **Rate Transparency**: Show merchant the rate being used

**Benefits:**
- ✅ Merchant thinks in VND (familiar)
- ✅ Tourist pays in crypto (convenient)
- ✅ No rate risk for merchant (locked rate)
- ✅ Automatic conversion (no manual calculation)

**Rate Display:**
```
╔═══════════════════════════════════════════╗
║  Payment Amount                           ║
╠═══════════════════════════════════════════╣
║  VND Amount:        2,300,000 ₫          ║
║  Exchange Rate:     23,000 VND/USDT      ║
║  Crypto Amount:     100.00 USDT          ║
║                                           ║
║  ℹ️ Rate locked for 30 minutes           ║
║  Last updated: 2 minutes ago              ║
╚═══════════════════════════════════════════╝
```

**MVP Scope:**
- ✅ VND to crypto conversion
- ✅ Rate from public APIs
- ✅ 1.5% volatility buffer
- ❌ Merchant-set custom rates (future)
- ❌ Multi-currency support (future)

---

### Feature 5: VND Settlement & Payouts

**Description:**
Merchants can withdraw their accumulated VND balance to their Vietnamese bank account.

**Payout Flow:**

**Step 1: Request Payout**
```
╔═══════════════════════════════════════════╗
║  Request Payout                           ║
╠═══════════════════════════════════════════╣
║  Available Balance: 12,450,000 VND        ║
║                                           ║
║  Payout Amount:     [10,000,000] VND      ║
║  Fee (1%):          100,000 VND           ║
║  You'll Receive:    9,900,000 VND         ║
║                                           ║
║  Bank Account:      Vietcombank           ║
║                     *********7890         ║
║                                           ║
║  [Confirm Payout]  [Cancel]               ║
╚═══════════════════════════════════════════╝
```

**Step 2: Admin Approval** (MVP - Manual)
- Ops team reviews request
- Checks for fraud indicators
- Approves or rejects with reason

**Step 3: Bank Transfer** (MVP - Manual)
- Ops team executes bank transfer
- Typically T+0 or T+1 settlement
- Upload bank receipt for records

**Step 4: Confirmation**
```
✅ Payout Completed

Amount:        9,900,000 VND
Bank Account:  Vietcombank ***7890
Reference:     PO-20251118-001
Completed:     2025-11-18 14:30 ICT

[Download Receipt] [View Transaction]
```

**Key Capabilities:**
- **Flexible Amounts**: Withdraw any amount above minimum (500,000 VND)
- **Clear Fees**: 1% payout fee, shown upfront
- **Multiple Bank Support**: All major Vietnamese banks
- **Status Tracking**: Real-time status updates
- **Email Notifications**: Confirmation at each stage

**Benefits:**
- ✅ Merchants receive familiar VND currency
- ✅ Direct to business bank account
- ✅ No crypto wallet needed
- ✅ Clear audit trail for accounting

**Payout Limits (MVP):**
- Minimum: 500,000 VND
- Maximum: 50,000,000 VND per payout
- Frequency: 1 payout per day (initial merchants)
- Processing Time: 1-2 business days

**MVP Scope:**
- ✅ Manual approval process
- ✅ Manual bank transfer execution
- ✅ Basic fraud checks
- ❌ Automatic approval (future)
- ❌ Instant settlement (future)
- ❌ API-based bank transfers (future)

---

### Feature 6: Webhook Notifications

**Description:**
Real-time webhooks notify merchant systems when payment events occur, enabling automated workflows.

**Supported Events:**

**1. payment.created**
```json
{
  "event": "payment.created",
  "payment_id": "uuid",
  "amount_vnd": 2300000,
  "amount_crypto": 100.00,
  "crypto_currency": "USDT",
  "status": "created",
  "expires_at": "2025-11-18T11:00:00Z",
  "timestamp": "2025-11-18T10:30:00Z"
}
```

**2. payment.completed**
```json
{
  "event": "payment.completed",
  "payment_id": "uuid",
  "amount_vnd": 2300000,
  "tx_hash": "abc123...",
  "blockchain": "solana",
  "confirmed_at": "2025-11-18T10:32:45Z",
  "timestamp": "2025-11-18T10:32:46Z"
}
```

**3. payment.failed**
```json
{
  "event": "payment.failed",
  "payment_id": "uuid",
  "reason": "expired",
  "timestamp": "2025-11-18T11:00:01Z"
}
```

**4. payout.completed**
```json
{
  "event": "payout.completed",
  "payout_id": "uuid",
  "amount_vnd": 10000000,
  "bank_reference": "BT20251118001",
  "completed_at": "2025-11-18T14:30:00Z"
}
```

**Key Capabilities:**
- **HMAC Signature**: Verify webhook authenticity
- **Automatic Retry**: Exponential backoff (30s, 1m, 5m, 15m, 1h)
- **Delivery Logs**: Track all webhook attempts
- **Webhook Testing**: Test endpoint from dashboard
- **Event Filtering**: Subscribe to specific events only

**Benefits:**
- ✅ Automate order fulfillment
- ✅ Real-time booking confirmations
- ✅ Integrate with existing systems
- ✅ No polling required

**Use Cases:**

**Hotel Booking System:**
```
payment.completed webhook received
  → Confirm room reservation
  → Send confirmation email to guest
  → Update inventory system
  → Generate digital room key
```

**Tour Booking Platform:**
```
payment.completed webhook received
  → Mark tour slot as booked
  → Send tour details and meeting point
  → Notify tour guide
  → Add to day's manifest
```

**MVP Scope:**
- ✅ Core payment events
- ✅ HMAC verification
- ✅ Automatic retry with backoff
- ❌ Webhook logs in dashboard (future)
- ❌ Custom webhook headers (future)

---

### Feature 7: Multi-Chain Support

**Description:**
Accept payments on multiple blockchain networks, giving tourists flexibility in how they pay.

**Supported Chains (MVP):**

#### Solana
**Advantages:**
- ⚡ Ultra-fast: ~400ms confirmation
- 💰 Low fees: ~$0.001 per transaction
- ✅ Fast finality: ~13 seconds
- 🌐 Growing adoption in Asia

**Tokens:**
- USDT (Tether USD)
- USDC (USD Coin)

**Best For:**
- Quick payments (coffee, taxi, small purchases)
- Cost-sensitive tourists
- High-volume merchants

#### BNB Smart Chain (BSC)
**Advantages:**
- 🏦 Mature ecosystem
- 🌏 Popular in Asia-Pacific
- 💳 Wide token support
- 📱 Many wallet integrations

**Tokens:**
- USDT (BEP20)
- BUSD (Binance USD)

**Best For:**
- Binance wallet users (very common in Asia)
- Larger transactions (hotel bookings)
- Tourists familiar with BSC ecosystem

**Chain Selection:**

**Automatic (Default):**
- System recommends Solana for speed and cost
- Tourist can switch chain in wallet if needed

**Manual (Dashboard Setting):**
- Merchant can prefer specific chain
- E.g., BSC-only for Binance-heavy clientele

**Chain Comparison:**

| Feature | Solana | BSC |
|---------|--------|-----|
| Confirmation Time | 13 seconds | 45 seconds |
| Transaction Fee | $0.001 | $0.10-0.30 |
| Finality | Native | 15 blocks |
| Wallet Support | Growing | Extensive |
| Asia Adoption | Medium | High |

**Benefits:**
- ✅ Tourist flexibility (use preferred wallet)
- ✅ Geographic optimization (BSC popular in Asia)
- ✅ Risk diversification (multi-chain)
- ✅ Network redundancy (fallback option)

**MVP Scope:**
- ✅ Solana (USDT, USDC)
- ✅ BSC (USDT, BUSD)
- ❌ Ethereum (Phase 2)
- ❌ Polygon (Phase 2)
- ❌ Automatic chain selection based on fees (future)

---

### Feature 8: Security & Compliance

**Description:**
Enterprise-grade security and compliance features to protect merchants and users.

**Security Features:**

#### 8.1 Merchant Authentication
- **API Keys**: Secure authentication for API access
- **Key Rotation**: Regenerate keys from dashboard
- **Rate Limiting**: 100 requests/minute per merchant
- **IP Whitelisting**: Restrict API access by IP (future)

#### 8.2 Payment Security
- **Amount Validation**: Exact amount match required
- **Memo Verification**: Payment ID must match exactly
- **Expiry Enforcement**: 30-minute payment window
- **Duplicate Prevention**: Block duplicate transaction processing
- **Blockchain Finality**: Wait for full confirmation before crediting

#### 8.3 Payout Security
- **Two-Factor Review**: Admin approval required (MVP)
- **Bank Account Verification**: Match KYC documents
- **Velocity Limits**: Daily/monthly payout caps
- **Fraud Detection**: Pattern analysis for suspicious activity
- **Balance Locks**: Prevent double-spending

#### 8.4 Data Security
- **Encryption at Rest**: Database encryption for sensitive data
- **TLS 1.3**: All connections encrypted in transit
- **PII Redaction**: Personal info redacted from logs
- **Secure Key Storage**: Environment variables (→ Vault in future)
- **Audit Logs**: Immutable audit trail for all actions

**Compliance Features:**

#### 8.5 KYC (Know Your Customer)
**Required Documents:**
- Business registration certificate
- Tax identification number
- Owner ID card/passport
- Bank account verification
- Business license (for regulated industries)

**Process:**
- Document upload via encrypted channel
- Manual review by compliance team
- Verification against government registries
- Sanctions list screening
- Approval/rejection within 24 hours

#### 8.6 Transaction Monitoring
- **AML Screening**: Flag suspicious patterns
- **Velocity Checks**: Unusual volume triggers review
- **Source of Funds**: Monitor crypto source addresses
- **Reporting**: Generate reports for regulators

#### 8.7 Record Keeping
- **7-Year Retention**: All transaction records
- **Audit Trail**: Complete history of all actions
- **Export Capability**: CSV/PDF for audits
- **Compliance Reports**: Monthly summaries

**Regulatory Compliance:**
- ✅ Vietnam data localization laws
- ✅ Da Nang regulatory sandbox requirements
- ✅ AML/CFT basic compliance
- ❌ Full financial institution compliance (not required for MVP)

**Benefits:**
- ✅ Merchant trust and confidence
- ✅ Regulatory approval maintained
- ✅ Reduced fraud losses
- ✅ Clear audit trail for disputes

---

### Feature 9: API Integration

**Description:**
RESTful API for developers to integrate payment gateway into existing systems.

**Core Endpoints:**

#### Create Payment
```http
POST /api/v1/payments
Authorization: Bearer {api_key}
Content-Type: application/json

{
  "amount_vnd": 2300000,
  "description": "Hotel booking #12345",
  "customer_email": "tourist@example.com",
  "callback_url": "https://yourdomain.com/webhook",
  "metadata": {
    "booking_id": "12345",
    "room_type": "deluxe"
  }
}

Response 201:
{
  "data": {
    "payment_id": "uuid",
    "amount_vnd": 2300000,
    "amount_crypto": 100.00,
    "crypto_currency": "USDT",
    "blockchain": "solana",
    "qr_code_url": "https://cdn.../qr.png",
    "payment_url": "solana:...",
    "status": "created",
    "expires_at": "2025-11-18T11:00:00Z"
  }
}
```

#### Check Payment Status
```http
GET /api/v1/payments/{payment_id}
Authorization: Bearer {api_key}

Response 200:
{
  "data": {
    "payment_id": "uuid",
    "status": "completed",
    "amount_vnd": 2300000,
    "tx_hash": "abc123...",
    "confirmed_at": "2025-11-18T10:32:45Z"
  }
}
```

#### List Payments
```http
GET /api/v1/payments?status=completed&limit=50&offset=0
Authorization: Bearer {api_key}

Response 200:
{
  "data": {
    "payments": [...],
    "total": 156,
    "limit": 50,
    "offset": 0
  }
}
```

#### Request Payout
```http
POST /api/v1/payouts
Authorization: Bearer {api_key}

{
  "amount_vnd": 10000000,
  "bank_account_id": "uuid"
}

Response 201:
{
  "data": {
    "payout_id": "uuid",
    "amount_vnd": 10000000,
    "fee_vnd": 100000,
    "status": "requested"
  }
}
```

#### Get Balance
```http
GET /api/v1/balance
Authorization: Bearer {api_key}

Response 200:
{
  "data": {
    "available_vnd": 12450000,
    "pending_vnd": 0,
    "total_received_vnd": 45680000,
    "total_paid_out_vnd": 33230000
  }
}
```

**API Features:**
- **RESTful Design**: Standard HTTP methods and status codes
- **JSON Format**: Request and response bodies in JSON
- **Pagination**: Limit/offset for list endpoints
- **Filtering**: Query parameters for filtering results
- **Error Handling**: Consistent error response format
- **Rate Limiting**: Clear limits and headers
- **Versioning**: `/v1/` in URL for future compatibility

**Developer Resources:**
- 📚 API Documentation (OpenAPI/Swagger)
- 💻 Code Examples (Python, Node.js, PHP, Go)
- 🧪 Postman Collection
- 🔑 Sandbox Environment (Testnet)
- 📞 Developer Support

**Benefits:**
- ✅ Easy integration with existing systems
- ✅ Automated payment processing
- ✅ Programmatic payout management
- ✅ Build custom UIs and workflows

**MVP Scope:**
- ✅ Core CRUD operations
- ✅ Basic filtering and pagination
- ✅ API key authentication
- ❌ OAuth2 (future)
- ❌ GraphQL endpoint (future)
- ❌ Websocket for real-time updates (future)

---

### Feature 10: Reporting & Analytics

**Description:**
Insights and reports to help merchants understand their business performance.

**Dashboard Metrics:**

#### Overview Cards
```
┌─────────────────────────────┬─────────────────────────────┐
│  Today's Revenue            │  This Month                 │
│  5,750,000 VND              │  87,340,000 VND             │
│  ↑ +15% vs yesterday        │  ↑ +23% vs last month       │
└─────────────────────────────┴─────────────────────────────┘

┌─────────────────────────────┬─────────────────────────────┐
│  Successful Payments        │  Average Transaction        │
│  234 transactions           │  1,873,500 VND              │
│  98.3% success rate         │  ↑ +8% vs last month        │
└─────────────────────────────┴─────────────────────────────┘
```

#### Revenue Chart
```
Revenue Trend (Last 30 Days)

4M ┤                                    ╭─╮
3M ┤                          ╭─╮  ╭─╮│ │
2M ┤              ╭─╮    ╭─╮ │ ╰──╯ ╰╯ │
1M ┤     ╭─╮  ╭─╮│ ╰────╯ ╰─╯          │
0  ┼─────┴─┴──┴─┘                      │
   Nov 1      Nov 15      Nov 30
```

#### Payment Breakdown
```
By Blockchain:
  Solana:  85%  ████████████████░░░
  BSC:     15%  ███░░░░░░░░░░░░░░░

By Token:
  USDT:    78%  ███████████████░░░░
  USDC:    22%  ████░░░░░░░░░░░░░░

By Amount:
  < 1M VND:     45%  █████████░░░
  1M-5M VND:    40%  ████████░░░░
  > 5M VND:     15%  ███░░░░░░░░░
```

**Report Types:**

#### Transaction Report
- Date range selector
- Filters: Status, blockchain, amount range
- Columns: Date, amount (VND/crypto), customer, status, tx hash
- Export: CSV, PDF

#### Settlement Report
- Monthly VND received
- Fees paid
- Payout history
- Net revenue
- Export: CSV for accounting software

#### Tax Report
- Total revenue by month/quarter
- Fee breakdown
- VAT calculations (if applicable)
- Export: PDF for tax filing

**Analytics (Future):**
- Customer analytics (repeat customers, geographic distribution)
- Peak hours/days analysis
- Conversion funnel (QR shown → payment completed)
- Blockchain performance comparison

**Benefits:**
- ✅ Business insights for decision-making
- ✅ Easy accounting reconciliation
- ✅ Tax filing support
- ✅ Performance monitoring

**MVP Scope:**
- ✅ Basic dashboard metrics
- ✅ Transaction list with filters
- ✅ CSV export
- ❌ Advanced analytics (future)
- ❌ Custom report builder (future)
- ❌ Scheduled reports (future)

---

## User Journeys

### Journey 1: First-Time Hotel Booking Payment

**Actor:** International tourist Alex staying at Sunrise Hotel

**Steps:**

1. **Check-in at Hotel**
   - Alex arrives at hotel
   - Receptionist calculates total: 4,600,000 VND for 2 nights

2. **Payment Option Presented**
   - Receptionist: "We accept cash, card, or crypto"
   - Alex: "I can pay with USDT?"
   - Receptionist: "Yes! Let me generate a QR code"

3. **QR Code Generation**
   - Receptionist opens dashboard
   - Enters: 4,600,000 VND
   - Description: "Room 305 - 2 nights"
   - Clicks "Create Payment"
   - QR code appears on screen

4. **Payment by Tourist**
   - Receptionist shows QR code
   - Alex opens Phantom wallet (Solana)
   - Scans QR code
   - Wallet shows: "Pay 200 USDT to Sunrise Hotel"
   - Alex confirms payment
   - Transaction broadcast to Solana

5. **Confirmation**
   - Dashboard shows: "Payment pending..."
   - 10 seconds later: "✅ Payment confirmed!"
   - Receipt prints automatically
   - Alex receives email receipt
   - Receptionist hands room key

6. **Post-Payment**
   - Payment appears in hotel's dashboard
   - Balance increases: +4,600,000 VND
   - Transaction recorded for accounting
   - Alex checks out 2 days later - smooth experience

**Time:** 2 minutes from QR generation to confirmation

**Satisfaction Factors:**
- ✅ Fast (no waiting for card machine)
- ✅ Clear (amount shown in both VND and USDT)
- ✅ Secure (blockchain confirmation)
- ✅ Convenient (no cash needed)

---

### Journey 2: Tour Booking via Website

**Actor:** Tourist planning trip online

**Steps:**

1. **Browse Tours**
   - Alex visits DaNangTours.com
   - Finds "Marble Mountains & Beach Tour"
   - Price: 1,500,000 VND per person
   - Adds 2 tickets to cart

2. **Checkout**
   - Proceeds to checkout
   - Total: 3,000,000 VND
   - Payment options: Credit Card / Crypto
   - Selects: "Pay with Crypto"

3. **Payment Page**
   - Redirected to payment page
   - Shows QR code + payment details
   - Amount: 130.43 USDT
   - Timer: "29:45 remaining"

4. **Payment from Desktop**
   - Alex copies wallet address
   - Opens MetaMask (BSC)
   - Manually enters details:
     - To: [wallet address]
     - Amount: 130.43 USDT (BEP20)
     - Memo: TOUR-123456
   - Confirms transaction

5. **Confirmation & Booking**
   - Page updates: "Payment received! Confirming..."
   - 30 seconds later: "Booking confirmed!"
   - Webhook triggers tour booking system
   - Booking confirmation email sent
   - Tour details, meeting point, guide contact
   - Add to calendar option

6. **Day of Tour**
   - Alex shows booking confirmation
   - Guide checks manifest
   - Enjoys tour

**Time:** 3 minutes for payment + 30 seconds for blockchain confirmation

**Satisfaction Factors:**
- ✅ Can book from abroad before arrival
- ✅ No credit card foreign transaction fees
- ✅ Instant confirmation
- ✅ Digital receipt for records

---

### Journey 3: Merchant Payout Request

**Actor:** Hotel Manager Minh checking weekly earnings

**Steps:**

1. **Review Balance**
   - Minh logs into dashboard
   - Sees: Available balance 45,000,000 VND
   - Has received 23 payments this week
   - All payments completed successfully

2. **Initiate Payout**
   - Clicks "Request Payout"
   - Enters amount: 40,000,000 VND
   - System shows:
     - Amount: 40,000,000 VND
     - Fee (1%): 400,000 VND
     - You'll receive: 39,600,000 VND
   - Confirms bank account: Vietcombank ***7890
   - Clicks "Submit Request"

3. **Confirmation**
   - Success message: "Payout requested"
   - Status: "Pending approval"
   - Estimated: 1-2 business days
   - Email confirmation received
   - Balance updates:
     - Available: 5,000,000 VND
     - Pending: 40,000,000 VND

4. **Admin Review** (Backend)
   - Ops team reviews request
   - Checks merchant history (all good)
   - Verifies bank account matches KYC
   - Approves payout
   - System sends email: "Payout approved"

5. **Bank Transfer**
   - Ops team executes bank transfer
   - Same day (T+0) via internet banking
   - Reference: PAYOUT-20251118-001

6. **Completion**
   - 2 hours later: Bank transfer received
   - Ops marks payout complete
   - Email: "Payout completed"
   - Minh checks bank: +39,600,000 VND
   - Downloads receipt for accounting

**Time:** Request to bank account: 3-6 hours (same day payout)

**Satisfaction Factors:**
- ✅ Simple process
- ✅ Clear fees upfront
- ✅ Fast settlement (same day)
- ✅ Reliable VND in bank account

---

### Journey 4: Merchant Onboarding

**Actor:** New restaurant owner Thanh wants to accept crypto

**Steps:**

1. **Discovery**
   - Thanh hears about gateway from hotel colleague
   - Visits website
   - Reads about benefits
   - Clicks "Sign Up"

2. **Registration**
   - Fills form:
     - Business name: "Pho 24 Da Nang"
     - Email: thanh@pho24dn.vn
     - Phone: +84 901 234 567
     - Business type: Restaurant
   - Receives verification email
   - Clicks link to verify

3. **KYC Submission**
   - Logs into dashboard
   - Prompted to complete KYC
   - Uploads documents:
     - Business registration certificate
     - Tax ID
     - Owner ID card
     - Bank account statement
   - Submits for review

4. **Review Period**
   - Status: "Under review"
   - Receives email: "Documents received"
   - 12 hours later: Admin reviews
   - All documents check out
   - Status updated: "Approved"

5. **Activation Email**
   - Email: "Welcome! Your account is active"
   - Contains:
     - API key
     - Dashboard link
     - Quick start guide
     - Support contact

6. **First Payment Setup**
   - Thanh logs in
   - Reviews dashboard tutorial
   - Creates test payment: 50,000 VND
   - Scans with own wallet to test
   - Payment completes successfully
   - Prints QR code stand for counter
   - Ready to accept payments!

7. **First Real Payment**
   - Tourist pays for lunch: 350,000 VND
   - QR scanned, payment received
   - Thanh sees balance update in real-time
   - Very satisfied!

**Time:** Registration to first payment: 24 hours (mostly waiting for KYC)

**Satisfaction Factors:**
- ✅ Easy setup process
- ✅ Clear instructions
- ✅ Fast approval
- ✅ Test payment capability
- ✅ Support available if needed

---

## Feature Matrix (MVP vs Future)

### MVP Features (Launch - Phase 1)

| Feature | Status | Priority | Notes |
|---------|--------|----------|-------|
| QR Code Payment | ✅ MVP | P0 | Core value prop |
| Solana Support | ✅ MVP | P0 | Primary chain |
| BSC Support | ✅ MVP | P0 | Secondary chain |
| Real-time Tracking | ✅ MVP | P0 | Status updates |
| Merchant Dashboard | ✅ MVP | P0 | Basic version |
| VND Payouts | ✅ MVP | P0 | Manual approval |
| KYC | ✅ MVP | P0 | Manual review |
| Webhooks | ✅ MVP | P1 | Core integrations |
| API | ✅ MVP | P1 | Basic endpoints |
| Email Notifications | ✅ MVP | P1 | Confirmations |
| Exchange Rate API | ✅ MVP | P0 | Real-time rates |
| Transaction Reports | ✅ MVP | P2 | CSV export |

### Phase 2 Features (Month 2-3)

| Feature | Status | Priority | Notes |
|---------|--------|----------|-------|
| Automatic Payout Approval | 🔜 Phase 2 | P1 | Risk-based automation |
| Ethereum Support | 🔜 Phase 2 | P1 | Third chain |
| SMS Notifications | 🔜 Phase 2 | P2 | Alternative to email |
| Advanced Analytics | 🔜 Phase 2 | P2 | Charts, trends |
| Multi-user Accounts | 🔜 Phase 2 | P2 | Team access |
| API Webhooks Log | 🔜 Phase 2 | P2 | Debugging |
| Refund Capability | 🔜 Phase 2 | P1 | Dispute handling |
| Partial Payments | 🔜 Phase 2 | P3 | Installments |

### Phase 3 Features (Month 4-6)

| Feature | Status | Priority | Notes |
|---------|--------|----------|-------|
| Instant Payouts | 🔮 Phase 3 | P1 | API-based bank transfer |
| Payment Links | 🔮 Phase 3 | P2 | No-code payment pages |
| Subscription Payments | 🔮 Phase 3 | P2 | Recurring billing |
| Mobile App | 🔮 Phase 3 | P2 | iOS/Android |
| Multi-currency | 🔮 Phase 3 | P3 | USD, EUR support |
| Payment Terminal | 🔮 Phase 3 | P3 | Hardware POS |
| Loyalty Program | 🔮 Phase 3 | P3 | Customer rewards |

### Future Consideration

| Feature | Status | Priority | Notes |
|---------|--------|----------|-------|
| Lightning Network | 💭 Future | P3 | Bitcoin support |
| Stablecoin Savings | 💭 Future | P3 | Earn yield on balance |
| Invoice Management | 💭 Future | P3 | B2B features |
| Marketplace | 💭 Future | P3 | Multi-merchant |

---

## Competitive Advantages

### Why Choose Us?

#### 1. Vietnam-Focused
- **Local Expertise**: Built specifically for Vietnamese market
- **VND Settlement**: Direct to Vietnamese bank accounts
- **Regulatory Compliance**: Da Nang regulatory sandbox participant
- **Local Support**: Vietnamese-speaking customer support

#### 2. Tourism-Optimized
- **No App Required**: Works with any crypto wallet
- **QR Code Standard**: Universal payment method
- **Multi-language**: English, Vietnamese, Chinese (future)
- **Tourist-Friendly**: Simple UX for one-time users

#### 3. Merchant-First Design
- **Zero Crypto Knowledge**: Merchant never touches crypto
- **VND Thinking**: Quote prices in VND, receive VND
- **Fast Onboarding**: 24-hour approval process
- **Fair Pricing**: 1% transaction fee, transparent payout fees

#### 4. Technical Excellence
- **Multi-Chain**: Solana + BSC + more
- **Fast Settlement**: Solana ~13 seconds
- **Low Fees**: <$0.01 per transaction on Solana
- **Reliable**: 99%+ uptime SLA

#### 5. Business Integration
- **API-First**: Easy integration with existing systems
- **Webhooks**: Real-time notifications
- **Developer-Friendly**: Great documentation, code samples
- **Flexible**: Works with POS, booking systems, e-commerce

### Competitive Comparison

| Feature | Us | Traditional Payment Gateway | Other Crypto Solutions |
|---------|-----|----------------------------|----------------------|
| Transaction Fee | 1% | 2-4% | 1-2% |
| Settlement Time | Same day | T+2 to T+7 | Instant (crypto) |
| Settlement Currency | VND | VND | Crypto (merchant risk) |
| International Cards | ❌ (not needed) | ✅ | ❌ |
| Crypto Payments | ✅ | ❌ | ✅ |
| Vietnam Banks | ✅ | ✅ | ❌ (limited) |
| Regulatory Compliant | ✅ | ✅ | ⚠️ (uncertain) |
| Setup Time | 24 hours | 1-2 weeks | Hours |
| Monthly Fees | $0 | $20-50 | $0-30 |

---

## Success Metrics

### North Star Metric
**Total Payment Volume (TPV)**: Total VND value processed through the platform

**Target:** 1 billion VND in Month 1

### Key Performance Indicators (KPIs)

#### Product Metrics

**Merchant Acquisition:**
- New merchants onboarded per week
- Target: 5 merchants in Month 1
- KYC approval rate: >90%
- Time to first payment: <48 hours

**Payment Success Rate:**
- Completed payments / Total payment attempts
- Target: >98%
- Reasons for failure tracking

**Payment Volume:**
- Daily/weekly/monthly TPV in VND
- Average transaction value
- Payments per merchant per day

**Payout Performance:**
- Payout request to completion time
- Target: <24 hours
- Payout approval rate: >95%

#### User Experience Metrics

**Payment Speed:**
- QR generation time: <1 second
- Payment confirmation time: <20 seconds (Solana), <60 seconds (BSC)
- Dashboard load time: <2 seconds

**Reliability:**
- System uptime: >99%
- API response time: <200ms (p95)
- Webhook delivery rate: >95%

**Merchant Satisfaction:**
- Net Promoter Score (NPS): >30
- Dashboard session length
- Feature adoption rate

**Tourist Experience:**
- Payment completion rate
- Time to scan → confirm
- Repeat customer rate (same crypto address)

#### Business Metrics

**Revenue:**
- Transaction fees collected
- Payout fees collected
- Monthly recurring revenue (MRR)
- Target Month 1: 10M+ VND revenue

**Cost:**
- Blockchain transaction fees
- OTC conversion costs
- Ops team time per payout

**Unit Economics:**
- Revenue per merchant
- Cost per transaction
- Gross margin: Target >60%

#### Operational Metrics

**KYC Process:**
- Review time: <4 hours during business hours
- Approval rate: >90%
- Rejection reasons tracking

**Payout Operations:**
- Manual review time per payout
- Bank transfer execution time
- Payout error rate: <1%

**Customer Support:**
- Response time: <2 hours
- Resolution time: <24 hours
- Support tickets per merchant

---

## Feature Prioritization Framework

### How We Prioritize

**RICE Score = (Reach × Impact × Confidence) / Effort**

**Reach**: How many users affected per month?
**Impact**: Scale of 0.25 (minimal) to 3 (massive)
**Confidence**: % certainty (50%, 80%, 100%)
**Effort**: Person-months of work

### Example Scoring

| Feature | Reach | Impact | Confidence | Effort | RICE | Priority |
|---------|-------|--------|------------|--------|------|----------|
| QR Payment | 1000 | 3.0 | 100% | 2 | 1500 | P0 |
| Webhooks | 500 | 2.0 | 100% | 1 | 1000 | P1 |
| Analytics | 200 | 1.0 | 80% | 1.5 | 107 | P2 |
| Mobile App | 500 | 1.5 | 50% | 4 | 94 | P3 |

---

## Conclusion

This payment gateway provides a **complete solution** for Vietnamese tourism merchants to accept crypto payments and receive VND settlements.

**Core Value:**
- Merchants: Accept global payments, get local currency
- Tourists: Pay with crypto, support more businesses
- Platform: Enable crypto commerce in Vietnam

**MVP Focus:**
- Fast, reliable QR code payments
- Multi-chain support (Solana + BSC)
- Simple merchant dashboard
- Secure VND payouts
- Developer-friendly API

**Success Criteria:**
- 5+ merchants by end of Month 1
- 1B+ VND payment volume
- 98%+ payment success rate
- 99%+ system uptime
- Positive merchant feedback (NPS >30)

The product is designed to be **simple for users** while being **powerful for developers**, balancing ease of use with comprehensive functionality.

---

**Document Version**: 1.0
**Last Updated**: 2025-11-18
**Owner**: Product Team
**Next Review**: After MVP launch

For technical implementation details, see `APPLICATION_FLOWS.md` and `ARCHITECTURE.md`.
