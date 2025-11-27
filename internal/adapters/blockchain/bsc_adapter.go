package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/bsc"
	"github.com/hxuan190/stable_payment_gateway/internal/ports"
)

// BSCListenerAdapter adapts the existing BSC TransactionListener to implement BlockchainListener interface
// This implements the ADAPTER pattern in Ports & Adapters (Hexagonal Architecture)
type BSCListenerAdapter struct {
	// Underlying BSC listener implementation
	listener *bsc.TransactionListener

	// Configuration
	config ports.BlockchainListenerConfig
	client *bsc.Client
	wallet *bsc.Wallet

	// Confirmation handler
	confirmationHandler ports.PaymentConfirmationHandler
	handlerMu           sync.RWMutex

	// Health tracking
	health       ports.ListenerHealth
	healthMu     sync.RWMutex
	lastActivity time.Time
	errorCount   uint64
	successCount uint64
}

// NewBSCListenerAdapter creates a new BSC blockchain listener adapter
func NewBSCListenerAdapter(config ports.BlockchainListenerConfig) (*BSCListenerAdapter, error) {
	if config.BlockchainType != ports.BlockchainTypeBSC {
		return nil, fmt.Errorf("invalid blockchain type: expected %s, got %s", ports.BlockchainTypeBSC, config.BlockchainType)
	}

	// Create BSC client
	client, err := bsc.NewClient(bsc.ClientConfig{
		RPCURL: config.RPCURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create BSC client: %w", err)
	}

	// Create wallet from private key
	wallet, err := bsc.LoadWallet(config.WalletPrivateKey, config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	adapter := &BSCListenerAdapter{
		config:       config,
		client:       client,
		wallet:       wallet,
		lastActivity: time.Now(),
		health: ports.ListenerHealth{
			IsHealthy:        true,
			ConnectionStatus: "initialized",
		},
	}

	// Set confirmation handler if provided
	if config.ConfirmationHandler != nil {
		adapter.confirmationHandler = config.ConfirmationHandler
	}

	return adapter, nil
}

// Start begins listening for transactions on the BSC blockchain
func (a *BSCListenerAdapter) Start(ctx context.Context) error {
	// Build supported token contracts map
	supportedTokenContracts := make(map[string]bsc.TokenContractInfo)
	for symbol, contractAddress := range a.config.SupportedTokens {
		addr := common.HexToAddress(contractAddress)

		// Determine decimals based on token (most BEP20 tokens = 18 decimals)
		decimals := uint8(18)

		supportedTokenContracts[symbol] = bsc.TokenContractInfo{
			ContractAddress: addr,
			Symbol:          symbol,
			Decimals:        decimals,
		}
	}

	// Set poll interval
	pollInterval := time.Duration(a.config.PollIntervalSeconds) * time.Second
	if pollInterval == 0 {
		pollInterval = 10 * time.Second // BSC block time is ~3 seconds
	}

	// Set required confirmations
	requiredConfirmations := a.config.RequiredConfirmations
	if requiredConfirmations == 0 {
		requiredConfirmations = 15 // BSC finality is around 15 blocks
	}

	// Create the underlying BSC listener with our adapter's callback
	listenerConfig := bsc.ListenerConfig{
		Client:                  a.client,
		Wallet:                  a.wallet,
		ConfirmationCallback:    a.handlePaymentConfirmation,
		SupportedTokenContracts: supportedTokenContracts,
		PollInterval:            pollInterval,
		RequiredConfirmations:   requiredConfirmations,
		MaxRetries:              a.config.MaxRetries,
	}

	listener, err := bsc.NewTransactionListener(listenerConfig)
	if err != nil {
		a.updateHealth(false, "failed to create listener")
		return fmt.Errorf("failed to create BSC transaction listener: %w", err)
	}

	a.listener = listener

	// Start the listener
	if err := a.listener.Start(); err != nil {
		a.updateHealth(false, "failed to start listener")
		return fmt.Errorf("failed to start BSC listener: %w", err)
	}

	a.updateHealth(true, "running")
	return nil
}

// Stop gracefully stops the listener and cleans up resources
func (a *BSCListenerAdapter) Stop(ctx context.Context) error {
	if a.listener == nil {
		return fmt.Errorf("listener not initialized")
	}

	err := a.listener.Stop()
	if err != nil {
		a.updateHealth(false, "failed to stop")
		return fmt.Errorf("failed to stop BSC listener: %w", err)
	}

	a.updateHealth(false, "stopped")
	return nil
}

// IsRunning returns whether the listener is currently active
func (a *BSCListenerAdapter) IsRunning() bool {
	if a.listener == nil {
		return false
	}
	return a.listener.IsRunning()
}

// GetBlockchainType returns the type of blockchain this listener monitors
func (a *BSCListenerAdapter) GetBlockchainType() ports.BlockchainType {
	return ports.BlockchainTypeBSC
}

// GetWalletAddress returns the wallet address being monitored
func (a *BSCListenerAdapter) GetWalletAddress() string {
	return a.wallet.GetAddress()
}

// SetConfirmationHandler sets the callback for when payments are confirmed
func (a *BSCListenerAdapter) SetConfirmationHandler(handler ports.PaymentConfirmationHandler) {
	a.handlerMu.Lock()
	defer a.handlerMu.Unlock()
	a.confirmationHandler = handler
}

// GetSupportedTokens returns a list of token symbols supported by this listener
func (a *BSCListenerAdapter) GetSupportedTokens() []string {
	tokens := make([]string, 0, len(a.config.SupportedTokens))
	for symbol := range a.config.SupportedTokens {
		tokens = append(tokens, symbol)
	}
	return tokens
}

// GetListenerHealth returns health status and metrics
func (a *BSCListenerAdapter) GetListenerHealth() ports.ListenerHealth {
	a.healthMu.RLock()
	defer a.healthMu.RUnlock()

	health := a.health
	health.LastActivityTimestamp = a.lastActivity.Unix()
	health.ErrorCount = a.errorCount
	health.SuccessfulConfirmations = a.successCount

	return health
}

// handlePaymentConfirmation is the adapter function that converts BSC-specific callback to the generic port interface
func (a *BSCListenerAdapter) handlePaymentConfirmation(paymentID string, txHash string, amount decimal.Decimal, tokenSymbol string) error {
	a.handlerMu.RLock()
	handler := a.confirmationHandler
	a.handlerMu.RUnlock()

	if handler == nil {
		// No handler set, just log and return
		fmt.Printf("Warning: Payment confirmed but no handler set: %s\n", paymentID)
		return nil
	}

	// Update last activity
	a.lastActivity = time.Now()

	// Create generic payment confirmation from BSC-specific data
	confirmation := ports.PaymentConfirmation{
		PaymentID:      paymentID,
		TxHash:         txHash,
		Amount:         amount,
		TokenSymbol:    tokenSymbol,
		BlockchainType: ports.BlockchainTypeBSC,
		Recipient:      a.wallet.GetAddress(),
		// Sender is not easily available from the callback, would need to parse transaction
		// BlockNumber and Timestamp would require additional RPC calls
		// For MVP, we'll leave these empty and can enhance later
	}

	// Call the handler with context
	ctx := context.Background()
	err := handler(ctx, confirmation)

	if err != nil {
		a.incrementErrorCount()
		return fmt.Errorf("confirmation handler failed: %w", err)
	}

	a.incrementSuccessCount()
	return nil
}

// updateHealth updates the health status
func (a *BSCListenerAdapter) updateHealth(isHealthy bool, status string) {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()

	a.health.IsHealthy = isHealthy
	a.health.ConnectionStatus = status
}

// incrementErrorCount increments the error counter
func (a *BSCListenerAdapter) incrementErrorCount() {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	a.errorCount++
}

// incrementSuccessCount increments the success counter
func (a *BSCListenerAdapter) incrementSuccessCount() {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	a.successCount++
}
