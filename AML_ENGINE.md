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

#### aml_travel_rule_data
```sql
-- IVMS101 compliant Travel Rule storage for crypto transactions ≥ $1,000 USD
-- Reference: FATF Recommendation 16 (Wire Transfer Rule for VASPs)
CREATE TABLE aml_travel_rule_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    transaction_amount_usd DECIMAL(20, 2) NOT NULL,
    transaction_threshold_met BOOLEAN DEFAULT TRUE, -- TRUE if ≥ $1,000

    -- Originator Information (Sender - ENCRYPTED PII)
    originator_full_name TEXT NOT NULL, -- AES-256 encrypted
    originator_wallet_address VARCHAR(255) NOT NULL,
    originator_account_number TEXT, -- For VASP accounts (encrypted)
    originator_address_line TEXT, -- Physical address (encrypted)
    originator_country_code VARCHAR(2),
    originator_id_number TEXT, -- National ID/Passport (encrypted)
    originator_id_type VARCHAR(50), -- 'passport', 'national_id', 'drivers_license'
    originator_id_country VARCHAR(2),
    originator_date_of_birth DATE, -- Encrypted

    -- Originator VASP Information
    originator_vasp_name VARCHAR(255), -- e.g., 'Binance', 'Coinbase'
    originator_vasp_identifier VARCHAR(255), -- LEI or registration number
    originator_vasp_country VARCHAR(2),

    -- Beneficiary Information (Receiver - our merchant)
    beneficiary_full_name TEXT NOT NULL, -- AES-256 encrypted (merchant owner)
    beneficiary_wallet_address VARCHAR(255) NOT NULL, -- Our hot wallet
    beneficiary_account_number TEXT, -- Merchant account ID (encrypted)
    beneficiary_address_line TEXT, -- Business address (encrypted)
    beneficiary_country_code VARCHAR(2) DEFAULT 'VN',

    -- Beneficiary VASP (Us)
    beneficiary_vasp_name VARCHAR(255) DEFAULT 'Stablecoin Payment Gateway',
    beneficiary_vasp_identifier VARCHAR(255), -- Our business registration
    beneficiary_vasp_country VARCHAR(2) DEFAULT 'VN',

    -- Transaction Details
    blockchain VARCHAR(20) NOT NULL, -- 'solana', 'tron', 'bsc'
    crypto_currency VARCHAR(10) NOT NULL, -- 'USDT', 'USDC'
    crypto_amount DECIMAL(20, 8) NOT NULL,
    tx_hash VARCHAR(255),

    -- Compliance Fields
    data_source VARCHAR(50) NOT NULL, -- 'kyc_verification', 'vasp_api', 'manual_input'
    verification_status VARCHAR(30) DEFAULT 'pending', -- 'pending', 'verified', 'failed'
    verification_method VARCHAR(50), -- 'sumsub_kyc', 'proof_of_ownership', 'vasp_transfer'
    verified_at TIMESTAMP,
    verified_by UUID REFERENCES admin_users(id),

    -- Data Retention & Privacy
    data_hash VARCHAR(64), -- SHA-256 hash for integrity verification
    encryption_key_version INTEGER DEFAULT 1, -- Track encryption key rotation

    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT check_threshold CHECK (transaction_amount_usd >= 1000)
);

CREATE INDEX idx_travel_payment ON aml_travel_rule_data(payment_id);
CREATE INDEX idx_travel_originator_wallet ON aml_travel_rule_data(originator_wallet_address);
CREATE INDEX idx_travel_threshold ON aml_travel_rule_data(transaction_threshold_met);
CREATE INDEX idx_travel_verification ON aml_travel_rule_data(verification_status);
CREATE INDEX idx_travel_created ON aml_travel_rule_data(created_at);

-- Comments for PII fields requiring encryption
COMMENT ON COLUMN aml_travel_rule_data.originator_full_name IS 'ENCRYPTED: AES-256-GCM at application level';
COMMENT ON COLUMN aml_travel_rule_data.originator_account_number IS 'ENCRYPTED: AES-256-GCM at application level';
COMMENT ON COLUMN aml_travel_rule_data.originator_address_line IS 'ENCRYPTED: AES-256-GCM at application level';
COMMENT ON COLUMN aml_travel_rule_data.originator_id_number IS 'ENCRYPTED: AES-256-GCM at application level';
COMMENT ON COLUMN aml_travel_rule_data.originator_date_of_birth IS 'ENCRYPTED: AES-256-GCM at application level';
COMMENT ON COLUMN aml_travel_rule_data.beneficiary_full_name IS 'ENCRYPTED: AES-256-GCM at application level';
COMMENT ON COLUMN aml_travel_rule_data.beneficiary_account_number IS 'ENCRYPTED: AES-256-GCM at application level';
COMMENT ON COLUMN aml_travel_rule_data.beneficiary_address_line IS 'ENCRYPTED: AES-256-GCM at application level';
```

