# AML Engine - Anti-Money Laundering System

**Project**: Stablecoin Payment Gateway - AML Compliance Module
**Last Updated**: 2025-11-19
**Status**: Design Phase

---

## 🎯 Overview

This document describes the **in-house AML (Anti-Money Laundering) engine** designed to provide cost-effective, standards-compliant transaction monitoring and risk assessment for the stablecoin payment gateway.

### Why Build In-House?

**Cost Savings**:
- 3rd party AML services: $500-5,000/month + per-transaction fees
- In-house solution: Development cost + maintenance (more scalable)

**Customization**:
- Vietnam-specific compliance rules
- Tourism industry risk patterns
- Crypto-specific transaction monitoring
- Direct integration with payment flow

**Data Control**:
- Keep sensitive customer data in-house
- Comply with local data residency requirements
- Full audit trail ownership

---

## 📋 AML Standards & Regulations

### International Standards

**FATF (Financial Action Task Force)**:
- 40 Recommendations for AML/CFT compliance
- Risk-based approach to customer due diligence
- Travel Rule for crypto transfers (>$1,000 USD)
- Beneficial ownership identification

**Key FATF Requirements**:
1. **Customer Due Diligence (CDD)**: Verify customer identity
2. **Enhanced Due Diligence (EDD)**: For high-risk customers
3. **Ongoing Monitoring**: Continuous transaction surveillance
4. **Suspicious Activity Reporting (SAR)**: Report suspicious transactions
5. **Record Keeping**: Maintain records for 5-7 years
6. **Sanctions Screening**: Check against global watchlists

### Vietnam-Specific Compliance

**Law on Anti-Money Laundering (2022)**:
- Decree 74/2023/ND-CP (implementing regulations)
- Required for all "reporting entities" (includes payment processors)
- Threshold reporting: Transactions > 400M VND (~$16,000 USD)
- Suspicious transaction reporting to State Bank of Vietnam

**Key Vietnam Requirements**:
- Customer identification and verification
- Transaction monitoring and reporting
- Internal AML policies and procedures
- AML officer designation
- Annual AML training for staff
- Cooperation with authorities

### Crypto-Specific Standards

**FATF Travel Rule**:
- Applies to crypto transfers ≥ $1,000 USD
- Must collect originator and beneficiary information
- Transmit information to counterparty VASP (Virtual Asset Service Provider)

**Crypto Risk Factors**:
- Source of funds (wallet history analysis)
- Mixing/tumbling services usage
- High-risk jurisdiction exposure
- Rapid movement of funds
- Structuring (smurfing) patterns

---

## 🏗️ AML Engine Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        AML ENGINE                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────┐  ┌──────────────────┐  ┌────────────────┐ │
│  │ Customer Risk   │  │ Transaction      │  │ Sanctions      │ │
│  │ Scoring         │  │ Monitoring       │  │ Screening      │ │
│  └─────────────────┘  └──────────────────┘  └────────────────┘ │
│                                                                   │
│  ┌─────────────────┐  ┌──────────────────┐  ┌────────────────┐ │
│  │ Alert           │  │ Case             │  │ Reporting      │ │
│  │ Management      │  │ Management       │  │ Engine         │ │
│  └─────────────────┘  └──────────────────┘  └────────────────┘ │
│                                                                   │
│  ┌─────────────────┐  ┌──────────────────┐  ┌────────────────┐ │
│  │ Wallet          │  │ Rule             │  │ ML Anomaly     │ │
│  │ Screening       │  │ Engine           │  │ Detection      │ │
│  └─────────────────┘  └──────────────────┘  └────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
                              ↕
                    ┌──────────────────┐
                    │  Payment Gateway │
                    │  (Core System)   │
                    └──────────────────┘
```

### Component Descriptions

#### 1. Customer Risk Scoring
**Purpose**: Assess and assign risk levels to merchants and end-users

**Risk Levels**:
- **Low Risk**: Established business, clean history, normal patterns
- **Medium Risk**: New customer, moderate volume, some red flags
- **High Risk**: PEP, high-risk jurisdiction, unusual patterns
- **Prohibited**: Sanctioned entity, blacklisted

**Risk Factors**:
- Business type (tourism = lower risk, forex/gambling = higher)
- Geographic location (Vietnam = lower, sanctioned countries = prohibited)
- Transaction volume and velocity
- KYC completeness
- Historical compliance issues
- PEP status
- Adverse media

**Scoring Algorithm**:
```
Customer Risk Score = Σ (Factor Weight × Factor Score)

Thresholds:
- 0-30: Low Risk
- 31-60: Medium Risk
- 61-85: High Risk
- 86-100: Prohibited
```

#### 2. Transaction Monitoring
**Purpose**: Real-time and batch analysis of all payment transactions

**Monitoring Rules**:

**Threshold Monitoring**:
- Single transaction > 400M VND (Vietnam legal requirement)
- Single transaction > 10M VND (internal threshold for crypto)
- Daily aggregate > 1B VND per merchant
- Weekly aggregate > 5B VND per merchant

**Pattern Detection**:
- **Structuring**: Multiple transactions just below threshold
- **Rapid Movement**: Payment received → immediate payout
- **Round Amount**: Suspicious round numbers (1M, 5M, 10M exactly)
- **Unusual Hours**: Transactions outside business hours
- **Velocity**: Sudden spike in transaction frequency
- **Geographic Anomaly**: Payments from unexpected countries

**Behavioral Analysis**:
- Baseline: Establish normal merchant behavior (30-day rolling average)
- Deviation: Flag transactions >3 standard deviations from baseline
- Peer Comparison: Compare to similar merchants in same industry

**Example Rules**:
```
Rule: STRUCT_001 - Structuring Detection
Condition:
  - 3+ transactions within 24 hours
  - Each transaction 80-95% of threshold (e.g., 8-9.5M VND if threshold is 10M)
  - Same merchant
Action: Generate MEDIUM alert

Rule: RAPID_002 - Rapid Cash-Out
Condition:
  - Payment confirmed
  - Payout request within 1 hour
  - >80% of balance withdrawn
Action: Generate HIGH alert

