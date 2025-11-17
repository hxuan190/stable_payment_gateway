package solana

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	// This is a test private key - NEVER use in production
	testPrivateKeyBase58 = "5JCz9xMrCW8yGN3P9nKLqGvSYN5F8VG7vTxMYWKPLZNNqhFYYh3V3X1X8aQ5YfYJN6MQbMZ8Z1Z2Z3Z4Z5Z6Z7Z8"
	testRPCURL           = "https://api.devnet.solana.com"

	// Devnet USDT and USDC mint addresses (may vary, check current devnet)
	testUSDTMint = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
	testUSDCMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
)

func generateTestWallet() (*Wallet, ed25519.PrivateKey) {
	// Generate a random keypair for testing
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	privateKeyBase58 := base58.Encode(privateKey)
	wallet, err := LoadWallet(privateKeyBase58, testRPCURL)
	if err != nil {
		panic(err)
	}

	return wallet, privateKey
}

func TestLoadWallet(t *testing.T) {
	tests := []struct {
		name        string
		privateKey  string
		rpcURL      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Empty private key",
			privateKey:  "",
			rpcURL:      testRPCURL,
			expectError: true,
			errorMsg:    "private key cannot be empty",
		},
		{
			name:        "Empty RPC URL",
			privateKey:  testPrivateKeyBase58,
			rpcURL:      "",
			expectError: true,
			errorMsg:    "RPC URL cannot be empty",
		},
		{
			name:        "Invalid base58 private key",
			privateKey:  "invalid-base58-!@#$%",
			rpcURL:      testRPCURL,
			expectError: true,
			errorMsg:    "failed to decode private key",
		},
		{
			name:        "Invalid private key length",
			privateKey:  base58.Encode([]byte("short")),
			rpcURL:      testRPCURL,
			expectError: true,
			errorMsg:    "invalid private key length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wallet, err := LoadWallet(tt.privateKey, tt.rpcURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, wallet)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wallet)
			}
		})
	}
}

func TestLoadWallet_ValidKey(t *testing.T) {
	// Generate a valid keypair
	_, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	privateKeyBase58 := base58.Encode(privateKey)

	wallet, err := LoadWallet(privateKeyBase58, testRPCURL)
	require.NoError(t, err)
	assert.NotNil(t, wallet)
	assert.NotNil(t, wallet.privateKey)
	assert.NotNil(t, wallet.publicKey)
	assert.NotNil(t, wallet.rpcClient)

	// Verify the public key matches
	expectedPublicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	assert.Equal(t, expectedPublicKey, []byte(wallet.publicKey[:]))
}

func TestGetAddress(t *testing.T) {
	wallet, privateKey := generateTestWallet()

	address := wallet.GetAddress()
	assert.NotEmpty(t, address)

	// Verify the address is valid base58
	_, err := solana.PublicKeyFromBase58(address)
	assert.NoError(t, err)

	// Verify the address matches the public key derived from private key
	expectedPublicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	expectedAddress := solana.PublicKeyFromBytes(expectedPublicKey).String()
	assert.Equal(t, expectedAddress, address)
}

func TestGetPublicKey(t *testing.T) {
	wallet, privateKey := generateTestWallet()

	publicKey := wallet.GetPublicKey()
	assert.NotNil(t, publicKey)

	// Verify it matches the expected public key
	expectedPublicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	assert.Equal(t, expectedPublicKey, []byte(publicKey[:]))
}

func TestGetPrivateKey(t *testing.T) {
	wallet, originalPrivateKey := generateTestWallet()

	privateKey := wallet.GetPrivateKey()
	assert.NotNil(t, privateKey)
	assert.Equal(t, originalPrivateKey, []byte(privateKey))
}

// TestGetSOLBalance tests SOL balance retrieval (requires network connection)
func TestGetSOLBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	wallet, _ := generateTestWallet()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	balance, err := wallet.GetSOLBalance(ctx)

	// Should not error even if balance is zero
	assert.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))
}

// TestGetTokenBalance tests SPL token balance retrieval (requires network connection)
func TestGetTokenBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		tokenMint   string
		expectError bool
	}{
		{
			name:        "Empty token mint",
			tokenMint:   "",
			expectError: true,
		},
		{
			name:        "Invalid token mint",
			tokenMint:   "invalid",
			expectError: true,
		},
		{
			name:        "Valid USDT mint (may have zero balance)",
			tokenMint:   testUSDTMint,
			expectError: false,
		},
	}

	wallet, _ := generateTestWallet()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := wallet.GetTokenBalance(ctx, tt.tokenMint)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Balance should be >= 0, even if the account doesn't exist
				assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))
			}
		})
	}
}

