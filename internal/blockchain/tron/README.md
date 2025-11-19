# TRON Blockchain Integration

This package provides comprehensive TRON blockchain integration for the Stablecoin Payment Gateway, supporting both native TRX and TRC20 token (USDT) transactions.

## Features

- ✅ **TRON gRPC Client** - High-performance gRPC connection to TRON network
- ✅ **Wallet Management** - Create, load, and manage TRON wallets
- ✅ **TRC20 Token Support** - Full support for USDT (TRC20) and other TRC20 tokens
- ✅ **Transaction Parsing** - Parse TRX and TRC20 transfers with memo extraction
- ✅ **Blockchain Listener** - Real-time monitoring of incoming transactions
- ✅ **Transaction Validation** - Validate payments against expected criteria
- ✅ **Multi-sig Support** - Compatible with multi-signature wallet setups

## Why TRON? (Priority HIGH)

According to PRD v2.2, TRON is the **highest priority blockchain** for this payment gateway:

1. **Cheapest Fees**: ~$1 per transaction (vs $5-20 on Ethereum)
2. **Fast Confirmation**: ~3 seconds per block, 19 blocks (~57s) for solid confirmation
3. **Massive Adoption in Asia**: Preferred by Chinese tourists and Vietnamese merchants
4. **USDT (TRC20)**: Most popular stablecoin format in Asia-Pacific
5. **High Throughput**: 2000+ TPS capacity

## Quick Start

### 1. Create a TRON Client

```go
import (
    "context"
    "time"
    "github.com/hxuan190/stable_payment_gateway/internal/blockchain/tron"
)

// For testnet (Shasta)
client, err := tron.NewClientWithURL("grpc.shasta.trongrid.io:50051", false)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// For mainnet
client, err := tron.NewClientWithURL("grpc.trongrid.io:50051", true)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Health check
err = client.HealthCheck(context.Background())
if err != nil {
    log.Printf("TRON node unhealthy: %v", err)
}
```

### 2. Load a Wallet

```go
// Load wallet from private key (hex format)
privateKey := "your_private_key_here" // 64 hex characters
rpcURL := "grpc.shasta.trongrid.io:50051"

wallet, err := tron.LoadWallet(privateKey, rpcURL, false)
if err != nil {
    log.Fatal(err)
}
defer wallet.Close()

log.Printf("Wallet address: %s", wallet.GetAddress())

// Get TRX balance
trxBalance, err := wallet.GetTRXBalance(context.Background())
if err != nil {
    log.Fatal(err)
}
log.Printf("TRX Balance: %s TRX", trxBalance.String())

// Get USDT (TRC20) balance
usdtContract := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // Mainnet USDT
usdtBalance, decimals, err := wallet.GetTRC20Balance(context.Background(), usdtContract)
if err != nil {
    log.Fatal(err)
}
log.Printf("USDT Balance: %s USDT (decimals: %d)", usdtBalance.String(), decimals)
```

### 3. Listen for Incoming Transactions

```go
import (
    "github.com/sirupsen/logrus"
)

// Create listener configuration
config := tron.ListenerConfig{
    Client:           client,
    WatchAddress:     "TYourHotWalletAddressHere",
    TRC20Contracts:   []string{"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"}, // USDT
    PollInterval:     3 * time.Second,
    MinConfirmations: 19, // TRON solid block
    StartBlock:       0,  // Start from current block
    Logger:           logrus.New(),
}

listener, err := tron.NewListener(config)
if err != nil {
    log.Fatal(err)
}

// Add transaction handler
listener.AddHandler(func(event *tron.TransactionEvent) error {
    log.Printf("Transaction detected!")
    log.Printf("  TX ID: %s", event.TxID)
    log.Printf("  From: %s", event.FromAddress)
    log.Printf("  To: %s", event.ToAddress)
    log.Printf("  Amount: %s %s", event.Amount.String(), event.TokenSymbol)
    log.Printf("  Memo: %s", event.Memo)
    log.Printf("  Confirmations: %d/%d", event.Confirmations, 19)
    log.Printf("  Confirmed: %v", event.IsConfirmed)

    // Process payment here
    return processPayment(event)
})

// Start listener
err = listener.Start(context.Background())
if err != nil {
    log.Fatal(err)
}

// Keep running
select {}
```

### 4. Parse and Validate Transactions

```go
// Create parser
parser := tron.NewTransactionParser(client)

// Parse transaction
txID := "transaction_id_here"
parsed, err := parser.ParseTransaction(txID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Parsed transaction:")
log.Printf("  From: %s", parsed.FromAddress)
log.Printf("  To: %s", parsed.ToAddress)
log.Printf("  Amount: %s %s", parsed.Amount.String(), parsed.TokenSymbol)
log.Printf("  Memo: %s", parsed.Memo)
log.Printf("  Is TRC20: %v", parsed.IsTRC20Transfer)

// Validate payment
expectedAddress := "TYourHotWalletAddressHere"
expectedAmount := decimal.NewFromFloat(100.0) // 100 USDT
expectedMemo := "PAYMENT:uuid-1234"

validation, err := parser.ValidatePayment(txID, expectedAddress, expectedAmount, expectedMemo, 19)
if err != nil {
    log.Fatal(err)
}

log.Printf("Payment validation:")
log.Printf("  Valid: %v", validation.IsValid)
log.Printf("  Address match: %v", validation.AddressMatch)
log.Printf("  Amount match: %v", validation.AmountMatch)
log.Printf("  Memo match: %v", validation.MemoMatch)
log.Printf("  Confirmed: %v (%s)", validation.IsConfirmed, validation.ConfirmationText)
```