Rule: VEL_003 - Transaction Velocity Spike
Condition:
  - Transaction count today > 5x daily average
  - Merchant age > 30 days (to establish baseline)
Action: Generate MEDIUM alert
```

#### 3. Sanctions Screening
**Purpose**: Check customers and wallet addresses against global sanctions lists

**Watchlists** (to be integrated):
- **OFAC SDN List** (US Treasury - Specially Designated Nationals)
- **UN Sanctions List**
- **EU Consolidated Sanctions List**
- **Interpol Red Notices**
- **Vietnam Government Blacklists**

**Screening Triggers**:
- New merchant registration (KYC phase)
- Ongoing: Daily batch screening of all active merchants
- Transaction-level: Wallet address screening for high-value payments (>$1,000)

**Matching Algorithm**:
- **Exact Match**: Name/address/ID exactly matches (auto-reject)
- **Fuzzy Match** (>85% similarity): Generate alert for manual review
- **False Positive Management**: Allow admin to mark false positives

**Data Sources** (Open Source):
- OFAC XML feed: https://sanctionslistservice.ofac.treas.gov/api/PublicationPreview/exports/SDN.XML
- UN Consolidated List: https://www.un.org/securitycouncil/content/un-sc-consolidated-list
- EU Sanctions Map: https://www.sanctionsmap.eu/

**Implementation**:
```
1. Daily update: Fetch latest sanctions lists
2. Parse and normalize names (handle diacritics, aliases)
3. Store in searchable database table
4. Screening: Use fuzzy matching algorithm (Levenshtein distance, Soundex)
5. Cache results to avoid repeated checks
```

#### 4. Wallet Screening (Blockchain Intelligence)
**Purpose**: Analyze source of crypto funds and wallet risk

**Wallet Risk Factors**:
- **Mixing Services**: Tornado Cash, ChipMixer, etc. (HIGH RISK)
- **Darknet Markets**: Known illicit marketplace wallets (PROHIBITED)
- **Ransomware**: Addresses linked to ransomware payments (PROHIBITED)
- **Sanctioned Addresses**: OFAC-sanctioned crypto wallets (PROHIBITED)
- **High-Risk Exchanges**: Non-KYC exchanges (MEDIUM RISK)

**Data Sources** (for MVP - free/open options):
- TRM Labs Sanctions Screener API (free tier: 100 requests/day)
- Chainalysis Sanctions Oracle (on-chain, free)
- Elliptic Navigator (paid, but industry standard)
- Open-source lists: https://github.com/0xB10C/ofac-sanctioned-digital-currency-addresses

**Screening Process**:
1. Extract sender wallet address from blockchain transaction
2. Check against local blacklist cache
3. If not cached, query external API
4. Analyze wallet transaction history (depth: 3 hops)
5. Calculate wallet risk score
6. Cache result for 7 days

**Wallet Risk Score**:
```
Score = Direct Risk (60%) + Indirect Risk (30%) + Metadata (10%)

Direct Risk:
- Wallet on sanctions list: 100 points
- Direct interaction with mixer: 80 points
- Direct interaction with darknet: 100 points

Indirect Risk:
- 1-hop from mixer: 40 points
- 2-hop from mixer: 20 points
- 3-hop from mixer: 10 points

