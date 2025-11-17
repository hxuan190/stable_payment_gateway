# Solana Blockchain Integration

This package provides Solana blockchain integration for the payment gateway, including wallet management, transaction listening, and payment confirmation.

## Overview

The Solana integration enables the payment gateway to:
- Accept USDT and USDC payments on Solana blockchain
- Monitor wallet for incoming transactions
- Verify transaction finality
- Sign and send transactions

## Components

### Wallet (`wallet.go`)

The wallet component provides secure wallet operations including:
- Loading wallet from private key
- Querying SOL and SPL token balances
- Signing transactions
- Sending transactions

## Wallet Usage

### Loading a Wallet

```go
import (
    "context"
    "github.com/yourusername/stable_payment_gateway/internal/blockchain/solana"
    "github.com/yourusername/stable_payment_gateway/internal/config"
)

// Load configuration
cfg, err := config.Load()
if err != nil {
    panic(err)
}

// Load wallet from private key
wallet, err := solana.LoadWallet(
    cfg.Solana.WalletPrivateKey,
    cfg.Solana.RPCURL,
)
if err != nil {
    panic(err)
}

// Get wallet address
address := wallet.GetAddress()
fmt.Println("Wallet address:", address)
```

### Checking Balances

```go
ctx := context.Background()

// Get SOL balance
solBalance, err := wallet.GetSOLBalance(ctx)
if err != nil {
    panic(err)
}
fmt.Printf("SOL Balance: %s SOL\n", solBalance.String())

// Get USDT balance
usdtBalance, err := wallet.GetUSDTBalance(ctx, cfg.Solana.USDTMint)
if err != nil {
    panic(err)
}
fmt.Printf("USDT Balance: %s USDT\n", usdtBalance.String())

// Get USDC balance
usdcBalance, err := wallet.GetUSDCBalance(ctx, cfg.Solana.USDCMint)
if err != nil {
    panic(err)
}
fmt.Printf("USDC Balance: %s USDC\n", usdcBalance.String())

// Get comprehensive wallet balance
tokenMints := map[string]string{
    "USDT": cfg.Solana.USDTMint,
    "USDC": cfg.Solana.USDCMint,
}
walletBalance, err := wallet.GetWalletBalance(ctx, tokenMints)
if err != nil {
    panic(err)
}
fmt.Printf("SOL: %s\n", walletBalance.SOL.String())
for symbol, tokenInfo := range walletBalance.Tokens {
    fmt.Printf("%s: %s\n", symbol, tokenInfo.Balance.String())
}
```

### Signing and Sending Transactions

```go
import (
    "github.com/gagliardetto/solana-go"
)

// Create a transfer instruction
recipientAddress := solana.MustPublicKeyFromBase58("RecipientAddressHere...")
lamports := uint64(1_000_000) // 0.001 SOL

instruction := wallet.CreateTransferInstruction(recipientAddress, lamports)

// Create transaction
tx, err := solana.NewTransaction(
    []solana.Instruction{instruction},
    solana.Hash{}, // Will be filled with recent blockhash
    solana.TransactionPayer(wallet.GetPublicKey()),
)
if err != nil {
    panic(err)
}

// Sign and send transaction
signature, err := wallet.SignAndSendTransaction(ctx, tx)
if err != nil {
    panic(err)
}
fmt.Printf("Transaction sent: %s\n", signature.String())

// Wait for finalization
err = wallet.VerifyTransaction(ctx, signature, 30)
if err != nil {
    panic(err)
}
fmt.Println("Transaction finalized!")
```

### Health Check

```go
// Verify wallet connectivity
err := wallet.HealthCheck(ctx)
if err != nil {
    log.Printf("Wallet health check failed: %v", err)
} else {
    log.Println("Wallet is healthy")
}
```

## Security Considerations

### Private Key Management

**CRITICAL SECURITY REQUIREMENTS:**

1. **Never hardcode private keys** - Always load from environment variables or secure vault
2. **Never log private keys** - The `GetPrivateKey()` method should only be used internally
3. **Use secure storage** - In production, use HashiCorp Vault or AWS Secrets Manager
4. **Rotate keys regularly** - Implement key rotation procedures
5. **Minimum balance** - Keep hot wallet balance below $10,000 threshold

### Environment Configuration

```bash
# Required environment variables
SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
SOLANA_WALLET_PRIVATE_KEY=<base58-encoded-private-key>
SOLANA_WALLET_ADDRESS=<public-address>
SOLANA_NETWORK=mainnet
SOLANA_CONFIRMATION_LEVEL=finalized
SOLANA_USDT_MINT=Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB
SOLANA_USDC_MINT=EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v
```

### Testing

The package includes comprehensive unit and integration tests.

#### Running Unit Tests

```bash
# Run all tests
go test ./internal/blockchain/solana/...

# Run only unit tests (skip integration tests)
go test -short ./internal/blockchain/solana/...

# Run with coverage
go test -cover ./internal/blockchain/solana/...

# Run with verbose output
go test -v ./internal/blockchain/solana/...
```

#### Running Integration Tests

Integration tests require network connectivity to Solana devnet:

```bash
# Run all tests including integration tests
go test ./internal/blockchain/solana/...

# Run specific test
go test -run TestGetSOLBalance ./internal/blockchain/solana/
```

