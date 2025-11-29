# Project Overview: Stablecoin Payment Gateway

## Introduction

The **Stablecoin Payment Gateway** is a compliant, multi-chain payment solution designed to enable merchants in Vietnam (initially Da Nang) to accept cryptocurrency payments (USDT, USDC, BUSD) while receiving settlements in local currency (VND). The system is built with a focus on regulatory compliance, security, and scalability, leveraging a modular architecture to handle blockchain interactions, fund management, and merchant services.

## 🏗️ Architecture

The project follows a modular microservices-ready architecture (monorepo) implemented in **Go**. It distinguishes between the core API, blockchain listeners, and background workers.

### System Components

1.  **Core API (`cmd/api`)**:
    -   RESTful API server using the **Gin** framework.
    -   Handles merchant requests, payment creation, user management, and admin operations.
    -   Implements the "Clean Architecture" pattern with distinct layers: `adapters` (HTTP), `modules` (Business Logic), and `ports` (Interfaces).

2.  **Blockchain Listener (`cmd/listener`)**:
    -   A dedicated service that monitors blockchain networks (TRON, Solana, BSC) for incoming transactions to merchant wallets.
    -   Detects payments and updates the internal ledger state.

3.  **Background Worker (`cmd/worker`)**:
    -   Powered by `hibiken/asynq` (Redis-based).
    -   Handles asynchronous tasks such as:
        -   Transaction confirmation polling.
        -   Merchant balance updates.
        -   Treasury sweeps (hot wallet to cold wallet).
        -   Sending notifications (Email, Webhooks).
        -   Running AML/KYC checks.

4.  **Admin Dashboard (`cmd/admin`)**:
    -   Internal tool for operations teams to manage merchants, approve KYC, process payouts, and view audit logs.

5.  **Frontend (`web/payment-ui`)**:
    -   A Next.js + TailwindCSS application providing the payment interface for end-users and the dashboard for merchants.

## 🛠️ Technology Stack

### Backend
-   **Language**: Go (1.24+)
-   **Web Framework**: Gin
-   **Database ORM**: GORM (PostgreSQL) / sqlx
-   **Async Queue**: Asynq (Redis)
-   **Validation**: go-playground/validator
-   **Logging**: Logrus

### Infrastructure & Data
-   **Database**: PostgreSQL (Primary data store)
-   **Cache/Queue**: Redis
-   **Blockchains**:
    -   **Solana** (USDT, USDC) - High speed, low cost.
    -   **TRON** (USDT) - Industry standard for stablecoin transfers.
    -   **BSC** (USDT, BUSD) - Widely used in SEA.
-   **Storage**: AWS S3 / MinIO (Document storage for KYC).

### Compliance & Security
-   **AML Engine**: In-house rule-based engine for transaction monitoring and sanctions screening.
-   **Travel Rule**: FATF compliance implementation for crypto transactions.
-   **Security**: JWT authentication, HMAC webhook signing, AES database encryption.

## 📂 Codebase Structure

```text
stable_payment_gateway/
├── cmd/                    # Application entry points
│   ├── api/                # Main REST API server
│   ├── listener/           # Blockchain event listener
│   ├── worker/             # Asynchronous task worker
│   └── admin/              # Internal admin tools
├── internal/               # Private application code
│   ├── api/                # API handlers and router setup
│   ├── config/             # Configuration loading (env vars)
│   ├── model/              # Database models (GORM structs)
│   ├── modules/            # Business logic modules (Domain Services)
│   │   ├── payment/        # Payment processing logic
│   │   ├── merchant/       # Merchant management
│   │   ├── wallet/         # Blockchain wallet management
│   │   ├── aml/            # Anti-Money Laundering engine
│   │   └── ...
│   ├── pkg/                # Shared libraries and utilities
│   └── worker/             # Worker task definitions
├── migrations/             # SQL database migrations
├── docs/                   # Project documentation
└── web/                    # Frontend source code
```

## 🔑 Key Features

-   **Dynamic QR Generation**: Creates unique payment addresses or memos for each transaction.
-   **Real-time Settlement**: Instantly detects blockchain transactions and credits merchant ledgers.
-   **Merchant Payouts**: Automated flows for merchants to request VND withdrawals to their bank accounts.
-   **Compliance First**: Integrated KYC (Know Your Customer) and AML (Anti-Money Laundering) checks.
-   **Treasury Management**: Automated sweeping of funds from hot wallets to secure cold storage.
-   **Ledger System**: Double-entry ledger ensuring mathematical accuracy of all balances.

## 🚀 Getting Started

For development instructions, please refer to [GETTING_STARTED.md](./GETTING_STARTED.md).
For a detailed roadmap, see [PRD_v2.2_ROADMAP.md](./PRD_v2.2_ROADMAP.md).