Metadata:
- Wallet age < 7 days: 20 points
- Single-use wallet: 15 points
- No prior transactions: 10 points
```

#### 5. Alert Management
**Purpose**: Generate, prioritize, and manage AML alerts for review

**Alert Severity**:
- **LOW**: Minor threshold breach, informational
- **MEDIUM**: Pattern detected, requires review within 48 hours
- **HIGH**: Serious red flag, requires review within 24 hours
- **CRITICAL**: Sanctions hit or prohibited activity, immediate action

**Alert Lifecycle**:
```
Created → Assigned → Under Review → Resolved (Cleared | Escalated | SAR Filed)
```

**Alert Generation**:
```go
type Alert struct {
    ID            string
    Type          string    // "THRESHOLD", "PATTERN", "SANCTIONS", "WALLET"
    Severity      string    // "LOW", "MEDIUM", "HIGH", "CRITICAL"
    MerchantID    string
    PaymentID     *string   // nullable
    RuleName      string
    Description   string
    Metadata      JSONB     // rule-specific data
    Status        string    // "created", "assigned", "under_review", "resolved"
    AssignedTo    *string   // admin user ID
    CreatedAt     time.Time
    ReviewedAt    *time.Time
    Resolution    *string   // "cleared", "escalated", "sar_filed"
    ResolutionNotes *string
}
```

**Alert Prioritization** (for admin dashboard):
1. CRITICAL alerts first
2. Then by creation time (oldest first)
3. Assigned alerts to show for assigned reviewer

**Auto-Resolution** (to reduce noise):
- LOW alerts: Auto-close after 7 days if no escalation
- FALSE POSITIVE: If same rule triggers 3+ times and always cleared, reduce sensitivity

#### 6. Case Management
**Purpose**: Investigate complex alerts and document findings

**Case Creation Triggers**:
- Multiple related alerts (same merchant, pattern)
- HIGH or CRITICAL alert escalated
- External tip or regulatory inquiry
- Periodic review of high-risk customers

**Case Workflow**:
```
Opened → Investigation → Evidence Collection → Decision → Closed
```

**Investigation Checklist**:
- [ ] Review all related alerts
- [ ] Check customer KYC documents
- [ ] Analyze transaction history
- [ ] Review payout patterns
- [ ] Check external sources (Google, social media)
- [ ] Document findings
- [ ] Make decision (clear, monitor, SAR, reject)

**Case Documentation**:
```go
type Case struct {
    ID              string
    MerchantID      string
    CaseType        string    // "investigation", "sar", "periodic_review"
    Severity        string
    Status          string    // "open", "investigating", "closed"
    AlertIDs        []string  // related alerts
    AssignedTo      string    // compliance officer
    OpenedAt        time.Time
    ClosedAt        *time.Time
    Outcome         *string   // "cleared", "sar_filed", "account_suspended", "account_terminated"
    Summary         string
    Notes           string    // investigation notes
    Documents       []string  // file paths to supporting documents
}
```

#### 7. Reporting Engine
**Purpose**: Generate regulatory reports and compliance dashboards

**Required Reports**:

**Vietnam Regulatory Reports**:
- **Threshold Report**: All transactions > 400M VND (monthly to State Bank)
- **Suspicious Activity Report (SAR)**: As needed for suspicious patterns
- **Annual AML Compliance Report**: Summary of AML activities

**Internal Reports**:
- **Daily AML Summary**: Alerts generated, resolved, pending
- **Monthly Risk Report**: Customer risk distribution, trends
- **Quarterly Executive Summary**: KPIs, SAR filed, system effectiveness

**Report Templates**:
```
Vietnam SAR Format:
- Reporting Entity Information
- Subject Information (merchant details)
- Transaction Details (amount, date, payment method)
- Suspicious Activity Description
- Supporting Evidence
- Officer Name and Signature
```

**KPIs to Track**:
- Total alerts generated (by type, severity)
- Alert resolution time (avg, median, p95)
- False positive rate (target: <20%)
- SAR filed count
- Customer risk distribution (low/medium/high/prohibited %)
- Sanctions screening hit rate
- High-risk wallet detection rate

#### 8. Rule Engine
**Purpose**: Flexible, configurable rule definitions for transaction monitoring

**Rule Structure**:
```go
type AMLRule struct {
    ID          string
    Name        string
    Description string
    Category    string    // "threshold", "pattern", "velocity", "behavioral"
    Enabled     bool
    Severity    string    // alert severity if triggered
    Conditions  []Condition
    Actions     []Action
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Condition struct {
    Field       string    // "amount_vnd", "tx_count_24h", "merchant_age_days"
    Operator    string    // ">", "<", "==", "between", "in"
    Value       interface{}
}

type Action struct {
    Type        string    // "create_alert", "block_transaction", "require_review"
    Params      map[string]interface{}
}
```

**Example Rule Definition** (JSON config):
```json
{
  "id": "THRESHOLD_VN_001",
  "name": "Vietnam Legal Threshold",
  "description": "Flag transactions exceeding 400M VND (Vietnam legal requirement)",
  "category": "threshold",
  "enabled": true,
  "severity": "MEDIUM",
  "conditions": [
    {
      "field": "amount_vnd",
      "operator": ">=",
      "value": 400000000
    }
  ],
  "actions": [
    {
      "type": "create_alert",
      "params": {
        "alert_type": "THRESHOLD",
        "notify_admin": true
      }
    },
    {
      "type": "require_review",
      "params": {
        "review_type": "manual_approval"
      }
    }
  ]
}
```

**Rule Categories**:

1. **Threshold Rules**:
   - Single transaction thresholds
   - Aggregate thresholds (daily, weekly, monthly)
   - Country-specific legal thresholds

2. **Pattern Rules**:
   - Structuring detection
   - Rapid cash-out
   - Round amount detection
   - Time-based patterns (unusual hours)

3. **Velocity Rules**:
   - Transaction frequency spikes
   - Volume spikes
   - Sudden changes in behavior

4. **Behavioral Rules**:
   - Deviation from baseline
   - Peer comparison
   - Geographic anomalies

**Rule Management**:
- Admin can enable/disable rules via dashboard
- Rule changes logged in audit trail
- Test mode: Run rule without triggering alerts (dry-run)
- Rule effectiveness tracking: Alert→SAR conversion rate

#### 9. ML Anomaly Detection (Phase 2 - Optional)
**Purpose**: Use machine learning to detect unusual patterns

**Approach** (for future implementation):
- **Unsupervised Learning**: Isolation Forest, One-Class SVM
- **Features**: Transaction amount, frequency, time, merchant profile, wallet metadata
- **Training**: Use historical normal transactions
- **Anomaly Score**: 0-100 (>80 = potential anomaly)
- **Integration**: ML score as input to rule engine

**MVP Approach**:
- Start with rule-based system (sufficient for MVP)
- Collect data for 6 months
- Evaluate ML models in Phase 2
- Consider open-source libraries: scikit-learn, TensorFlow

---

## 🗄️ Database Schema

### AML Tables

#### aml_customer_risk_scores
```sql
CREATE TABLE aml_customer_risk_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    risk_level VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'prohibited'
    risk_score INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_factors JSONB NOT NULL, -- detailed breakdown of factors
    last_assessed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    next_review_date DATE NOT NULL, -- periodic review schedule
    assessed_by VARCHAR(50), -- 'system' or admin user
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_aml_risk_merchant ON aml_customer_risk_scores(merchant_id);
CREATE INDEX idx_aml_risk_level ON aml_customer_risk_scores(risk_level);
CREATE INDEX idx_aml_risk_review ON aml_customer_risk_scores(next_review_date);
```

#### aml_transaction_monitoring
```sql
CREATE TABLE aml_transaction_monitoring (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID REFERENCES payments(id),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    amount_vnd DECIMAL(20, 2) NOT NULL,
    amount_crypto DECIMAL(20, 8),
    crypto_currency VARCHAR(10),
    monitoring_status VARCHAR(20) NOT NULL, -- 'clean', 'flagged', 'blocked'
    risk_score INTEGER, -- 0-100
    rules_triggered TEXT[], -- array of rule IDs that fired
    wallet_address VARCHAR(255),
    wallet_risk_score INTEGER,
    metadata JSONB, -- additional context
    reviewed BOOLEAN DEFAULT FALSE,
    reviewed_by UUID REFERENCES admin_users(id),
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_aml_txn_payment ON aml_transaction_monitoring(payment_id);
CREATE INDEX idx_aml_txn_merchant ON aml_transaction_monitoring(merchant_id);
CREATE INDEX idx_aml_txn_status ON aml_transaction_monitoring(monitoring_status);
CREATE INDEX idx_aml_txn_reviewed ON aml_transaction_monitoring(reviewed);
CREATE INDEX idx_aml_txn_created ON aml_transaction_monitoring(created_at);
```

#### aml_alerts
```sql
CREATE TABLE aml_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_type VARCHAR(50) NOT NULL, -- 'THRESHOLD', 'PATTERN', 'SANCTIONS', 'WALLET'
    severity VARCHAR(20) NOT NULL, -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    payment_id UUID REFERENCES payments(id),
    rule_id VARCHAR(100), -- reference to rule that triggered
    rule_name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    metadata JSONB, -- rule-specific data
    status VARCHAR(30) NOT NULL DEFAULT 'created', -- 'created', 'assigned', 'under_review', 'resolved'
    assigned_to UUID REFERENCES admin_users(id),
    resolution VARCHAR(30), -- 'cleared', 'false_positive', 'escalated', 'sar_filed'
    resolution_notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    assigned_at TIMESTAMP,
    reviewed_at TIMESTAMP,
    resolved_at TIMESTAMP
);

CREATE INDEX idx_aml_alert_merchant ON aml_alerts(merchant_id);
CREATE INDEX idx_aml_alert_status ON aml_alerts(status);
CREATE INDEX idx_aml_alert_severity ON aml_alerts(severity);
CREATE INDEX idx_aml_alert_assigned ON aml_alerts(assigned_to);
CREATE INDEX idx_aml_alert_created ON aml_alerts(created_at);
```

#### aml_cases
```sql
CREATE TABLE aml_cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_number VARCHAR(50) UNIQUE NOT NULL, -- human-readable: CASE-2025-001
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    case_type VARCHAR(50) NOT NULL, -- 'investigation', 'sar', 'periodic_review'
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'open', -- 'open', 'investigating', 'closed'
    alert_ids UUID[], -- related alerts
    assigned_to UUID NOT NULL REFERENCES admin_users(id),
    opened_at TIMESTAMP NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP,
    outcome VARCHAR(50), -- 'cleared', 'sar_filed', 'account_suspended', 'account_terminated'
    summary TEXT,
    investigation_notes TEXT,
    supporting_documents JSONB, -- file paths/URLs
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_aml_case_merchant ON aml_cases(merchant_id);
CREATE INDEX idx_aml_case_status ON aml_cases(status);
CREATE INDEX idx_aml_case_assigned ON aml_cases(assigned_to);
CREATE SEQUENCE case_number_seq START 1;
```

#### aml_sanctions_list
```sql
CREATE TABLE aml_sanctions_list (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_source VARCHAR(50) NOT NULL, -- 'OFAC', 'UN', 'EU', 'INTERPOL'
    entry_type VARCHAR(30) NOT NULL, -- 'individual', 'entity', 'vessel', 'aircraft'
    name_primary VARCHAR(500) NOT NULL,
    name_aliases TEXT[], -- array of alternative names
    id_numbers TEXT[], -- passport, national ID, tax ID, etc.
    addresses TEXT[],
    date_of_birth DATE,
    place_of_birth VARCHAR(255),
    nationalities VARCHAR(10)[],
    programs TEXT[], -- sanction programs (e.g., 'IRGQ', 'SDGT')
    remarks TEXT,
    list_updated_at DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sanctions_name ON aml_sanctions_list USING gin(to_tsvector('simple', name_primary));
CREATE INDEX idx_sanctions_source ON aml_sanctions_list(list_source);
CREATE INDEX idx_sanctions_type ON aml_sanctions_list(entry_type);
```

#### aml_wallet_screening
```sql
CREATE TABLE aml_wallet_screening (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL, -- 'solana', 'bsc', 'ethereum'
    risk_level VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'prohibited'
    risk_score INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_factors JSONB NOT NULL, -- detailed risk breakdown
    is_sanctioned BOOLEAN DEFAULT FALSE,
    screening_source VARCHAR(50), -- 'chainalysis', 'trm', 'elliptic', 'internal'
    last_screened_at TIMESTAMP NOT NULL DEFAULT NOW(),
    cache_expires_at TIMESTAMP NOT NULL, -- cache for 7 days
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_wallet_screen_addr_chain ON aml_wallet_screening(wallet_address, blockchain);
CREATE INDEX idx_wallet_screen_risk ON aml_wallet_screening(risk_level);
CREATE INDEX idx_wallet_screen_sanctioned ON aml_wallet_screening(is_sanctioned);
CREATE INDEX idx_wallet_screen_expires ON aml_wallet_screening(cache_expires_at);
```

#### aml_rules
```sql
CREATE TABLE aml_rules (
    id VARCHAR(100) PRIMARY KEY, -- e.g., 'THRESHOLD_VN_001'
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL, -- 'threshold', 'pattern', 'velocity', 'behavioral'
    enabled BOOLEAN DEFAULT TRUE,
    severity VARCHAR(20) NOT NULL, -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    conditions JSONB NOT NULL,
    actions JSONB NOT NULL,
    metadata JSONB, -- additional config
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES admin_users(id),
    updated_by UUID REFERENCES admin_users(id)
);

CREATE INDEX idx_aml_rule_category ON aml_rules(category);
CREATE INDEX idx_aml_rule_enabled ON aml_rules(enabled);
```

#### aml_reports
```sql
CREATE TABLE aml_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type VARCHAR(50) NOT NULL, -- 'SAR', 'THRESHOLD', 'MONTHLY', 'QUARTERLY'
    report_period_start DATE,
    report_period_end DATE,
    generated_by UUID NOT NULL REFERENCES admin_users(id),
    generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    file_path VARCHAR(500), -- path to generated PDF/CSV
    status VARCHAR(30) NOT NULL DEFAULT 'draft', -- 'draft', 'finalized', 'submitted'
    submitted_to VARCHAR(100), -- 'State Bank of Vietnam', 'Internal'
    submitted_at TIMESTAMP,
    metadata JSONB, -- report-specific data
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_aml_report_type ON aml_reports(report_type);
CREATE INDEX idx_aml_report_status ON aml_reports(status);
CREATE INDEX idx_aml_report_period ON aml_reports(report_period_start, report_period_end);
```

#### aml_audit_log
```sql
CREATE TABLE aml_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_type VARCHAR(30) NOT NULL, -- 'system', 'admin', 'compliance_officer'
    actor_id UUID, -- admin user ID if applicable
    action VARCHAR(100) NOT NULL, -- 'alert_created', 'case_opened', 'rule_updated', etc.
    resource_type VARCHAR(50) NOT NULL, -- 'alert', 'case', 'rule', 'merchant'
    resource_id UUID,
    changes JSONB, -- before/after state
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_aml_audit_actor ON aml_audit_log(actor_id);
CREATE INDEX idx_aml_audit_resource ON aml_audit_log(resource_type, resource_id);
CREATE INDEX idx_aml_audit_created ON aml_audit_log(created_at);
```

---

## 🔄 Integration with Payment Gateway

### Payment Flow with AML Checks

```
┌──────────────────────────────────────────────────────────────┐
│ PAYMENT CREATION                                              │
└──────────────────────────────────────────────────────────────┘
    ↓
1. Merchant creates payment via API
    ↓
2. Check merchant risk level
   - If PROHIBITED: Reject immediately
   - If HIGH: Require manual approval
   - If MEDIUM/LOW: Proceed
    ↓
3. Create payment record
    ↓
┌──────────────────────────────────────────────────────────────┐
│ PAYMENT CONFIRMATION                                          │
└──────────────────────────────────────────────────────────────┘
    ↓
4. Blockchain listener detects transaction
    ↓
5. AML PRE-SCREENING (real-time):
   a. Extract wallet address
   b. Screen wallet (check cache or query API)
   c. If wallet PROHIBITED: Block transaction, create CRITICAL alert
   d. If wallet HIGH risk: Create HIGH alert, hold for review
   e. If wallet MEDIUM/LOW: Proceed
    ↓
6. AML TRANSACTION MONITORING:
   a. Create aml_transaction_monitoring record
   b. Run all enabled rules against transaction
   c. Calculate transaction risk score
   d. If rules triggered:
      - Create alerts based on severity
      - If CRITICAL: Block confirmation, require manual approval
      - If HIGH/MEDIUM: Create alert, allow confirmation
      - If LOW: Log only
    ↓
7. Confirm payment (if not blocked)
    ↓
8. Update merchant balance
    ↓
9. Send webhook to merchant
    ↓
┌──────────────────────────────────────────────────────────────┐
│ BACKGROUND PROCESSING (async)                                 │
└──────────────────────────────────────────────────────────────┘
    ↓
10. Update merchant risk score (daily batch)
11. Generate reports (daily/weekly/monthly)
12. Update sanctions lists (daily)
13. Periodic customer reviews (based on risk level)
```

### API Integration Points

**Merchant Registration** (KYC Phase):
```go
// In merchant registration flow
func (s *MerchantService) RegisterMerchant(req RegisterRequest) error {
    // ... existing KYC checks ...

    // AML: Sanctions screening
    sanctionsHit, err := s.amlService.ScreenSanctions(req.OwnerName, req.OwnerID, req.BusinessName)
    if err != nil {
        return fmt.Errorf("sanctions screening failed: %w", err)
    }

    if sanctionsHit {
        // Create CRITICAL alert
        s.amlService.CreateAlert(Alert{
            Type: "SANCTIONS",
            Severity: "CRITICAL",
            Description: fmt.Sprintf("Sanctions hit: %s", req.OwnerName),
            // ...
        })
        return ErrSanctionsHit
    }

    // Calculate initial risk score
    riskScore, err := s.amlService.CalculateCustomerRiskScore(merchant)
    if err != nil {
        return err
    }

    if riskScore.RiskLevel == "prohibited" {
        return ErrHighRiskCustomer
    }

    // Store risk score
    s.amlService.SaveRiskScore(riskScore)

    // ... continue registration ...
}
```

**Payment Confirmation**:
```go
func (s *PaymentService) ConfirmPayment(txHash string) error {
    // ... blockchain validation ...

    // AML: Wallet screening
    walletRisk, err := s.amlService.ScreenWallet(senderAddress, blockchain)
    if err != nil {
        log.Errorf("Wallet screening failed: %v", err)
        // Don't block payment on screening error, but create alert
        s.amlService.CreateAlert(Alert{
            Type: "WALLET_SCREENING_ERROR",
            Severity: "MEDIUM",
            // ...
        })
    }

    if walletRisk != nil && walletRisk.IsProhibited {
        // Block transaction
        s.amlService.CreateAlert(Alert{
            Type: "WALLET",
            Severity: "CRITICAL",
            Description: fmt.Sprintf("Prohibited wallet: %s (sanctioned)", senderAddress),
            // ...
        })
        return ErrProhibitedWallet
    }

    // AML: Transaction monitoring
    monitoringResult, err := s.amlService.MonitorTransaction(MonitorTransactionRequest{
        PaymentID: payment.ID,
        MerchantID: payment.MerchantID,
        AmountVND: payment.AmountVND,
        AmountCrypto: payment.AmountCrypto,
        CryptoCurrency: payment.CryptoCurrency,
        WalletAddress: senderAddress,
        WalletRiskScore: walletRisk.RiskScore,
    })

    if monitoringResult.ShouldBlock {
        // Block confirmation, require manual approval
        payment.Status = "aml_review"
        s.paymentRepo.Update(payment)

        return ErrAMLReviewRequired
    }

    // Proceed with confirmation
    payment.Status = "completed"
    // ... continue ...
}
```

**Payout Request**:
```go
func (s *PayoutService) RequestPayout(req PayoutRequest) error {
    // ... existing validation ...

    // AML: Check customer risk level
    riskScore, err := s.amlService.GetCurrentRiskScore(req.MerchantID)
    if err != nil {
        return err
    }

    if riskScore.RiskLevel == "high" || riskScore.RiskLevel == "prohibited" {
        // Require enhanced review
        payout.RequiresEnhancedReview = true

        // Create alert
        s.amlService.CreateAlert(Alert{
            Type: "PAYOUT",
            Severity: "HIGH",
            Description: "Payout request from high-risk merchant",
            // ...
        })
    }

    // AML: Check recent alerts
    recentAlerts, err := s.amlService.GetRecentAlerts(req.MerchantID, 30) // last 30 days
    if err != nil {
        return err
    }

    if len(recentAlerts) > 5 {
        // Multiple recent alerts, require review
        payout.RequiresEnhancedReview = true
    }

    // ... continue payout creation ...
}
```

---

## 🛠️ Implementation Guide

### Phase 1: Core AML Engine (Week 1-2)

**Week 1: Database & Models**
- [ ] Create database migrations for all AML tables
- [ ] Implement Go models for AML entities
- [ ] Set up repository layer (GORM/sqlx)
- [ ] Write unit tests for models and repositories

**Week 2: Core Services**
- [ ] Implement Customer Risk Scoring service
- [ ] Implement Transaction Monitoring service
- [ ] Implement Alert Management service
- [ ] Implement Rule Engine (basic rules)
- [ ] Write integration tests

### Phase 2: Sanctions & Wallet Screening (Week 3)

**Week 3: External Integrations**
- [ ] Implement sanctions list downloader (OFAC, UN)
- [ ] Implement sanctions screening (fuzzy matching)
- [ ] Integrate wallet screening API (Chainalysis or TRM Labs)
- [ ] Implement caching for screening results
- [ ] Set up daily batch jobs for list updates
- [ ] Write integration tests with mock data

### Phase 3: Admin Interface (Week 4)

**Week 4: Admin Dashboard**
- [ ] Build alert dashboard (list, filter, assign)
- [ ] Build case management interface
- [ ] Build merchant risk profile page
- [ ] Build rule management interface
- [ ] Build reporting interface
- [ ] Implement real-time notifications for CRITICAL alerts

### Phase 4: Reporting & Compliance (Week 5)

**Week 5: Reporting**
- [ ] Implement SAR report generator
- [ ] Implement threshold report generator
- [ ] Implement monthly/quarterly summary reports
- [ ] Build audit log viewer
- [ ] Create compliance dashboard for executives
- [ ] Document all reports and processes

### Phase 5: Testing & Deployment (Week 6)

**Week 6: QA & Launch**
- [ ] End-to-end testing with test data
- [ ] Load testing (simulate high-alert scenarios)
- [ ] Security audit of AML system
- [ ] Train compliance team on system usage
- [ ] Deploy to production
- [ ] Monitor and tune rules based on false positive rate

---

## 📊 Rule Definitions (MVP)

### Threshold Rules

```json
{
  "id": "THRESHOLD_VN_001",
  "name": "Vietnam Legal Threshold",
  "description": "Vietnam requires reporting of transactions ≥ 400M VND",
  "category": "threshold",
  "enabled": true,
  "severity": "MEDIUM",
  "conditions": [
    {"field": "amount_vnd", "operator": ">=", "value": 400000000}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "THRESHOLD"}},
    {"type": "flag_for_reporting", "params": {"report_type": "threshold_report"}}
  ]
}
```

```json
{
  "id": "THRESHOLD_CRYPTO_001",
  "name": "High-Value Crypto Transaction",
  "description": "Internal threshold for crypto payments (10M VND ≈ $400 USD)",
  "category": "threshold",
  "enabled": true,
  "severity": "LOW",
  "conditions": [
    {"field": "amount_vnd", "operator": ">=", "value": 10000000}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "THRESHOLD"}}
  ]
}
```

```json
{
  "id": "THRESHOLD_DAILY_001",
  "name": "Daily Aggregate Threshold",
  "description": "Flag merchants exceeding 1B VND in single day",
  "category": "threshold",
  "enabled": true,
  "severity": "MEDIUM",
  "conditions": [
    {"field": "daily_total_vnd", "operator": ">=", "value": 1000000000}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "THRESHOLD_AGGREGATE"}}
  ]
}
```

### Structuring Rules

```json
{
  "id": "STRUCT_001",
  "name": "Structuring Detection - 24 Hour Window",
  "description": "Multiple transactions just below threshold within 24 hours",
  "category": "pattern",
  "enabled": true,
  "severity": "MEDIUM",
  "conditions": [
    {"field": "tx_count_24h", "operator": ">=", "value": 3},
    {"field": "amount_vnd", "operator": "between", "value": [8000000, 9500000]},
    {"field": "same_merchant", "operator": "==", "value": true}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "STRUCTURING"}}
  ]
}
```

### Velocity Rules

```json
{
  "id": "VEL_001",
  "name": "Transaction Velocity Spike",
  "description": "Transaction count today > 5x daily average (for established merchants)",
  "category": "velocity",
  "enabled": true,
  "severity": "MEDIUM",
  "conditions": [
    {"field": "merchant_age_days", "operator": ">", "value": 30},
    {"field": "tx_count_today", "operator": ">", "value": "5 * avg_daily_tx"}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "VELOCITY_SPIKE"}}
  ]
}
```

```json
{
  "id": "VEL_002",
  "name": "Volume Spike",
  "description": "Daily volume > 3x average",
  "category": "velocity",
  "enabled": true,
  "severity": "MEDIUM",
  "conditions": [
    {"field": "merchant_age_days", "operator": ">", "value": 30},
    {"field": "daily_volume_vnd", "operator": ">", "value": "3 * avg_daily_volume"}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "VOLUME_SPIKE"}}
  ]
}
```

### Rapid Cash-Out Rules

```json
{
  "id": "RAPID_001",
  "name": "Rapid Cash-Out",
  "description": "Payout request within 1 hour of payment confirmation",
  "category": "pattern",
  "enabled": true,
  "severity": "HIGH",
  "conditions": [
    {"field": "time_since_payment_minutes", "operator": "<=", "value": 60},
    {"field": "payout_percentage", "operator": ">=", "value": 80}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "RAPID_CASHOUT"}},
    {"type": "require_review", "params": {"review_type": "enhanced"}}
  ]
}
```

### Round Amount Rules

```json
{
  "id": "ROUND_001",
  "name": "Suspicious Round Amounts",
  "description": "Exact round numbers (1M, 5M, 10M, 50M VND) may indicate suspicious activity",
  "category": "pattern",
  "enabled": true,
  "severity": "LOW",
  "conditions": [
    {"field": "amount_vnd", "operator": "in", "value": [1000000, 5000000, 10000000, 50000000, 100000000]}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "ROUND_AMOUNT"}}
  ]
}
```

### Geographic Rules

```json
{
  "id": "GEO_001",
  "name": "High-Risk Jurisdiction",
  "description": "Payment from FATF high-risk or non-cooperative jurisdiction",
  "category": "pattern",
  "enabled": true,
  "severity": "HIGH",
  "conditions": [
    {"field": "wallet_jurisdiction", "operator": "in", "value": ["KP", "IR", "MM"]}
  ],
  "actions": [
    {"type": "create_alert", "params": {"alert_type": "HIGH_RISK_JURISDICTION"}},
    {"type": "require_review", "params": {"review_type": "enhanced"}}
  ]
}
```

---

## 🧪 Testing Strategy

### Unit Tests

**Risk Scoring**:
```go
func TestCalculateCustomerRiskScore(t *testing.T) {
    tests := []struct {
        name     string
        merchant Merchant
        expected RiskLevel
    }{
        {
            name: "Low risk - established tourism business",
            merchant: Merchant{
                BusinessType: "hotel",
                KYCStatus: "approved",
                Country: "VN",
                MonthlyVolume: decimal.NewFromFloat(50000000), // 50M VND
            },
            expected: RiskLevelLow,
        },
        {
            name: "High risk - new business, high volume",
            merchant: Merchant{
                BusinessType: "other",
                KYCStatus: "pending",
                CreatedAt: time.Now().Add(-7 * 24 * time.Hour), // 7 days old
                MonthlyVolume: decimal.NewFromFloat(500000000), // 500M VND
            },
            expected: RiskLevelHigh,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            score := CalculateCustomerRiskScore(tt.merchant)
            assert.Equal(t, tt.expected, score.RiskLevel)
        })
    }
}
```

**Rule Engine**:
```go
func TestStructuringRule(t *testing.T) {
    rule := GetRule("STRUCT_001")

    // Test: 3 transactions in 24h, each 9M VND
    transactions := []Transaction{
        {AmountVND: decimal.NewFromFloat(9000000), CreatedAt: time.Now()},
        {AmountVND: decimal.NewFromFloat(9200000), CreatedAt: time.Now().Add(-5 * time.Hour)},
        {AmountVND: decimal.NewFromFloat(8800000), CreatedAt: time.Now().Add(-10 * time.Hour)},
    }

    result := rule.Evaluate(transactions)
    assert.True(t, result.Triggered)
    assert.Equal(t, "MEDIUM", result.Severity)
}
```

### Integration Tests

**Sanctions Screening**:
```go
func TestSanctionsScreening_Integration(t *testing.T) {
    // Use test data from actual OFAC list
    testCases := []struct {
        name     string
        expected bool
    }{
        {"Vladimir Putin", true}, // Known sanctioned individual
        {"John Smith", false},    // Common name, not sanctioned
    }

    for _, tc := range testCases {
        hit, err := amlService.ScreenSanctions(tc.name, "", "")
        require.NoError(t, err)
        assert.Equal(t, tc.expected, hit)
    }
}
```

**End-to-End AML Flow**:
```go
func TestE2E_PaymentWithAMLChecks(t *testing.T) {
    // 1. Register merchant
    merchant := createTestMerchant(t)

    // 2. Create payment
    payment := createTestPayment(t, merchant.ID, 9500000) // Just below 10M threshold

    // 3. Simulate blockchain transaction from clean wallet
    confirmPayment(t, payment.ID, "clean_wallet_address")

    // 4. Verify: Payment confirmed, no alerts
    alerts := getAlerts(t, merchant.ID)
    assert.Empty(t, alerts)

    // 5. Create second payment (structuring attempt)
    payment2 := createTestPayment(t, merchant.ID, 9200000)
    confirmPayment(t, payment2.ID, "clean_wallet_address")

    // 6. Create third payment (should trigger structuring alert)
    payment3 := createTestPayment(t, merchant.ID, 9000000)
    confirmPayment(t, payment3.ID, "clean_wallet_address")

    // 7. Verify: Structuring alert created
    alerts = getAlerts(t, merchant.ID)
    assert.Len(t, alerts, 1)
    assert.Equal(t, "STRUCTURING", alerts[0].Type)
    assert.Equal(t, "MEDIUM", alerts[0].Severity)
}
```

---

## 🚀 Deployment

### Environment Variables

```bash
# AML Configuration
AML_ENABLED=true
AML_ALERT_EMAIL=compliance@yourcompany.com

