package solana

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/shopspring/decimal"
)

// PaymentConfirmationCallback is called when a payment is confirmed
// paymentID: the payment ID from the transaction memo
// txHash: the transaction signature
// amount: the amount transferred in decimal form
type PaymentConfirmationCallback func(paymentID string, txHash string, amount decimal.Decimal, tokenMint string) error

// TransactionListener monitors Solana blockchain for incoming transactions
// to a specific wallet address
type TransactionListener struct {
	client               *Client
	wallet               *Wallet
	wsClient             *ws.Client
	wsURL                string
	confirmationCallback PaymentConfirmationCallback

	// Supported token mints for filtering
	supportedTokenMints  map[string]TokenMintInfo

	// Control channels
	ctx                  context.Context
	cancel               context.CancelFunc
	shutdownCh           chan struct{}
	wg                   sync.WaitGroup

	// Configuration
	pollInterval         time.Duration
	maxRetries           int

	// State
	isRunning            bool
	mu                   sync.RWMutex
}

// TokenMintInfo contains information about supported SPL tokens
type TokenMintInfo struct {
	MintAddress solana.PublicKey
	Symbol      string
	Decimals    uint8
}

// ListenerConfig holds configuration for the transaction listener
type ListenerConfig struct {
	Client               *Client
	Wallet               *Wallet
	WSURL                string
	ConfirmationCallback PaymentConfirmationCallback
	SupportedTokenMints  map[string]TokenMintInfo
	PollInterval         time.Duration
	MaxRetries           int
}

// NewTransactionListener creates a new transaction listener
func NewTransactionListener(config ListenerConfig) (*TransactionListener, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	if config.Wallet == nil {
		return nil, fmt.Errorf("wallet cannot be nil")
	}

	if config.ConfirmationCallback == nil {
		return nil, fmt.Errorf("confirmation callback cannot be nil")
	}

	// Set defaults
	pollInterval := config.PollInterval
	if pollInterval == 0 {
		pollInterval = 5 * time.Second
	}

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	// Derive WebSocket URL from RPC URL if not provided
	wsURL := config.WSURL
	if wsURL == "" {
		wsURL = deriveWSURL(config.Client.GetRPCURL())
	}

	ctx, cancel := context.WithCancel(context.Background())

	listener := &TransactionListener{
		client:               config.Client,
		wallet:               config.Wallet,
		wsURL:                wsURL,
		confirmationCallback: config.ConfirmationCallback,
		supportedTokenMints:  config.SupportedTokenMints,
		ctx:                  ctx,
		cancel:               cancel,
		shutdownCh:           make(chan struct{}),
		pollInterval:         pollInterval,
		maxRetries:           maxRetries,
		isRunning:            false,
	}

	return listener, nil
}

// deriveWSURL converts an RPC HTTP URL to a WebSocket URL
func deriveWSURL(rpcURL string) string {
	// Simple conversion: replace http(s) with ws(s)
	if len(rpcURL) >= 7 && rpcURL[:7] == "http://" {
		return "ws://" + rpcURL[7:]
	}
	if len(rpcURL) >= 8 && rpcURL[:8] == "https://" {
		return "wss://" + rpcURL[8:]
	}
	return rpcURL
}

// Start begins listening for transactions
func (l *TransactionListener) Start() error {
	l.mu.Lock()
	if l.isRunning {
		l.mu.Unlock()
		return fmt.Errorf("listener is already running")
	}
	l.isRunning = true
	l.mu.Unlock()

	// Start WebSocket listener goroutine
	l.wg.Add(1)
	go l.listenWebSocket()

	// Start polling fallback goroutine
	l.wg.Add(1)
	go l.pollTransactions()

	return nil
}

// Stop gracefully stops the listener
func (l *TransactionListener) Stop() error {
	l.mu.Lock()
	if !l.isRunning {
		l.mu.Unlock()
		return fmt.Errorf("listener is not running")
	}
	l.mu.Unlock()

	// Cancel context to signal shutdown
	l.cancel()

	// Close WebSocket client if connected
	if l.wsClient != nil {
		l.wsClient.Close()
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		l.mu.Lock()
		l.isRunning = false
		l.mu.Unlock()
		close(l.shutdownCh)
		return nil
	case <-time.After(10 * time.Second):
		l.mu.Lock()
		l.isRunning = false
		l.mu.Unlock()
		return fmt.Errorf("listener shutdown timeout")
	}
}

// IsRunning returns whether the listener is currently running
func (l *TransactionListener) IsRunning() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.isRunning
}

// listenWebSocket listens for transactions via WebSocket subscription
func (l *TransactionListener) listenWebSocket() {
	defer l.wg.Done()

	retryDelay := 2 * time.Second
	maxRetryDelay := 60 * time.Second

	for {
		select {
		case <-l.ctx.Done():
			return
		default:
		}

		// Create WebSocket client
		wsClient, err := ws.Connect(l.ctx, l.wsURL)
		if err != nil {
			fmt.Printf("Failed to connect to WebSocket: %v, retrying in %v\n", err, retryDelay)
			time.Sleep(retryDelay)
			retryDelay = min(retryDelay*2, maxRetryDelay)
			continue
		}

		l.mu.Lock()
		l.wsClient = wsClient
		l.mu.Unlock()

		// Reset retry delay on successful connection
		retryDelay = 2 * time.Second

		// Subscribe to account changes (this will notify us of incoming transactions)
		sub, err := wsClient.AccountSubscribe(
			l.wallet.GetPublicKey(),
			rpc.CommitmentFinalized,
		)
		if err != nil {
			fmt.Printf("Failed to subscribe to account: %v, retrying...\n", err)
			wsClient.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		// Listen for notifications
		for {
			select {
			case <-l.ctx.Done():
				sub.Unsubscribe()
				wsClient.Close()
				return
			default:
			}

			// Receive notification
			_, err := sub.Recv()
			if err != nil {
				fmt.Printf("WebSocket receive error: %v, reconnecting...\n", err)
				sub.Unsubscribe()
				wsClient.Close()
				break // Break inner loop to reconnect
			}

			// When we receive a notification, fetch recent transactions
			l.fetchAndProcessRecentTransactions()
		}
	}
}

// pollTransactions periodically polls for new transactions as a fallback
func (l *TransactionListener) pollTransactions() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			l.fetchAndProcessRecentTransactions()
		}
	}
}