#### aml_proof_of_ownership
```sql
-- Cryptographic proof that a wallet address belongs to a specific user
-- Critical for unhosted wallet compliance (Phantom, MetaMask, Trust Wallet)
CREATE TABLE aml_proof_of_ownership (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID, -- From wallet_identity_mappings or merchant KYC
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL, -- 'solana', 'tron', 'bsc', 'ethereum'

    -- Cryptographic Proof
    signed_message TEXT NOT NULL, -- Original message user signed
    signature TEXT NOT NULL, -- Cryptographic signature (hex encoded)
    signature_algorithm VARCHAR(50) NOT NULL, -- 'ed25519' (Solana), 'secp256k1' (ETH/BSC)
    public_key TEXT, -- Public key if applicable

    -- Verification
    verification_status VARCHAR(30) NOT NULL DEFAULT 'pending', -- 'pending', 'verified', 'failed', 'expired'
    verified_at TIMESTAMP,
    verified_by VARCHAR(50) DEFAULT 'system', -- 'system' or admin user ID
    verification_method VARCHAR(50) NOT NULL, -- 'signature_verification', 'test_transaction'

    -- Additional Context
    ip_address INET, -- IP when proof was submitted
    user_agent TEXT,
    device_fingerprint TEXT, -- Optional: device identification

    -- Expiration & Renewal
    proof_expires_at TIMESTAMP NOT NULL, -- Proof valid for 1 year
    renewal_count INTEGER DEFAULT 0, -- Track how many times renewed
    last_used_at TIMESTAMP, -- Last time this proof was used for verification

    -- Linking to Transactions
    first_payment_id UUID REFERENCES payments(id), -- First payment with this wallet
    total_payments_count INTEGER DEFAULT 0, -- Track usage

    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    UNIQUE(wallet_address, blockchain)
);

CREATE INDEX idx_proof_wallet ON aml_proof_of_ownership(wallet_address, blockchain);
CREATE INDEX idx_proof_user ON aml_proof_of_ownership(user_id);
CREATE INDEX idx_proof_status ON aml_proof_of_ownership(verification_status);
CREATE INDEX idx_proof_expires ON aml_proof_of_ownership(proof_expires_at);
CREATE INDEX idx_proof_last_used ON aml_proof_of_ownership(last_used_at);

COMMENT ON TABLE aml_proof_of_ownership IS 'Stores cryptographic signatures proving wallet ownership for compliance';
COMMENT ON COLUMN aml_proof_of_ownership.signed_message IS 'Message format: "I own wallet {address} on {blockchain}. Timestamp: {unix_timestamp}"';
COMMENT ON COLUMN aml_proof_of_ownership.signature IS 'Hex-encoded cryptographic signature verifiable on-chain or via SDK';
```

---

## 🔄 Integration with Payment Gateway

### Payment Flow with AML Checks (Updated for Travel Rule + Proof of Ownership)

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
   b. Check if wallet has Proof of Ownership (aml_proof_of_ownership)
      - If YES and verified: Mark as "known user", reduce friction
      - If NO: Flag for identity verification (if > $1,000)
   c. Screen wallet (check cache or query API)
      - If wallet PROHIBITED: Block transaction, create CRITICAL alert
      - If wallet HIGH risk: Create HIGH alert, hold for review
      - If wallet MEDIUM/LOW: Proceed
    ↓
6. TRAVEL RULE CHECK (for transactions ≥ $1,000 USD):
   a. Calculate USD equivalent of payment
   b. If amount_usd >= 1000:
      - Check if originator info exists (from KYC or VASP API)
      - If NOT exists:
          * Trigger identity verification flow
          * Request Proof of Ownership signature
          * Collect originator PII (name, address, ID)
          * Block confirmation until verified
      - If exists:
          * Create aml_travel_rule_data record (ENCRYPT PII)
          * Proceed with confirmation
   c. If amount_usd < 1000: Skip Travel Rule (optional data collection)
    ↓