# Sanctions Lists
SANCTIONS_UPDATE_INTERVAL=24h
OFAC_API_URL=https://sanctionslistservice.ofac.treas.gov/api/PublicationPreview/exports/SDN.XML
UN_API_URL=https://www.un.org/securitycouncil/content/un-sc-consolidated-list

# Wallet Screening (choose one)
WALLET_SCREENING_PROVIDER=chainalysis  # or 'trm' or 'elliptic'
CHAINALYSIS_API_KEY=<secret>
TRM_API_KEY=<secret>

# Alert Thresholds
AML_THRESHOLD_VND=10000000
AML_VELOCITY_MULTIPLIER=5
AML_STRUCTURING_COUNT=3
AML_STRUCTURING_WINDOW_HOURS=24

# Performance
AML_CACHE_TTL=168h  # 7 days for wallet screening cache
AML_BATCH_SIZE=100  # for daily risk score updates
```

### Cron Jobs

```bash
# Daily sanctions list update (2 AM UTC)
0 2 * * * /app/bin/aml-updater update-sanctions

# Daily merchant risk score recalculation (3 AM UTC)
0 3 * * * /app/bin/aml-worker recalculate-risk-scores

# Weekly case review reminder (Monday 9 AM UTC)
0 9 * * 1 /app/bin/aml-worker send-case-reminders

