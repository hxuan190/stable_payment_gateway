# Treasury Service

## 1. Overview
The **Treasury Service** is the "Bank Vault" of the platform. It manages the organization's cryptocurrency assets, enforcing strict security protocols like **Hot/Cold Wallet Separation** and **Multi-Signature (Multi-Sig)** authorization. It also handles automated liquidity management (Sweeping) to minimize risk exposure.

**Responsibility**:
- **Asset Custody**: Tracking and managing Hot (Online) and Cold (Offline) wallets.
- **Risk Management**: Automatically sweeping excess funds from Hot to Cold storage.
- **Security**: Enforcing Multi-Sig workflows for high-value or sensitive operations.
- **Auditability**: Immutable logging of every fund movement (Sweep, Settlement, Emergency Withdrawal).

## 2. Architecture & Flow

The service implements a tiered security architecture to protect assets.

```mermaid
graph TD
    subgraph Wallets
        Hot[Hot Wallet (Online)]
        Cold[Cold Wallet (Offline)]
    end

    subgraph Automation
        Monitor[Balance Monitor] -->|Check Threshold| Hot
        Monitor -- Exceeds Limit --> Sweep[Trigger Sweep]
    end

    subgraph Operations
        Sweep -->|Create Op| Op[Treasury Operation]
        Op -->|Broadcast| Chain[Blockchain]
        Chain -->|Confirm| Op
        Op -->|Update Balance| Hot
        Op -->|Update Balance| Cold
    end

    subgraph Multi-Sig
        Manual[Manual Transfer] -->|Create Op| MSOp[Multi-Sig Op]
        MSOp -->|Wait| Pending[Pending Signatures]
        Signer1[Signer A] -->|Sign| Pending
        Signer2[Signer B] -->|Sign| Pending
        Pending -- Threshold Met --> Broadcast[Broadcast to Chain]
    end
```

### Flow Description
1.  **Auto-Sweep**:
    *   The system monitors Hot Wallets.
    *   If `Balance > SweepThreshold` (e.g., $10k), it triggers a **Sweep**.
    *   Excess funds are moved to the designated **Cold Wallet**, leaving a safe buffer (e.g., $5k).
2.  **Multi-Sig**:
    *   Critical operations (e.g., moving funds out of Cold Storage) require `M-of-N` signatures.
    *   The operation stays in `Pending Signatures` until enough authorized keys sign it.

## 3. Key Components

### Core Interfaces & Structs
-   **`TreasuryWallet`** (`domain/treasury.go`): Represents a blockchain wallet with configuration for type (Hot/Cold) and security (Multi-Sig).
-   **`TreasuryOperation`** (`domain/treasury.go`): A record of any asset movement, tracking status, gas fees, and signatures.

### Critical Functions
-   **`NeedsSweep()`**: Determines if a wallet holds too much risk and needs balancing.
-   **`CalculateSweepAmount()`**: Computes the exact amount to move (`Balance - Buffer`).
-   **`AddSignature()`**: Collects cryptographic proofs for Multi-Sig operations and checks if the threshold is met.

## 4. Critical Business Logic

### 🌡️ Hot vs. Cold Storage
-   **Hot Wallets**: Connected to the internet for automated payouts. High risk, low balance.
-   **Cold Wallets**: Offline/Air-gapped. Low risk, high balance.
-   **Logic**: We never keep more than necessary in Hot Wallets.

### 🧹 Auto-Sweep Logic
To minimize loss in case of a key compromise:
`IF HotWallet.Balance > Threshold ($10,000) THEN Move (Balance - Buffer ($5,000)) TO ColdWallet`

### 🔐 Multi-Signature Security
For Cold Wallets or high-value transactions:
-   **Scheme**: `M-of-N` (e.g., 2-of-3).
-   **Process**: The transaction is constructed but not broadcast until `M` valid signatures are collected and appended.

## 5. Database Schema

### `treasury_wallets`
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Unique Wallet ID. |
| `type` | VARCHAR | `hot`, `cold`. |
| `address` | VARCHAR | Public key. |
| `sweep_threshold_usd` | DECIMAL | Trigger for auto-sweep. |
| `is_multisig` | BOOLEAN | Security flag. |

### `treasury_operations`
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Unique Op ID. |
| `type` | VARCHAR | `sweep`, `manual_transfer`. |
| `status` | VARCHAR | `initiated`, `pending_signatures`, `confirmed`. |
| `signatures_collected` | INT | Current count of approvals. |

## 6. Configuration & Env

| Variable | Description | Example |
| :--- | :--- | :--- |
| `DEFAULT_SWEEP_THRESHOLD` | Max hot wallet balance. | `10000` ($10k) |
| `DEFAULT_SWEEP_BUFFER` | Amount to keep hot. | `5000` ($5k) |
| `KMS_KEY_ID` | AWS KMS ID for hot wallet signing. | `arn:aws:kms...` |
