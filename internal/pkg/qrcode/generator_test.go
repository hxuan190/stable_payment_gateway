package qrcode

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()
	assert.NotNil(t, gen)
	assert.Equal(t, QRCodeSizeMedium, gen.defaultSize)
}

func TestGeneratePaymentQR_Success(t *testing.T) {
	gen := NewGenerator()

	config := PaymentQRConfig{
		WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
		Amount:        decimal.NewFromFloat(100.50),
		TokenMint:     USDTMintSolana,
		Memo:          "payment-123",
		Label:         "Test Payment",
		Size:          QRCodeSizeSmall,
	}

	qrCode, err := gen.GeneratePaymentQR(config)

	require.NoError(t, err)
	assert.NotEmpty(t, qrCode)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(qrCode)
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)

	// Verify it's a PNG (starts with PNG magic bytes)
	assert.Equal(t, byte(0x89), decoded[0])
	assert.Equal(t, byte(0x50), decoded[1])
	assert.Equal(t, byte(0x4E), decoded[2])
	assert.Equal(t, byte(0x47), decoded[3])
}

func TestGeneratePaymentQR_DefaultSize(t *testing.T) {
	gen := NewGenerator()

	config := PaymentQRConfig{
		WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
		Amount:        decimal.NewFromFloat(50.0),
		TokenMint:     USDCMintSolana,
		Memo:          "payment-456",
		Label:         "Test",
		// Size not specified - should use default
	}

	qrCode, err := gen.GeneratePaymentQR(config)

	require.NoError(t, err)
	assert.NotEmpty(t, qrCode)
}

func TestGeneratePaymentQR_InvalidConfig(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name        string
		config      PaymentQRConfig
		expectedErr string
	}{
		{
			name: "missing wallet address",
			config: PaymentQRConfig{
				Amount:    decimal.NewFromFloat(100),
				TokenMint: USDTMintSolana,
				Memo:      "payment-123",
			},
			expectedErr: "wallet address is required",
		},
		{
			name: "zero amount",
			config: PaymentQRConfig{
				WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				Amount:        decimal.Zero,
				TokenMint:     USDTMintSolana,
				Memo:          "payment-123",
			},
			expectedErr: "amount must be greater than zero",
		},
		{
			name: "negative amount",
			config: PaymentQRConfig{
				WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				Amount:        decimal.NewFromFloat(-50),
				TokenMint:     USDTMintSolana,
				Memo:          "payment-123",
			},
			expectedErr: "amount must be greater than zero",
		},
		{
			name: "missing token mint",
			config: PaymentQRConfig{
				WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				Amount:        decimal.NewFromFloat(100),
				Memo:          "payment-123",
			},
			expectedErr: "token mint is required",
		},
		{
			name: "missing memo",
			config: PaymentQRConfig{
				WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				Amount:        decimal.NewFromFloat(100),
				TokenMint:     USDTMintSolana,
			},
			expectedErr: "memo is required",
		},
		{
			name: "invalid size",
			config: PaymentQRConfig{
				WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				Amount:        decimal.NewFromFloat(100),
				TokenMint:     USDTMintSolana,
				Memo:          "payment-123",
				Size:          999, // Invalid size
			},
			expectedErr: "invalid QR code size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gen.GeneratePaymentQR(tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestBuildSolanaPayURL(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name           string
		config         PaymentQRConfig
		expectedInURL  []string
		notExpectedInURL []string
	}{
		{
			name: "USDT payment with label",
			config: PaymentQRConfig{
				WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				Amount:        decimal.NewFromFloat(100.50),
				TokenMint:     USDTMintSolana,
				Memo:          "payment-123",
				Label:         "Test Payment",
			},
			expectedInURL: []string{
				"solana:7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				"amount=100.5",
				"spl-token=" + USDTMintSolana,
				"memo=payment-123",
				"label=Test+Payment",
			},
		},
		{
			name: "USDC payment without label",
			config: PaymentQRConfig{
				WalletAddress: "ABC123",
				Amount:        decimal.NewFromFloat(50),
				TokenMint:     USDCMintSolana,
				Memo:          "order-456",
			},
			expectedInURL: []string{
				"solana:ABC123",
				"amount=50",
				"spl-token=" + USDCMintSolana,
				"memo=order-456",
			},
			notExpectedInURL: []string{
				"label=",
			},
		},
		{
			name: "decimal amount precision",
			config: PaymentQRConfig{
				WalletAddress: "WALLET123",
				Amount:        decimal.NewFromFloat(123.456789),
				TokenMint:     USDTMintSolana,
				Memo:          "test",
			},
			expectedInURL: []string{
				"amount=123.456789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := gen.buildSolanaPayURL(tt.config)
			require.NoError(t, err)

			for _, expected := range tt.expectedInURL {
				assert.Contains(t, url, expected, "URL should contain: %s", expected)
			}

			for _, notExpected := range tt.notExpectedInURL {
				assert.NotContains(t, url, notExpected, "URL should not contain: %s", notExpected)
			}
		})
	}
}

