# KYC Provider Package

This package provides KYC (Know Your Customer) verification integration for the payment gateway.

## Overview

The Identity Mapping Service (PRD v2.2) requires KYC verification for new wallet addresses. This package implements a provider interface that can be used with different KYC services.

## Providers

### 1. MockKYCProvider (Development/Testing)

A mock implementation for local development and testing.

**Features:**
- No external API calls
- Configurable auto-approval
- Simulated face liveness checks
- Realistic API delays (optional)

**Usage:**

```go
// Create mock provider (auto-approve enabled)
provider := kyc.NewMockKYCProvider(
    true,  // autoApprove
    true,  // autoFaceLiveness
    false, // simulateDelay
)

// Create applicant
applicantID, err := provider.CreateApplicant(ctx, "John Doe", "john@example.com")

// Check status
status, err := provider.GetApplicantStatus(ctx, applicantID)
// Returns: KYCStatusApproved (if autoApprove = true)

// Verify face liveness
passed, score, err := provider.VerifyFaceLiveness(ctx, applicantID)
// Returns: true, 0.95-1.0 (if autoFaceLiveness = true)
```

### 2. SumsubProvider (Production - Not Yet Implemented)

Production KYC provider using Sumsub API.

**Status:** Stub implementation (TODO)

**Features:**
- Face liveness detection
- Document verification
- AML screening
- PEP (Politically Exposed Person) checks

**Configuration:**

```go
config := &kyc.SumsubConfig{
    AppToken:  "your-app-token",
    SecretKey: "your-secret-key",
    BaseURL:   "https://api.sumsub.com", // Optional, defaults to production URL
}

provider, err := kyc.NewSumsubProvider(config)
```

**Environment Variables:**

```bash
SUMSUB_APP_TOKEN=<your-app-token>
SUMSUB_SECRET_KEY=<your-secret-key>
SUMSUB_BASE_URL=https://api.sumsub.com
```

## Interface

All KYC providers must implement the `KYCProvider` interface:

```go
type KYCProvider interface {
    // CreateApplicant creates a new KYC applicant
    CreateApplicant(ctx context.Context, fullName, email string) (applicantID string, err error)

    // GetApplicantStatus retrieves the current KYC status
    GetApplicantStatus(ctx context.Context, applicantID string) (status model.KYCStatus, err error)

    // VerifyFaceLiveness triggers face liveness check
    VerifyFaceLiveness(ctx context.Context, applicantID string) (passed bool, score float64, err error)

    // GetApplicantData retrieves KYC data (name, DOB, nationality, etc.)
    GetApplicantData(ctx context.Context, applicantID string) (data map[string]interface{}, err error)
}
```

## KYC Status Flow

```
pending → in_progress → approved ✓
                     ↘ rejected ✗
```

- `pending`: Initial state, waiting for user to submit documents
- `in_progress`: Documents submitted, under review
- `approved`: KYC passed, user can make payments
- `rejected`: KYC failed, user cannot make payments
- `expired`: KYC expired, needs re-verification

## Integration with Identity Mapping Service

The Identity Mapping Service uses the KYC provider for wallet verification:

```go
// Create identity mapping service with KYC provider
kycProvider := kyc.NewMockKYCProvider(true, true, false)
identityService := service.NewIdentityMappingService(
    walletMappingRepo,
    userRepo,
    cache,
    kycProvider,
)

// When a new wallet is detected
mapping, user, needsKYC, err := identityService.GetOrCreateWalletMapping(
    ctx,
    walletAddress,
    blockchain,
    "John Doe",
    "john@example.com",
)

if needsKYC {
    // Trigger KYC flow
    applicantID, err := kycProvider.CreateApplicant(ctx, "John Doe", "john@example.com")

    // Verify face liveness
    passed, score, err := kycProvider.VerifyFaceLiveness(ctx, applicantID)

    // After KYC approval, create wallet mapping
    mapping, err := identityService.CreateWalletMappingAfterKYC(
        ctx,
        walletAddress,
        blockchain,
        userID,
    )
}
```

## Testing

Run tests:

```bash
go test -v ./internal/pkg/kyc/...
```

## Future Providers

Other KYC providers can be added:
- Onfido
- Jumio
- Veriff
- Persona
- Manual KYC (admin review)

Each provider should implement the `KYCProvider` interface.

## Security Considerations

- **NEVER** log sensitive KYC data (passport numbers, photos, etc.)
- Store KYC documents encrypted (S3 with AES-256)
- Redact PII from application logs
- Use HTTPS for all KYC API calls
- Implement rate limiting to prevent abuse
- Audit all KYC status changes

## Compliance Requirements

As per PRD v2.2:
- Face liveness detection required for new wallets
- KYC data retention: 7 years (Vietnam law)
- PEP and sanction screening required
- Enhanced due diligence for high-risk users
- Transaction limits based on KYC tier

## References

- [Sumsub API Documentation](https://developers.sumsub.com/api-reference/)
- [PRD v2.2 - Identity Mapping](../../../IDENTITY_MAPPING.md)
- [Vietnam KYC Requirements](../../../COMPLIANCE_ENGINE_INTEGRATION.md)
