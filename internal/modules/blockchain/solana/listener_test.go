package solana

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/memo"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveWSURL(t *testing.T) {
	tests := []struct {
		name     string
		rpcURL   string
		expected string
	}{
		{
			name:     "http URL",
			rpcURL:   "http://localhost:8899",
			expected: "ws://localhost:8899",
		},
		{
			name:     "https URL",
			rpcURL:   "https://api.mainnet-beta.solana.com",
			expected: "wss://api.mainnet-beta.solana.com",
		},
		{
			name:     "already ws URL",
			rpcURL:   "ws://localhost:8900",
			expected: "ws://localhost:8900",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveWSURL(tt.rpcURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTransactionListener(t *testing.T) {
	tests := []struct {
		name        string
		config      ListenerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: ListenerConfig{
				Client: &Client{},
				Wallet: &Wallet{},
				ConfirmationCallback: func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
					return nil
				},
			},
			expectError: false,
		},
		{
			name: "missing client",
			config: ListenerConfig{
				Wallet: &Wallet{},
				ConfirmationCallback: func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
					return nil
				},
			},
			expectError: true,
			errorMsg:    "client cannot be nil",
		},
		{
			name: "missing wallet",
			config: ListenerConfig{
				Client: &Client{},
				ConfirmationCallback: func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
					return nil
				},
			},
			expectError: true,
			errorMsg:    "wallet cannot be nil",
		},
		{
			name: "missing callback",
			config: ListenerConfig{
				Client: &Client{},
				Wallet: &Wallet{},
			},
			expectError: true,
			errorMsg:    "confirmation callback cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := NewTransactionListener(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, listener)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, listener)
				assert.False(t, listener.IsRunning())
			}
		})
	}
}

func TestNewTransactionListener_Defaults(t *testing.T) {
	listener, err := NewTransactionListener(ListenerConfig{
		Client: &Client{},
		Wallet: &Wallet{},
		ConfirmationCallback: func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, listener)

	// Check defaults
	assert.Equal(t, 5*time.Second, listener.pollInterval)
	assert.Equal(t, 3, listener.maxRetries)
}

func TestIsSupportedToken(t *testing.T) {
	usdtMint := solana.MustPublicKeyFromBase58("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB")
	usdcMint := solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
	unknownMint := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	listener := &TransactionListener{
		supportedTokenMints: map[string]TokenMintInfo{
			"USDT": {
				MintAddress: usdtMint,
				Symbol:      "USDT",
				Decimals:    6,
			},
			"USDC": {
				MintAddress: usdcMint,
				Symbol:      "USDC",
				Decimals:    6,
			},
		},
	}

	tests := []struct {
		name          string
		mint          solana.PublicKey
		expectFound   bool
		expectedSymbol string
	}{
		{
			name:          "USDT supported",
			mint:          usdtMint,
			expectFound:   true,
			expectedSymbol: "USDT",
		},
		{
			name:          "USDC supported",
			mint:          usdcMint,
			expectFound:   true,
			expectedSymbol: "USDC",
		},
		{
			name:        "unknown token",
			mint:        unknownMint,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenInfo, found := listener.isSupportedToken(tt.mint)

			assert.Equal(t, tt.expectFound, found)
			if tt.expectFound {
				assert.Equal(t, tt.expectedSymbol, tokenInfo.Symbol)
			}
		})
	}
}

func TestParsePaymentTransaction(t *testing.T) {
	// Setup test data
	amount := uint64(1000000) // 1 USDT (6 decimals)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	usdtMint := solana.MustPublicKeyFromBase58("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB")
	walletAddress := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	sourceAccount := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	// Create TransferChecked instruction
	transferData := append([]byte{12}, amountBytes...)
	transferData = append(transferData, 6) // decimals

	accountKeys := []solana.PublicKey{
		token.ProgramID,
		sourceAccount,
		usdtMint,
		walletAddress,
		authority,
		memo.ProgramID,
	}

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys: accountKeys,
			Instructions: []solana.CompiledInstruction{
				{
					ProgramIDIndex: 0,
					Accounts:       []uint16{1, 2, 3, 4},
					Data:           transferData,
				},
				{
					ProgramIDIndex: 5,
					Data:           []byte("payment-12345"),
				},
			},
		},
	}

	signature := solana.MustSignatureFromBase58("5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnbJLgp8uirBgmQpjKhoR4tjF3ZpRzrFmBV6UjKdiSZkQUW")

	txInfo := &TransactionInfo{
		Signature:   signature,
		Transaction: tx,
		IsFinalized: true,
	}

	listener := &TransactionListener{
		wallet: &Wallet{
			publicKey: walletAddress,
		},
		supportedTokenMints: map[string]TokenMintInfo{
			"USDT": {
				MintAddress: usdtMint,
				Symbol:      "USDT",
				Decimals:    6,
			},
		},
	}

	details, err := listener.parsePaymentTransaction(txInfo)
	require.NoError(t, err)
	require.NotNil(t, details)

	assert.Equal(t, walletAddress, details.Recipient)
	assert.Equal(t, authority, details.Sender)
	assert.Equal(t, "payment-12345", details.PaymentID)
	assert.Equal(t, "USDT", details.TokenMint)

	// Verify amount (1 USDT = 1000000 micro-USDT / 10^6 = 1.0)
	expectedAmount := decimal.NewFromInt(1)
	assert.True(t, details.Amount.Equal(expectedAmount),
		"Expected amount %s, got %s", expectedAmount.String(), details.Amount.String())
}

