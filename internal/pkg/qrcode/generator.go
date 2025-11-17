package qrcode

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/shopspring/decimal"
	"github.com/skip2/go-qrcode"
)

// QRCodeSize represents the size of the QR code in pixels
type QRCodeSize int

const (
	// QRCodeSizeSmall is 256x256 pixels
	QRCodeSizeSmall QRCodeSize = 256
	// QRCodeSizeMedium is 512x512 pixels (default)
	QRCodeSizeMedium QRCodeSize = 512
	// QRCodeSizeLarge is 1024x1024 pixels
	QRCodeSizeLarge QRCodeSize = 1024
)

// Solana token mint addresses
const (
	// USDTMintSolana is the USDT SPL token mint address on Solana
	USDTMintSolana = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
	// USDCMintSolana is the USDC SPL token mint address on Solana
	USDCMintSolana = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
)

// PaymentQRConfig holds configuration for payment QR code generation
type PaymentQRConfig struct {
	// WalletAddress is the recipient's Solana wallet address
	WalletAddress string
	// Amount is the payment amount in token units
	Amount decimal.Decimal
	// TokenMint is the SPL token mint address (USDT or USDC)
	TokenMint string
	// Memo is the payment reference/ID
	Memo string
	// Label is an optional label for the payment
	Label string
	// Size is the QR code size in pixels
	Size QRCodeSize
}

// Generator handles QR code generation
type Generator struct {
	defaultSize QRCodeSize
}

// NewGenerator creates a new QR code generator
func NewGenerator() *Generator {
	return &Generator{
		defaultSize: QRCodeSizeMedium,
	}
}

// GeneratePaymentQR generates a Solana Pay QR code for a payment
// Returns a base64-encoded PNG image
func (g *Generator) GeneratePaymentQR(config PaymentQRConfig) (string, error) {
	// Validate inputs
	if err := g.validateConfig(config); err != nil {
		return "", err
	}

	// Use default size if not specified
	size := config.Size
	if size == 0 {
		size = g.defaultSize
	}

	// Build Solana Pay URL
	solanaPayURL, err := g.buildSolanaPayURL(config)
	if err != nil {
		return "", fmt.Errorf("failed to build Solana Pay URL: %w", err)
	}

	// Generate QR code
	qrCode, err := qrcode.New(solanaPayURL, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to create QR code: %w", err)
	}

	// Generate PNG bytes
	pngBytes, err := qrCode.PNG(int(size))
	if err != nil {
		return "", fmt.Errorf("failed to generate PNG: %w", err)
	}

	// Encode to base64
	base64String := base64.StdEncoding.EncodeToString(pngBytes)

	return base64String, nil
}

// buildSolanaPayURL constructs a Solana Pay URL
// Format: solana:<recipient>?amount=<amount>&spl-token=<mint>&memo=<memo>&label=<label>
func (g *Generator) buildSolanaPayURL(config PaymentQRConfig) (string, error) {
	// Start with base URL
	baseURL := fmt.Sprintf("solana:%s", config.WalletAddress)

	// Build query parameters
	params := url.Values{}

	// Add amount (convert to string with proper precision)
	params.Add("amount", config.Amount.String())

	// Add SPL token mint
	params.Add("spl-token", config.TokenMint)

	// Add memo (payment reference)
	params.Add("memo", config.Memo)

	// Add label if provided
	if config.Label != "" {
		params.Add("label", config.Label)
	}

	// Construct full URL
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	return fullURL, nil
}

// validateConfig validates the QR code configuration
func (g *Generator) validateConfig(config PaymentQRConfig) error {
	if config.WalletAddress == "" {
		return fmt.Errorf("wallet address is required")
	}

	if config.Amount.IsZero() || config.Amount.IsNegative() {
		return fmt.Errorf("amount must be greater than zero")
	}

	if config.TokenMint == "" {
		return fmt.Errorf("token mint is required")
	}

	if config.Memo == "" {
		return fmt.Errorf("memo is required")
	}

	// Validate size if specified
	if config.Size != 0 {
		if config.Size != QRCodeSizeSmall &&
		   config.Size != QRCodeSizeMedium &&
		   config.Size != QRCodeSizeLarge {
			return fmt.Errorf("invalid QR code size: %d", config.Size)
		}
	}

	return nil
}

// GeneratePaymentQRSimple is a convenience function for simple QR code generation
// Uses default size and USDT token
func GeneratePaymentQRSimple(walletAddress string, amount decimal.Decimal, memo string) (string, error) {
	generator := NewGenerator()
	return generator.GeneratePaymentQR(PaymentQRConfig{
		WalletAddress: walletAddress,
		Amount:        amount,
		TokenMint:     USDTMintSolana,
		Memo:          memo,
		Label:         "Payment",
		Size:          QRCodeSizeMedium,
	})
}

// GetSolanaPayURL returns the Solana Pay URL without generating a QR code
// Useful for testing or when you just need the URL
func GetSolanaPayURL(walletAddress string, amount decimal.Decimal, tokenMint, memo, label string) (string, error) {
	generator := NewGenerator()

	config := PaymentQRConfig{
		WalletAddress: walletAddress,
		Amount:        amount,
		TokenMint:     tokenMint,
		Memo:          memo,
		Label:         label,
	}

	if err := generator.validateConfig(config); err != nil {
		return "", err
	}

	return generator.buildSolanaPayURL(config)
}
