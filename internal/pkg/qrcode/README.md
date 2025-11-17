# QR Code Generator

This package provides QR code generation for Solana Pay payments.

## Features

- Generate QR codes for Solana Pay transactions
- Support for USDT and USDC SPL tokens
- Configurable QR code sizes (256x256, 512x512, 1024x1024)
- Base64-encoded PNG output
- Solana Pay URL format compliance

## Usage

### Simple QR Code Generation

The simplest way to generate a QR code for a USDT payment:

```go
import (
    "github.com/hxuan190/stable_payment_gateway/internal/pkg/qrcode"
    "github.com/shopspring/decimal"
)

// Generate QR code with default settings (USDT, medium size)
qrCodeBase64, err := qrcode.GeneratePaymentQRSimple(
    "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU", // wallet address
    decimal.NewFromFloat(100.50),                      // amount
    "payment-123",                                     // memo/reference
)
if err != nil {
    // handle error
}

// qrCodeBase64 is a base64-encoded PNG image
// Can be embedded directly in HTML: <img src="data:image/png;base64,{qrCodeBase64}" />
```

### Custom Configuration

For more control over the QR code generation:

```go
import (
    "github.com/hxuan190/stable_payment_gateway/internal/pkg/qrcode"
    "github.com/shopspring/decimal"
)

generator := qrcode.NewGenerator()

config := qrcode.PaymentQRConfig{
    WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
    Amount:        decimal.NewFromFloat(100.50),
    TokenMint:     qrcode.USDCMintSolana, // Use USDC instead of USDT
    Memo:          "order-456",
    Label:         "Coffee Shop Payment",
    Size:          qrcode.QRCodeSizeLarge, // 1024x1024 pixels
}

qrCodeBase64, err := generator.GeneratePaymentQR(config)
if err != nil {
    // handle error
}
```

### Getting Solana Pay URL

If you only need the URL without generating the QR code:

```go
url, err := qrcode.GetSolanaPayURL(
    "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU", // wallet
    decimal.NewFromFloat(50.0),                      // amount
    qrcode.USDTMintSolana,                          // token
    "payment-789",                                   // memo
    "Payment",                                       // label
)
// url: "solana:7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU?amount=50&spl-token=Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB&memo=payment-789&label=Payment"
```

## QR Code Sizes

Three predefined sizes are available:

- `QRCodeSizeSmall` - 256x256 pixels (suitable for mobile)
- `QRCodeSizeMedium` - 512x512 pixels (default, good balance)
- `QRCodeSizeLarge` - 1024x1024 pixels (high quality for printing)

## Token Mint Addresses

The package provides constants for common Solana SPL tokens:

- `USDTMintSolana` - USDT on Solana: `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB`
- `USDCMintSolana` - USDC on Solana: `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`

## Solana Pay URL Format

The generated QR codes follow the Solana Pay specification:

```
solana:<recipient>?amount=<amount>&spl-token=<mint>&memo=<memo>&label=<label>
```

Example:
```
solana:7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU?amount=100.5&spl-token=Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB&memo=payment-123&label=Test+Payment
```

## Validation

The package validates all inputs:
- Wallet address must not be empty
- Amount must be greater than zero
- Token mint must not be empty
- Memo must not be empty
- Size (if specified) must be one of the predefined sizes

## Integration Example

Here's how to integrate QR code generation in a payment API:

```go
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
    var req CreatePaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Create payment in database
    payment, err := h.paymentService.CreatePayment(req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Generate QR code
    qrCode, err := qrcode.GeneratePaymentQRSimple(
        h.config.WalletAddress,
        payment.AmountCrypto,
        payment.ID, // Use payment ID as memo
    )
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to generate QR code"})
        return
    }

    c.JSON(200, gin.H{
        "payment_id": payment.ID,
        "amount_vnd": payment.AmountVND,
        "amount_usdt": payment.AmountCrypto,
        "qr_code": qrCode, // Base64-encoded PNG
        "expires_at": payment.ExpiresAt,
    })
}
```

## Performance

Benchmark results on Intel Xeon @ 2.60GHz:
- ~5.4ms per QR code generation (medium size)
- ~1.6MB memory allocated per operation

This is fast enough for real-time API requests.

## Testing

Run tests:
```bash
go test ./internal/pkg/qrcode/
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./internal/pkg/qrcode/
```

## Dependencies

- `github.com/skip2/go-qrcode` - QR code generation
- `github.com/shopspring/decimal` - Precise decimal arithmetic for amounts
