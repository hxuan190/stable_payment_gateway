# Velocity Check Implementation

**Date**: 2025-11-20
**Feature**: Basic Transaction Velocity Limiting
**Status**: ✅ IMPLEMENTED

---

## Overview

Implemented a basic velocity check to prevent spam and high-frequency trading/laundering attempts by limiting wallet addresses to **10 transactions per 24 hours**.

## Changes Made

### 1. Repository Layer (`internal/repository/payment.go`)

**Added Method**: `CountByAddressAndWindow(fromAddress string, window time.Duration) (int64, error)`

- **SQL Query**: Counts non-failed payments from a specific wallet address within a time window
- **Performance Note**: Added comment suggesting index creation:
  ```sql
  CREATE INDEX idx_payments_from_address_created ON payments(from_address, created_at DESC);
  ```

**Implementation Details**:
- Excludes failed payments from velocity count
- Uses `time.Now().Add(-window)` for time window calculation
- Filters out soft-deleted records

### 2. Service Layer (`internal/service/aml.go`)

**Added Error**: `ErrVelocityLimitExceeded`
```go
ErrVelocityLimitExceeded = errors.New("velocity limit exceeded: too many transactions from this address")
```

**Added Constant**: `VelocityLimit24H = 10`
- Hardcoded limit for MVP (no database rules yet)
- Can be made configurable in future iterations

**Modified Function**: `ValidateTransaction()`
- **New Step 1**: Velocity check (first line of defense)
- Executes BEFORE external API calls (TRM Labs)
- Non-blocking on velocity check failure (logs error but continues)
- Blocks transaction if `txCount >= VelocityLimit24H`

**Validation Order** (updated):
1. ✅ **Velocity check** (NEW - 10 tx/24h limit)
2. Screen wallet address (TRM Labs API)
3. Record screening result
4. Check if sanctioned
5. Check risk score threshold
6. Check blocked flags

### 3. Interface Update (`internal/service/payment.go`)

**Updated Interface**: `PaymentRepository`
- Added `CountByAddressAndWindow(fromAddress string, window time.Duration) (int64, error)`

### 4. Tests (`internal/service/aml_test.go`)

**Added Mock**: `MockPaymentRepository`
- Full implementation of payment repository interface for testing

**New Tests**:
1. `TestValidateTransaction_VelocityLimitExceeded`
   - Verifies rejection when wallet hits 10 tx limit
   - Confirms TRM Labs is NOT called (short-circuit)

2. `TestValidateTransaction_VelocityLimitBelowThreshold`
   - Verifies normal flow when velocity is acceptable (5 tx in 24h)
   - Confirms all subsequent checks are executed

---

## Security Benefits

| Attack Vector | Protection |
|:--------------|:-----------|
| **Spam Attacks** | ✅ Limits wallet to 10 tx/24h |
| **High-Frequency Trading Abuse** | ✅ Prevents rapid transaction patterns |
| **Basic Money Laundering** | ✅ Detects rapid fund movement |
| **API Cost Optimization** | ✅ Blocks spam before expensive TRM Labs API call |

---

## Future Improvements

1. **Database-Backed Rules** (from audit report):
   - Move `VelocityLimit24H` to `aml_rules` table
   - Support dynamic rule updates without code deployment

2. **Advanced Velocity Checks**:
   - Implement `VEL_001`: Transaction count spike (5x daily average)
   - Implement `VEL_002`: Volume spike (3x daily average)
   - Implement `STRUCT_001`: Structuring detection (smurfing)

3. **Performance Optimization**:
   - Add recommended database index: `(from_address, created_at DESC)`
   - Consider Redis cache for frequent offenders

4. **Configurable Limits**:
   - Per-merchant velocity limits based on KYC tier
   - Per-chain velocity limits (Solana vs BSC vs TRON)

---

## Testing

### Manual Verification
```bash
# Code compiles successfully
go build ./internal/service/...
```

### Unit Tests
```bash
# Run velocity check tests
go test ./internal/service -run TestValidateTransaction_Velocity -v
```

**Expected Results**:
- ✅ Velocity limit exceeded: Transaction rejected with `ErrVelocityLimitExceeded`
- ✅ Below threshold: Transaction proceeds to sanctions screening

---

## Integration Points

### Payment Service (`internal/service/payment.go`)
- Already wraps AML errors: `fmt.Errorf("AML pre-screening failed: %w", err)`
- **HTTP Status Mapping**:
  - `ErrVelocityLimitExceeded` → HTTP 429 (Too Many Requests) or 403 (Forbidden)
  - Not treated as HTTP 500 (Internal Server Error)

### Error Handling Chain
```
CreatePayment()
  → ValidatePaymentCompliance()
    → ValidateTransaction()
      → CountByAddressAndWindow()
        → [Database Query]
      → if count >= 10: return ErrVelocityLimitExceeded
      → [Continue to sanctions check...]
```

---

## Addressing Audit Report Gaps

This implementation addresses **1 of 4 missing AML checks** identified in the audit report:

| Check | Status | Notes |
|:------|:-------|:------|
| ✅ **Basic Velocity** | IMPLEMENTED | 10 tx/24h hardcoded limit |
| ❌ Velocity Spike (5x avg) | PENDING | Requires baseline computation |
| ❌ Structuring Detection | PENDING | Requires multi-tx pattern matching |
| ❌ Rapid Cash-Out | PENDING | Requires payout time comparison |

---

## Commit Message

```
feat: Add velocity check for AML compliance (10 tx/24h limit)

- Repository: Add CountByAddressAndWindow for time-window tx counting
- Service: Add velocity limit check (first AML validation step)
- Service: Add ErrVelocityLimitExceeded error type
- Tests: Add MockPaymentRepository and velocity limit tests
- Interface: Update PaymentRepository interface

This implements basic velocity limiting to prevent spam and high-frequency
abuse. Future work: Migrate to database-backed rule engine.

Addresses: Audit Report AML Gap #1 (basic velocity check)
```

---

**Implementation Complete** ✅
