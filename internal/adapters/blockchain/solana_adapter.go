package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/shopspring/decimal"

	"stable_payment_gateway/internal/blockchain/solana"
	"stable_payment_gateway/internal/ports"
)

// SolanaListenerAdapter adapts the existing Solana TransactionListener to implement BlockchainListener interface
// This implements the ADAPTER pattern in Ports & Adapters (Hexagonal Architecture)
type SolanaListenerAdapter struct {
	// Underlying Solana listener implementation
	listener *solana.TransactionListener

	// Configuration
	config        ports.BlockchainListenerConfig
	client        *solana.Client
	wallet        *solana.Wallet

	// Confirmation handler
	confirmationHandler ports.PaymentConfirmationHandler
	handlerMu           sync.RWMutex

	// Health tracking
	health           ports.ListenerHealth
	healthMu         sync.RWMutex
	lastActivity     time.Time
	errorCount       uint64
	successCount     uint64
}

// NewSolanaListenerAdapter creates a new Solana blockchain listener adapter
func NewSolanaListenerAdapter(config ports.BlockchainListenerConfig) (*SolanaListenerAdapter, error) {
	if config.BlockchainType != ports.BlockchainTypeSolana {
		return nil, fmt.Errorf("invalid blockchain type: expected %s, got %s", ports.BlockchainTypeSolana, config.BlockchainType)
	}

	// Create Solana client
	client, err := solana.NewClient(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	// Create wallet from private key
	wallet, err := solana.NewWalletFromPrivateKey(config.WalletPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	adapter := &SolanaListenerAdapter{
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

// Start begins listening for transactions on the Solana blockchain
func (a *SolanaListenerAdapter) Start(ctx context.Context) error {
	// Build supported token mints map
	supportedTokenMints := make(map[string]solana.TokenMintInfo)
	for symbol, mintAddress := range a.config.SupportedTokens {
		pubKey, err := solana.PublicKeyFromString(mintAddress)
		if err != nil {
			return fmt.Errorf("invalid mint address for %s: %w", symbol, err)
		}

		// Determine decimals based on token (USDT/USDC = 6 decimals on Solana)
		decimals := uint8(6)

		supportedTokenMints[symbol] = solana.TokenMintInfo{
			MintAddress: pubKey,
			Symbol:      symbol,
			Decimals:    decimals,
		}
	}

	// Set poll interval
	pollInterval := time.Duration(a.config.PollIntervalSeconds) * time.Second
	if pollInterval == 0 {
		pollInterval = 5 * time.Second
	}

	// Create the underlying Solana listener with our adapter's callback
	listenerConfig := solana.ListenerConfig{
		Client:               a.client,
		Wallet:               a.wallet,
		WSURL:                "", // Will be derived from RPC URL
		ConfirmationCallback: a.handlePaymentConfirmation,
		SupportedTokenMints:  supportedTokenMints,
		PollInterval:         pollInterval,
		MaxRetries:           a.config.MaxRetries,
	}

	listener, err := solana.NewTransactionListener(listenerConfig)
	if err != nil {
		a.updateHealth(false, "failed to create listener")
		return fmt.Errorf("failed to create Solana transaction listener: %w", err)
	}

	a.listener = listener

	// Start the listener
	if err := a.listener.Start(); err != nil {
		a.updateHealth(false, "failed to start listener")
		return fmt.Errorf("failed to start Solana listener: %w", err)
	}

	a.updateHealth(true, "running")
	return nil
}

// Stop gracefully stops the listener and cleans up resources
func (a *SolanaListenerAdapter) Stop(ctx context.Context) error {
	if a.listener == nil {
		return fmt.Errorf("listener not initialized")
	}

	err := a.listener.Stop()
	if err != nil {
		a.updateHealth(false, "failed to stop")
		return fmt.Errorf("failed to stop Solana listener: %w", err)
	}

	a.updateHealth(false, "stopped")
	return nil
}

// IsRunning returns whether the listener is currently active
func (a *SolanaListenerAdapter) IsRunning() bool {
	if a.listener == nil {
		return false
	}
	return a.listener.IsRunning()
}

// GetBlockchainType returns the type of blockchain this listener monitors
func (a *SolanaListenerAdapter) GetBlockchainType() ports.BlockchainType {
	return ports.BlockchainTypeSolana
}

// GetWalletAddress returns the wallet address being monitored
func (a *SolanaListenerAdapter) GetWalletAddress() string {
	return a.wallet.GetAddress()
}

// SetConfirmationHandler sets the callback for when payments are confirmed
func (a *SolanaListenerAdapter) SetConfirmationHandler(handler ports.PaymentConfirmationHandler) {
	a.handlerMu.Lock()
	defer a.handlerMu.Unlock()
	a.confirmationHandler = handler
}

// GetSupportedTokens returns a list of token symbols supported by this listener
func (a *SolanaListenerAdapter) GetSupportedTokens() []string {
	tokens := make([]string, 0, len(a.config.SupportedTokens))
	for symbol := range a.config.SupportedTokens {
		tokens = append(tokens, symbol)
	}
	return tokens
}

// GetListenerHealth returns health status and metrics
func (a *SolanaListenerAdapter) GetListenerHealth() ports.ListenerHealth {
	a.healthMu.RLock()
	defer a.healthMu.RUnlock()

	health := a.health
	health.LastActivityTimestamp = a.lastActivity.Unix()
	health.ErrorCount = a.errorCount
	health.SuccessfulConfirmations = a.successCount

	return health
}

// handlePaymentConfirmation is the adapter function that converts Solana-specific callback to the generic port interface
func (a *SolanaListenerAdapter) handlePaymentConfirmation(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error {
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

	// Create generic payment confirmation from Solana-specific data
	confirmation := ports.PaymentConfirmation{
		PaymentID:      paymentID,
		TxHash:         txHash,
		Amount:         amount,
		TokenSymbol:    tokenMint,
		BlockchainType: ports.BlockchainTypeSolana,
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
func (a *SolanaListenerAdapter) updateHealth(isHealthy bool, status string) {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()

	a.health.IsHealthy = isHealthy
	a.health.ConnectionStatus = status
}

// incrementErrorCount increments the error counter
func (a *SolanaListenerAdapter) incrementErrorCount() {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	a.errorCount++
}

// incrementSuccessCount increments the success counter
func (a *SolanaListenerAdapter) incrementSuccessCount() {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	a.successCount++
}
