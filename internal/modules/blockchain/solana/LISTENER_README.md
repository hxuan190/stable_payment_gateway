# Solana Transaction Listener

## Overview

The Solana Transaction Listener monitors the blockchain for incoming SPL token transfers to a designated wallet address. It automatically detects payments, validates them, and triggers payment confirmation callbacks.

## Implementation Status

✅ **Task 4.3: Solana Transaction Listener** - COMPLETED

This implementation includes:
- Real-time transaction monitoring via WebSocket subscription
- Polling fallback for reliability
- SPL token transfer parsing
- Memo field extraction for payment ID
- Transaction validation and finality checking
- Error handling and retry logic
- Comprehensive unit tests

## Components

### 1. listener.go

Main listener implementation with:
- **WebSocket Listening**: Real-time notifications of account changes
- **Polling Fallback**: Periodic polling for missed transactions
- **Transaction Processing**: Parse and validate incoming payments
- **Graceful Shutdown**: Clean resource cleanup

### 2. parser.go

Transaction parsing utilities:
- **extractMemoFromTransaction()**: Extracts payment ID from memo instruction
- **parseSPLTokenTransfer()**: Decodes SPL token transfer details
- **decodeTransferInstruction()**: Handles Transfer instruction (discriminator 3)
- **decodeTransferCheckedInstruction()**: Handles TransferChecked instruction (discriminator 12)
- **ValidatePaymentTransaction()**: Validates payment amount and recipient

### 3. Tests

Comprehensive test coverage:
- **parser_test.go**: Tests for all parsing functions
- **listener_test.go**: Tests for listener configuration and logic

## Usage

### Basic Setup

```go
import (
    "github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
    "github.com/shopspring/decimal"
)

// Create RPC client
client, err := solana.NewClient(solana.ClientConfig{
    RPCURL: "https://api.mainnet-beta.solana.com",
})

// Load wallet
wallet, err := solana.LoadWallet(privateKey, rpcURL)

// Define supported tokens
supportedTokens := map[string]solana.TokenMintInfo{
    "USDT": {
        MintAddress: solana.MustPublicKeyFromBase58("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"),
        Symbol:      "USDT",
        Decimals:    6,
    },
    "USDC": {
        MintAddress: solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"),
        Symbol:      "USDC",
        Decimals:    6,
    },
}

// Create listener
listener, err := solana.NewTransactionListener(solana.ListenerConfig{
    Client: client,
    Wallet: wallet,
    ConfirmationCallback: func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
        // Handle confirmed payment
        log.Printf("Payment confirmed: %s, amount: %s %s, tx: %s",
            paymentID, amount.String(), tokenMint, txHash)

        // Update payment in database
        return paymentService.ConfirmPayment(paymentID, txHash, amount)
    },
    SupportedTokenMints: supportedTokens,
    PollInterval: 10 * time.Second,
})

// Start listening
err = listener.Start()

// ... later, graceful shutdown
err = listener.Stop()
```

### Payment Flow

1. **Merchant creates payment** → generates payment ID
2. **User sends tokens** → includes payment ID in memo field
3. **Listener detects transaction** → WebSocket or polling
4. **Parser extracts details** → amount, token, memo
5. **Validation** → verify recipient, token support, finality
6. **Callback triggered** → confirm payment in database

### Transaction Structure

Expected transaction format:
```
Transaction {
    Instructions: [
        TokenTransferInstruction {  // Transfer or TransferChecked
            from: user_wallet
            to: merchant_wallet (our wallet)
            amount: 1000000  // 1 USDT (6 decimals)
            mint: USDT_MINT_ADDRESS
        },
        MemoInstruction {
            data: "payment-abc123"  // Payment ID
        }
    ]
}
```

## Configuration

### Listener Configuration

- **PollInterval**: How often to poll for transactions (default: 5s)
- **MaxRetries**: Maximum retries for RPC calls (default: 3)
- **WSURL**: WebSocket endpoint (auto-derived from RPC URL if not provided)

### Recommended Settings

**Development/Testnet:**
```go
PollInterval: 5 * time.Second
MaxRetries: 3
```

**Production:**
```go
PollInterval: 10 * time.Second  // Less frequent for cost savings
MaxRetries: 5  // More retries for reliability
```

