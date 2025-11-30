# Identity Service

## 1. Overview
The **Identity Service** manages the "Smart Identity Mapping" between blockchain wallets and internal users. Its primary responsibility is to **link anonymous wallet addresses to verified identities**, ensuring that every payment is attributable to a KYC-verified user while maintaining a seamless "One-Click" experience for returning users.

**Responsibility**:
- **Wallet Recognition**: Instantly identifying if a wallet belongs to a known user.
- **KYC Orchestration**: Triggering identity verification for new wallets.
- **Identity Caching**: High-performance caching of wallet-to-user links.
- **Privacy**: Securely hashing and storing wallet associations.

## 2. Architecture & Flow

The service uses a **Cache-Aside** pattern to minimize database lookups and external API calls during the critical payment path.

```mermaid
graph TD
    subgraph Payment Request
        Req[Incoming Payment] --> IDS[Identity Service]
    end

    subgraph Identity Service
        IDS -->|1. Check Cache| Redis[(Redis Cache)]
        Redis -- Hit --> Return[Return User Identity]
        Redis -- Miss --> DB[(Database)]
        
        DB -- Found --> Cache[Update Cache (7 Days)]
        Cache --> Return
        
        DB -- Not Found --> KYC[Trigger KYC Flow]
        KYC -->|Verify| Provider[KYC Provider (Sumsub)]
        Provider -->|Success| Create[Create Mapping]
        Create --> Return
    end
```

### Flow Description
1.  **Recognize**: When a payment arrives, the service checks Redis for the `wallet_address`.
2.  **Fast Path**: If cached (Hit), the User ID is returned immediately (< 5ms).
3.  **Slow Path**: If not cached (Miss), it queries the DB. If found, it populates the cache with a **7-day TTL**.
4.  **Onboarding**: If the wallet is unknown, the flow halts, and the user is redirected to the KYC Provider. Once verified, a new `WalletIdentityMapping` is created.

## 3. Key Components

### Core Interfaces & Structs
-   **`IdentityMappingService`** (`service/identity_mapping.go`): The brain of the module. Handles recognition logic, caching strategies, and KYC coordination.
-   **`WalletIdentityMapping`** (`domain/wallet_identity_mapping.go`): The entity representing the link between a `WalletAddress` (on-chain) and a `UserID` (internal).
-   **`KYCProvider`** (`service/identity_mapping.go`): Interface for external verification services (e.g., Sumsub, Onfido).

### Critical Functions
-   **`RecognizeWallet()`**: The high-speed lookup function used by the Payment Service. It handles the Cache -> DB fallback logic.
-   **`GetOrCreateWalletMapping()`**: Idempotent method to handle new user onboarding.
-   **`ComputeWalletHash()`**: Generates a SHA-256 hash of the wallet address for privacy-preserving indexing.

## 4. Critical Business Logic

### 🧠 Smart Identity Mapping
The "Secret Sauce" is the **One-Time KYC** model.
-   We do **not** require users to log in for every payment.
-   Instead, we treat the **Wallet as the Credential**.
-   Once a wallet is linked to a verified User ID, all future payments from that wallet are automatically attributed to that user.

### 🚀 Performance Caching
-   **Strategy**: Cache-Aside with Sliding Expiration.
-   **TTL**: 7 Days.
-   **Logic**: Every time a user makes a payment, their cache TTL is reset. Active users never hit the database.

### 🔒 Privacy & Hashing
-   To protect user privacy, we store a `WalletAddressHash` (SHA-256).
-   This allows us to perform lookups and uniqueness checks without exposing raw wallet addresses in all logs or indexes.

## 5. Database Schema

### `wallet_identity_mappings`
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Unique Mapping ID. |
| `wallet_address` | VARCHAR | The raw blockchain address. |
| `wallet_address_hash` | VARCHAR | SHA-256 Hash for indexing. |
| `blockchain` | VARCHAR | Chain identifier (e.g., `solana`, `ethereum`). |
| `user_id` | UUID | Foreign Key to `users` table. |
| `kyc_status` | VARCHAR | `pending`, `approved`, `rejected`. |
| `last_seen_at` | TIMESTAMP | Used for cache eviction logic. |
| `is_flagged` | BOOLEAN | If true, auto-rejects payments. |

## 6. Configuration & Env

| Variable | Description | Example |
| :--- | :--- | :--- |
| `REDIS_URL` | Connection string for identity caching. | `redis://localhost:6379` |
| `KYC_PROVIDER` | The active provider. | `sumsub` |
| `KYC_API_KEY` | Credentials for the provider. | `app_token_...` |
| `CACHE_TTL_DAYS` | Duration for wallet mapping cache. | `7` |
