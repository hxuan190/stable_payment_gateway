package blockchain_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stable_payment_gateway/internal/adapters/blockchain"
	"stable_payment_gateway/internal/ports"
	"stable_payment_gateway/internal/shared/events"
)

// TestSolanaListenerAdapter_Integration tests the Solana adapter with devnet
func TestSolanaListenerAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get configuration from environment
	rpcURL := os.Getenv("SOLANA_RPC_URL")
	privateKey := os.Getenv("SOLANA_WALLET_PRIVATE_KEY")

	if rpcURL == "" || privateKey == "" {
		t.Skip("SOLANA_RPC_URL and SOLANA_WALLET_PRIVATE_KEY must be set for integration tests")
	}

	// Create adapter
	adapter, err := blockchain.NewSolanaListenerAdapter(ports.BlockchainListenerConfig{
		BlockchainType:   ports.BlockchainTypeSolana,
		RPCURL:           rpcURL,
		WalletPrivateKey: privateKey,
		SupportedTokens: map[string]string{
			// Devnet USDC mint
			"USDC": "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU",
		},
		PollIntervalSeconds:   5,
		RequiredConfirmations: 1,
		MaxRetries:            3,
	})

	require.NoError(t, err, "Failed to create Solana adapter")

	// Start the adapter
	ctx := context.Background()
	err = adapter.Start(ctx)
	require.NoError(t, err, "Failed to start adapter")

	defer func() {
		err := adapter.Stop(ctx)
		assert.NoError(t, err, "Failed to stop adapter")
	}()

	// Verify adapter is running
	assert.True(t, adapter.IsRunning(), "Adapter should be running")

	// Verify blockchain type
	assert.Equal(t, ports.BlockchainTypeSolana, adapter.GetBlockchainType())

	// Verify wallet address is set
	walletAddress := adapter.GetWalletAddress()
	assert.NotEmpty(t, walletAddress, "Wallet address should not be empty")

	// Verify supported tokens
	supportedTokens := adapter.GetSupportedTokens()
	assert.Contains(t, supportedTokens, "USDC", "Should support USDC")

	// Check health
	health := adapter.GetListenerHealth()
	assert.True(t, health.IsHealthy, "Adapter should be healthy")
	assert.Equal(t, "running", health.ConnectionStatus)

	// Let it run for a few seconds to ensure no crashes
	time.Sleep(10 * time.Second)

	// Verify still running
	assert.True(t, adapter.IsRunning(), "Adapter should still be running")
}

// TestBSCListenerAdapter_Integration tests the BSC adapter with testnet
func TestBSCListenerAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get configuration from environment
	rpcURL := os.Getenv("BSC_RPC_URL")
	privateKey := os.Getenv("BSC_WALLET_PRIVATE_KEY")

	if rpcURL == "" || privateKey == "" {
		t.Skip("BSC_RPC_URL and BSC_WALLET_PRIVATE_KEY must be set for integration tests")
	}

	// Create adapter
	adapter, err := blockchain.NewBSCListenerAdapter(ports.BlockchainListenerConfig{
		BlockchainType:   ports.BlockchainTypeBSC,
		RPCURL:           rpcURL,
		WalletPrivateKey: privateKey,
		SupportedTokens: map[string]string{
			// BSC Testnet USDT
			"USDT": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd",
		},
		PollIntervalSeconds:   10,
		RequiredConfirmations: 15,
		MaxRetries:            3,
	})

	require.NoError(t, err, "Failed to create BSC adapter")

	// Start the adapter
	ctx := context.Background()
	err = adapter.Start(ctx)
	require.NoError(t, err, "Failed to start adapter")

	defer func() {
		err := adapter.Stop(ctx)
		assert.NoError(t, err, "Failed to stop adapter")
	}()

	// Verify adapter is running
	assert.True(t, adapter.IsRunning(), "Adapter should be running")

	// Verify blockchain type
	assert.Equal(t, ports.BlockchainTypeBSC, adapter.GetBlockchainType())

	// Verify wallet address is set
	walletAddress := adapter.GetWalletAddress()
	assert.NotEmpty(t, walletAddress, "Wallet address should not be empty")

	// Verify supported tokens
	supportedTokens := adapter.GetSupportedTokens()
	assert.Contains(t, supportedTokens, "USDT", "Should support USDT")

	// Check health
	health := adapter.GetListenerHealth()
	assert.True(t, health.IsHealthy, "Adapter should be healthy")

	// Let it run for a few seconds
	time.Sleep(10 * time.Second)

	// Verify still running
	assert.True(t, adapter.IsRunning(), "Adapter should still be running")
}