7. AML TRANSACTION MONITORING:
   a. Create aml_transaction_monitoring record
   b. Run all enabled rules against transaction
   c. Calculate transaction risk score
   d. If rules triggered:
      - Create alerts based on severity
      - If CRITICAL: Block confirmation, require manual approval
      - If HIGH/MEDIUM: Create alert, allow confirmation
      - If LOW: Log only
    ↓
8. Confirm payment (if not blocked)
    ↓
9. Update merchant balance
    ↓
10. Send webhook to merchant
    ↓
┌──────────────────────────────────────────────────────────────┐
│ BACKGROUND PROCESSING (async)                                 │
└──────────────────────────────────────────────────────────────┘
    ↓
11. Update merchant risk score (daily batch)
12. Generate reports (daily/weekly/monthly)
13. Update sanctions lists (daily)
14. Periodic customer reviews (based on risk level)
15. Travel Rule reporting (monthly - transactions ≥ $1,000)
```

### Travel Rule Workflow (≥ $1,000 USD Transactions)

```
┌──────────────────────────────────────────────────────────────┐
│ TRAVEL RULE DATA COLLECTION                                   │
└──────────────────────────────────────────────────────────────┘

USER SCANS QR → SENDS CRYPTO
    ↓
Blockchain listener detects transaction
    ↓
Calculate USD equivalent: crypto_amount × current_rate
    ↓
IF amount_usd >= 1000:
    ↓
    ┌─────────────────────────────────────────┐
    │ SCENARIO 1: Custodial Wallet (Binance)  │
    └─────────────────────────────────────────┘
    ↓
    1. Check if originator_wallet_address is known VASP
       - Query VASP directory (TRP, CipherTrace, etc.)
       - If VASP found: Binance, Coinbase, Kraken, etc.
    ↓
    2. Request Travel Rule data via VASP API (if available)
       - Originator name, address, account number
       - VASP automatically provides data
    ↓
    3. Store in aml_travel_rule_data (ENCRYPTED)
       - data_source: 'vasp_api'
       - verification_method: 'vasp_transfer'
    ↓
    Proceed with payment confirmation

    ┌─────────────────────────────────────────┐
    │ SCENARIO 2: Unhosted Wallet (Phantom)   │
    └─────────────────────────────────────────┘
    ↓
    1. Check aml_proof_of_ownership for this wallet
       - If EXISTS and verified and NOT expired:
           → Retrieve user_id and KYC data
           → Skip identity verification
       - If NOT exists or expired:
           → HOLD payment confirmation
           → Send notification to user: "Identity verification required"
    ↓
    2. Identity Verification Flow:
       a. User receives email/SMS with verification link
       b. User clicks link → redirected to verification page
       c. Request Proof of Ownership signature:
          - Generate challenge: "I own wallet {address} on {blockchain}. Nonce: {random}"
          - User signs message with Phantom wallet
          - Verify signature cryptographically
       d. If signature valid:
          - Collect originator PII:
              * Full name
              * Physical address
              * National ID/Passport number
              * Date of birth
          - Optionally: Face liveness check (Sumsub) for high-value
       e. Store Proof of Ownership (aml_proof_of_ownership)
       f. Store Travel Rule data (aml_travel_rule_data, ENCRYPTED)
    ↓
    3. Verification complete → Resume payment confirmation
    ↓
    Future payments from this wallet → Auto-recognized (cached for 1 year)

ELSE (amount_usd < 1000):
    ↓
    Travel Rule NOT required
    Optional: Still collect Proof of Ownership for future
    Proceed with standard AML checks
```

### Proof of Ownership Verification Flow

```
┌──────────────────────────────────────────────────────────────┐
│ PROOF OF OWNERSHIP - SIGNATURE VERIFICATION                   │
└──────────────────────────────────────────────────────────────┘

1. USER INITIATES VERIFICATION
   ↓
   User clicks "Verify Wallet Ownership" link (from email/notification)
   ↓
2. GENERATE CHALLENGE MESSAGE
   ↓
   Backend generates unique message:
   "I own wallet {wallet_address} on {blockchain}. Timestamp: {unix_timestamp}. Nonce: {random}"

   Example (Solana):
   "I own wallet 7XaHW...xyz on solana. Timestamp: 1700000000. Nonce: a3b9c8d1"
   ↓