## Supported Token Standards

- **SPL Token** (Token Program: `TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA`)
- **Transfer Instruction** (Discriminator: 3)
- **TransferChecked Instruction** (Discriminator: 12) - Preferred for security

## Error Handling

The listener includes robust error handling:

1. **WebSocket Reconnection**: Automatic reconnection with exponential backoff
2. **RPC Failures**: Retry logic with configurable attempts
3. **Invalid Transactions**: Gracefully skip non-payment transactions
4. **Unsupported Tokens**: Filter out unsupported token transfers
5. **Missing Memos**: Ignore transactions without payment ID

## Finality

The listener waits for **finalized** commitment before confirming payments:
- Solana finality: ~13 seconds (32 confirmed blocks)
- No risk of transaction reversal after finalization

## Security Considerations

1. **Exact Amount Matching**: No tolerance for amount discrepancies (configurable)
2. **Recipient Verification**: Only process payments to our wallet
3. **Token Whitelist**: Only accept approved stablecoins
4. **Memo Validation**: Payment ID must be present and valid
5. **Finality Check**: Wait for finalized commitment

## Performance

**Expected Performance:**
- WebSocket latency: ~400ms (Solana block time)
- Polling latency: depends on poll interval (5-10s)
- Processing time: <100ms per transaction
- Memory usage: ~50MB baseline + 10KB per transaction

## Monitoring

Key metrics to monitor:
- Transaction detection latency
- WebSocket connection uptime
- RPC error rate
- Callback success rate
- Transaction processing rate

## Testing

Run tests:
```bash
go test ./internal/blockchain/solana/... -v
```

Run specific tests:
```bash
go test -v ./internal/blockchain/solana -run TestParsePaymentTransaction
```

## Integration with Payment Service

```go
// In payment service
func (s *PaymentService) StartBlockchainListener() error {
    listener, err := solana.NewTransactionListener(solana.ListenerConfig{
        Client: s.blockchainClient,
        Wallet: s.wallet,
        ConfirmationCallback: s.handlePaymentConfirmation,
        SupportedTokenMints: s.supportedTokens,
    })

    if err != nil {
        return err
    }

    s.listener = listener
    return listener.Start()
}

func (s *PaymentService) handlePaymentConfirmation(
    paymentID string,
    txHash string,
    amount decimal.Decimal,
    tokenMint string,
) error {
    // Get payment from database
    payment, err := s.paymentRepo.GetByID(paymentID)
    if err != nil {
        return err
    }

    // Validate amount matches
    if !amount.Equal(payment.ExpectedAmount) {
        return fmt.Errorf("amount mismatch")
    }

    // Update payment status
    err = s.paymentRepo.UpdateStatus(paymentID, "completed")
    if err != nil {
        return err
    }

    // Record ledger entry
    err = s.ledgerService.RecordPaymentConfirmed(paymentID, amount)
    if err != nil {
        return err
    }

    // Send webhook notification
    go s.notificationService.SendWebhook(payment.MerchantID, "payment.completed", payment)

    return nil
}
```

## Next Steps

After implementing the listener, the next tasks are:

1. **Task 4.4**: Solana Transaction Parser ✅ (Completed as part of this task)
2. **Task 4.5**: Blockchain Transaction Repository (track tx in database)
3. **Task 4.6**: Wallet Balance Monitor (alert on low/high balance)

## Resources

- [Solana RPC Documentation](https://docs.solana.com/api/http)
- [SPL Token Program](https://spl.solana.com/token)
- [Solana Go SDK](https://github.com/gagliardetto/solana-go)

## Troubleshooting

**WebSocket not connecting:**
- Check WSURL format (should be wss:// for HTTPS endpoints)
- Verify RPC provider supports WebSocket
- Check firewall/network restrictions

**Transactions not detected:**
- Verify wallet address is correct
- Check transaction includes memo instruction
- Ensure token mint is in supported list
- Confirm transaction is finalized

**Callback errors:**
- Check database connection
- Verify payment exists in database
- Review error logs for details
- Ensure callback doesn't panic

## License

Part of the Stablecoin Payment Gateway MVP project.
