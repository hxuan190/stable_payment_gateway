# Merchant Service

## 1. Overview
The **Merchant Service** is the central registry for all business entities on the platform. It manages the **Merchant Lifecycle**, from initial registration and KYC/KYB submission to account approval, API key issuance, and ongoing compliance monitoring (Tier limits). It also acts as the configuration hub for integration settings like Webhooks.

**Responsibility**:
- **Onboarding**: Handling registration and KYC document submission.
- **Compliance**: Enforcing Tier-based monthly transaction limits.
- **Authentication**: Managing API Keys (`spg_live_...`).
- **Integration**: Configuring Webhook endpoints and secrets.

## 2. Architecture & Flow

The service acts as the gatekeeper for the platform. No merchant can transact without a valid, approved account managed here.

```mermaid
graph TD
    subgraph Onboarding
        Reg[Registration] --> KYC[KYC Submission]
        KYC -->|Pending| Admin[Admin Review]
        Admin -->|Approve| Active[Active Account]
        Admin -->|Reject| Rejected[Rejected]
    end

    subgraph Integration
        Active -->|Generate| API[API Key]
        Active -->|Configure| Hook[Webhook Settings]
    end

    subgraph Runtime Checks
        Tx[Transaction Request] -->|Check Status| Active
        Tx -->|Check Limit| Tier[Tier Limit Check]
        Tier -- Pass --> Allow[Allow Transaction]
        Tier -- Fail --> Block[Block Transaction]
    end
```

### Flow Description
1.  **Registration**: Merchant signs up; status is `Pending`.
2.  **KYC/KYB**: Merchant submits business details (Tax ID, License).
3.  **Approval**: Admin reviews documents. On approval, `KYCStatus` becomes `Approved`.
4.  **Activation**: System generates a secure API Key.
5.  **Runtime**: Every payment checks the merchant's `Status` (must be Active) and `MonthlyLimit` (must have remaining volume).

## 3. Key Components

### Core Interfaces & Structs
-   **`Merchant`** (`domain/merchant.go`): The aggregate root containing profile, KYC status, and configuration.
-   **`MerchantBalance`** (`domain/balance.go`): A read-optimized view of the merchant's financial state (Available vs. Pending vs. Reserved).
-   **`MerchantService`** (`service/merchant_impl.go`): Handles the business logic for state transitions and key management.

### Critical Functions
-   **`RegisterMerchant()`**: Creates the initial record with `Pending` status.
-   **`ApproveKYC()`**: Transitions state to `Approved` and generates the initial API Key.
-   **`IsWithinMonthlyLimit()`**: The gatekeeper function that enforces volume caps.
-   **`RotateAPIKey()`**: Invalidates the old key and issues a new one for security rotation.

## 4. Critical Business Logic

### 🛡️ KYC Tiers & Limits
To manage risk, merchants are assigned a **Tier** based on their verification level.
-   **Tier 1** ($5,000/mo): Basic email/phone verification.
-   **Tier 2** ($50,000/mo): ID Card + Bank Account verification.
-   **Tier 3** ($500,000/mo): Full Business License + Tax ID verification.

**Logic**:
`TotalVolumeThisMonthUSD + TxAmount <= MonthlyLimitUSD`

### 🔑 API Key Security
-   **Format**: `spg_live_<base64_string>` (48 random bytes).
-   **Storage**: Stored in the DB (hashed in future versions, currently raw for MVP).
-   **Rotation**: Merchants can rotate keys if compromised. This immediately invalidates the old key.

### 💰 Balance States
While the **Ledger** is the source of truth, the Merchant Service maintains a **Cached Balance** for UI performance:
-   **Pending**: Funds received on-chain but not yet settled (waiting for confirmations/OTC).
-   **Available**: Funds ready for payout.
-   **Reserved**: Funds locked for an in-flight payout request.

## 5. Database Schema

### `merchants`
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Unique Merchant ID. |
| `email` | VARCHAR | Login email (Unique). |
| `kyc_status` | VARCHAR | `pending`, `approved`, `rejected`. |
| `kyc_tier` | VARCHAR | `tier1`, `tier2`, `tier3`. |
| `monthly_limit_usd` | DECIMAL | Max allowed volume. |
| `api_key` | VARCHAR | The active API key. |
| `webhook_url` | VARCHAR | URL for event notifications. |
| `webhook_secret` | VARCHAR | Secret for HMAC signature. |

### `merchant_balances`
| Column | Type | Description |
| :--- | :--- | :--- |
| `available_vnd` | DECIMAL | Withdrawable funds. |
| `pending_vnd` | DECIMAL | Unsettled funds. |
| `reserved_vnd` | DECIMAL | Locked for payout. |
| `version` | INT | Optimistic locking version. |

## 6. Configuration & Env

| Variable | Description | Example |
| :--- | :--- | :--- |
| `DATABASE_URL` | Postgres connection. | `postgres://user:pass@localhost:5432/db` |
| `ENCRYPTION_KEY` | For encrypting sensitive PII (Tax IDs). | `32_byte_hex_string` |