# Monthly compliance report generation (1st of month, 10 AM UTC)
0 10 1 * * /app/bin/aml-reporter generate-monthly-report
```

---

## 📈 KPIs & Monitoring

### Alert Metrics

```sql
-- Daily alert summary
SELECT
    date_trunc('day', created_at) as date,
    severity,
    COUNT(*) as alert_count,
    COUNT(CASE WHEN resolution = 'false_positive' THEN 1 END) as false_positives,
    AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600) as avg_resolution_hours
FROM aml_alerts
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY date, severity
ORDER BY date DESC, severity;
```

### False Positive Rate (Target: <20%)

```sql
-- False positive rate by alert type
SELECT
    alert_type,
    COUNT(*) as total_alerts,
    COUNT(CASE WHEN resolution = 'false_positive' THEN 1 END) as false_positives,
    ROUND(100.0 * COUNT(CASE WHEN resolution = 'false_positive' THEN 1 END) / COUNT(*), 2) as fp_rate
FROM aml_alerts
WHERE status = 'resolved'
    AND created_at >= NOW() - INTERVAL '90 days'
GROUP BY alert_type
ORDER BY fp_rate DESC;
```

### Customer Risk Distribution

```sql
-- Current risk distribution
SELECT
    risk_level,
    COUNT(*) as merchant_count,
    ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (), 2) as percentage