3. USER SIGNS MESSAGE
   ↓
   Frontend (Next.js) requests signature from wallet:
   - Phantom: window.solana.signMessage(message)
   - MetaMask: ethereum.request({ method: 'personal_sign', params: [message, address] })
   ↓
   User approves signature in wallet popup
   ↓
   Wallet returns signature (base58/hex encoded)
   ↓
4. BACKEND VERIFIES SIGNATURE
   ↓
   For Solana (Ed25519):
   ```go
   import "github.com/gagliardetto/solana-go"

   func VerifySignature(walletAddress, message, signatureBase58 string) (bool, error) {
       pubKey := solana.MustPublicKeyFromBase58(walletAddress)
       sig := solana.SignatureFromBase58(signatureBase58)

       messageBytes := []byte(message)

       // Verify signature matches public key
       if sig.Verify(pubKey, messageBytes) {
           return true, nil
       }
       return false, nil
   }
   ```

   For Ethereum/BSC (secp256k1):
   ```go
   import "github.com/ethereum/go-ethereum/crypto"

   func VerifyETHSignature(walletAddress, message, signatureHex string) (bool, error) {
       messageHash := crypto.Keccak256Hash([]byte(message))
       signature := hexutil.MustDecode(signatureHex)

       // Recover public key from signature
       pubKey, err := crypto.SigToPub(messageHash.Bytes(), signature)
       if err != nil {
           return false, err
       }

       // Verify recovered address matches claimed address
       recoveredAddr := crypto.PubkeyToAddress(*pubKey)
       return recoveredAddr.Hex() == walletAddress, nil
   }
   ```
   ↓
5. STORE PROOF OF OWNERSHIP
   ↓
   If signature valid:
   - Create record in aml_proof_of_ownership:
       * wallet_address
       * blockchain
       * signed_message (original challenge)
       * signature (hex/base58)
       * signature_algorithm ('ed25519' or 'secp256k1')
       * verification_status: 'verified'
       * proof_expires_at: NOW() + 1 YEAR
   - Link to user_id (from KYC or create new user record)
   ↓
6. COLLECT IDENTITY INFORMATION
   ↓
   After signature verified:
   - Request user to provide:
       * Full legal name
       * Physical address
       * National ID / Passport number
       * Date of birth
   - For transactions ≥ $5,000: Face liveness check (Sumsub)
   ↓
7. ENCRYPT AND STORE TRAVEL RULE DATA
   ↓
   Create aml_travel_rule_data record (ENCRYPT all PII)
   - originator_full_name (ENCRYPTED)
   - originator_address_line (ENCRYPTED)
   - originator_id_number (ENCRYPTED)
   - originator_date_of_birth (ENCRYPTED)
   - data_source: 'proof_of_ownership'
   - verification_method: 'signature_verification'
   ↓
8. VERIFICATION COMPLETE
   ↓
   User notified: "Wallet verified! Future payments will be auto-approved."
   Payment confirmation proceeds

FUTURE PAYMENTS FROM SAME WALLET:
   ↓
   1. Check aml_proof_of_ownership
   2. If exists and NOT expired (< 1 year old):
      - Auto-approve without re-verification
      - Update last_used_at timestamp
   3. If expired (> 1 year):
      - Request signature renewal (same flow)
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

### Data Privacy & PII Encryption

⚠️ **CRITICAL REQUIREMENT**: All PII fields MUST be encrypted at application level before INSERT into database.

#### 1. Application-Level Encryption (AES-256-GCM)

**Mandatory Encrypted Fields**:

**Travel Rule Data (`aml_travel_rule_data`)**:
- `originator_full_name`
- `originator_account_number`
- `originator_address_line`
- `originator_id_number`
- `originator_date_of_birth`
- `beneficiary_full_name`
- `beneficiary_account_number`
- `beneficiary_address_line`

**Customer Risk Scores** (if storing PII in `risk_factors` JSONB):
- Any personally identifiable information in metadata

**Merchants Table** (extend existing schema):
- `owner_full_name`
- `owner_id_number`
- `business_address`
- `phone_number`
- `tax_id`

**Implementation Guide (Golang)**:

