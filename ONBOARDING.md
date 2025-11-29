# New Team Member Onboarding Guide

Welcome to the **Stablecoin Payment Gateway** project! This guide will help you get up to speed quickly and start contributing effectively.

---

## 📖 Table of Contents

1.  [Project Overview](#1-project-overview)
2.  [Prerequisites](#2-prerequisites)
3.  [Development Environment Setup](#3-development-environment-setup)
4.  [Codebase Walkthrough](#4-codebase-walkthrough)
5.  [Running the Application](#5-running-the-application)
6.  [Core Workflows](#6-core-workflows)
7.  [Coding Standards & Conventions](#7-coding-standards--conventions)
8.  [Testing](#8-testing)
9.  [Git Workflow](#9-git-workflow)
10. [Key Documentation](#10-key-documentation)
11. [Common Tasks](#11-common-tasks)
12. [Troubleshooting](#12-troubleshooting)
13. [Who to Ask](#13-who-to-ask)

---

## 1. Project Overview

### What is this project?

A compliant, multi-chain stablecoin payment gateway for Vietnam's tourism market. The core flow is:

> **Merchant creates QR → User scans → Pays with Crypto (USDT/USDC) → System confirms → Merchant receives VND settlement**

### Key Business Goals

-   Enable merchants in Da Nang to accept crypto payments.
-   Settle merchant balances in Vietnamese Dong (VND).
-   Ensure full regulatory compliance (KYC/AML).

### Supported Blockchains

| Chain  | Tokens       | Use Case                     |
| :----- | :----------- | :--------------------------- |
| TRON   | USDT (TRC20) | Lowest fees (~$1), high volume |
| Solana | USDT, USDC   | Fastest finality (~13s)      |
| BSC    | USDT, BUSD   | Popular in Southeast Asia    |

---

## 2. Prerequisites

Ensure you have the following installed on your machine:

| Tool             | Version   | Purpose                         |
| :--------------- | :-------- | :------------------------------ |
| **Go**           | 1.24+     | Backend language                |
| **Node.js**      | 20+ LTS   | Frontend (Next.js)              |
| **Docker**       | Latest    | Running databases locally       |
| **PostgreSQL**   | 15+       | Primary database (via Docker)   |
| **Redis**        | 7+        | Caching & job queues (via Docker)|
| **Make**         | Any       | Running project commands        |
| **golang-migrate** | Latest  | Database migrations             |

### Installing Go

```bash
# macOS
brew install go

# Linux (Ubuntu/Debian)
sudo apt update && sudo apt install golang-go

# Verify
go version
```

### Installing Docker

Follow the official guide: [https://docs.docker.com/get-docker/](https://docs.docker.com/get-docker/)

### Installing `golang-migrate`

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

## 3. Development Environment Setup

### Step 1: Clone the Repository

```bash
git clone git@github.com:hxuan190/stable_payment_gateway.git
cd stable_payment_gateway
```

### Step 2: Create Environment File

Create a `.env` file in the project root:

```bash
cp .env.example .env  # If .env.example exists
# OR create manually:
```

```dotenv
# .env - Local Development Configuration

# Environment
ENV=development
VERSION=1.0.0

# API Server
API_PORT=8080
API_HOST=0.0.0.0

# Admin Server
ADMIN_PORT=8081
ADMIN_HOST=0.0.0.0

# Database (PostgreSQL)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=payment_gateway
DB_SSL_MODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis_password
REDIS_DB=0

# JWT Authentication
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRATION_HOURS=24

# Solana (Devnet for development)
SOLANA_RPC_URL=https://api.devnet.solana.com
SOLANA_WS_URL=wss://api.devnet.solana.com
SOLANA_WALLET_ADDRESS=YOUR_DEVNET_WALLET_ADDRESS
SOLANA_WALLET_PRIVATE_KEY=
SOLANA_NETWORK=devnet
SOLANA_USDT_MINT=Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB
SOLANA_USDC_MINT=EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v

# TRON (Shasta Testnet for development)
TRON_RPC_URL=grpc.shasta.trongrid.io:50051
TRON_WALLET_ADDRESS=YOUR_TESTNET_WALLET_ADDRESS
TRON_WALLET_PRIVATE_KEY=
TRON_NETWORK=testnet
TRON_USDT_CONTRACT=TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t

# BSC (Testnet for development)
BSC_RPC_URL=https://data-seed-prebsc-1-s1.binance.org:8545
BSC_WALLET_ADDRESS=YOUR_TESTNET_WALLET_ADDRESS
BSC_WALLET_PRIVATE_KEY=
BSC_NETWORK=testnet
BSC_CHAIN_ID=97

# Storage (MinIO for local development)
STORAGE_PROVIDER=local
STORAGE_BUCKET=kyc-documents
STORAGE_BASE_URL=/uploads

# Email (disable for local dev)
EMAIL_PROVIDER=smtp
SMTP_HOST=localhost
SMTP_PORT=1025
```

### Step 3: Start Infrastructure Services

```bash
# Start PostgreSQL, Redis, and MinIO
docker-compose up -d postgres redis minio

# Verify services are running
docker-compose ps
```

### Step 4: Run Database Migrations

```bash
# Install migration tool (if not already installed)
make install-tools

# Run migrations
make migrate-up

# Verify migration status
make migrate-version
```

### Step 5: Install Go Dependencies

```bash
go mod download
go mod tidy
```

---

## 4. Codebase Walkthrough

### Directory Structure

```
stable_payment_gateway/
├── cmd/                        # Application entry points
│   ├── api/main.go             # REST API server (port 8080)
│   ├── admin/main.go           # Admin API server (port 8081)
│   ├── listener/main.go        # Blockchain transaction listener
│   └── worker/main.go          # Background job processor
│
├── internal/                   # Private application code
│   ├── api/                    # HTTP handlers, middleware, routing
│   ├── config/                 # Configuration loading
│   ├── model/                  # Database models (GORM structs)
│   ├── modules/                # Domain-driven business modules
│   │   ├── payment/            # Payment processing
│   │   ├── merchant/           # Merchant management
│   │   ├── payout/             # Withdrawal processing
│   │   ├── blockchain/         # Chain-specific listeners
│   │   ├── compliance/         # AML/KYC checks
│   │   ├── ledger/             # Double-entry accounting
│   │   └── notification/       # Webhooks, emails
│   ├── pkg/                    # Shared utilities
│   └── shared/                 # Cross-module interfaces & events
│
├── migrations/                 # SQL migration files
├── web/payment-ui/             # Next.js frontend
├── docker-compose.yml          # Local infrastructure
├── Makefile                    # Project commands
└── go.mod                      # Go dependencies
```

### Key Modules (in `internal/modules/`)

| Module       | Purpose                                      |
| :----------- | :------------------------------------------- |
| `payment`    | Creates payments, generates QR, confirms tx  |
| `merchant`   | Merchant registration, KYC, API keys         |
| `payout`     | Merchant withdrawal requests                 |
| `blockchain` | Listens for on-chain transactions            |
| `compliance` | AML screening, Travel Rule, sanctions        |
| `ledger`     | Immutable financial record-keeping           |
| `notification`| Sends webhooks and emails                   |

For detailed module documentation, see [MODULES.md](./MODULES.md).

---

## 5. Running the Application

### Using Make (Recommended)

```bash
# See all available commands
make help

# Run the API server
make run-api

# Run the blockchain listener (in a separate terminal)
make run-listener

# Run the background worker (in a separate terminal)
make run-worker
```

### Using Go Directly

```bash
# API Server
go run ./cmd/api/main.go

# Blockchain Listener
go run ./cmd/listener/main.go

# Background Worker
go run ./cmd/worker/main.go
```

### Access Points

| Service        | URL                         |
| :------------- | :-------------------------- |
| API Server     | `http://localhost:8080`     |
| Admin Server   | `http://localhost:8081`     |
| pgAdmin        | `http://localhost:5050`     |
| MinIO Console  | `http://localhost:9001`     |

---

## 6. Core Workflows

### Payment Flow

```
1. Merchant calls POST /api/v1/payments
2. System generates unique payment address + QR code
3. User scans QR, sends crypto
4. Blockchain Listener detects transaction
5. System confirms payment, updates ledger
6. Webhook sent to merchant
```

### Payout Flow

```
1. Merchant requests payout via API
2. System validates balance
3. Admin approves in admin panel
4. Ops team executes bank transfer
5. Payout marked complete, ledger updated
```

---

## 7. Coding Standards & Conventions

### Go Code Style

-   **Formatting**: Always run `gofmt` or `make fmt`.
-   **Linting**: Run `make lint` before committing.
-   **Error Handling**: Always check and wrap errors.

```go
// Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
result, _ := doSomething()
```

### Money Calculations

**CRITICAL**: Never use `float64` for money. Always use `decimal.Decimal`.

```go
import "github.com/shopspring/decimal"

amount := decimal.NewFromFloat(100.50)
fee := amount.Mul(decimal.NewFromFloat(0.01)) // 1% fee
```

### API Response Format

```json
{
  "data": { ... },
  "error": null,
  "timestamp": "2025-11-28T10:00:00Z"
}
```

---

## 8. Testing

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run with coverage report
make test-coverage
```

### Writing Tests

-   Place tests in `*_test.go` files next to the code.
-   Use `github.com/stretchr/testify` for assertions.
-   Mock external dependencies (database, blockchain RPC).

```go
func TestPaymentService_CreatePayment(t *testing.T) {
    // Arrange
    mockRepo := &MockPaymentRepository{}
    svc := NewPaymentService(mockRepo)

    // Act
    payment, err := svc.CreatePayment(ctx, input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "created", payment.Status)
}
```

---

## 9. Git Workflow

### Branching Strategy

-   `main`: Production-ready code. **Never push directly.**
-   `develop`: Integration branch (if used).
-   `feature/<name>`: New features.
-   `fix/<name>`: Bug fixes.

### Commit Messages

Write clear, descriptive messages:

```
feat: add payment confirmation webhook

- Implement HMAC signature for webhook security
- Add retry logic with exponential backoff
- Log all webhook delivery attempts
```

### Pull Request Process

1.  Create a feature branch: `git checkout -b feature/my-feature`
2.  Make changes and commit.
3.  Push: `git push -u origin feature/my-feature`
4.  Open a Pull Request on GitHub.
5.  Request review from a team member.
6.  Address feedback, then merge.

---

## 10. Key Documentation

Read these documents in order:

| Document                | Description                                 |
| :---------------------- | :------------------------------------------ |
| `README.md`             | Project overview and quick start            |
| `OVERVIEW.md`           | High-level architecture summary             |
| `MODULES.md`            | Detailed module documentation               |
| `ARCHITECTURE.md`       | System design and database schema           |
| `CLAUDE.md`             | Comprehensive technical guide               |
| `AML_ENGINE.md`         | Compliance and AML rules                    |

---

## 11. Common Tasks

### Create a New Database Migration

```bash
make migrate-create NAME=add_new_column_to_payments
# Edit the generated files in migrations/
make migrate-up
```

### Add a New API Endpoint

1.  Define the handler in `internal/modules/<module>/handler/`.
2.  Register the route in `internal/api/router.go`.
3.  Implement service logic in `internal/modules/<module>/service/`.
4.  Write tests.

### Build for Production

```bash
make build
# Binaries are in ./bin/
```

---

## 12. Troubleshooting

### Database Connection Failed

```
Error: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solution**: Ensure PostgreSQL is running.

```bash
docker-compose up -d postgres
docker-compose ps  # Check status
```

### Migration Dirty State

```
Error: Dirty database version X. Fix and force version.
```

**Solution**: Force the migration version after fixing the issue.

```bash
make migrate-force VERSION=X
```

### Redis Connection Failed

**Solution**: Ensure Redis is running and password matches `.env`.

```bash
docker-compose up -d redis
```

---

## 13. Who to Ask

| Topic                  | Contact                |
| :--------------------- | :--------------------- |
| Architecture & Design  | Tech Lead              |
| Blockchain Integration | Backend Team           |
| Frontend / UI          | Frontend Team          |
| Compliance / AML       | Product Owner          |
| DevOps / Infrastructure| DevOps Engineer        |

---

## ✅ Onboarding Checklist

- [ ] Cloned the repository
- [ ] Created `.env` file
- [ ] Started Docker services (`docker-compose up -d`)
- [ ] Ran database migrations (`make migrate-up`)
- [ ] Successfully ran the API server (`make run-api`)
- [ ] Read `README.md` and `OVERVIEW.md`
- [ ] Read `MODULES.md`
- [ ] Ran the test suite (`make test`)
- [ ] Made a small test commit on a feature branch

---

**Welcome aboard! 🚀**

*Last Updated: 2025-11-28*