// TestEventBasedListenerAdapter tests event publishing
func TestEventBasedListenerAdapter(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create event bus
	eventBus := events.NewInMemoryEventBus(logger)

	// Create a mock listener
	mockListener := &MockBlockchainListener{
		blockchainType: ports.BlockchainTypeSolana,
		walletAddress:  "TestWalletAddress",
		supportedTokens: []string{"USDT", "USDC"},
	}

	// Create event-based adapter
	adapter := blockchain.NewEventBasedListenerAdapter(mockListener, eventBus)

	// Subscribe to payment confirmed events
	eventReceived := make(chan *events.PaymentConfirmedEvent, 1)
	eventBus.Subscribe("payment.confirmed", func(ctx context.Context, event events.Event) error {
		paymentEvent := event.(*events.PaymentConfirmedEvent)
		eventReceived <- paymentEvent
		return nil
	})

	// Start the adapter
	ctx := context.Background()
	err := adapter.Start(ctx)
	require.NoError(t, err)

	// Simulate a payment confirmation by triggering the handler
	if mockListener.handler != nil {
		confirmation := ports.PaymentConfirmation{
			PaymentID:      "test-payment-123",
			TxHash:         "test-tx-hash-abc",
			Amount:         decimal.NewFromInt(100),
			TokenSymbol:    "USDT",
			BlockchainType: ports.BlockchainTypeSolana,
			Sender:         "sender-address",
			Recipient:      "recipient-address",
			BlockNumber:    12345,
			Timestamp:      time.Now().Unix(),
		}

		err := mockListener.handler(ctx, confirmation)
		require.NoError(t, err)

		// Wait for event to be received
		select {
		case event := <-eventReceived:
			assert.Equal(t, "test-payment-123", event.PaymentID)
			assert.Equal(t, "test-tx-hash-abc", event.TxHash)
			assert.Equal(t, decimal.NewFromInt(100), event.Amount)
			assert.Equal(t, "USDT", event.TokenSymbol)
			assert.Equal(t, events.BlockchainTypeSolana, event.Blockchain)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for payment confirmed event")
		}
	}

	// Stop the adapter
	err = adapter.Stop(ctx)
	assert.NoError(t, err)

	// Shutdown event bus
	err = eventBus.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestListenerManager tests the listener manager
func TestListenerManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create event bus
	eventBus := events.NewInMemoryEventBus(logger)

	// Create listener manager
	manager := blockchain.NewListenerManager(eventBus, logger)

	// Create mock listeners
	solanaListener := &MockBlockchainListener{
		blockchainType:  ports.BlockchainTypeSolana,
		walletAddress:   "SolanaWalletAddress",
		supportedTokens: []string{"USDT", "USDC"},
	}

	bscListener := &MockBlockchainListener{
		blockchainType:  ports.BlockchainTypeBSC,
		walletAddress:   "0xBSCWalletAddress",
		supportedTokens: []string{"USDT", "BUSD"},
	}

	// Add listeners
	err := manager.AddListener(solanaListener)
	assert.NoError(t, err)

	err = manager.AddListener(bscListener)
	assert.NoError(t, err)

	// Verify listener count
	assert.Equal(t, 2, manager.GetListenerCount())

	// Start all listeners
	ctx := context.Background()
	err = manager.StartAll(ctx)
	assert.NoError(t, err)

	// Verify all listeners are running
	assert.True(t, solanaListener.isRunning)
	assert.True(t, bscListener.isRunning)

	// Get health status
	healthStatus := manager.GetHealthStatus()
	assert.Len(t, healthStatus, 2)
	assert.True(t, manager.IsAllHealthy())

	// Stop all listeners
	err = manager.StopAll(ctx)
	assert.NoError(t, err)

	// Verify all listeners are stopped
	assert.False(t, solanaListener.isRunning)
	assert.False(t, bscListener.isRunning)

	// Shutdown event bus
	err = eventBus.Shutdown(ctx)
	assert.NoError(t, err)
}

// MockBlockchainListener is a mock implementation for testing
type MockBlockchainListener struct {
	blockchainType  ports.BlockchainType
	walletAddress   string
	supportedTokens []string
	isRunning       bool
	handler         ports.PaymentConfirmationHandler
}

func (m *MockBlockchainListener) Start(ctx context.Context) error {
	m.isRunning = true
	return nil
}

func (m *MockBlockchainListener) Stop(ctx context.Context) error {
	m.isRunning = false
	return nil
}

func (m *MockBlockchainListener) IsRunning() bool {
	return m.isRunning
}

func (m *MockBlockchainListener) GetBlockchainType() ports.BlockchainType {
	return m.blockchainType
}

func (m *MockBlockchainListener) GetWalletAddress() string {
	return m.walletAddress
}

func (m *MockBlockchainListener) SetConfirmationHandler(handler ports.PaymentConfirmationHandler) {
	m.handler = handler
}

func (m *MockBlockchainListener) GetSupportedTokens() []string {
	return m.supportedTokens
}

func (m *MockBlockchainListener) GetListenerHealth() ports.ListenerHealth {
	return ports.ListenerHealth{
		IsHealthy:               m.isRunning,
		LastProcessedBlock:      12345,
		LastActivityTimestamp:   time.Now().Unix(),
		ErrorCount:              0,
		SuccessfulConfirmations: 10,
		ConnectionStatus:        "running",
	}
}