```go
// pkg/crypto/encryption.go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
)

type PIIEncryptor struct {
    key []byte // 32 bytes for AES-256
    keyVersion int
}

// NewPIIEncryptor creates encryptor from environment variable
func NewPIIEncryptor() (*PIIEncryptor, error) {
    keyBase64 := os.Getenv("PII_ENCRYPTION_KEY")
    if keyBase64 == "" {
        return nil, errors.New("PII_ENCRYPTION_KEY not set")
    }

    key, err := base64.StdEncoding.DecodeString(keyBase64)
    if err != nil {
        return nil, fmt.Errorf("invalid encryption key: %w", err)
    }

    if len(key) != 32 {
        return nil, errors.New("encryption key must be 32 bytes (AES-256)")
    }

    return &PIIEncryptor{
        key: key,
        keyVersion: 1, // Track for key rotation
    }, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *PIIEncryptor) Encrypt(plaintext string) (string, error) {
    if plaintext == "" {
        return "", nil // Don't encrypt empty strings
    }

    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("failed to create GCM: %w", err)
    }

    // Generate random nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Encrypt and authenticate
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

    // Return base64 encoded (nonce + ciphertext)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *PIIEncryptor) Decrypt(ciphertext string) (string, error) {
    if ciphertext == "" {
        return "", nil
    }

    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", fmt.Errorf("failed to decode ciphertext: %w", err)
    }

    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("failed to create GCM: %w", err)
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt: %w", err)
    }

    return string(plaintext), nil
}

// EncryptDate encrypts date field (store as encrypted string)
func (e *PIIEncryptor) EncryptDate(date time.Time) (string, error) {
    if date.IsZero() {
        return "", nil
    }
    return e.Encrypt(date.Format("2006-01-02"))
}

// DecryptDate decrypts date field
func (e *PIIEncryptor) DecryptDate(encrypted string) (time.Time, error) {
    if encrypted == "" {
        return time.Time{}, nil
    }

    dateStr, err := e.Decrypt(encrypted)
    if err != nil {
        return time.Time{}, err
    }

    return time.Parse("2006-01-02", dateStr)
}
```

**Usage in Repository Layer**:

