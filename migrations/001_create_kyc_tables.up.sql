-- Migration: Create KYC tables for comprehensive merchant verification
-- This supports three merchant types: individual, household_business, company

-- Enum types for KYC
CREATE TYPE merchant_type_enum AS ENUM ('individual', 'household_business', 'company');
CREATE TYPE kyc_status_enum AS ENUM ('not_started', 'in_progress', 'pending_review', 'approved', 'rejected');
CREATE TYPE kyc_document_type_enum AS ENUM (
    'cccd_front',           -- CCCD mặt trước
    'cccd_back',            -- CCCD mặt sau
    'selfie',               -- Ảnh chân dung
    'business_license',     -- Giấy phép kinh doanh (HKT)
    'business_registration',-- Giấy đăng ký kinh doanh (Company)
    'business_charter',     -- Điều lệ (Company)
    'director_id',          -- CCCD giám đốc
    'appointment_decision', -- Quyết định bổ nhiệm
    'shop_photo',           -- Ảnh bảng hiệu
    'other'
);
CREATE TYPE verification_status_enum AS ENUM ('pending', 'success', 'failed', 'skipped');
CREATE TYPE risk_level_enum AS ENUM ('low', 'medium', 'high', 'critical');

-- Update merchants table to include merchant type
ALTER TABLE merchants
    ADD COLUMN merchant_type merchant_type_enum DEFAULT 'individual',
    ADD COLUMN full_name VARCHAR(255),           -- For individuals
    ADD COLUMN phone VARCHAR(20),
    ADD COLUMN bank_account_number VARCHAR(50),
    ADD COLUMN bank_account_name VARCHAR(255),
    ADD COLUMN bank_name VARCHAR(100),
    ADD COLUMN daily_limit_vnd DECIMAL(15, 2) DEFAULT 200000000,   -- Default limits by type
    ADD COLUMN monthly_limit_vnd DECIMAL(15, 2) DEFAULT 3000000000;

-- Create index on merchant type
CREATE INDEX idx_merchants_type ON merchants(merchant_type);
CREATE INDEX idx_merchants_kyc_status ON merchants(kyc_status);

-- KYC Submissions table
-- Tracks the overall KYC submission for a merchant
CREATE TABLE kyc_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    merchant_type merchant_type_enum NOT NULL,

    -- Status tracking
    status kyc_status_enum DEFAULT 'in_progress',
    submitted_at TIMESTAMP,
    reviewed_at TIMESTAMP,
    approved_at TIMESTAMP,
    rejected_at TIMESTAMP,

    -- Individual-specific fields
    individual_full_name VARCHAR(255),
    individual_cccd_number VARCHAR(20),
    individual_date_of_birth DATE,
    individual_address TEXT,

    -- Household business-specific fields
    hkt_business_name VARCHAR(255),
    hkt_tax_id VARCHAR(50),
    hkt_business_address TEXT,
    hkt_owner_name VARCHAR(255),
    hkt_owner_cccd VARCHAR(20),

    -- Company-specific fields
    company_legal_name VARCHAR(255),
    company_registration_number VARCHAR(50),
    company_tax_id VARCHAR(50),
    company_headquarters_address TEXT,
    company_website VARCHAR(500),
    company_director_name VARCHAR(255),
    company_director_cccd VARCHAR(20),

    -- Common fields
    business_description TEXT,
    business_website VARCHAR(500),
    business_facebook VARCHAR(500),
    product_category VARCHAR(100),

    -- Risk and verification
    auto_approved BOOLEAN DEFAULT FALSE,
    requires_manual_review BOOLEAN DEFAULT FALSE,
    rejection_reason TEXT,
    admin_notes TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_kyc_submissions_merchant ON kyc_submissions(merchant_id);
CREATE INDEX idx_kyc_submissions_status ON kyc_submissions(status);
CREATE INDEX idx_kyc_submissions_type ON kyc_submissions(merchant_type);

-- KYC Documents table
-- Stores references to uploaded documents (actual files in S3/MinIO)
CREATE TABLE kyc_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kyc_submission_id UUID NOT NULL REFERENCES kyc_submissions(id) ON DELETE CASCADE,
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,

    document_type kyc_document_type_enum NOT NULL,
    file_path VARCHAR(500) NOT NULL,       -- S3/MinIO path
    file_name VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),

    -- Metadata extracted from document
    extracted_data JSONB,  -- OCR results, etc.

    -- Verification status
    verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP,
    verification_method VARCHAR(50),  -- 'ocr', 'manual', 'ai'

    uploaded_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_kyc_documents_submission ON kyc_documents(kyc_submission_id);