// TestGetUSDTBalance tests USDT balance convenience method
func TestGetUSDTBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	wallet, _ := generateTestWallet()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	balance, err := wallet.GetUSDTBalance(ctx, testUSDTMint)
	assert.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))
}

// TestGetUSDCBalance tests USDC balance convenience method
func TestGetUSDCBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	wallet, _ := generateTestWallet()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	balance, err := wallet.GetUSDCBalance(ctx, testUSDCMint)
	assert.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))
}

// TestGetWalletBalance tests comprehensive balance retrieval
func TestGetWalletBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	wallet, _ := generateTestWallet()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenMints := map[string]string{
		"USDT": testUSDTMint,
		"USDC": testUSDCMint,
	}

	balance, err := wallet.GetWalletBalance(ctx, tokenMints)
	assert.NoError(t, err)
	assert.NotNil(t, balance)
	assert.True(t, balance.SOL.GreaterThanOrEqual(decimal.Zero))
	assert.NotNil(t, balance.Tokens)
}

func TestSignTransaction(t *testing.T) {
	wallet, _ := generateTestWallet()

	// Create a test transaction
	recipient := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	instruction := wallet.CreateTransferInstruction(recipient, 1000)

	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		solana.Hash{}, // Empty blockhash for testing
		solana.TransactionPayer(wallet.publicKey),
	)
	require.NoError(t, err)

	// Test signing
	err = wallet.SignTransaction(tx)
	assert.NoError(t, err)

	// Verify transaction has signatures
	assert.NotEmpty(t, tx.Signatures)
}

func TestSignTransaction_NilTransaction(t *testing.T) {
	wallet, _ := generateTestWallet()

	err := wallet.SignTransaction(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction cannot be nil")
}

func TestCreateTransferInstruction(t *testing.T) {
	wallet, _ := generateTestWallet()

	recipient := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	lamports := uint64(1000000) // 0.001 SOL

	instruction := wallet.CreateTransferInstruction(recipient, lamports)
	assert.NotNil(t, instruction)

	// Verify instruction program ID is the system program
	programID := instruction.ProgramID()
	assert.Equal(t, solana.SystemProgramID, programID)
}

// TestHealthCheck tests the wallet health check (requires network connection)
func TestHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	wallet, _ := generateTestWallet()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := wallet.HealthCheck(ctx)
	assert.NoError(t, err)
}

// TestHealthCheck_InvalidRPC tests health check with invalid RPC
func TestHealthCheck_InvalidRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Generate a wallet with invalid RPC URL
	_, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	privateKeyBase58 := base58.Encode(privateKey)
	wallet, err := LoadWallet(privateKeyBase58, "http://invalid-rpc-url:9999")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = wallet.HealthCheck(ctx)
	assert.Error(t, err)
}

func TestGetRPCClient(t *testing.T) {
	wallet, _ := generateTestWallet()

	client := wallet.GetRPCClient()
	assert.NotNil(t, client)
}

// Benchmark tests
func BenchmarkLoadWallet(b *testing.B) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(b, err)
	privateKeyBase58 := base58.Encode(privateKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadWallet(privateKeyBase58, testRPCURL)
	}
}

func BenchmarkSignTransaction(b *testing.B) {
	wallet, _ := generateTestWallet()
	recipient := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	instruction := wallet.CreateTransferInstruction(recipient, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, _ := solana.NewTransaction(
			[]solana.Instruction{instruction},
			solana.Hash{},
			solana.TransactionPayer(wallet.publicKey),
		)
		_ = wallet.SignTransaction(tx)
	}
}

// Example test demonstrating wallet usage
func ExampleWallet_usage() {
	// Generate a keypair for demonstration
	_, privateKey, _ := ed25519.GenerateKey(nil)
	privateKeyBase58 := base58.Encode(privateKey)

	// Load wallet
	wallet, err := LoadWallet(privateKeyBase58, testRPCURL)
	if err != nil {
		panic(err)
	}

	// Get address
	address := wallet.GetAddress()
	println("Wallet address:", address)

	// Get SOL balance (requires network connection)
	ctx := context.Background()
	balance, err := wallet.GetSOLBalance(ctx)
	if err != nil {
		panic(err)
	}
	println("SOL balance:", balance.String())

	// Get USDT balance
	usdtBalance, err := wallet.GetUSDTBalance(ctx, testUSDTMint)
	if err != nil {
		panic(err)
	}
	println("USDT balance:", usdtBalance.String())
}