```go
// internal/repository/travel_rule_repository.go
package repository

import (
    "context"
    "github.com/yourproject/internal/model"
    "github.com/yourproject/pkg/crypto"
)

type TravelRuleRepository struct {
    db        *sql.DB
    encryptor *crypto.PIIEncryptor
}

func NewTravelRuleRepository(db *sql.DB, encryptor *crypto.PIIEncryptor) *TravelRuleRepository {
    return &TravelRuleRepository{
        db:        db,
        encryptor: encryptor,
    }
}

func (r *TravelRuleRepository) Create(ctx context.Context, data *model.TravelRuleData) error {
    // ENCRYPT PII BEFORE INSERT
    encryptedOriginatorName, err := r.encryptor.Encrypt(data.OriginatorFullName)
    if err != nil {
        return fmt.Errorf("failed to encrypt originator name: %w", err)
    }

    encryptedOriginatorAddress, err := r.encryptor.Encrypt(data.OriginatorAddressLine)
    if err != nil {
        return fmt.Errorf("failed to encrypt originator address: %w", err)
    }

    encryptedOriginatorID, err := r.encryptor.Encrypt(data.OriginatorIDNumber)
    if err != nil {
        return fmt.Errorf("failed to encrypt originator ID: %w", err)
    }

    encryptedDOB, err := r.encryptor.EncryptDate(data.OriginatorDateOfBirth)
    if err != nil {
        return fmt.Errorf("failed to encrypt DOB: %w", err)
    }

    encryptedBeneficiaryName, err := r.encryptor.Encrypt(data.BeneficiaryFullName)
    if err != nil {
        return fmt.Errorf("failed to encrypt beneficiary name: %w", err)
    }

    // Compute SHA-256 hash for integrity
    dataHash := computeDataHash(data)

    query := `
        INSERT INTO aml_travel_rule_data (
            payment_id, transaction_amount_usd, transaction_threshold_met,
            originator_full_name, originator_wallet_address,
            originator_address_line, originator_id_number, originator_date_of_birth,
            beneficiary_full_name, beneficiary_wallet_address,
            blockchain, crypto_currency, crypto_amount, tx_hash,
            data_source, verification_method, data_hash, encryption_key_version
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
        )
    `

    _, err = r.db.ExecContext(ctx, query,
        data.PaymentID,
        data.TransactionAmountUSD,
        data.TransactionAmountUSD.Cmp(decimal.NewFromInt(1000)) >= 0,
        encryptedOriginatorName,           // ENCRYPTED
        data.OriginatorWalletAddress,      // NOT encrypted (needed for queries)
        encryptedOriginatorAddress,        // ENCRYPTED
        encryptedOriginatorID,             // ENCRYPTED
        encryptedDOB,                      // ENCRYPTED
        encryptedBeneficiaryName,          // ENCRYPTED
        data.BeneficiaryWalletAddress,
        data.Blockchain,
        data.CryptoCurrency,
        data.CryptoAmount,
        data.TxHash,
        data.DataSource,
        data.VerificationMethod,
        dataHash,
        1, // encryption_key_version
    )

    if err != nil {
        return fmt.Errorf("failed to insert travel rule data: %w", err)
    }

    return nil
}

func (r *TravelRuleRepository) GetByPaymentID(ctx context.Context, paymentID string) (*model.TravelRuleData, error) {
    var data model.TravelRuleData
    var encryptedFields struct {
        OriginatorFullName    string
        OriginatorAddressLine sql.NullString
        OriginatorIDNumber    sql.NullString
        OriginatorDOB         sql.NullString
        BeneficiaryFullName   string
    }

    query := `
        SELECT
            id, payment_id, transaction_amount_usd,
            originator_full_name, originator_wallet_address,
            originator_address_line, originator_id_number, originator_date_of_birth,
            beneficiary_full_name, beneficiary_wallet_address,
            blockchain, crypto_currency, crypto_amount, tx_hash,
            data_source, verification_status, verification_method
        FROM aml_travel_rule_data
        WHERE payment_id = $1
    `

    err := r.db.QueryRowContext(ctx, query, paymentID).Scan(
        &data.ID, &data.PaymentID, &data.TransactionAmountUSD,
        &encryptedFields.OriginatorFullName,
        &data.OriginatorWalletAddress,
        &encryptedFields.OriginatorAddressLine,
        &encryptedFields.OriginatorIDNumber,
        &encryptedFields.OriginatorDOB,
        &encryptedFields.BeneficiaryFullName,
        &data.BeneficiaryWalletAddress,
        &data.Blockchain,
        &data.CryptoCurrency,
        &data.CryptoAmount,
        &data.TxHash,
        &data.DataSource,
        &data.VerificationStatus,
        &data.VerificationMethod,
    )

    if err != nil {
        return nil, err
    }

    // DECRYPT PII AFTER SELECT
    data.OriginatorFullName, err = r.encryptor.Decrypt(encryptedFields.OriginatorFullName)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt originator name: %w", err)
    }

    if encryptedFields.OriginatorAddressLine.Valid {
        data.OriginatorAddressLine, err = r.encryptor.Decrypt(encryptedFields.OriginatorAddressLine.String)
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt address: %w", err)
        }
    }

    if encryptedFields.OriginatorIDNumber.Valid {
        data.OriginatorIDNumber, err = r.encryptor.Decrypt(encryptedFields.OriginatorIDNumber.String)
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt ID number: %w", err)
        }
    }

    if encryptedFields.OriginatorDOB.Valid {
        data.OriginatorDateOfBirth, err = r.encryptor.DecryptDate(encryptedFields.OriginatorDOB.String)
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt DOB: %w", err)
        }
    }

    data.BeneficiaryFullName, err = r.encryptor.Decrypt(encryptedFields.BeneficiaryFullName)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt beneficiary name: %w", err)
    }

    return &data, nil
}
```

#### 2. Environment Configuration

```bash
# .env (NEVER commit to git!)

# PII Encryption Key (AES-256 requires 32 bytes = 44 chars base64)
# Generate: openssl rand -base64 32
PII_ENCRYPTION_KEY=your_base64_encoded_32_byte_key_here

# Key Rotation (for future)
PII_ENCRYPTION_KEY_V2=new_key_for_rotation
PII_ACTIVE_KEY_VERSION=1
```

**Key Generation Script**:
```bash
#!/bin/bash
# scripts/generate_encryption_key.sh
echo "Generating new AES-256 encryption key..."
openssl rand -base64 32 > encryption_key.txt
echo "Key saved to encryption_key.txt"
echo "Add to .env as: PII_ENCRYPTION_KEY=$(cat encryption_key.txt)"
echo ""
echo "⚠️  CRITICAL: Store this key in a secure vault (AWS KMS, HashiCorp Vault)"
echo "⚠️  NEVER commit this key to git"
echo "⚠️  Losing this key means PERMANENT data loss"
```

