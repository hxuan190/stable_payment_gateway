# KYC Implementation Documentation

## Overview

This document describes the comprehensive KYC (Know Your Customer) implementation for the Stablecoin Payment Gateway, supporting three merchant types with automated verification and risk assessment.

**Last Updated**: 2025-11-16

---

## Table of Contents

1. [Merchant Types](#merchant-types)
2. [Architecture](#architecture)
3. [Database Schema](#database-schema)
4. [API Endpoints](#api-endpoints)
5. [Verification Flows](#verification-flows)
6. [Auto-Approval Logic](#auto-approval-logic)
7. [Risk Assessment](#risk-assessment)
8. [Admin Review Workflow](#admin-review-workflow)
9. [Implementation Guide](#implementation-guide)
10. [Testing](#testing)

---

## Merchant Types

The system supports three distinct merchant types, each with different verification requirements and transaction limits:

### 1. Individual Merchant (Cá nhân)

**Target**: Developers, freelancers, small sellers, digital service providers

**Required Information**:
- Full name (matching CCCD)
- Email
- Phone number
- Bank account (name must match declared name)
- CCCD number
- Date of birth
- Address

**Required Documents**:
- CCCD (front)
- CCCD (back)
- Selfie (for face matching)

**Auto-Verification Checks**:
- OCR extraction from CCCD
- Face match (selfie ↔ CCCD photo)
- Age verification (≥18 years)
- Name consistency check
- Sanctions screening

**Default Limits**:
- Daily: 200,000,000 VND (200M)
- Monthly: 3,000,000,000 VND (3B)

**Auto-Approval Rate**: Target 80%

---

### 2. Household Business (Hộ Kinh Doanh - HKT)

**Target**: Small shops, offline service providers, retail stores

**Required Information**:
- Business name
- Tax ID (Mã số thuế)
- Business address
- Owner name (must match CCCD)
- Owner CCCD number
- Bank account (in HKT name)

**Required Documents**:
- Owner CCCD (front & back)
- Selfie
- Business license (Giấy phép kinh doanh)
- Shop photo (bảng hiệu)
- Facebook/website (optional)

**Auto-Verification Checks**:
- All individual checks
- MST lookup via Tổng Cục Thuế API
- Business name verification
- Owner name matching

**Default Limits**:
- Daily: 500,000,000 VND (500M)
- Monthly: 10,000,000,000 VND (10B)

**Auto-Approval Rate**: Target 60%

---

### 3. Company Merchant (Doanh Nghiệp)

**Target**: Large apps, game studios, top-up platforms, fintech integrations

**Required Information**:
- Legal name
- Company registration number
- Tax ID
- Headquarters address
- Website
- Director name & CCCD
- Bank account (company account)

**Required Documents**:
- Business registration certificate (Giấy đăng ký kinh doanh)
- Company charter (Điều lệ)
- Director CCCD
- Appointment decision (if applicable)

**Auto-Verification Checks**:
- All previous checks
- Business registration lookup (Cổng thông tin QG)
- Legal entity validation
- AML screening
- Beneficial owner check

**Default Limits**:
- Daily: 1,000,000,000 VND (1B)
- Monthly: 30,000,000,000 VND (30B)
- Can be higher with partnership agreements

**Auto-Approval Rate**: Target 40% (more require manual review)

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Merchant API Layer                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  POST /api/v1/kyc/submissions                       │   │
│  │  GET  /api/v1/kyc/status                            │   │
│  │  POST /api/v1/kyc/documents                         │   │
│  │  POST /api/v1/kyc/submit                            │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      KYC Service Layer                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Submission   │  │ Document     │  │ Auto-        │     │
│  │ Management   │  │ Upload       │  │ Approval     │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Verification │  │ Risk         │  │ Audit        │
│ Service      │  │ Assessment   │  │ Service      │
│              │  │ Service      │  │              │
│ - OCR        │  │ - Scoring    │  │ - Logging    │
│ - Face Match │  │ - Industry   │  │ - Tracking   │
│ - MST Lookup │  │ - Sanctions  │  │              │
│ - AML Check  │  │ - Limits     │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Repository Layer                           │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  KYC Repository (PostgreSQL)                         │  │
│  │  - Submissions  - Documents  - Verification Results  │  │
│  │  - Risk Assess  - Reviews    - Audit Logs          │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Storage Layer                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ PostgreSQL   │  │ S3/MinIO     │  │ Redis        │     │
│  │ (KYC data)   │  │ (Documents)  │  │ (Cache)      │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Admin Panel                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  GET  /api/v1/admin/kyc/pending                     │   │
│  │  GET  /api/v1/admin/kyc/submissions/:id             │   │
│  │  POST /api/v1/admin/kyc/submissions/:id/approve     │   │
│  │  POST /api/v1/admin/kyc/submissions/:id/reject      │   │
│  │  POST /api/v1/admin/kyc/submissions/:id/request-info│   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Database Schema

### Core Tables

**kyc_submissions**
- Stores main KYC submission data for each merchant
- Supports all three merchant types (polymorphic)
- Tracks status lifecycle: `in_progress` → `pending_review` → `approved` / `rejected`

**kyc_documents**
- Stores references to uploaded documents (files in S3/MinIO)
- Supports multiple document types per submission
- Tracks verification status per document

**kyc_verification_results**
- Stores results from automated verification checks
- Each verification type gets its own record
- Includes confidence scores and detailed results (JSONB)

**kyc_risk_assessments**
- Stores risk scoring and categorization
- Includes industry risk, sanctions check results
- Provides recommended actions (auto_approve, manual_review, reject)

**kyc_review_actions**
- Tracks all manual review actions by admin staff
- Audit trail for approvals, rejections, info requests
- Stores reviewer identity and decision reasoning

**kyc_audit_logs**
- Comprehensive audit trail for all KYC operations
- Immutable log (append-only)
- Required for compliance (7-year retention)

### Relationships

```sql
merchants (1) ←→ (0..n) kyc_submissions
kyc_submissions (1) ←→ (0..n) kyc_documents
kyc_submissions (1) ←→ (0..n) kyc_verification_results
kyc_submissions (1) ←→ (0..1) kyc_risk_assessments
kyc_submissions (1) ←→ (0..n) kyc_review_actions
```

---

## API Endpoints

### Merchant-Facing Endpoints

#### 1. Create KYC Submission
```http
POST /api/v1/kyc/submissions
Authorization: Bearer {merchant_jwt}
Content-Type: application/json

{
  "merchant_type": "individual",
  "individual_full_name": "Nguyễn Văn A",
  "individual_cccd_number": "001234567890",
  "individual_date_of_birth": "1990-01-15",
  "individual_address": "123 Đường ABC, Quận 1, TP.HCM",
  "product_category": "digital_services"
}
```

**Response**:
```json
{
  "data": {
    "id": "uuid",
    "merchant_id": "uuid",
    "merchant_type": "individual",
    "status": "in_progress",
    "created_at": "2025-11-16T10:00:00Z"
  }
}
```

#### 2. Upload Document
```http
POST /api/v1/kyc/documents
Authorization: Bearer {merchant_jwt}
Content-Type: multipart/form-data

file: [binary]
document_type: "cccd_front"
```

**Response**:
```json
{
  "data": {
    "id": "uuid",
    "document_type": "cccd_front",
    "file_name": "cccd_front.jpg",
    "verified": false,
    "uploaded_at": "2025-11-16T10:05:00Z"
  }
}
```

#### 3. Get KYC Status
```http
GET /api/v1/kyc/status
Authorization: Bearer {merchant_jwt}
```

**Response**:
```json
{
  "data": {
    "submission": { ... },
    "documents": [
      { "document_type": "cccd_front", ... },
      { "document_type": "cccd_back", ... }
    ],
    "required_documents": ["cccd_front", "cccd_back", "selfie"],
    "can_submit": false
  }
}
```

#### 4. Submit for Review
```http
POST /api/v1/kyc/submit
Authorization: Bearer {merchant_jwt}
```

**Response**:
```json
{
  "data": {
    "id": "uuid",
    "status": "approved",  // or "pending_review"
    "auto_approved": true,
    "approved_at": "2025-11-16T10:15:00Z"
  },
  "message": "Congratulations! Your KYC has been automatically approved."
}
```

### Admin Endpoints

#### 1. List Pending Reviews
```http
GET /api/v1/admin/kyc/pending?limit=20&offset=0
Authorization: Bearer {admin_jwt}
```

#### 2. Get Submission Detail
```http
GET /api/v1/admin/kyc/submissions/:id
Authorization: Bearer {admin_jwt}
```

**Response**:
```json
{
  "data": {
    "submission": { ... },
    "documents": [ ... ],
    "verification_results": [
      {
        "verification_type": "ocr",
        "status": "success",
        "passed": true,
        "confidence_score": 0.95
      },
      {
        "verification_type": "face_match",
        "status": "success",
        "passed": true,
        "confidence_score": 0.92
      }
    ],
    "risk_assessment": {
      "risk_level": "low",
      "risk_score": 10,
      "recommended_action": "auto_approve"
    }
  }
}
```

#### 3. Approve KYC
```http
POST /api/v1/admin/kyc/submissions/:id/approve
Authorization: Bearer {admin_jwt}
Content-Type: application/json

{
  "notes": "All documents verified",
  "daily_limit_vnd": 500000000,
  "monthly_limit_vnd": 10000000000
}
```

#### 4. Reject KYC
```http
POST /api/v1/admin/kyc/submissions/:id/reject
Authorization: Bearer {admin_jwt}
Content-Type: application/json

{
  "reason": "Invalid CCCD - photo quality too low",
  "notes": "Please resubmit with clearer photos"
}
```

#### 5. Request More Information
```http
POST /api/v1/admin/kyc/submissions/:id/request-info
Authorization: Bearer {admin_jwt}
Content-Type: application/json

{
  "required_documents": ["business_license", "shop_photo"],
  "notes": "We need proof of business registration"
}
```

---

## Verification Flows

### Individual Merchant Flow

```
1. Merchant creates submission
   ↓
2. Uploads CCCD (front, back)
   ↓
3. Uploads selfie
   ↓
4. Submits for review
   ↓
5. System runs auto-verification:
   ├─ OCR: Extract CCCD data
   ├─ Face Match: Compare selfie ↔ CCCD photo
   ├─ Age Check: Verify age ≥ 18
   ├─ Name Match: Verify name consistency
   └─ Sanctions: Check against sanctions lists
   ↓
6. System assesses risk:
   ├─ Calculate risk score (0-100)
   ├─ Check industry category
   └─ Determine recommended action
   ↓
7. Decision:
   ├─ Low risk + all checks passed → AUTO-APPROVE
   ├─ Medium/High risk → MANUAL REVIEW
   └─ Critical risk / Sanctions hit → REJECT
```

### Household Business Flow

```
Same as Individual +
   ↓
Upload business license
   ↓
Upload shop photo
   ↓
Additional verifications:
   ├─ MST Lookup: Validate tax ID
   ├─ Business Name Match
   └─ Owner verification
   ↓
Risk assessment (higher threshold for auto-approval)
```

### Company Flow

```
Same as above +
   ↓
Upload business registration
Upload company charter
Upload director CCCD
   ↓
Additional verifications:
   ├─ Business Registration Lookup
   ├─ Legal Entity Validation
   ├─ Beneficial Owner Check
   └─ Enhanced AML screening
   ↓
Risk assessment (mostly manual review)
```

---

## Auto-Approval Logic

### Criteria for Auto-Approval

An application is auto-approved if **ALL** of the following are true:

1. **All verification checks passed**
   - OCR: success
   - Face match: confidence ≥ 0.85
   - Age: ≥ 18 years
   - Name match: success
   - Sanctions: no hits

2. **Risk level**: Low or Medium-Low
   - Risk score < 40
   - No high-risk industry flags
   - No suspicious patterns

3. **Merchant type thresholds**:
   - Individual: 80% auto-approval target
   - Household Business: 60% target
   - Company: 40% target

4. **No manual review flags**:
   - No previous rejections
   - No document quality issues
   - No data inconsistencies

### Auto-Approval Rates by Type

| Merchant Type | Target | Typical Reasons for Manual Review |
|---------------|--------|-----------------------------------|
| Individual | 80% | Low-quality photos, age near threshold, unusual patterns |
| Household Business | 60% | MST lookup failures, business name mismatches, high-risk industries |
| Company | 40% | Complex ownership, international connections, large volume expectations |

---

## Risk Assessment

### Risk Scoring (0-100)

**Factors that increase risk score**:

| Factor | Score Impact | Weight |
|--------|--------------|--------|
| Verification failure | +20 per check | High |
| Sanctions hit | +50 | Critical |
| High-risk industry | +30 | High |
| Document quality issues | +20 | Medium |
| Age below threshold | +20 | High |
| Name mismatch | +20 | High |
| Company (vs Individual) | +5 | Low |

**Risk Levels**:
- **Low** (0-19): Auto-approve
- **Medium** (20-39): Manual review (can approve quickly)
- **High** (40-69): Detailed manual review required
- **Critical** (70+): Reject or extreme scrutiny

### High-Risk Industries

Industries flagged for additional scrutiny:
- Gambling / Casino
- Cryptocurrency trading
- OTC desks
- Adult content
- Forex trading
- Money lending
- Pawn shops
- Money exchange

### Recommended Limit Adjustments

Based on risk level, default limits are adjusted:

- **Low risk**: 100% of base limits
- **Medium risk**: 70% of base limits
- **High risk**: 50% of base limits
- **Critical risk**: Minimal limits (50M daily, 500M monthly)

---

## Admin Review Workflow

### Review Queue

Admins see submissions ordered by:
1. Priority (high-risk first)
2. Submission time (oldest first)
3. Merchant type (companies before individuals)

### Review Process

1. **View Submission Details**
   - All merchant information
   - Uploaded documents (with viewer)
   - Verification results
   - Risk assessment

2. **Review Documents**
   - Click to view each document
   - Check quality and authenticity
   - Verify consistency across documents

3. **Make Decision**:

   **Option A: Approve**
   - Set transaction limits (or use defaults)
   - Add approval notes
   - Merchant immediately notified

   **Option B: Reject**
   - Provide rejection reason
   - Add detailed notes
   - Merchant can resubmit after fixing issues

   **Option C: Request More Info**
   - Specify additional documents needed
   - Add clarification notes
   - Submission returns to "in_progress"
   - Merchant uploads requested items
   - Returns to review queue

### Admin Dashboard Metrics

- Total pending reviews
- Average review time
- Auto-approval rate
- Rejection rate
- By merchant type breakdown

---

## Implementation Guide

### Setting Up KYC System

#### 1. Run Database Migrations

```bash
# Apply KYC migrations
psql -U postgres -d payment_gateway -f migrations/001_create_kyc_tables.up.sql
```

#### 2. Configure Environment Variables

```bash
# Storage
S3_BUCKET_NAME=kyc-documents
S3_REGION=ap-southeast-1

# Verification Services (Optional for MVP)
OCR_API_KEY=your_ocr_api_key
FACE_MATCH_API_KEY=your_face_api_key
```

#### 3. Initialize Services

```go
// In main.go or dependency injection setup
import (
    "github.com/hxuan190/stable_payment_gateway/internal/repository"
    "github.com/hxuan190/stable_payment_gateway/internal/service"
    "github.com/hxuan190/stable_payment_gateway/internal/api/handler"
)

// Initialize repository
kycRepo := repository.NewKYCRepository(db)

// Initialize services
verificationService := service.NewVerificationService(
    ocrClient,
    faceMatchClient,
    mstLookupClient,
    sanctionsClient,
)
riskService := service.NewRiskAssessmentService()
auditService := service.NewAuditService(kycRepo)
storageService := service.NewStorageService(s3Backend, "/kyc-documents")

// Initialize KYC service
kycService := service.NewKYCService(
    kycRepo,
    verificationService,
    riskService,
    auditService,
    storageService,
)

// Initialize handlers
kycHandler := handler.NewKYCHandler(kycService)
adminKYCHandler := handler.NewAdminKYCHandler(kycService)
```

#### 4. Register Routes

```go
// Merchant routes
merchant := r.Group("/api/v1/kyc")
merchant.Use(authMiddleware.MerchantAuth())
{
    merchant.POST("/submissions", kycHandler.CreateSubmission)
    merchant.GET("/status", kycHandler.GetSubmissionStatus)
    merchant.POST("/documents", kycHandler.UploadDocument)
    merchant.POST("/submit", kycHandler.SubmitForReview)
    merchant.GET("/detail/:submission_id", kycHandler.GetDetail)
}

// Admin routes
admin := r.Group("/api/v1/admin/kyc")
admin.Use(authMiddleware.AdminAuth())
{
    admin.GET("/pending", adminKYCHandler.ListPendingReviews)
    admin.GET("/list", adminKYCHandler.ListByStatus)
    admin.GET("/submissions/:id", adminKYCHandler.GetSubmissionDetail)
    admin.POST("/submissions/:id/approve", adminKYCHandler.ApproveKYC)
    admin.POST("/submissions/:id/reject", adminKYCHandler.RejectKYC)
    admin.POST("/submissions/:id/request-info", adminKYCHandler.RequestMoreInfo)
    admin.GET("/documents/:id/url", adminKYCHandler.GetDocumentURL)
}
```

---

## Testing

### Unit Tests

Test files to create:
- `internal/service/kyc_service_test.go`
- `internal/service/verification_service_test.go`
- `internal/service/risk_assessment_service_test.go`
- `internal/repository/kyc_repository_test.go`

### Integration Tests

Test scenarios:
1. **Individual merchant happy path** (auto-approved)
2. **Household business with manual review**
3. **Company with rejection**
4. **Request more info workflow**
5. **High-risk industry flagging**
6. **Sanctions hit detection**

### Manual Testing Checklist

- [ ] Create individual merchant KYC
- [ ] Upload all required documents
- [ ] Submit and verify auto-approval
- [ ] Test document upload limits (file size)
- [ ] Test invalid document types
- [ ] Create household business KYC
- [ ] Verify manual review queue
- [ ] Admin: approve a submission
- [ ] Admin: reject a submission
- [ ] Admin: request more info
- [ ] Verify audit logs created
- [ ] Test transaction limits applied

---

## Future Enhancements

### Phase 2 Improvements

1. **Enhanced Verification**
   - Integrate with AWS Rekognition for face matching
   - Use AWS Textract for OCR
   - Integrate with Vietnam Tax Authority API for real-time MST validation
   - Add liveness detection for selfies

2. **Advanced Risk Scoring**
   - Machine learning model for risk prediction
   - Behavioral analytics
   - Transaction pattern analysis
   - Velocity checks

3. **User Experience**
   - Real-time document quality feedback
   - Mobile app for document capture
   - Progress indicators
   - Estimated approval time

4. **Compliance**
   - AML transaction monitoring
   - PEP (Politically Exposed Person) screening
   - Ongoing monitoring and re-verification
   - Compliance reporting dashboard

---

## Support

For questions or issues:
- Check the main `README.md`
- Review `ARCHITECTURE.md` for system design
- Consult `CLAUDE.md` for AI assistant guidance

---

**Document Version**: 1.0
**Last Updated**: 2025-11-16
**Maintained By**: Development Team