func TestParsePaymentTransaction_NoMemo(t *testing.T) {
	amount := uint64(1000000)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	usdtMint := solana.MustPublicKeyFromBase58("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB")
	walletAddress := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	sourceAccount := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	transferData := append([]byte{12}, amountBytes...)
	transferData = append(transferData, 6)

	accountKeys := []solana.PublicKey{
		token.ProgramID,
		sourceAccount,
		usdtMint,
		walletAddress,
		authority,
	}

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys: accountKeys,
			Instructions: []solana.CompiledInstruction{
				{
					ProgramIDIndex: 0,
					Accounts:       []uint16{1, 2, 3, 4},
					Data:           transferData,
				},
			},
		},
	}

	signature := solana.MustSignatureFromBase58("5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnbJLgp8uirBgmQpjKhoR4tjF3ZpRzrFmBV6UjKdiSZkQUW")

	txInfo := &TransactionInfo{
		Signature:   signature,
		Transaction: tx,
		IsFinalized: true,
	}

	listener := &TransactionListener{
		wallet: &Wallet{
			publicKey: walletAddress,
		},
		supportedTokenMints: map[string]TokenMintInfo{
			"USDT": {
				MintAddress: usdtMint,
				Symbol:      "USDT",
				Decimals:    6,
			},
		},
	}

	_, err := listener.parsePaymentTransaction(txInfo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no memo found")
}

func TestParsePaymentTransaction_UnsupportedToken(t *testing.T) {
	amount := uint64(1000000)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	unknownMint := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	walletAddress := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	sourceAccount := solana.MustPublicKeyFromBase58("33333333333333333333333333333333")
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	transferData := append([]byte{12}, amountBytes...)
	transferData = append(transferData, 6)

	accountKeys := []solana.PublicKey{
		token.ProgramID,
		sourceAccount,
		unknownMint,
		walletAddress,
		authority,
		memo.ProgramID,
	}

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys: accountKeys,
			Instructions: []solana.CompiledInstruction{
				{
					ProgramIDIndex: 0,
					Accounts:       []uint16{1, 2, 3, 4},
					Data:           transferData,
				},
				{
					ProgramIDIndex: 5,
					Data:           []byte("payment-12345"),
				},
			},
		},
	}

	signature := solana.MustSignatureFromBase58("5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnbJLgp8uirBgmQpjKhoR4tjF3ZpRzrFmBV6UjKdiSZkQUW")

	txInfo := &TransactionInfo{
		Signature:   signature,
		Transaction: tx,
		IsFinalized: true,
	}

	listener := &TransactionListener{
		wallet: &Wallet{
			publicKey: walletAddress,
		},
		supportedTokenMints: map[string]TokenMintInfo{
			"USDT": {
				MintAddress: solana.MustPublicKeyFromBase58("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"),
				Symbol:      "USDT",
				Decimals:    6,
			},
		},
	}

	_, err := listener.parsePaymentTransaction(txInfo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported token")
}

func TestListenerStartStop(t *testing.T) {
	// Create a mock client (we won't actually connect)
	client := &Client{
		rpcURL:  "http://localhost:8899",
		timeout: 30 * time.Second,
	}

	wallet := &Wallet{
		publicKey: solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
	}

	callbackCalled := false
	callback := func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
		callbackCalled = true
		return nil
	}

	listener, err := NewTransactionListener(ListenerConfig{
		Client:               client,
		Wallet:               wallet,
		ConfirmationCallback: callback,
		PollInterval:         100 * time.Millisecond,
	})

	require.NoError(t, err)
	require.NotNil(t, listener)

	// Initially not running
	assert.False(t, listener.IsRunning())

	// Note: We can't actually start the listener in tests without a real RPC endpoint
	// In a real test environment, you would:
	// 1. Start the listener
	// 2. Send a test transaction
	// 3. Verify the callback was called
	// 4. Stop the listener

	// For now, just verify the listener was created correctly
	assert.Equal(t, 100*time.Millisecond, listener.pollInterval)
	assert.Equal(t, client, listener.client)
	assert.Equal(t, wallet, listener.wallet)
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        time.Duration
		b        time.Duration
		expected time.Duration
	}{
		{
			name:     "a smaller",
			a:        1 * time.Second,
			b:        2 * time.Second,
			expected: 1 * time.Second,
		},
		{
			name:     "b smaller",
			a:        5 * time.Second,
			b:        3 * time.Second,
			expected: 3 * time.Second,
		},
		{
			name:     "equal",
			a:        10 * time.Second,
			b:        10 * time.Second,
			expected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