#### 3. Key Management Best Practices

**Production Key Storage**:
- **DO NOT** store keys in environment variables in production
- **USE** AWS KMS, HashiCorp Vault, or Google Cloud KMS
- **IMPLEMENT** key rotation policy (every 12 months)
- **BACKUP** encryption keys in secure offline storage

**Key Rotation Strategy**:
```go
// Support for multiple encryption key versions
func (r *TravelRuleRepository) Migrate ToNewKey(ctx context.Context) error {
    // 1. Fetch all records encrypted with old key (version 1)
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, originator_full_name, originator_address_line, ...
        FROM aml_travel_rule_data
        WHERE encryption_key_version = 1
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        // 2. Decrypt with old key
        plaintext, err := oldEncryptor.Decrypt(encryptedData)

        // 3. Re-encrypt with new key
        newCiphertext, err := newEncryptor.Encrypt(plaintext)

        // 4. Update record with new ciphertext and version 2
        _, err = r.db.ExecContext(ctx, `
            UPDATE aml_travel_rule_data
            SET originator_full_name = $1, encryption_key_version = 2
            WHERE id = $2
        `, newCiphertext, id)
    }

    return nil
}
```

#### 4. GDPR/Vietnam Personal Data Protection Compliance

1. **Data Subject Rights**:
   - **Right to Access**: Decrypt and provide PII to user on request
   - **Right to Rectification**: Update encrypted PII with new values
   - **Right to Erasure**: Replace PII with "[REDACTED]" after retention period
   - **Right to Portability**: Export decrypted PII in structured format

2. **Access Controls**:
   - Only compliance officers can decrypt PII
   - Implement role-based access control (RBAC)
   - Audit log all PII decryption operations

```go
// Audit PII access
func (r *TravelRuleRepository) GetByPaymentID(ctx context.Context, paymentID string, actorID string) (*model.TravelRuleData, error) {
    data, err := r.getAndDecrypt(ctx, paymentID)
    if err != nil {
        return nil, err
    }

    // LOG PII ACCESS
    r.auditLog.Log(AuditEntry{
        ActorID: actorID,
        Action: "PII_ACCESS",
        ResourceType: "travel_rule_data",
        ResourceID: data.ID,
        Metadata: map[string]interface{}{
            "payment_id": paymentID,
            "fields_accessed": []string{"originator_full_name", "originator_address"},
        },
        IPAddress: getIPFromContext(ctx),
        Timestamp: time.Now(),
    })

    return data, nil
}
```

3. **Data Retention**:
   - Keep encrypted PII for 7 years (Vietnam AML requirement)
   - After 7 years: Securely delete or pseudonymize
   - Implement automated retention policy

```sql
-- Automated cleanup job (run annually)
UPDATE aml_travel_rule_data
SET
    originator_full_name = '[REDACTED]',
    originator_address_line = '[REDACTED]',
    originator_id_number = '[REDACTED]',
    originator_date_of_birth = NULL,
    beneficiary_full_name = '[REDACTED]',
    beneficiary_address_line = '[REDACTED]'
WHERE created_at < NOW() - INTERVAL '7 years';
```

4. **Logging & Redaction**:
   - **NEVER** log PII in application logs
   - Redact sensitive fields before logging

```go
// Safe logging with PII redaction
log.Infof("Processing travel rule for payment %s, originator wallet: %s (name: [REDACTED])",
    paymentID,
    maskWalletAddress(originatorWallet), // Show only last 4 chars
)

func maskWalletAddress(addr string) string {
    if len(addr) <= 8 {
        return "****"
    }
    return addr[:4] + "****" + addr[len(addr)-4:]
}
```

#### 5. Disaster Recovery

**What if encryption key is lost?**
- ❌ **PII is PERMANENTLY UNRECOVERABLE**
- Critical: Implement key backup strategy
- Store key backups in:
  - AWS KMS (encrypted at rest)
  - Hardware Security Module (HSM)
  - Offline secure storage (printed, in safe)

**Backup Procedure**:
```bash
# Encrypt encryption key with GPG before storing
gpg --encrypt --recipient compliance@yourcompany.com encryption_key.txt
# Store encryption_key.txt.gpg in offline vault
```

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

