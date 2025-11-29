# Module Documentation

This document provides a comprehensive overview of all modules within the **Stablecoin Payment Gateway** system. The project follows a modular architecture where each module represents a specific business domain.

## 📦 Module Inventory

| Module | Status | Description |
| :--- | :--- | :--- |
| **[Payment](#1-payment-module)** | 🟢 Active | Core payment processing, QR generation, and lifecycle management. |
| **[Merchant](#2-merchant-module)** | 🟢 Active | Merchant onboarding, KYC, settings, and API key management. |
| **[Payout](#3-payout-module)** | 🟢 Active | Withdrawal requests, approval workflows, and bank transfers. |
| **[Blockchain](#4-blockchain-module)** | 🟢 Active | Listeners for Solana, BSC, and TRON networks. |
| **[Compliance](#5-compliance-module)** | 🟢 Active | AML screening, sanctions checks, and Travel Rule enforcement. |
| **[Ledger](#6-ledger-module)** | 🟢 Active | Double-entry accounting system for all financial movements. |
| **[Notification](#7-notification-module)** | 🟢 Active | Multi-channel notifications (Webhook, Email, etc.). |
| **[Infrastructure](#8-infrastructure-module)** | 🟡 Shared | Shared services like archival, exchange rates, and common repositories. |
| **[Identity](#9-identity-module)** | 🟡 New | User identity mapping and wallet ownership verification. |
| **[Audit](#10-audit-module)** | 🟡 New | Comprehensive system-wide audit logging. |
| **[Treasury](#11-treasury-module)** | 🟡 New | Hot/Cold wallet management and funds sweeping. |
| **[OTC](#12-otc-module)** | ⚪ Planned | Liquidity management and crypto-to-fiat conversion. |
| **[User](#13-user-module)** | ⚪ Planned | End-user management (payers/consumers). |
| **[Wallet](#14-wallet-module)** | ⚪ Planned | General wallet balance tracking (distinct from merchant ledger). |
| **[Reconciliation](#15-reconciliation-module)**| ⚪ Planned | Automated reconciliation between ledger, blockchain, and bank. |

---

## 🟢 Core Modules

### 1. Payment Module
**Directory:** `internal/modules/payment/`

The central module responsible for the end-to-end payment flow.

*   **Responsibilities:**
    *   Creates payment requests (`CreatePayment`).
    *   Generates dynamic QR codes for crypto payments.
    *   Validates incoming payment confirmations.
    *   Manages payment expiration and status updates.
    *   Calculates real-time exchange rates (Crypto ↔ VND).
*   **Key Components:**
    *   `Service`: Orchestrates payment logic.
    *   `Handler`: REST API endpoints for payment creation.
    *   `Repository`: PostgreSQL storage for payment records.

### 2. Merchant Module
**Directory:** `internal/modules/merchant/`

Handles all merchant-related operations.

*   **Responsibilities:**
    *   Merchant registration and authentication.
    *   KYC (Know Your Customer) document submission and review.
    *   Management of API keys and webhook secrets.
    *   Merchant profile and settings configuration.
*   **Key Components:**
    *   `Service`: Business logic for merchant accounts.
    *   `Handler`: API endpoints for dashboard and settings.
    *   `Repository`: Stores merchant data and KYC status.

### 3. Payout Module
**Directory:** `internal/modules/payout/`

Manages the outflow of funds from the system to merchants.

*   **Responsibilities:**
    *   Processes payout requests (VND withdrawals).
    *   Enforces approval workflows (e.g., manual admin approval).
    *   Tracks bank transfer status.
    *   Validates merchant balances before processing.
*   **Key Components:**
    *   `Service`: Payout lifecycle management.
    *   `Handler`: Endpoints for requesting and approving payouts.

### 4. Blockchain Module
**Directory:** `internal/modules/blockchain/`

The bridge between the gateway and external blockchain networks.

*   **Responsibilities:**
    *   **Listeners**: Monitors blockchains (Solana, BSC, TRON) for transactions.
    *   **Parsers**: Decodes transaction data (amount, sender, memo).
    *   **Wallet**: Manages hot wallet addresses for receiving payments.
    *   **Broadcaster**: Sends transactions (e.g., for sweeping or refunds).
*   **Sub-modules:**
    *   `solana/`: Solana-specific client and listener.
    *   `bsc/`: Binance Smart Chain client and listener.

### 5. Compliance Module
**Directory:** `internal/modules/compliance/`

Ensures the system adheres to financial regulations.

*   **Responsibilities:**
    *   **AML Screening**: Checks transactions against risk lists (TRM Labs, etc.).
    *   **Travel Rule**: Enforces FATF Travel Rule for crypto transfers.
    *   **Sanctions**: Screens wallet addresses against sanction lists (OFAC).
    *   **Reporting**: Generates reports for regulatory bodies (e.g., SBV).
*   **Key Components:**
    *   `Service`: Runs rule engines and third-party checks.
    *   `Repository`: Stores compliance logs and alerts.

### 6. Ledger Module
**Directory:** `internal/modules/ledger/`

The source of truth for all financial balances.

*   **Responsibilities:**
    *   Implements double-entry bookkeeping principles.
    *   Records every balance change as an immutable ledger entry.
    *   Calculates current merchant balances.
    *   Prevents overdrafts and ensures data integrity.
*   **Key Components:**
    *   `Service`: Transaction recording logic.
    *   `Repository`: Optimized queries for balance calculation.

### 7. Notification Module
**Directory:** `internal/modules/notification/`

Handles communication with external parties.

*   **Responsibilities:**
    *   Sends Webhooks to merchant servers upon payment success.
    *   Sends Emails for account alerts, KYC updates, etc.
    *   Manages notification templates and delivery retries.
*   **Key Components:**
    *   `Service`: Delivery logic and retry mechanism.

---

## 🟡 Support & Infrastructure Modules

### 8. Infrastructure Module
**Directory:** `internal/modules/infrastructure/`

Contains shared technical services and repositories used across multiple domains.

*   **Contents:**
    *   **Archival**: Logic for archiving old data to cold storage (S3/Glacier).
    *   **Exchange Rates**: Fetches and caches crypto/fiat exchange rates.
    *   **Shared Repositories**: Common data access patterns.

### 9. Identity Module
**Directory:** `internal/modules/identity/`

*   **Status**: In Development
*   **Purpose**: Maps blockchain wallet addresses to real-world user identities.
*   **Features**: Wallet ownership verification, user profiling.

### 10. Audit Module
**Directory:** `internal/modules/audit/`

*   **Status**: In Development
*   **Purpose**: Centralized logging of all critical system actions (who did what and when) for security and compliance audits.

### 11. Treasury Module
**Directory:** `internal/modules/treasury/`

*   **Status**: In Development
*   **Purpose**: Manages the platform's liquidity and asset security. Handles automated "sweeping" of funds from temporary reception wallets (hot) to secure storage (cold).

---

## ⚪ Emerging / Planned Modules

These modules primarily contain domain definitions and are part of the roadmap for future expansion.

### 12. OTC (Over-The-Counter) Module
**Directory:** `internal/modules/otc/`
*   **Purpose**: To manage relationships and automated trading with OTC partners for converting received stablecoins into fiat currency (VND) for merchant settlement.

### 13. User Module
**Directory:** `internal/modules/user/`
*   **Purpose**: To manage end-user (payer) accounts, if the platform evolves to offer a consumer wallet app in the future. Currently focuses on `User` domain models.

### 14. Wallet Module
**Directory:** `internal/modules/wallet/`
*   **Purpose**: A generic wallet management domain. Likely to be integrated with `Ledger` or `Treasury` to track system-owned wallet balances across different chains.

### 15. Reconciliation Module
**Directory:** `internal/modules/reconciliation/`
*   **Purpose**: Automated jobs to compare internal Ledger records with Blockchain data and Bank statements to ensure 100% financial accuracy.