CREATE INDEX idx_kyc_documents_merchant ON kyc_documents(merchant_id);
CREATE INDEX idx_kyc_documents_type ON kyc_documents(document_type);

-- KYC Verification Results table
-- Stores results of automated verification checks
CREATE TABLE kyc_verification_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kyc_submission_id UUID NOT NULL REFERENCES kyc_submissions(id) ON DELETE CASCADE,
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,

    verification_type VARCHAR(50) NOT NULL,  -- 'ocr', 'face_match', 'mst_lookup', 'age_check', 'name_match', etc.
    status verification_status_enum DEFAULT 'pending',

    -- Results
    passed BOOLEAN,
    confidence_score DECIMAL(5, 4),  -- 0.0000 to 1.0000
    result_data JSONB,               -- Detailed results
    error_message TEXT,

    -- Timing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_kyc_verification_submission ON kyc_verification_results(kyc_submission_id);
CREATE INDEX idx_kyc_verification_type ON kyc_verification_results(verification_type);
CREATE INDEX idx_kyc_verification_status ON kyc_verification_results(status);

-- KYC Risk Assessment table
-- Stores risk scoring and flagging
CREATE TABLE kyc_risk_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kyc_submission_id UUID NOT NULL REFERENCES kyc_submissions(id) ON DELETE CASCADE,
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,

    -- Overall risk
    risk_level risk_level_enum DEFAULT 'low',
    risk_score INTEGER DEFAULT 0,  -- 0-100

    -- Risk factors
    risk_factors JSONB,  -- Array of detected risk factors
    /*
    Example:
    {
        "high_risk_industry": true,
        "sanctioned_entity": false,
        "suspicious_business_pattern": false,
        "age_below_threshold": false,
        "name_mismatch": false,
        "document_quality_issues": false
    }
    */

    -- Industry risk
    industry_category VARCHAR(100),
    is_high_risk_industry BOOLEAN DEFAULT FALSE,

    -- Sanctions and compliance
    sanctions_checked BOOLEAN DEFAULT FALSE,
    sanctions_hit BOOLEAN DEFAULT FALSE,
    sanctions_details JSONB,

    -- Recommendations
    recommended_action VARCHAR(50),  -- 'auto_approve', 'manual_review', 'reject'
    recommended_limits JSONB,
    /*
    {
        "daily_limit_vnd": 200000000,
        "monthly_limit_vnd": 3000000000
    }
    */

    assessed_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_kyc_risk_submission ON kyc_risk_assessments(kyc_submission_id);
CREATE INDEX idx_kyc_risk_level ON kyc_risk_assessments(risk_level);

-- KYC Review Actions table
-- Tracks manual review actions by admin staff
CREATE TABLE kyc_review_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kyc_submission_id UUID NOT NULL REFERENCES kyc_submissions(id) ON DELETE CASCADE,
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,

    -- Reviewer info
    reviewer_id UUID,  -- References admin user (to be created)
    reviewer_email VARCHAR(255),
    reviewer_name VARCHAR(255),

    -- Action taken
    action VARCHAR(50) NOT NULL,  -- 'approve', 'reject', 'request_more_info', 'flag', 'comment'
    decision VARCHAR(50),          -- 'approved', 'rejected', 'pending'

    -- Details
    notes TEXT,
    internal_notes TEXT,  -- Not visible to merchant
    required_documents JSONB,  -- List of additional docs needed

    -- Limits set (if approved)
    approved_daily_limit_vnd DECIMAL(15, 2),
    approved_monthly_limit_vnd DECIMAL(15, 2),

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_kyc_review_submission ON kyc_review_actions(kyc_submission_id);
CREATE INDEX idx_kyc_review_merchant ON kyc_review_actions(merchant_id);
CREATE INDEX idx_kyc_review_action ON kyc_review_actions(action);

-- Audit log for KYC changes
CREATE TABLE kyc_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kyc_submission_id UUID REFERENCES kyc_submissions(id) ON DELETE SET NULL,
    merchant_id UUID REFERENCES merchants(id) ON DELETE SET NULL,

    actor_type VARCHAR(50),  -- 'system', 'admin', 'merchant'
    actor_id VARCHAR(255),
    actor_email VARCHAR(255),

    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,

    -- Change details
    old_status VARCHAR(50),
    new_status VARCHAR(50),
    changes JSONB,

    -- Request details
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_kyc_audit_submission ON kyc_audit_logs(kyc_submission_id);
CREATE INDEX idx_kyc_audit_merchant ON kyc_audit_logs(merchant_id);
CREATE INDEX idx_kyc_audit_action ON kyc_audit_logs(action);
CREATE INDEX idx_kyc_audit_created ON kyc_audit_logs(created_at);