// fetchAndProcessRecentTransactions fetches and processes recent transactions
func (l *TransactionListener) fetchAndProcessRecentTransactions() {
	ctx, cancel := context.WithTimeout(l.ctx, 30*time.Second)
	defer cancel()

	// Get recent transaction signatures for the wallet
	sigs, err := l.client.GetRPCClient().GetSignaturesForAddress(
		ctx,
		l.wallet.GetPublicKey(),
	)
	if err != nil {
		fmt.Printf("Failed to get signatures: %v\n", err)
		return
	}

	if sigs == nil {
		return
	}

	// Process each signature (most recent first)
	for _, sig := range sigs {
		if sig.Err != nil {
			// Skip failed transactions
			continue
		}

		// Only process finalized transactions
		if sig.ConfirmationStatus != rpc.ConfirmationStatusFinalized {
			continue
		}

		// Process the transaction
		l.handleTransaction(sig.Signature)
	}
}

// handleTransaction processes a single transaction
func (l *TransactionListener) handleTransaction(signature solana.Signature) {
	ctx, cancel := context.WithTimeout(l.ctx, 30*time.Second)
	defer cancel()

	// Get transaction details with retries
	var txInfo *TransactionInfo
	var err error

	for i := 0; i < l.maxRetries; i++ {
		txInfo, err = l.client.GetTransaction(ctx, signature)
		if err == nil {
			break
		}

		if i < l.maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		fmt.Printf("Failed to get transaction %s after %d retries: %v\n", signature, l.maxRetries, err)
		return
	}

	// Check if transaction is finalized
	if !txInfo.IsFinalized {
		// Wait for finalization
		err = l.client.WaitForConfirmation(ctx, signature, rpc.CommitmentFinalized, 30)
		if err != nil {
			fmt.Printf("Transaction %s not finalized: %v\n", signature, err)
			return
		}
	}

	// Check if transaction succeeded
	if txInfo.Error != nil {
		fmt.Printf("Transaction %s failed: %v\n", signature, txInfo.Error)
		return
	}

	// Parse the transaction to extract payment details
	paymentDetails, err := l.parsePaymentTransaction(txInfo)
	if err != nil {
		// Not a payment transaction or parsing error
		return
	}

	// Verify the payment is to our wallet
	if paymentDetails.Recipient != l.wallet.GetPublicKey() {
		return
	}

	// Call the confirmation callback
	err = l.confirmationCallback(
		paymentDetails.PaymentID,
		signature.String(),
		paymentDetails.Amount,
		paymentDetails.TokenMint,
	)

	if err != nil {
		fmt.Printf("Payment confirmation callback failed for %s: %v\n", signature, err)
	}
}

// PaymentDetails contains extracted payment information from a transaction
type PaymentDetails struct {
	Recipient  solana.PublicKey
	Sender     solana.PublicKey
	Amount     decimal.Decimal
	TokenMint  string
	PaymentID  string
	TxHash     string
}

// parsePaymentTransaction extracts payment details from a transaction
func (l *TransactionListener) parsePaymentTransaction(txInfo *TransactionInfo) (*PaymentDetails, error) {
	if txInfo == nil || txInfo.Transaction == nil {
		return nil, fmt.Errorf("invalid transaction info")
	}

	// Extract memo from transaction
	memo, err := extractMemoFromTransaction(txInfo.Transaction)
	if err != nil {
		return nil, fmt.Errorf("no memo found: %w", err)
	}

	// Parse SPL token transfer
	transfer, err := parseSPLTokenTransfer(txInfo.Transaction, l.wallet.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to parse SPL token transfer: %w", err)
	}

	// Verify token is supported
	tokenMintStr := transfer.TokenMint.String()
	tokenInfo, supported := l.isSupportedToken(transfer.TokenMint)
	if !supported {
		return nil, fmt.Errorf("unsupported token: %s", tokenMintStr)
	}

	// Convert amount based on token decimals
	divisor := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(tokenInfo.Decimals)))
	amount := decimal.NewFromBigInt(transfer.Amount, 0).Div(divisor)

	details := &PaymentDetails{
		Recipient:  transfer.Recipient,
		Sender:     transfer.Sender,
		Amount:     amount,
		TokenMint:  tokenInfo.Symbol,
		PaymentID:  memo,
		TxHash:     txInfo.Signature.String(),
	}

	return details, nil
}

// isSupportedToken checks if a token mint is supported
func (l *TransactionListener) isSupportedToken(mint solana.PublicKey) (TokenMintInfo, bool) {
	for _, tokenInfo := range l.supportedTokenMints {
		if tokenInfo.MintAddress.Equals(mint) {
			return tokenInfo, true
		}
	}
	return TokenMintInfo{}, false
}

// GetWalletAddress returns the wallet address being monitored
func (l *TransactionListener) GetWalletAddress() string {
	return l.wallet.GetAddress()
}

// min returns the minimum of two durations
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
