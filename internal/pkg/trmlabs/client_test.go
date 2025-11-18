package trmlabs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockClient(t *testing.T) {
	client := NewMockClient()
	assert.NotNil(t, client)
	assert.NotNil(t, client.SanctionedAddresses)
	assert.NotNil(t, client.HighRiskAddresses)
	assert.False(t, client.SimulateError)
}

func TestMockClient_ScreenAddress_SafeAddress(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	result, err := client.ScreenAddress(ctx, "SafeWalletAddress123", "solana")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "SafeWalletAddress123", result.Address)
	assert.Equal(t, "solana", result.Chain)
	assert.Equal(t, 0, result.RiskScore)
	assert.False(t, result.IsSanctioned)
	assert.Empty(t, result.Flags)
}

func TestMockClient_ScreenAddress_SanctionedAddress(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	result, err := client.ScreenAddress(ctx, "TornadoCash123456789", "solana")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.RiskScore)
	assert.True(t, result.IsSanctioned)
	assert.Contains(t, result.Flags, "sanctions")
	assert.Contains(t, result.Flags, "ofac")
}

func TestMockClient_ScreenAddress_HighRiskAddress(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	result, err := client.ScreenAddress(ctx, "MixerWallet123", "solana")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 85, result.RiskScore)
	assert.False(t, result.IsSanctioned)
	assert.Contains(t, result.Flags, "mixer")
	assert.Contains(t, result.Flags, "high-risk")
}

func TestMockClient_ScreenAddress_MixerPattern(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	result, err := client.ScreenAddress(ctx, "MyMixerWallet456", "bsc")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.RiskScore, 50)
	assert.Contains(t, result.Flags, "mixer")
}

func TestMockClient_ScreenAddress_TornadoPattern(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	result, err := client.ScreenAddress(ctx, "TornadoWallet789", "ethereum")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.RiskScore, 90)
	assert.Contains(t, result.Flags, "tornado-cash")
	assert.Contains(t, result.Flags, "mixer")
}

func TestMockClient_ScreenAddress_DarknetPattern(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	result, err := client.ScreenAddress(ctx, "DarknetAddress123", "solana")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.RiskScore, 80)
	assert.Contains(t, result.Flags, "darknet")
}

func TestMockClient_ScreenAddress_SimulateError(t *testing.T) {
	client := NewMockClient()
	client.SimulateError = true
	ctx := context.Background()

	result, err := client.ScreenAddress(ctx, "AnyAddress", "solana")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrAPIUnavailable, err)
}

func TestMockClient_AddSanctionedAddress(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Initially safe
	result, err := client.ScreenAddress(ctx, "NewSanctionedAddress", "solana")
	require.NoError(t, err)
	assert.Equal(t, 0, result.RiskScore)
	assert.False(t, result.IsSanctioned)

	// Add to sanctioned list
	client.AddSanctionedAddress("NewSanctionedAddress")

	// Now should be flagged
	result, err = client.ScreenAddress(ctx, "NewSanctionedAddress", "solana")
	require.NoError(t, err)
	assert.Equal(t, 100, result.RiskScore)
	assert.True(t, result.IsSanctioned)
}

func TestMockClient_AddHighRiskAddress(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Initially safe
	result, err := client.ScreenAddress(ctx, "NewHighRiskAddress", "solana")
	require.NoError(t, err)
	assert.Equal(t, 0, result.RiskScore)

	// Add to high-risk list
	client.AddHighRiskAddress("NewHighRiskAddress")

	// Now should be flagged as high risk
	result, err = client.ScreenAddress(ctx, "NewHighRiskAddress", "solana")
	require.NoError(t, err)
	assert.Equal(t, 85, result.RiskScore)
	assert.Contains(t, result.Flags, "high-risk")
}

func TestMockClient_BatchScreenAddresses(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	addresses := []string{
		"SafeAddress1",
		"TornadoCash123456789",
		"MixerWallet123",
		"SafeAddress2",
	}

	results, err := client.BatchScreenAddresses(ctx, addresses, "solana")
	require.NoError(t, err)
	assert.Len(t, results, 4)

	// Check safe address
	assert.Equal(t, 0, results["SafeAddress1"].RiskScore)
	assert.False(t, results["SafeAddress1"].IsSanctioned)

	// Check sanctioned address
	assert.Equal(t, 100, results["TornadoCash123456789"].RiskScore)
	assert.True(t, results["TornadoCash123456789"].IsSanctioned)

	// Check high-risk address
	assert.Equal(t, 85, results["MixerWallet123"].RiskScore)
	assert.False(t, results["MixerWallet123"].IsSanctioned)
}

func TestNormalizeChainName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"solana", "solana"},
		{"Solana", "solana"},
		{"SOLANA", "solana"},
		{"bsc", "binance-smart-chain"},
		{"BSC", "binance-smart-chain"},
		{"binance-smart-chain", "binance-smart-chain"},
		{"ethereum", "ethereum"},
		{"Ethereum", "ethereum"},
		{"eth", "ethereum"},
		{"ETH", "ethereum"},
		{"polygon", "polygon"},
		{"matic", "polygon"},
		{"unknown-chain", "unknown-chain"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeChainName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewClient(t *testing.T) {
	// Test with custom base URL
	client := NewClient("test-api-key", "https://custom.api.com")
	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, "https://custom.api.com", client.baseURL)

	// Test with empty base URL (should use default)
	client = NewClient("test-api-key", "")
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.trmlabs.com/public/v1", client.baseURL)
}