func TestGeneratePaymentQRSimple(t *testing.T) {
	walletAddress := "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU"
	amount := decimal.NewFromFloat(100)
	memo := "payment-789"

	qrCode, err := GeneratePaymentQRSimple(walletAddress, amount, memo)

	require.NoError(t, err)
	assert.NotEmpty(t, qrCode)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(qrCode)
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)
}

func TestGetSolanaPayURL(t *testing.T) {
	walletAddress := "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU"
	amount := decimal.NewFromFloat(75.25)
	tokenMint := USDTMintSolana
	memo := "payment-999"
	label := "Test"

	url, err := GetSolanaPayURL(walletAddress, amount, tokenMint, memo, label)

	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.True(t, strings.HasPrefix(url, "solana:"))
	assert.Contains(t, url, walletAddress)
	assert.Contains(t, url, "amount=75.25")
	assert.Contains(t, url, "spl-token="+tokenMint)
	assert.Contains(t, url, "memo="+memo)
	assert.Contains(t, url, "label="+label)
}

func TestGetSolanaPayURL_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		walletAddress string
		amount        decimal.Decimal
		tokenMint     string
		memo          string
		label         string
		expectedErr   string
	}{
		{
			name:          "empty wallet",
			walletAddress: "",
			amount:        decimal.NewFromFloat(100),
			tokenMint:     USDTMintSolana,
			memo:          "test",
			label:         "Test",
			expectedErr:   "wallet address is required",
		},
		{
			name:          "zero amount",
			walletAddress: "WALLET123",
			amount:        decimal.Zero,
			tokenMint:     USDTMintSolana,
			memo:          "test",
			label:         "Test",
			expectedErr:   "amount must be greater than zero",
		},
		{
			name:          "empty token mint",
			walletAddress: "WALLET123",
			amount:        decimal.NewFromFloat(100),
			tokenMint:     "",
			memo:          "test",
			label:         "Test",
			expectedErr:   "token mint is required",
		},
		{
			name:          "empty memo",
			walletAddress: "WALLET123",
			amount:        decimal.NewFromFloat(100),
			tokenMint:     USDTMintSolana,
			memo:          "",
			label:         "Test",
			expectedErr:   "memo is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetSolanaPayURL(tt.walletAddress, tt.amount, tt.tokenMint, tt.memo, tt.label)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestQRCodeSizes(t *testing.T) {
	gen := NewGenerator()

	sizes := []QRCodeSize{
		QRCodeSizeSmall,
		QRCodeSizeMedium,
		QRCodeSizeLarge,
	}

	for _, size := range sizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			config := PaymentQRConfig{
				WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				Amount:        decimal.NewFromFloat(100),
				TokenMint:     USDTMintSolana,
				Memo:          "test",
				Size:          size,
			}

			qrCode, err := gen.GeneratePaymentQR(config)
			require.NoError(t, err)
			assert.NotEmpty(t, qrCode)

			// Decode and verify
			decoded, err := base64.StdEncoding.DecodeString(qrCode)
			require.NoError(t, err)

			// Larger sizes should produce larger images
			// This is a basic sanity check
			if size == QRCodeSizeSmall {
				assert.Less(t, len(decoded), 50000, "Small QR code should be under 50KB")
			}
		})
	}
}

func TestTokenMintConstants(t *testing.T) {
	// Verify the token mint addresses are correct
	assert.Equal(t, "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", USDTMintSolana)
	assert.Equal(t, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", USDCMintSolana)
}

func BenchmarkGeneratePaymentQR(b *testing.B) {
	gen := NewGenerator()
	config := PaymentQRConfig{
		WalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
		Amount:        decimal.NewFromFloat(100.50),
		TokenMint:     USDTMintSolana,
		Memo:          "payment-123",
		Label:         "Test Payment",
		Size:          QRCodeSizeMedium,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.GeneratePaymentQR(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}
