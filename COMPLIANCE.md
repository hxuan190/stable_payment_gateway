# AML/CTF Compliance Framework

> **Institutional-Grade Compliance for Crypto Payment Gateway**
>
> This document outlines our **compliance-first** approach to AML (Anti-Money Laundering) and CTF (Counter-Terrorism Financing), designed to meet Vietnam's regulatory requirements and prepare for licensed OTC integration.

---

## 📋 Table of Contents

1. [Compliance Philosophy](#compliance-philosophy)
2. [KYC/KYB Framework](#kyckyb-framework)
3. [AML Transaction Monitoring](#aml-transaction-monitoring)
4. [Risk Scoring System](#risk-scoring-system)
5. [Wallet Screening](#wallet-screening)
6. [Suspicious Activity Reporting (SAR)](#suspicious-activity-reporting-sar)
7. [Data Retention & Audit](#data-retention--audit)
8. [Technology Stack](#technology-stack)
9. [Operational Procedures](#operational-procedures)
10. [Regulatory Compliance](#regulatory-compliance)

---

## 🎯 Compliance Philosophy

### Core Principles

1. **Compliance is a Feature, Not a Cost**
   - Regulatory compliance = competitive moat
   - First-mover advantage when regulations tighten
   - Trust signal to merchants and partners

2. **Zero-Tolerance for Illegal Activity**
   - No sanctioned entities
   - No mixer/tumbler usage
   - No structuring/smurfing
   - Proactive risk management

3. **Regulatory Readiness**
   - Build for **licensed OTC era** from day one
   - Seamless transition when Vietnam licenses crypto exchanges
   - Data and procedures already in place

---

## 🔐 KYC/KYB Framework

### Merchant Onboarding Tiers

#### Tier 1: Basic KYB (< 50M VND/month)

**Required Documents:**
```
Business Entity:
├── Business license (Giấy phép kinh doanh)
├── Tax identification number (Mã số thuế)
├── Business address (proof: utility bill, lease agreement)
└── Business registration certificate

Legal Representative:
├── Vietnam ID card / Passport (front + back)
├── Proof of address (< 3 months old)
├── Phone number (verified via OTP)
└── Email (verified)

Banking:
├── Bank account name (must match business name)
├── Bank account number
└── Bank name + branch
```

**Verification Process:**
1. Submit documents via dashboard
2. **Automated checks** (eKYC for ID card):
   - VNPT eKYC / FPT.AI eKYC integration
   - OCR + facial recognition for legal rep
   - Business registry check (via Vietnam National Business Registration Portal)
3. **Manual review** (compliance officer):
   - Document authenticity
   - Cross-check business name with tax ID
   - Verify bank account ownership
4. **Decision**: Approve / Reject / Request more info
5. **Timeline**: 24-48 hours

**Transaction Limits:**
- Single transaction: **≤ 20M VND**
- Daily volume: **≤ 50M VND**
- Monthly volume: **≤ 50M VND**

---

#### Tier 2: Enhanced KYB (50M - 500M VND/month)

**Additional Requirements:**
```
Beneficial Ownership:
├── List of all shareholders/owners with > 25% ownership
├── ID documents for all beneficial owners
├── Corporate structure diagram
└── Proof of source of funds

Business Operations:
├── Business model description
├── Website / e-commerce platform URL
├── Expected transaction volume (with justification)
├── Customer demographics
└── Average transaction size

Financial Information:
├── Bank statements (last 3 months)
├── Audited financials (if available)
└── Revenue projections
```

**Enhanced Verification:**
1. All Tier 1 checks +
2. **Beneficial owner screening**:
   - PEP (Politically Exposed Person) check
   - Sanctions list screening (OFAC, UN, EU)
   - Adverse media search
3. **Business verification**:
   - Website visit + screenshot
   - Social media presence check
   - Google Maps business location verification
4. **Source of funds assessment**:
   - How does business generate revenue?
   - Is crypto payment gateway appropriate for this business?
5. **Decision**: Approve / Reject / Escalate to Tier 3
6. **Timeline**: 3-5 business days

**Transaction Limits:**
- Single transaction: **≤ 100M VND**
- Daily volume: **≤ 500M VND**
- Monthly volume: **≤ 500M VND**

---

#### Tier 3: Institutional KYB (> 500M VND/month)

**Additional Requirements:**
```
Corporate Governance:
├── Board resolution authorizing crypto payment gateway usage
├── List of all directors + their IDs
├── Articles of incorporation
└── Shareholder register

Enhanced Due Diligence:
├── On-site visit (for Da Nang merchants)
├── Interview with key management
├── Internal AML/CTF policy (if large company)
└── Proof of regulatory licenses (if applicable)

Ongoing Monitoring:
├── Quarterly financial statements
├── Annual audit report
├── Notification of ownership changes
└── Notification of business model changes
```

**Institutional Verification:**
1. All Tier 1 + Tier 2 checks +
2. **On-site inspection**:
   - Physical location visit
   - Interview with owner/manager
   - Verify operations match stated business model
3. **Enhanced PEP/sanctions screening**:
   - All directors and beneficial owners
   - Ongoing monitoring (not just one-time)
4. **Relationship manager assigned**:
   - Dedicated compliance officer
   - Quarterly check-ins
5. **Decision**: Approve / Reject
6. **Timeline**: 7-14 business days

**Transaction Limits:**
- Single transaction: **Negotiable** (case-by-case)
- Daily volume: **Negotiable**
- Monthly volume: **Negotiable** (with ongoing monitoring)

---

### KYC Database Schema

```sql
CREATE TABLE merchant_kyb (
    id UUID PRIMARY KEY,
    merchant_id UUID REFERENCES merchants(id),

    -- Tier
    kyb_tier VARCHAR(20), -- 'tier_1', 'tier_2', 'tier_3'

    -- Business Info
    business_license_number VARCHAR(100),
    tax_id VARCHAR(50) UNIQUE,
    business_type VARCHAR(100),
    business_address TEXT,

    -- Legal Representative
    legal_rep_name VARCHAR(255),
    legal_rep_id_number VARCHAR(50),
    legal_rep_id_type VARCHAR(20), -- 'vietnam_id', 'passport'
    legal_rep_dob DATE,
    legal_rep_nationality VARCHAR(50),

    -- Documents (encrypted S3/MinIO URLs)
    documents JSONB,

    -- Verification
    kyb_status VARCHAR(50), -- 'pending', 'approved', 'rejected', 'under_review'
    verified_by UUID, -- Admin user ID
    verified_at TIMESTAMP,
    rejection_reason TEXT,

    -- Risk Assessment
    risk_score INTEGER, -- 0-100
    risk_level VARCHAR(20), -- 'low', 'medium', 'high', 'critical'

    -- PEP/Sanctions Screening
    pep_screening_result JSONB,
    sanctions_screening_result JSONB,

    -- Ongoing Monitoring
    last_reviewed_at TIMESTAMP,
    next_review_due DATE,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE beneficial_owners (
    id UUID PRIMARY KEY,
    merchant_id UUID REFERENCES merchants(id),

    name VARCHAR(255),
    id_number VARCHAR(50),
    nationality VARCHAR(50),
    ownership_percentage DECIMAL(5, 2), -- e.g., 35.50%

    pep_status BOOLEAN DEFAULT FALSE,
    sanctions_match BOOLEAN DEFAULT FALSE,

    documents JSONB,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_kyb_merchant ON merchant_kyb(merchant_id);
CREATE INDEX idx_kyb_status ON merchant_kyb(kyb_status);
CREATE INDEX idx_kyb_risk_level ON merchant_kyb(risk_level);
```

---

## 🔍 AML Transaction Monitoring

### Real-Time Monitoring Rules

#### Rule 1: Velocity Checks

```go
type VelocityRule struct {
    MaxTxPerHour  int     // 10
    MaxTxPerDay   int     // 50
    MaxAmountPerDay Decimal // 500M VND
}

func (r *VelocityRule) Check(merchantID string, tx Transaction) (bool, string) {
    // Get merchant transactions in last hour
    txLastHour := repo.GetTransactionsLastHour(merchantID)
    if len(txLastHour) >= r.MaxTxPerHour {
        return false, "VELOCITY_HOUR_EXCEEDED"
    }

    // Get merchant transactions in last 24 hours
    txLast24h := repo.GetTransactionsLast24Hours(merchantID)
    if len(txLast24h) >= r.MaxTxPerDay {
        return false, "VELOCITY_DAY_EXCEEDED"
    }

    // Check daily amount
    totalToday := sumTransactions(txLast24h)
    if totalToday.Add(tx.Amount).GreaterThan(r.MaxAmountPerDay) {
        return false, "DAILY_AMOUNT_EXCEEDED"
    }

    return true, ""
}
```

**Alert Thresholds:**
- **Warning**: > 5 tx/hour OR > 100M VND/hour
- **Block**: > 10 tx/hour OR > 200M VND/hour

---

#### Rule 2: Amount-Based Thresholds

```go
type AmountRule struct {
    SingleTxAlert      Decimal // 50M VND
    SingleTxBlock      Decimal // 200M VND
    DailyCumulativeAlert Decimal // 200M VND
}

func (r *AmountRule) Check(tx Transaction) (RiskLevel, string) {
    if tx.Amount.GreaterThan(r.SingleTxBlock) {
        return RISK_CRITICAL, "SINGLE_TX_EXCEEDS_LIMIT"
    }

    if tx.Amount.GreaterThan(r.SingleTxAlert) {
        return RISK_HIGH, "SINGLE_TX_ALERT_THRESHOLD"
    }

    // Check daily cumulative
    dailyTotal := repo.GetDailyTotal(tx.MerchantID)
    if dailyTotal.Add(tx.Amount).GreaterThan(r.DailyCumulativeAlert) {
        return RISK_HIGH, "DAILY_CUMULATIVE_ALERT"
    }

    return RISK_LOW, ""
}
```

---

#### Rule 3: Behavioral Analysis

```go
type BehaviorRule struct{}

func (r *BehaviorRule) Check(tx Transaction) []AMLFlag {
    flags := []AMLFlag{}

    // Unusual time (2 AM - 5 AM)
    if tx.CreatedAt.Hour() >= 2 && tx.CreatedAt.Hour() < 5 {
        flags = append(flags, AMLFlag{
            Type: "UNUSUAL_TIME",
            Severity: "MEDIUM",
            Description: "Transaction during unusual hours (2-5 AM)",
        })
    }

    // Rapid onboarding (KYC approved < 24 hours ago)
    merchant := repo.GetMerchant(tx.MerchantID)
    if time.Since(merchant.KYCApprovedAt) < 24*time.Hour {
        if tx.Amount.GreaterThan(Decimal.NewFromInt(50000000)) { // > 50M VND
            flags = append(flags, AMLFlag{
                Type: "RAPID_ONBOARDING_HIGH_VALUE",
                Severity: "HIGH",
                Description: "High-value transaction within 24h of KYC approval",
            })
        }
    }

    // Structuring detection (multiple transactions just below threshold)
    recentTxs := repo.GetRecentTransactions(tx.MerchantID, 24*time.Hour)
    if detectStructuring(recentTxs, Decimal.NewFromInt(50000000)) {
        flags = append(flags, AMLFlag{
            Type: "STRUCTURING_PATTERN",
            Severity: "CRITICAL",
            Description: "Multiple transactions just below reporting threshold",
        })
    }

    // Round amounts (exactly 10M, 20M, 50M, 100M)
    if tx.Amount.Mod(Decimal.NewFromInt(10000000)).Equal(Decimal.Zero) {
        if tx.Amount.GreaterThan(Decimal.NewFromInt(10000000)) {
            flags = append(flags, AMLFlag{
                Type: "ROUND_AMOUNT",
                Severity: "LOW",
                Description: "Transaction amount is suspiciously round",
            })
        }
    }

    return flags
}

func detectStructuring(txs []Transaction, threshold Decimal) bool {
    count := 0
    margin := Decimal.NewFromInt(1000000) // 1M VND margin

    for _, tx := range txs {
        diff := threshold.Sub(tx.Amount)
        if diff.GreaterThan(Decimal.Zero) && diff.LessThan(margin) {
            count++
        }
    }

    return count >= 3 // 3+ transactions just below threshold = structuring
}
```

---

#### Rule 4: Wallet Source Analysis

```go
type WalletSourceRule struct {
    chainanalysis *ChainanalysisClient
}

func (r *WalletSourceRule) Check(tx BlockchainTransaction) (RiskLevel, []AMLFlag) {
    flags := []AMLFlag{}

    // Check source wallet via Chainalysis
    walletRisk, err := r.chainanalysis.GetWalletRisk(tx.FromAddress, tx.Chain)
    if err != nil {
        log.Error("Chainalysis API error", err)
        return RISK_MEDIUM, flags
    }

    // Mixer/Tumbler interaction
    if walletRisk.MixerExposure > 0 {
        flags = append(flags, AMLFlag{
            Type: "MIXER_INTERACTION",
            Severity: "CRITICAL",
            Description: fmt.Sprintf("Source wallet has %f%% mixer exposure",
                walletRisk.MixerExposure),
        })
    }

    // Sanctioned entity
    if len(walletRisk.SanctionsMatches) > 0 {
        flags = append(flags, AMLFlag{
            Type: "SANCTIONS_MATCH",
            Severity: "CRITICAL",
            Description: "Source wallet linked to sanctioned entity",
            Details: walletRisk.SanctionsMatches,
        })
    }

    // Wallet age
    if walletRisk.WalletAge < 7 { // < 7 days old
        if tx.Amount.GreaterThan(Decimal.NewFromInt(100000000)) { // > 100M VND
            flags = append(flags, AMLFlag{
                Type: "NEW_WALLET_HIGH_VALUE",
                Severity: "HIGH",
                Description: "High-value transaction from newly created wallet",
            })
        }
    }

    // Risk score
    if walletRisk.RiskScore > 75 {
        return RISK_CRITICAL, flags
    } else if walletRisk.RiskScore > 50 {
        return RISK_HIGH, flags
    } else if walletRisk.RiskScore > 25 {
        return RISK_MEDIUM, flags
    }

    return RISK_LOW, flags
}
```

---

### Transaction Monitoring Database Schema

```sql
CREATE TABLE transaction_monitoring (
    id UUID PRIMARY KEY,
    tx_id UUID REFERENCES payments(id),
    merchant_id UUID REFERENCES merchants(id),

    -- Transaction Details
    amount_vnd DECIMAL(15, 2),
    amount_crypto DECIMAL(20, 8),
    chain VARCHAR(50),
    wallet_address VARCHAR(255),

    -- Risk Assessment
    risk_score INTEGER, -- 0-100
    risk_level VARCHAR(20), -- 'low', 'medium', 'high', 'critical'

    -- AML Flags
    aml_flags JSONB[], -- Array of flag objects

    -- Chainalysis/TRM Results
    wallet_risk_data JSONB,
    sanctions_screening_result JSONB,

    -- Review Status
    status VARCHAR(50), -- 'auto_approved', 'under_review', 'flagged', 'blocked', 'approved'
    reviewed_by UUID,
    reviewed_at TIMESTAMP,
    review_notes TEXT,

    -- Actions Taken
    action_taken VARCHAR(100), -- 'approved', 'blocked', 'sar_filed', 'funds_frozen'

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE aml_alerts (
    id UUID PRIMARY KEY,
    transaction_monitoring_id UUID REFERENCES transaction_monitoring(id),

    alert_type VARCHAR(100), -- 'VELOCITY_EXCEEDED', 'MIXER_INTERACTION', etc.
    severity VARCHAR(20), -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    description TEXT,
    details JSONB,

    status VARCHAR(50), -- 'open', 'investigating', 'resolved', 'escalated'
    assigned_to UUID,
    resolved_by UUID,
    resolution TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP
);

CREATE INDEX idx_monitoring_merchant ON transaction_monitoring(merchant_id);
CREATE INDEX idx_monitoring_risk_level ON transaction_monitoring(risk_level);
CREATE INDEX idx_monitoring_status ON transaction_monitoring(status);
CREATE INDEX idx_alerts_severity ON aml_alerts(severity);
CREATE INDEX idx_alerts_status ON aml_alerts(status);
```

---

## 📊 Risk Scoring System

### Composite Risk Score (0-100)

```go
type RiskScore struct {
    MerchantRisk      int // 0-25
    TransactionRisk   int // 0-25
    WalletRisk        int // 0-25
    BehavioralRisk    int // 0-25
}

func CalculateRiskScore(tx Transaction, merchant Merchant, walletData WalletRiskData) int {
    score := 0

    // 1. Merchant Risk (0-25)
    score += calculateMerchantRisk(merchant)

    // 2. Transaction Risk (0-25)
    score += calculateTransactionRisk(tx)

    // 3. Wallet Risk (0-25)
    score += calculateWalletRisk(walletData)

    // 4. Behavioral Risk (0-25)
    score += calculateBehavioralRisk(tx, merchant)

    return min(score, 100)
}

func calculateMerchantRisk(merchant Merchant) int {
    risk := 0

    // KYB tier (higher tier = lower risk)
    switch merchant.KYBTier {
    case "tier_3":
        risk += 0 // Institutional, lowest risk
    case "tier_2":
        risk += 5
    case "tier_1":
        risk += 10
    default:
        risk += 20 // No KYB = highest risk
    }

    // Account age (newer = higher risk)
    accountAge := time.Since(merchant.CreatedAt)
    if accountAge < 7*24*time.Hour {
        risk += 10
    } else if accountAge < 30*24*time.Hour {
        risk += 5
    }

    // PEP status
    if merchant.KYB.PEPStatus {
        risk += 10
    }

    // Historical issues
    if merchant.Stats.SARFiledCount > 0 {
        risk += 15
    }

    return min(risk, 25)
}

func calculateTransactionRisk(tx Transaction) int {
    risk := 0

    // Amount-based risk
    amountVND := tx.AmountVND.IntPart()
    if amountVND > 200000000 { // > 200M VND
        risk += 15
    } else if amountVND > 100000000 { // > 100M VND
        risk += 10
    } else if amountVND > 50000000 { // > 50M VND
        risk += 5
    }

    // Time-based risk
    hour := tx.CreatedAt.Hour()
    if hour >= 2 && hour < 5 {
        risk += 5
    }

    return min(risk, 25)
}

func calculateWalletRisk(walletData WalletRiskData) int {
    risk := 0

    // Direct Chainalysis risk score mapping
    if walletData.RiskScore > 75 {
        risk += 25
    } else if walletData.RiskScore > 50 {
        risk += 15
    } else if walletData.RiskScore > 25 {
        risk += 10
    } else {
        risk += 5
    }

    // Mixer exposure
    if walletData.MixerExposure > 0 {
        risk = 25 // Auto-max
    }

    // Sanctions
    if len(walletData.SanctionsMatches) > 0 {
        risk = 25 // Auto-max
    }

    return min(risk, 25)
}

func calculateBehavioralRisk(tx Transaction, merchant Merchant) int {
    risk := 0

    // Velocity
    recentTxCount := repo.CountTransactionsLast24Hours(merchant.ID)
    if recentTxCount > 20 {
        risk += 10
    } else if recentTxCount > 10 {
        risk += 5
    }

    // Structuring pattern
    if detectStructuring(repo.GetRecentTransactions(merchant.ID, 24*time.Hour), Decimal.NewFromInt(50000000)) {
        risk += 15
    }

    return min(risk, 25)
}
```

### Risk-Based Actions

| Risk Score | Risk Level | Action |
|------------|-----------|--------|
| 0-30 | **Low** | ✅ Auto-approve, routine monitoring |
| 31-60 | **Medium** | ⚠️ Auto-approve, flag for review within 24h |
| 61-80 | **High** | 🔍 Hold for manual review (< 2 hours) |
| 81-100 | **Critical** | 🚫 Block transaction, immediate escalation, possible SAR |

---

## 🚨 Suspicious Activity Reporting (SAR)

### SAR Triggers

Automatic SAR consideration when:
1. ✅ Risk score ≥ 81
2. ✅ Sanctioned wallet interaction
3. ✅ Mixer/tumbler usage detected
4. ✅ Structuring pattern (3+ transactions just below threshold)
5. ✅ PEP with insufficient source of funds documentation
6. ✅ Rapid high-value transactions from new merchant
7. ✅ Geographic mismatch (e.g., Vietnam merchant, North Korea source wallet)

### SAR Workflow

```
┌─────────────────────────────────────────────────────┐
│ 1. TRIGGER DETECTION                                 │
│    - Automated rule breach OR                        │
│    - Manual flag by compliance officer               │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│ 2. PRELIMINARY INVESTIGATION (24 hours)              │
│    - Gather all transaction data                    │
│    - Review merchant KYB                             │
│    - Check transaction history                       │
│    - Analyze wallet source chain                     │
│    - Document findings                               │
└──────────────────────┬──────────────────────────────┘
                       │
                   ┌───┴───┐
                   │       │
           Confirmed     False
          Suspicious     Positive
                   │       │
┌──────────────────▼───────────────────────────────────┐
│ 3. SAR PREPARATION (within 48 hours of confirmation) │
│    - Complete SAR form (Vietnam format)              │
│    - Attach supporting evidence                      │
│    - Compliance officer review                       │
│    - Legal review (if complex)                       │
└──────────────────────┬─────────────────────────────┘
                       │
┌──────────────────────▼─────────────────────────────┐
│ 4. SAR FILING                                       │
│    - Submit to State Bank of Vietnam (SBV)         │
│    - If terrorism-related: also to Police          │
│    - Record submission in database                  │
└──────────────────────┬─────────────────────────────┘
                       │
┌──────────────────────▼─────────────────────────────┐
│ 5. ACTIONS                                          │
│    - Freeze merchant account (if required)         │
│    - Block further transactions                     │
│    - Cooperate with authorities                     │
│    - Do NOT tip off the merchant (illegal!)        │
└─────────────────────────────────────────────────────┘
```

### SAR Database Schema

```sql
CREATE TABLE sar_reports (
    id UUID PRIMARY KEY,

    -- Related Entities
    merchant_id UUID REFERENCES merchants(id),
    related_transactions UUID[], -- Array of payment IDs

    -- SAR Details
    sar_type VARCHAR(100), -- 'STRUCTURING', 'SANCTIONS', 'MIXER', 'PEP', 'OTHER'
    description TEXT,
    total_amount_involved DECIMAL(20, 2),
    time_period_start DATE,
    time_period_end DATE,

    -- Investigation
    investigated_by UUID,
    investigation_summary TEXT,
    evidence JSONB, -- Chainalysis reports, screenshots, etc.

    -- Filing
    sar_number VARCHAR(100), -- Official SAR number from SBV
    filed_to VARCHAR(100), -- 'SBV', 'Police', 'Both'
    filed_by UUID,
    filed_at TIMESTAMP,
    filing_document_url TEXT, -- Encrypted

    -- Status
    status VARCHAR(50), -- 'investigating', 'prepared', 'filed', 'resolved'

    -- Authority Response
    authority_response TEXT,
    authority_action_taken TEXT,
    case_closed_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sar_merchant ON sar_reports(merchant_id);
CREATE INDEX idx_sar_status ON sar_reports(status);
CREATE INDEX idx_sar_filed_at ON sar_reports(filed_at);
```

---

## 💾 Data Retention & Audit

### Retention Policy

| Data Type | Retention Period | Format | Encryption |
|-----------|------------------|--------|------------|
| KYC/KYB documents | **10 years** after relationship ends | Encrypted PDFs | AES-256 |
| Transaction records | **10 years** | PostgreSQL | At-rest encryption |
| AML alerts | **10 years** | PostgreSQL | At-rest encryption |
| SAR reports | **Permanent** | PostgreSQL + encrypted files | AES-256 |
| Audit logs | **7 years** | Append-only log files | Hash-chained |
| Chainalysis API responses | **5 years** | JSONB | At-rest encryption |

### Audit Trail

**All critical operations logged:**
```go
type AuditLog struct {
    ID uuid.UUID
    Timestamp time.Time

    // Actor
    ActorType string // 'system', 'admin', 'merchant', 'api'
    ActorID   string
    ActorIP   string

    // Action
    Action       string // 'kyc.approved', 'payment.created', 'sar.filed', etc.
    ResourceType string // 'merchant', 'payment', 'payout', 'sar'
    ResourceID   uuid.UUID

    // Details
    OldValue JSONB // Before
    NewValue JSONB // After
    Metadata JSONB
}
```

**Example logs:**
- `kyc.submitted` - Merchant submits KYC
- `kyc.reviewed` - Admin reviews KYC
- `kyc.approved` - Admin approves KYC (WHO, WHEN, WHY)
- `payment.created` - Merchant creates payment
- `payment.confirmed` - System confirms payment
- `aml_alert.triggered` - AML rule triggered
- `aml_alert.reviewed` - Compliance officer reviews
- `sar.filed` - SAR filed to SBV
- `merchant.frozen` - Merchant account frozen

---

## 🛠️ Technology Stack

### AML/Compliance Tools

#### Option 1: Chainalysis (Recommended)

**Products:**
- **Chainalysis KYT (Know Your Transaction)**: Real-time transaction monitoring
- **Chainalysis Reactor**: Investigative tool for SAR preparation

**Pricing:**
- KYT: ~$10k-50k/year (volume-based)
- Reactor: ~$20k-40k/year

**Integration:**
```go
import "github.com/chainalysis/kyt-go"

client := kyt.NewClient(apiKey)

// Screen transaction
result, err := client.ScreenTransfer(kyt.TransferRequest{
    Network:      "bitcoin", // or "ethereum", "solana"
    Asset:        "USDT",
    TransferRef:  paymentID,
    Direction:    "received",
    Address:      sourceWallet,
    Amount:       amount,
})

if result.RiskScore > 75 {
    // Block transaction
}
```

---

#### Option 2: TRM Labs (Alternative)

**Products:**
- **TRM Transaction Monitoring**: Real-time AML
- **TRM Forensics**: Investigation tool

**Pricing:**
- Similar to Chainalysis
- Strong in Asia markets

---

#### Option 3: Merkle Science (Asia-Focused)

**Advantages:**
- Based in Singapore
- Strong Vietnam/SEA focus
- Lower pricing than Chainalysis

---

### KYC Providers (Vietnam)

#### VNPT eKYC
- OCR for Vietnam ID cards
- Facial recognition
- Live liveness detection
- API: RESTful
- Pricing: ~$0.50-1/check

#### FPT.AI eKYC
- Similar features
- Good accuracy
- Pricing: ~$0.50-1/check

#### Integration Example:
```go
import "github.com/vnpt/ekyc-go"

client := ekyc.NewClient(apiKey)

// OCR ID card
result, err := client.OCR(ekyc.OCRRequest{
    ImageFront: base64EncodedFront,
    ImageBack:  base64EncodedBack,
})

// Facial recognition
faceResult, err := client.FaceMatch(ekyc.FaceMatchRequest{
    IDCardImage: result.FaceImage,
    SelfieImage: base64EncodedSelfie,
})

if faceResult.MatchScore > 0.8 {
    // KYC passed
}
```

---

### Sanctions Screening

#### Dow Jones Risk & Compliance
- Comprehensive PEP/sanctions database
- API access
- Pricing: ~$5k-15k/year

#### ComplyAdvantage
- AI-powered screening
- Real-time updates
- Pricing: ~$10k-20k/year

---

## 📋 Operational Procedures

### Daily Compliance Tasks

**Compliance Officer Checklist:**
```
Morning (9 AM):
├── Review overnight AML alerts (HIGH/CRITICAL priority)
├── Check pending KYC applications (< 24h response time)
├── Review pending payouts (fraud check)
└── System health check (Chainalysis API, eKYC API)

Afternoon (2 PM):
├── Investigate flagged transactions (risk score > 60)
├── Prepare SARs if needed
├── Update merchant risk profiles (if new info)
└── Respond to merchant compliance queries

End of Day (5 PM):
├── Daily compliance report (to management)
│   ├── # of KYC approvals/rejections
│   ├── # of AML alerts (by severity)
│   ├── # of SARs filed
│   └── # of transactions processed
├── File any completed SARs
└── Schedule next-day priorities
```

---

### Weekly Compliance Tasks

**Every Monday:**
- Review last week's AML performance
- Update risk models (if patterns detected)
- Training for ops team (15 minutes)

**Every Friday:**
- Compliance metrics report to management
- Review upcoming merchant renewals (quarterly KYB review)

---

### Monthly Compliance Tasks

- Full system audit (sample check 10% of transactions)
- Review and update AML rules (if needed)
- Chainalysis/TRM performance review
- Compliance training (1 hour for all staff)

---

### Quarterly Compliance Tasks

- External compliance audit (hire 3rd party)
- Review all Tier 2/Tier 3 merchants (KYB refresh)
- Update compliance policies (if regulations change)
- Stress test AML system

---

## 📜 Regulatory Compliance (Vietnam)

### Applicable Laws & Regulations

1. **Law on Anti-Money Laundering (2012)**
   - Requires KYC for financial transactions
   - SAR filing obligations
   - Record retention (5-10 years)

2. **Decree 80/2019/ND-CP**
   - Updated AML/CTF regulations
   - Enhanced due diligence for high-risk customers

3. **Circular 35/2013/TT-NHNN** (State Bank of Vietnam)
   - KYC requirements for payment intermediaries
   - Transaction monitoring standards

4. **Vietnam Data Protection Law (Decree 13/2023)**
   - Personal data processing
   - Consent requirements
   - Data breach notification

### Compliance Checklist

- [ ] **Business Registration**: Register as "Payment Technology Service Provider"
- [ ] **AML Policy**: Written AML/CTF policy document
- [ ] **Data Protection**: GDPR-equivalent compliance
- [ ] **Customer Consent**: Explicit consent for data processing
- [ ] **SAR Reporting**: Process to report to SBV within 24-48 hours
- [ ] **Record Retention**: 10-year retention infrastructure
- [ ] **Staff Training**: All staff trained on AML/CTF (quarterly)
- [ ] **Audit**: Annual external compliance audit

---

## 🎯 Compliance KPIs

### Monthly Metrics

| Metric | Target | Critical Threshold |
|--------|--------|-------------------|
| KYC approval time (median) | < 24 hours | > 72 hours |
| KYC completion rate | > 95% | < 80% |
| AML screening latency (p99) | < 500ms | > 2 seconds |
| False positive rate | < 10% | > 30% |
| SAR filing time (from detection) | < 24 hours | > 48 hours |
| Compliance incidents | 0 | > 1 |

---

## 🚀 Compliance Roadmap

### Phase 1: MVP (Week 1-8)
- [ ] Basic KYB (Tier 1) implementation
- [ ] Chainalysis KYT integration (sandbox)
- [ ] Manual AML review workflow
- [ ] Audit logging infrastructure

### Phase 2: Automation (Month 2-3)
- [ ] Automated eKYC (VNPT/FPT.AI)
- [ ] Tier 2/Tier 3 KYB workflows
- [ ] Real-time AML screening (all chains)
- [ ] SAR automation tools

### Phase 3: Advanced (Month 4-6)
- [ ] ML-based fraud detection
- [ ] Behavioral analytics
- [ ] Predictive risk modeling
- [ ] Compliance dashboard (management)

---

**Built for Regulatory Excellence 🛡️**

*Last updated: 2025-11-16*