### 5. Send TRX or TRC20 Tokens

```go
// Send TRX
toAddress := "TRecipientAddressHere"
amount := decimal.NewFromFloat(10.5) // 10.5 TRX

txID, err := wallet.TransferTRX(context.Background(), toAddress, amount)
if err != nil {
    log.Fatal(err)
}
log.Printf("TRX transfer sent: %s", txID)

// Wait for confirmation
err = wallet.WaitForTransaction(context.Background(), txID, 19, 5*time.Minute)
if err != nil {
    log.Fatal(err)
}
log.Printf("Transaction confirmed!")

// Send USDT (TRC20)
usdtContract := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
usdtAmount := decimal.NewFromFloat(100.0) // 100 USDT

txID, err := wallet.TransferTRC20(context.Background(), usdtContract, toAddress, usdtAmount)
if err != nil {
    log.Fatal(err)
}
log.Printf("USDT transfer sent: %s", txID)
```

## Network Configuration

### Testnet (Shasta)
- **gRPC URL**: `grpc.shasta.trongrid.io:50051`
- **Explorer**: https://shasta.tronscan.org
- **Faucet**: https://www.trongrid.io/faucet
- **USDT Contract**: Use a testnet USDT contract or deploy your own

### Mainnet
- **gRPC URL**: `grpc.trongrid.io:50051`
- **Explorer**: https://tronscan.org
- **USDT Contract**: `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`

## TRC20 Token Addresses (Mainnet)

- **USDT**: `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`
- **USDC**: `TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8`
- **TUSD**: `TUpMhErZL2fhh4sVNULAbNKLokS4GjC1F4`

## Transaction Memo Format

For payment tracking, include a payment ID in the transaction memo:

```go
// Format: "PAYMENT:uuid"
memo := fmt.Sprintf("PAYMENT:%s", paymentID)

// Extract payment ID from memo
paymentID := tron.ExtractPaymentID(memo)
```

## Confirmation Requirements

According to TRON's finality model:

- **1 confirmation**: Transaction included in a block (~3 seconds)
- **19 confirmations**: Solid block, irreversible (~57 seconds)
- **Recommended**: Wait for 19 confirmations before crediting merchant

## Error Handling

```go
// Always check for errors
txInfo, err := client.GetTransaction(ctx, txID)
if err != nil {
    log.Printf("Failed to get transaction: %v", err)
    return
}

// Check if transaction was successful
if !txInfo.IsSuccess {
    log.Printf("Transaction failed: %v", txInfo.Error)
    return
}

// Validate transaction status
if !txInfo.IsConfirmed {
    log.Printf("Transaction not confirmed yet: %d/%d confirmations", txInfo.Confirmations, 19)
    return
}
```

## Best Practices

1. **Always use testnet first** - Test on Shasta before deploying to mainnet
2. **Wait for 19 confirmations** - Don't credit payments until solidly confirmed
3. **Validate all transactions** - Check address, amount, and memo match exactly
4. **Handle errors gracefully** - Network issues are common, implement retry logic
5. **Monitor hot wallet balance** - Implement auto-sweeping to cold wallet
6. **Use decimal.Decimal for amounts** - Never use float64 for money calculations
7. **Log all transactions** - Keep comprehensive audit trail
8. **Rate limiting** - TronGrid has rate limits, use paid API for production

## Integration with Payment Gateway

### Payment Flow

1. **Merchant creates payment**
   - Generate payment ID (UUID)
   - Calculate USDT amount from VND
   - Create QR code with hot wallet address + memo

2. **User scans QR and sends USDT**
   - Blockchain listener detects transaction
   - Parse transaction to extract amount and memo
   - Validate payment matches expected criteria

3. **Confirm payment**
   - Wait for 19 confirmations (~57 seconds)
   - Update payment status to confirmed
   - Credit merchant balance
   - Send webhook notification

4. **Settlement**
   - Merchant requests payout
   - OTC partner converts USDT to VND
   - Transfer VND to merchant bank account

## Troubleshooting

### Connection Issues
```go
// Check gRPC connection
err := client.HealthCheck(context.Background())
if err != nil {
    log.Printf("Connection failed: %v", err)
}

// Use retry logic
err = client.HealthCheckWithRetry(context.Background(), 3)
```

### Transaction Not Found
- Wait a few seconds and retry (transaction might not be propagated yet)
- Verify transaction ID format (64 hex characters)
- Check if transaction is on correct network (mainnet vs testnet)

### Balance Not Updating
- Check account has sufficient bandwidth/energy
- Verify TRC20 contract address is correct
- Wait for transaction confirmation (19 blocks)

## Testing

Run tests with:

```bash
go test ./internal/blockchain/tron/... -v
```

For integration tests with testnet:

```bash
export TRON_RPC_URL="grpc.shasta.trongrid.io:50051"
export TRON_PRIVATE_KEY="your_testnet_private_key"
go test ./internal/blockchain/tron/... -v -tags=integration
```

## References

- [TRON Documentation](https://developers.tron.network/)
- [TronGrid API](https://www.trongrid.io/)
- [gotron-sdk](https://github.com/fbsobreira/gotron-sdk)
- [TRC20 Standard](https://github.com/tronprotocol/tips/blob/master/tip-20.md)

## Support

For issues or questions:
- Check the main CLAUDE.md documentation
- Review ARCHITECTURE.md for system design
- See PRD_v2.2.md for feature requirements