FROM aml_customer_risk_scores
WHERE id IN (
    SELECT DISTINCT ON (merchant_id) id
    FROM aml_customer_risk_scores
    ORDER BY merchant_id, created_at DESC
)
GROUP BY risk_level
ORDER BY
    CASE risk_level
        WHEN 'low' THEN 1
        WHEN 'medium' THEN 2
        WHEN 'high' THEN 3
        WHEN 'prohibited' THEN 4
    END;
```

---

## ⚠️ Critical Considerations

### Legal & Compliance

1. **Designate AML Officer**:
   - Required by Vietnam law
   - Must be trained in AML/CFT
   - Responsible for SAR filings

2. **Documentation**:
   - Maintain AML policies and procedures manual
   - Document all rule changes and rationale
   - Keep training records for all staff

3. **Reporting Obligations**:
   - Threshold reports: Monthly to State Bank of Vietnam
   - SAR: Within 12 hours of detection (Vietnam requirement)
   - Keep copies of all reports for 7 years

### Data Privacy

1. **GDPR/Vietnam Personal Data Protection**:
   - Customer data must be encrypted at rest
   - Access controls for AML data (role-based)
   - Audit log all access to sensitive data
   - Data retention: 7 years (regulatory requirement)

2. **PII Handling**:
   - Redact PII from logs
   - Anonymize data for analytics/ML training
   - Secure disposal after retention period

### Security

1. **Sanctions List Updates**:
   - Verify HTTPS for all external API calls
   - Validate XML signatures (OFAC provides signed lists)
   - Implement fallback if update fails (use cached list)

2. **API Key Management**:
   - Rotate wallet screening API keys quarterly
   - Use separate keys for dev/staging/prod
   - Monitor API usage to detect abuse

### Performance

1. **Optimization**:
   - Index all foreign keys in AML tables
   - Cache wallet screening results (7 days)
   - Run heavy analytics queries on read replicas
   - Use connection pooling for database

2. **Scaling**:
   - Rule evaluation can be parallelized (goroutines)
   - Use message queue for async alert processing
   - Consider partitioning `aml_audit_log` by month

---

## 📚 Resources

### FATF Guidance
- [FATF 40 Recommendations](https://www.fatf-gafi.org/en/publications/Fatfrecommendations/Fatf-recommendations.html)
- [FATF Guidance for VASPs](https://www.fatf-gafi.org/en/publications/Fatfrecommendations/Guidance-rba-virtual-assets-2021.html)

### Vietnam Regulations
- [Law on Anti-Money Laundering (2022)](http://vbpl.vn/TW/Pages/vbpqen-toanvan.aspx?ItemID=17157)
- [Decree 74/2023/ND-CP](https://english.luatvietnam.vn/decree-no-74-2023-nd-cp-dated-september-15-2023-of-the-government-detailing-several-articles-of-the-law-on-anti-money-laundering-239938-doc1.html)

### Sanctions Lists (Free)
- [OFAC SDN List (XML)](https://sanctionslistservice.ofac.treas.gov/api/PublicationPreview/exports/SDN.XML)
- [UN Consolidated List](https://www.un.org/securitycouncil/content/un-sc-consolidated-list)
- [EU Sanctions Map](https://www.sanctionsmap.eu/)

### Wallet Screening APIs
- [Chainalysis KYT](https://www.chainalysis.com/solutions/kyt/) - Industry standard, expensive
- [TRM Labs](https://www.trmlabs.com/) - Good for crypto, competitive pricing
- [Elliptic Navigator](https://www.elliptic.co/) - Comprehensive, expensive
- [Chainalysis Sanctions Oracle](https://go.chainalysis.com/chainalysis-oracle-docs.html) - On-chain, FREE

### Open Source Tools
- [OFAC Sanctioned Addresses (GitHub)](https://github.com/0xB10C/ofac-sanctioned-digital-currency-addresses)
- [Fuzzy String Matching](https://github.com/xrash/smetrics) - Go library for fuzzy name matching

---

## 🎓 Next Steps

### Immediate Actions

1. **Review with Legal Team**:
   - Confirm Vietnam AML requirements
   - Review draft SAR template
   - Confirm reporting thresholds

2. **Evaluate Wallet Screening Options**:
   - Test Chainalysis free tier (Sanctions Oracle)
   - Compare TRM Labs pricing
   - Budget for wallet screening API

3. **Finalize Rule Configuration**:
   - Review rules with compliance officer
   - Adjust thresholds based on business model
   - Plan phased rollout (start conservative, tune based on false positives)

4. **Begin Implementation**:
   - Start with Phase 1 (database + core services)
   - Integrate with existing payment flow
   - Build admin dashboard for alert management

### Long-Term Roadmap

**Month 1-2**: MVP Launch
- Core AML engine operational
- Manual alert review process
- Basic reporting

**Month 3-6**: Optimization
- Tune rules based on false positive rate
- Implement advanced wallet risk analysis
- Add behavioral analytics

**Month 6-12**: Advanced Features
- Machine learning anomaly detection
- Automated low-risk alert resolution
- Predictive risk scoring

---

**Last Updated**: 2025-11-19
**Document Owner**: Product Team
**Next Review**: 2025-12-01