#### Benchmarks

```bash
# Run benchmarks
go test -bench=. ./internal/blockchain/solana/

# Run specific benchmark
go test -bench=BenchmarkLoadWallet ./internal/blockchain/solana/
```

## Token Addresses

### Mainnet

- **USDT (SPL)**: `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB`
- **USDC (SPL)**: `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`

### Devnet

- **USDT (SPL)**: `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB`
- **USDC (SPL)**: `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`

*Note: Devnet tokens can be minted for free for testing purposes.*

## RPC Endpoints

### Free Public RPCs

- **Mainnet**: `https://api.mainnet-beta.solana.com`
- **Devnet**: `https://api.devnet.solana.com`
- **Testnet**: `https://api.testnet.solana.com`

### Recommended Paid RPCs (Better reliability and rate limits)

- **Helius**: `https://rpc.helius.xyz/?api-key=<your-api-key>`
- **QuickNode**: `https://your-endpoint.solana-mainnet.quiknode.pro/`
- **Alchemy**: `https://solana-mainnet.g.alchemy.com/v2/<your-api-key>`

**For production, use paid RPC endpoints** to ensure reliability and avoid rate limiting.

## Performance Considerations

### RPC Call Optimization

- **Cache balances**: Don't query balance on every request
- **Batch requests**: Use RPC batch calls when possible
- **Rate limiting**: Implement rate limiting to avoid hitting RPC limits
- **Connection pooling**: Reuse RPC client connections

### Transaction Confirmation

- **Solana finality**: ~400ms to 13 seconds (depends on commitment level)
- **Commitment levels**:
  - `processed`: Fastest (~400ms) but can be rolled back
  - `confirmed`: Fast (~6s) with low risk of rollback
  - `finalized`: Slowest (~13s) but guaranteed final

**For payments, always use `finalized` commitment** to prevent double-spend attacks.

### Gas/Fee Estimation

- Solana transaction fees are typically 0.000005 SOL (~$0.001)
- Very predictable and low compared to Ethereum
- No need for complex gas estimation

## Error Handling

### Common Errors

```go
// Handle wallet loading errors
wallet, err := solana.LoadWallet(privateKey, rpcURL)
if err != nil {
    if strings.Contains(err.Error(), "invalid private key") {
        // Private key format error
        log.Fatal("Invalid private key format")
    }
    if strings.Contains(err.Error(), "RPC") {
        // RPC connection error
        log.Fatal("Cannot connect to Solana RPC")
    }
}

// Handle balance query errors
balance, err := wallet.GetSOLBalance(ctx)
if err != nil {
    if strings.Contains(err.Error(), "context deadline exceeded") {
        // Timeout
        log.Printf("RPC request timeout")
    }
    if strings.Contains(err.Error(), "failed to get SOL balance") {
        // Network error
        log.Printf("Network error while fetching balance")
    }
}

// Handle transaction errors
sig, err := wallet.SignAndSendTransaction(ctx, tx)
if err != nil {
    if strings.Contains(err.Error(), "insufficient funds") {
        // Not enough SOL for fee
        log.Printf("Insufficient SOL for transaction fee")
    }
    if strings.Contains(err.Error(), "blockhash not found") {
        // Blockhash expired
        log.Printf("Blockhash expired, retry transaction")
    }
}
```

## Monitoring and Alerts

### Wallet Balance Monitoring

```go
// Implement periodic balance checks
func monitorWalletBalance(wallet *solana.Wallet, thresholds map[string]decimal.Decimal) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        ctx := context.Background()

        // Check SOL balance
        solBalance, err := wallet.GetSOLBalance(ctx)
        if err != nil {
            log.Printf("Failed to check SOL balance: %v", err)
            continue
        }

        if solBalance.LessThan(thresholds["SOL"]) {
            sendAlert("Low SOL balance: " + solBalance.String())
        }

        // Check USDT balance
        usdtBalance, err := wallet.GetUSDTBalance(ctx, usdtMint)
        if err != nil {
            log.Printf("Failed to check USDT balance: %v", err)
            continue
        }

        if usdtBalance.GreaterThan(thresholds["USDT_MAX"]) {
            sendAlert("Hot wallet USDT balance exceeds threshold: " + usdtBalance.String())
        }
    }
}
```

## Future Enhancements

Planned features for future releases:

- [ ] Multi-signature wallet support
- [ ] Token swap integration (Jupiter, Raydium)
- [ ] Advanced transaction building with priority fees
- [ ] Compressed NFT support (if needed for receipts)
- [ ] WebSocket subscription for real-time transaction monitoring
- [ ] Automatic retry with exponential backoff
- [ ] Transaction simulation before sending

## References

- [Solana Documentation](https://docs.solana.com/)
- [Solana Go SDK](https://github.com/gagliardetto/solana-go)
- [SPL Token Program](https://spl.solana.com/token)
- [Solana Pay Specification](https://github.com/solana-labs/solana-pay)

## Support

For issues or questions:
1. Check the test files for usage examples
2. Review the error handling section
3. Consult Solana documentation
4. Open an issue in the project repository

---

**Last Updated**: 2025-11-17
**Maintained By**: Backend Team
