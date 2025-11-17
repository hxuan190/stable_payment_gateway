# Solana RPC Client

This package provides a robust RPC client for interacting with the Solana blockchain.

## Features

- **Connection Management**: Configurable RPC client with timeout support
- **Transaction Queries**: Retrieve transaction details and status
- **Health Checks**: Built-in health checking with retry logic
- **Confirmation Tracking**: Monitor transaction confirmations
- **Balance Queries**: Check SOL balances for any address
- **Slot Information**: Get current slot and recent blockhash

## Installation

```go
import "github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
```

## Usage

### Creating a Client

```go
// Simple creation with URL
client, err := solana.NewClientWithURL("https://api.mainnet-beta.solana.com")
if err != nil {
    log.Fatal(err)
}

// Advanced creation with custom config
client, err := solana.NewClient(solana.ClientConfig{
    RPCURL:  "https://api.mainnet-beta.solana.com",
    Timeout: 60 * time.Second,
})
if err != nil {
    log.Fatal(err)
}
```

### Health Check

```go
ctx := context.Background()

// Simple health check
if err := client.HealthCheck(ctx); err != nil {
    log.Printf("RPC is unhealthy: %v", err)
}

// Health check with retry
if err := client.HealthCheckWithRetry(ctx, 3); err != nil {
    log.Printf("RPC failed after retries: %v", err)
}
```

### Getting Transaction Information

```go
signature := solana.MustSignatureFromBase58("your-signature-here")

// Get transaction details
txInfo, err := client.GetTransaction(ctx, signature)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Transaction slot: %d\n", txInfo.Slot)
fmt.Printf("Is finalized: %v\n", txInfo.IsFinalized)
if txInfo.BlockTime != nil {
    fmt.Printf("Block time: %d\n", *txInfo.BlockTime)
}
```

### Getting Transaction Status

```go
status, err := client.GetSignatureStatus(ctx, signature)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Confirmation status: %s\n", status.ConfirmationStatus)
if status.Err != nil {
    fmt.Printf("Transaction error: %v\n", status.Err)
}
```

### Checking Confirmations

```go
confirmations, err := client.GetConfirmations(ctx, signature)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Confirmations: %d\n", confirmations)
// 0 = processed
// 1 = confirmed
// 32+ = finalized
```

### Waiting for Confirmation

```go
// Wait for transaction to be finalized
err := client.WaitForConfirmation(
    ctx,
    signature,
    rpc.CommitmentFinalized,
    30, // max retries
)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Transaction finalized!")
```

### Getting Balances

```go
address := solana.MustPublicKeyFromBase58("your-address-here")

balance, err := client.GetBalance(ctx, address)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Balance: %s SOL\n", balance.String())
```

### Getting Slot and Blockhash

```go
// Get current slot
slot, err := client.GetSlot(ctx, rpc.CommitmentFinalized)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Current slot: %d\n", slot)

// Get recent blockhash
blockhash, err := client.GetRecentBlockhash(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Recent blockhash: %s\n", blockhash.String())
```

### Getting Version

```go
version, err := client.GetVersion(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Solana version: %s\n", version.SolanaCore)
fmt.Printf("Feature set: %d\n", version.FeatureSet)
```

## Types

### ClientConfig

```go
type ClientConfig struct {
    RPCURL  string        // RPC endpoint URL
    Timeout time.Duration // Timeout for RPC calls (default: 30s)
}
```

### TransactionInfo

```go
type TransactionInfo struct {
    Signature       solana.Signature
    Slot            uint64
    BlockTime       *int64
    Meta            *rpc.TransactionMeta
    Transaction     *solana.Transaction
    Confirmations   uint64
    IsFinalized     bool
    Error           error
}
```

### SignatureStatusResult

```go
type SignatureStatusResult struct {
    Slot               uint64
    Confirmations      *uint64
    Err                interface{}
    ConfirmationStatus rpc.ConfirmationStatusType
}
```

### VersionInfo

```go
type VersionInfo struct {
    SolanaCore string
    FeatureSet int64
}
```

## Error Handling

All methods return descriptive errors that can be checked:

```go
txInfo, err := client.GetTransaction(ctx, signature)
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // Transaction doesn't exist
    } else {
        // Other error
    }
}

// Check for transaction execution errors
if txInfo.Error != nil {
    // Transaction failed on-chain
}
```

## Testing

The package includes comprehensive tests:

```bash
# Run unit tests (fast)
go test -v -short ./internal/blockchain/solana/...

# Run all tests including integration tests (requires devnet connection)
go test -v ./internal/blockchain/solana/...

# Run specific test
go test -v -run TestNewClient ./internal/blockchain/solana/...
```

## Performance

The client includes several performance optimizations:

- Configurable timeouts to prevent hanging
- Context support for cancellation
- Retry logic with exponential backoff
- Efficient connection reuse

## Best Practices

1. **Always use context with timeout** for production code:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

2. **Check both RPC errors and transaction errors**:
   ```go
   txInfo, err := client.GetTransaction(ctx, sig)
   if err != nil {
       // RPC or network error
   }
   if txInfo.Error != nil {
       // Transaction failed on-chain
   }
   ```

3. **Wait for finality** for important transactions:
   ```go
   err := client.WaitForConfirmation(ctx, sig, rpc.CommitmentFinalized, 30)
   ```

4. **Use health checks** before critical operations:
   ```go
   if err := client.HealthCheckWithRetry(ctx, 3); err != nil {
       return fmt.Errorf("RPC unhealthy: %w", err)
   }
   ```

## Network Support

The client works with any Solana network:

- **Devnet**: `https://api.devnet.solana.com`
- **Testnet**: `https://api.testnet.solana.com`
- **Mainnet-Beta**: `https://api.mainnet-beta.solana.com`
- **Custom RPC**: Any Solana RPC endpoint (Helius, QuickNode, etc.)

## Dependencies

- `github.com/gagliardetto/solana-go` - Official Solana Go SDK
- `github.com/shopspring/decimal` - Decimal math for SOL amounts

## License

Part of the Stablecoin Payment Gateway project.
