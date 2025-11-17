package solana

import (
	"context"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests the creation of a new Solana RPC client
func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      ClientConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with custom timeout",
			config: ClientConfig{
				RPCURL:  "https://api.devnet.solana.com",
				Timeout: 60 * time.Second,
			},
			expectError: false,
		},
		{
			name: "valid config with default timeout",
			config: ClientConfig{
				RPCURL: "https://api.devnet.solana.com",
			},
			expectError: false,
		},
		{
			name: "empty RPC URL",
			config: ClientConfig{
				RPCURL: "",
			},
			expectError: true,
			errorMsg:    "RPC URL cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.rpcClient)
				assert.Equal(t, tt.config.RPCURL, client.rpcURL)

				// Check timeout
				if tt.config.Timeout == 0 {
					assert.Equal(t, 30*time.Second, client.timeout)
				} else {
					assert.Equal(t, tt.config.Timeout, client.timeout)
				}
			}
		})
	}
}

// TestNewClientWithURL tests the convenience constructor
func TestNewClientWithURL(t *testing.T) {
	tests := []struct {
		name        string
		rpcURL      string
		expectError bool
	}{
		{
			name:        "valid URL",
			rpcURL:      "https://api.devnet.solana.com",
			expectError: false,
		},
		{
			name:        "empty URL",
			rpcURL:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientWithURL(tt.rpcURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, 30*time.Second, client.timeout) // Default timeout
			}
		})
	}
}

// TestGetSlot tests getting the current slot
func TestGetSlot(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	slot, err := client.GetSlot(ctx, rpc.CommitmentFinalized)
	assert.NoError(t, err)
	assert.Greater(t, slot, uint64(0))

	t.Logf("Current slot: %d", slot)
}

// TestGetBalance tests getting SOL balance
func TestGetBalance(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	// Test with a known devnet address (you can replace with any valid address)
	// Using the system program address which always exists
	systemProgram := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	balance, err := client.GetBalance(ctx, systemProgram)
	assert.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))

	t.Logf("Balance: %s SOL", balance.String())
}

// TestClientHealthCheck tests the health check functionality
func TestClientHealthCheck(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tests := []struct {
		name        string
		rpcURL      string
		expectError bool
	}{
		{
			name:        "valid devnet endpoint",
			rpcURL:      "https://api.devnet.solana.com",
			expectError: false,
		},
		{
			name:        "invalid endpoint",
			rpcURL:      "https://invalid-solana-endpoint-12345.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientWithURL(tt.rpcURL)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = client.HealthCheck(ctx)

			if tt.expectError {
				assert.Error(t, err)
				t.Logf("Expected error: %v", err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestHealthCheckWithRetry tests health check with retry logic
func TestHealthCheckWithRetry(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	// Test with valid endpoint (should succeed on first try)
	err = client.HealthCheckWithRetry(ctx, 3)
	assert.NoError(t, err)

	// Test with invalid endpoint (should fail after retries)
	invalidClient, err := NewClientWithURL("https://invalid-solana-endpoint-12345.com")
	require.NoError(t, err)

	err = invalidClient.HealthCheckWithRetry(ctx, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after")
}

// TestGetVersion tests getting Solana node version
func TestGetVersion(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	version, err := client.GetVersion(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, version)
	assert.NotEmpty(t, version.SolanaCore)

	t.Logf("Solana version: %s", version.SolanaCore)
}

// TestGetRecentBlockhash tests getting recent blockhash
func TestGetRecentBlockhash(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	blockhash, err := client.GetRecentBlockhash(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, blockhash)

	t.Logf("Recent blockhash: %s", blockhash.String())
}

// TestGetTransaction tests retrieving transaction details
func TestGetTransaction(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	// Test with an invalid signature (all zeros)
	invalidSig := solana.Signature{}
	_, err = client.GetTransaction(ctx, invalidSig)
	assert.Error(t, err)

	// To properly test GetTransaction, you would need a valid transaction signature
	// from devnet. This is left as an integration test that can be run manually
	t.Log("Note: Full transaction test requires a valid devnet transaction signature")
}

// TestGetSignatureStatus tests getting signature status
func TestGetSignatureStatus(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	// Test with invalid signature
	invalidSig := solana.Signature{}
	_, err = client.GetSignatureStatus(ctx, invalidSig)
	assert.Error(t, err)

	t.Log("Note: Full status test requires a valid devnet transaction signature")
}

// TestGetConfirmations tests getting transaction confirmations
func TestGetConfirmations(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	ctx := context.Background()

	// Test with invalid signature
	invalidSig := solana.Signature{}
	_, err = client.GetConfirmations(ctx, invalidSig)
	assert.Error(t, err)

	t.Log("Note: Full confirmations test requires a valid devnet transaction signature")
}

// TestWaitForConfirmation tests waiting for transaction confirmation
func TestWaitForConfirmation(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	// Test with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	invalidSig := solana.Signature{}
	err = client.WaitForConfirmation(ctx, invalidSig, rpc.CommitmentFinalized, 2)
	assert.Error(t, err)

	t.Log("Note: Full wait test requires a valid devnet transaction signature")
}

// TestClientGetters tests the getter methods
func TestClientGetters(t *testing.T) {
	rpcURL := "https://api.devnet.solana.com"
	timeout := 45 * time.Second

	client, err := NewClient(ClientConfig{
		RPCURL:  rpcURL,
		Timeout: timeout,
	})
	require.NoError(t, err)

	assert.Equal(t, rpcURL, client.GetRPCURL())
	assert.Equal(t, timeout, client.GetTimeout())
	assert.NotNil(t, client.GetRPCClient())
}

// TestSetTimeout tests setting a new timeout
func TestSetTimeout(t *testing.T) {
	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	// Initial timeout should be 30 seconds (default)
	assert.Equal(t, 30*time.Second, client.GetTimeout())

	// Set new timeout
	newTimeout := 60 * time.Second
	client.SetTimeout(newTimeout)
	assert.Equal(t, newTimeout, client.GetTimeout())

	// Try to set invalid timeout (0 or negative)
	client.SetTimeout(0)
	assert.Equal(t, newTimeout, client.GetTimeout()) // Should remain unchanged

	client.SetTimeout(-5 * time.Second)
	assert.Equal(t, newTimeout, client.GetTimeout()) // Should remain unchanged
}

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(t, err)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// These operations should fail quickly due to cancelled context
	_, err = client.GetSlot(ctx, rpc.CommitmentFinalized)
	assert.Error(t, err)

	systemProgram := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	_, err = client.GetBalance(ctx, systemProgram)
	assert.Error(t, err)

	err = client.HealthCheck(ctx)
	assert.Error(t, err)
}

// TestClientTimeout tests that operations timeout correctly
func TestClientTimeout(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create client with very short timeout
	client, err := NewClient(ClientConfig{
		RPCURL:  "https://api.devnet.solana.com",
		Timeout: 1 * time.Nanosecond, // Extremely short timeout
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Operations should timeout
	_, err = client.GetSlot(ctx, rpc.CommitmentFinalized)
	assert.Error(t, err)
	t.Logf("Timeout error: %v", err)
}

// Benchmark tests
func BenchmarkGetSlot(b *testing.B) {
	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetSlot(ctx, rpc.CommitmentFinalized)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	client, err := NewClientWithURL("https://api.devnet.solana.com")
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.HealthCheck(ctx)
	}
}
