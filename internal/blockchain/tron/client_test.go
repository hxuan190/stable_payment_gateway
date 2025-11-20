package tron

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		config    ClientConfig
		expectErr bool
	}{
		{
			name: "valid testnet config",
			config: ClientConfig{
				RPCURL:    "grpc.shasta.trongrid.io:50051",
				Timeout:   30 * time.Second,
				IsMainnet: false,
			},
			expectErr: false,
		},
		{
			name: "empty RPC URL",
			config: ClientConfig{
				RPCURL:    "",
				Timeout:   30 * time.Second,
				IsMainnet: false,
			},
			expectErr: true,
		},
		{
			name: "default timeout",
			config: ClientConfig{
				RPCURL:    "grpc.shasta.trongrid.io:50051",
				IsMainnet: false,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					assert.Equal(t, tt.config.RPCURL, client.GetRPCURL())
					if tt.config.Timeout > 0 {
						assert.Equal(t, tt.config.Timeout, client.GetTimeout())
					} else {
						assert.Equal(t, 30*time.Second, client.GetTimeout())
					}
					client.Close()
				}
			}
		})
	}
}

func TestNewClientWithURL(t *testing.T) {
	client, err := NewClientWithURL("grpc.shasta.trongrid.io:50051", false)
	require.NoError(t, err)
	require.NotNil(t, client)

	assert.Equal(t, "grpc.shasta.trongrid.io:50051", client.GetRPCURL())
	assert.Equal(t, 30*time.Second, client.GetTimeout())
	assert.False(t, client.IsMainnet())

	client.Close()
}

// TestHealthCheck tests connection to TRON testnet
// This is an integration test - requires network connection
func TestHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("grpc.shasta.trongrid.io:50051", false)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.HealthCheck(ctx)
	assert.NoError(t, err, "Health check should pass for testnet")
}

// TestGetCurrentBlockNumber tests retrieving current block
// This is an integration test - requires network connection
func TestGetCurrentBlockNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("grpc.shasta.trongrid.io:50051", false)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	blockNumber, err := client.GetCurrentBlockNumber(ctx)
	assert.NoError(t, err)
	assert.Greater(t, blockNumber, int64(0), "Block number should be greater than 0")
}

// TestGetChainInfo tests retrieving chain information
// This is an integration test - requires network connection
func TestGetChainInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("grpc.shasta.trongrid.io:50051", false)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chainInfo, err := client.GetChainInfo(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, chainInfo)
	assert.Greater(t, chainInfo.BlockNumber, int64(0))
	assert.NotEmpty(t, chainInfo.BlockHash)
}

func TestClientSetTimeout(t *testing.T) {
	client, err := NewClientWithURL("grpc.shasta.trongrid.io:50051", false)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// Test setting valid timeout
	newTimeout := 60 * time.Second
	client.SetTimeout(newTimeout)
	assert.Equal(t, newTimeout, client.GetTimeout())

	// Test setting zero timeout (should not change)
	originalTimeout := client.GetTimeout()
	client.SetTimeout(0)
	assert.Equal(t, originalTimeout, client.GetTimeout())
}

func TestValidateAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClientWithURL("grpc.shasta.trongrid.io:50051", false)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	tests := []struct {
		name      string
		address   string
		expectErr bool
		isValid   bool
	}{
		{
			name:      "valid testnet address",
			address:   "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8",
			expectErr: false,
			isValid:   true,
		},
		{
			name:      "invalid address format",
			address:   "invalid_address",
			expectErr: false,
			isValid:   false,
		},
		{
			name:      "empty address",
			address:   "",
			expectErr: false,
			isValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid, err := client.ValidateAddress(tt.address)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.isValid, isValid)
			}
		})
	}
}
