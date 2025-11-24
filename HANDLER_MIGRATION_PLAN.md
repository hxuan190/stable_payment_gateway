# Handler Migration Plan

**Date**: 2025-11-23
**Status**: In Progress

---

## Current State

### Handlers Already in Modules ✅

| Handler | Module Location | Status |
|---------|----------------|--------|
| Payment | `modules/payment/adapter/http/` | ✅ Complete |
| Merchant | `modules/merchant/handler/http.go` | ✅ Complete |
| Payout | `modules/payout/handler/http.go` | ✅ Complete |

### Handlers Moved to Modules (This PR) ✅

| Handler | Old Location | New Location | Status |
|---------|--------------|--------------|--------|
| KYC | `api/handler/kyc.go` | `modules/merchant/handler/kyc.go` | ✅ Copied |
| Travel Rule | `api/handler/travel_rule.go` | `modules/compliance/handler/` | ✅ Copied |
| Travel Rule Admin | `api/handler/travel_rule_admin.go` | `modules/compliance/handler/` | ✅ Copied |
| SBV Report | `api/handler/sbv_report.go` | `modules/compliance/handler/` | ✅ Copied |
| AML Rules | `api/handler/aml_rules.go` | `modules/compliance/handler/` | ✅ Copied |

### Handlers Remaining in api/handler (Cross-Module)

| Handler | Location | Reason |
|---------|----------|--------|
| Admin | `api/handler/admin.go` | Cross-module admin operations |
| Health | `api/handler/health.go` | Infrastructure health checks |

---

## Handler Duplicates Identified

### Duplicate Handlers (Can Be Removed)

| File | Reason |
|------|--------|
| `api/handler/merchant.go` | ❌ Duplicate of `modules/merchant/handler/http.go` |
| `api/handler/merchant_test.go` | ❌ Duplicate test |
| `api/handler/payout.go` | ❌ Duplicate of `modules/payout/handler/http.go` |
| `api/handler/payout_test.go` | ❌ Duplicate test |
| `api/handler/payment_test.go` | ❌ Payment handler now in `modules/payment/adapter/http/` |

---

## Issues Found

### Issue #1: server.go KYC Handler Constructor Mismatch

**Current code in server.go line 221:**
```go
kycHandler := handler.NewKYCHandler(kycDocumentRepo, merchantRepo, complianceService, s.config.Storage)
```

**Actual NewKYCHandler signature:**
```go
func NewKYCHandler(storageService StorageService, kycDocRepo KYCDocumentRepository) *KYCHandler
```

**Problem**: server.go is calling with 4 parameters, but the constructor only accepts 2.

**Solution**: Need to create a storage service adapter and call with correct parameters:
```go
// Create storage adapter
storageAdapter := storage.NewS3StorageAdapter(s.config.Storage)

// Call with correct parameters
kycHandler := merchanthandler.NewKYCHandler(storageAdapter, kycDocumentRepo)
```

### Issue #2: Handler Imports Need Update

**Current imports in server.go:**
```go
"github.com/hxuan190/stable_payment_gateway/internal/api/handler"
```

**Needed imports:**
```go
merchanthandler "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/handler"
payouthandler "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/handler"
compliancehandler "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/handler"
```

---

## Migration Steps (To Be Completed)

### Step 1: Fix server.go Handler Initialization ⏸️

```go
// Import module handlers
import (
    merchanthandler "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/handler"
    payouthandler "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/handler"
    compliancehandler "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/handler"
)

// Replace old handler initialization with module versions
payoutHandler := payouthandler.NewPayoutHandler(payoutService)

// Fix KYC handler initialization
storageAdapter := storage.NewS3StorageAdapter(s.config.Storage)
kycHandler := merchanthandler.NewKYCHandler(storageAdapter, kycDocumentRepo)

// Add compliance handlers
amlRulesHandler := compliancehandler.NewAMLRulesHandler(amlRuleRepo)
travelRuleHandler := compliancehandler.NewTravelRuleHandler(travelRuleRepo)
```

### Step 2: Update admin_server.go ⏸️

Same updates needed for admin_server.go which also uses KYC handler.

### Step 3: Remove Duplicate Handlers ⏸️

```bash
rm internal/api/handler/merchant.go
rm internal/api/handler/merchant_test.go
rm internal/api/handler/payout.go
rm internal/api/handler/payout_test.go
rm internal/api/handler/payment_test.go
rm internal/api/handler/kyc.go
rm internal/api/handler/travel_rule.go
rm internal/api/handler/travel_rule_admin.go
rm internal/api/handler/sbv_report.go
rm internal/api/handler/aml_rules.go
```

###Step 4: Test Compilation ⏸️

```bash
go build ./cmd/api
go build ./cmd/listener
go build ./cmd/worker
go build ./cmd/admin
```

### Step 5: Run Tests ⏸️

```bash
go test ./...
```

---

## Progress

| Phase | Status | Progress |
|-------|--------|----------|
| **Phase 1: Copy handlers to modules** | ✅ Complete | 100% |
| **Phase 2: Update server.go imports** | ⏸️ Pending | 0% |
| **Phase 3: Remove duplicate handlers** | ⏸️ Pending | 0% |
| **Phase 4: Test & verify** | ⏸️ Pending | 0% |

---

## Files Changed So Far

### Created:
- `internal/modules/compliance/handler/` (directory)
- `internal/modules/merchant/handler/kyc.go`
- `internal/modules/compliance/handler/travel_rule.go`
- `internal/modules/compliance/handler/travel_rule_admin.go`
- `internal/modules/compliance/handler/sbv_report.go`
- `internal/modules/compliance/handler/aml_rules.go`

### To Be Modified:
- `internal/api/server.go` - Update handler imports and initialization
- `internal/api/admin_server.go` - Update KYC handler
- Remove 10 duplicate files from `internal/api/handler/`

---

## Recommendations

### Option A: Complete Migration in This PR
- Fix server.go and admin_server.go
- Remove all duplicates
- Full testing required
- Higher risk but cleaner

### Option B: Incremental Migration (Recommended)
- Commit current progress (handlers copied to modules)
- Create separate PR for server.go updates
- Allows testing of each step
- Lower risk, easier to review

---

**Last Updated**: 2025-11-23
**Next Steps**: Decide on Option A vs Option B, then proceed
