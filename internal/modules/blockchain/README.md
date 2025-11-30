# Blockchain Service

## 1. Overview
The **Blockchain Service** acts as the bridge between the external blockchain networks (Solana, BSC) and the internal payment system. Its primary responsibility is to **monitor, validate, and ingest on-chain transactions**. It listens for incoming transfers to specific merchant wallets, parses them to extract payment references (Memos), and confirms them against pending payment requests.

**Responsibility**:
- **Real-time Monitoring**: Listens to blockchain nodes for new transactions.
- **Transaction Parsing**: Decodes raw transaction data to extract amounts, senders, and memos.
- **Payment Matching**: Links on-chain transfers to internal `PaymentID`s via memo correlation.
- **Finality Assurance**: Ensures transactions have reached the required confirmation depth before processing.

## 2. Architecture & Flow

The service uses a dual-mode listening strategy (WebSocket + Polling) to ensure no transactions are missed.

```mermaid
graph TD
    subgraph External [Blockchain Networks]
        Node[RPC / WebSocket Node]
    end

    subgraph Service [Blockchain Service]
        L[Transaction Listener]
        P[Parser]
        V[Validator]
        CB[Callback Handler]
    end

    subgraph Internal [Payment System]
        PS[Payment Service]
    end

    Node -->|Stream (WS)| L
    Node -->|Poll (RPC)| L
    L -->|Raw Tx| P
    P -->|Extract Memo & Amount| V
    V -->|Check Supported Token| V
    V -->|Check Finality| V
    V -->|Valid| CB
    CB -->|Confirm Payment| PS
```

### Flow Description
1.  **Listen**: The `Listener` connects to the blockchain node via WebSocket for real-time events and periodically polls via RPC as a fallback.
2.  **Parse**: When a transaction involves a monitored wallet, the `Parser` decodes the instruction data to find the transferred amount and the `Memo` field.
3.  **Validate**: The service checks if the token is supported (e.g., USDC, USDT) and if the transaction has reached `Finalized` status.
4.  **Match**: The extracted `Memo` (Payment ID) is used to trigger a callback to the Payment Service, completing the payment lifecycle.

## 3. Key Components

### Core Interfaces & Structs
-   **`Module`** (`module.go`): The entry point that initializes listeners for all configured chains (Solana, BSC).
-   **`TransactionListener`** (`listener.go`): Manages the lifecycle of blockchain connections. It handles connection retries, WebSocket subscriptions, and polling intervals.
-   **`Client`** (`client.go`): A high-level wrapper around the raw RPC client. It provides methods for fetching transaction details, checking health, and verifying finality.
-   **`BlockchainTransaction`** (`domain/transaction.go`): The standardized internal representation of a blockchain transaction, regardless of the underlying chain.

### Critical Functions
-   **`Start()`**: Initializes the WebSocket connection and starts the polling goroutine.
-   **`handleTransaction(signature)`**: The core processing pipeline. It fetches full transaction details, verifies status, parses the payload, and triggers the callback.
-   **`parsePaymentTransaction(txInfo)`**: Extracts the "Who, What, Where" from the raw blockchain byte stream. It specifically looks for SPL Token transfers and Memo instructions.
-   **`WaitForConfirmation()`**: A blocking function that polls the chain until a transaction reaches the `Finalized` commitment level.

## 4. Critical Business Logic

### 🔒 Dual-Mode Reliability
To prevent missing transactions due to WebSocket disconnections, the service implements a **"Race to Process"** pattern:
-   **WebSocket** pushes events immediately for low latency.
-   **Poller** runs every X seconds to fetch recent history.
-   Both feed into a deduplicated processing pipeline. If WS fails, Polling catches up.

### 📝 Memo-Based Matching
The "Secret Sauce" for payment reconciliation is the **Memo Field**.
-   We do not generate unique deposit addresses for every user (which is expensive and hard to manage).
-   Instead, we use a **Single Wallet Architecture** with unique `Memo` identifiers.
-   The logic: `Transaction Memo == Payment ID`. If the Memo matches a pending payment, it is processed.

### 🛡️ Finality & Safety
-   **Solana**: We wait for `commitment: "finalized"` (approx. 32+ confirmations) before crediting a payment. This prevents "optimistic confirmation" attacks where a block might be rolled back.
-   **Token Filtering**: The service maintains a whitelist of `SupportedTokenMints`. Any transfer of an unknown token (spam/dust) is silently ignored.

## 5. Database Schema

The service persists transaction data for audit and reconciliation purposes.

### `blockchain_transactions`
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Unique internal ID. |
| `chain` | VARCHAR | Blockchain name (e.g., `solana`, `bsc`). |
| `network` | VARCHAR | Network type (`mainnet`, `testnet`). |
| `tx_hash` | VARCHAR | The on-chain transaction signature. **Unique Index**. |
| `from_address` | VARCHAR | Sender's wallet address. |
| `to_address` | VARCHAR | Receiver's (Merchant) wallet address. |
| `amount` | DECIMAL | The amount transferred (normalized). |
| `currency` | VARCHAR | Token symbol (e.g., `USDC`). |
| `token_mint` | VARCHAR | The contract address of the token. |
| `memo` | VARCHAR | The extracted payment reference. |
| `status` | VARCHAR | `pending`, `confirmed`, `finalized`, `failed`. |
| `confirmations` | INT | Number of block confirmations. |
| `payment_id` | UUID | Linked Payment ID (if matched). |
| `raw_transaction` | JSONB | Full raw transaction data for debugging. |

## 6. Configuration & Env

The service relies on the following environment variables:

| Variable | Description | Example |
| :--- | :--- | :--- |
| `SOLANA_RPC_URL` | HTTP endpoint for Solana Node. | `https://api.mainnet-beta.solana.com` |
| `SOLANA_WS_URL` | WebSocket endpoint for Solana Node. | `wss://api.mainnet-beta.solana.com` |
| `BSC_RPC_URL` | HTTP endpoint for BSC Node. | `https://bsc-dataseed.binance.org` |
| `WALLET_PRIVATE_KEY` | (Optional) For signing outbound txs. | `[REDACTED]` |
| `MONITORED_WALLET_ADDRESS` | The public key to watch. | `EpZe...4D2` |
| `POLL_INTERVAL` | Fallback polling frequency. | `5s` |
