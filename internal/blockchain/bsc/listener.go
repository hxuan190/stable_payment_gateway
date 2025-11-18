package bsc

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// PaymentConfirmationCallback is called when a payment is confirmed
// paymentID: the payment ID extracted from the transaction
// txHash: the transaction hash
// amount: the amount transferred in decimal form
// tokenSymbol: the token symbol (e.g., "USDT", "BUSD")
type PaymentConfirmationCallback func(paymentID string, txHash string, amount decimal.Decimal, tokenSymbol string) error

// TransactionListener monitors BSC blockchain for incoming BEP20 token transfers
// to a specific wallet address
type TransactionListener struct {
	client               *Client
	wallet               *Wallet
	confirmationCallback PaymentConfirmationCallback

	// Supported token contracts for filtering
	supportedTokenContracts map[string]TokenContractInfo

	// Control channels
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownCh chan struct{}
	wg         sync.WaitGroup

	// Configuration
	pollInterval          time.Duration
	requiredConfirmations uint64
	maxRetries            int

	// State
	isRunning          bool
	mu                 sync.RWMutex
	lastProcessedBlock uint64
	processedTxs       map[string]bool // Track processed transactions to avoid duplicates
	processedTxsMu     sync.RWMutex
}

// TokenContractInfo contains information about supported BEP20 tokens
type TokenContractInfo struct {
	ContractAddress common.Address
	Symbol          string
	Decimals        uint8
}

// ListenerConfig holds configuration for the transaction listener
type ListenerConfig struct {
	Client                  *Client
	Wallet                  *Wallet
	ConfirmationCallback    PaymentConfirmationCallback
	SupportedTokenContracts map[string]TokenContractInfo
	PollInterval            time.Duration
	RequiredConfirmations   uint64
	MaxRetries              int
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
		pollInterval = 10 * time.Second // BSC block time is ~3 seconds, poll every 10 seconds
	}

	requiredConfirmations := config.RequiredConfirmations
	if requiredConfirmations == 0 {
		requiredConfirmations = 15 // BSC finality is around 15 blocks
	}

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	ctx, cancel := context.WithCancel(context.Background())

	listener := &TransactionListener{
		client:                  config.Client,
		wallet:                  config.Wallet,
		confirmationCallback:    config.ConfirmationCallback,
		supportedTokenContracts: config.SupportedTokenContracts,
		ctx:                     ctx,
		cancel:                  cancel,
		shutdownCh:              make(chan struct{}),
		pollInterval:            pollInterval,
		requiredConfirmations:   requiredConfirmations,
		maxRetries:              maxRetries,
		isRunning:               false,
		processedTxs:            make(map[string]bool),
	}

	return listener, nil
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

	// Get current block number as starting point
	ctx := context.Background()
	currentBlock, err := l.client.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	l.lastProcessedBlock = currentBlock

	fmt.Printf("Starting BSC listener from block %d\n", currentBlock)

	// Start polling goroutine
	l.wg.Add(1)
	go l.pollBlocks()

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

// pollBlocks periodically polls for new blocks and processes transactions
func (l *TransactionListener) pollBlocks() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			l.processNewBlocks()
		}
	}
}

// processNewBlocks fetches and processes new blocks since the last processed block
func (l *TransactionListener) processNewBlocks() {
	ctx, cancel := context.WithTimeout(l.ctx, 60*time.Second)
	defer cancel()

	// Get current block number
	currentBlock, err := l.client.GetBlockNumber(ctx)
	if err != nil {
		fmt.Printf("Failed to get current block number: %v\n", err)
		return
	}

	// Process blocks from last processed to current
	// Limit the number of blocks to process at once to avoid overwhelming the system
	maxBlocksPerIteration := uint64(100)
	fromBlock := l.lastProcessedBlock + 1
	toBlock := currentBlock

	if toBlock-fromBlock > maxBlocksPerIteration {
		toBlock = fromBlock + maxBlocksPerIteration
	}

	if fromBlock > toBlock {
		// No new blocks to process
		return
	}

	fmt.Printf("Processing BSC blocks %d to %d\n", fromBlock, toBlock)

	// Process each block
	for blockNum := fromBlock; blockNum <= toBlock; blockNum++ {
		select {
		case <-l.ctx.Done():
			return
		default:
		}

		l.processBlock(ctx, blockNum)
	}

	// Update last processed block
	l.lastProcessedBlock = toBlock
}

// processBlock processes a single block and looks for relevant transactions
func (l *TransactionListener) processBlock(ctx context.Context, blockNum uint64) {
	// Get block by number
	block, err := l.client.GetEthClient().BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	if err != nil {
		fmt.Printf("Failed to get block %d: %v\n", blockNum, err)
		return
	}

	// Process each transaction in the block
	for _, tx := range block.Transactions() {
		// Check if transaction is to one of our supported token contracts
		if tx.To() == nil {
			continue
		}

		// Check if this is a transaction to a supported token contract
		tokenInfo, isSupported := l.isSupportedTokenContract(*tx.To())
		if !isSupported {
			continue
		}

		// Get transaction receipt to check if it involves our wallet
		receipt, err := l.client.GetEthClient().TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			fmt.Printf("Failed to get receipt for tx %s: %v\n", tx.Hash().Hex(), err)
			continue
		}

		// Check if transaction succeeded
		if receipt.Status != types.ReceiptStatusSuccessful {
			continue
		}

		// Check if transaction has enough confirmations
		currentBlock, err := l.client.GetBlockNumber(ctx)
		if err != nil {
			continue
		}

		confirmations := uint64(0)
		if currentBlock >= receipt.BlockNumber.Uint64() {
			confirmations = currentBlock - receipt.BlockNumber.Uint64() + 1
		}

		if confirmations < l.requiredConfirmations {
			// Not enough confirmations yet, skip for now
			// Will be processed in next iteration
			continue
		}

		// Process this transaction
		l.handleTransaction(ctx, tx, receipt, tokenInfo)
	}
}

// handleTransaction processes a single transaction
func (l *TransactionListener) handleTransaction(ctx context.Context, tx *types.Transaction, receipt *types.Receipt, tokenInfo TokenContractInfo) {
	txHash := tx.Hash().Hex()

	// Check if already processed
	l.processedTxsMu.RLock()
	alreadyProcessed := l.processedTxs[txHash]
	l.processedTxsMu.RUnlock()

	if alreadyProcessed {
		return
	}

	// Parse BEP20 token transfer
	transfer, err := parseBEP20TokenTransfer(receipt, l.wallet.GetCommonAddress(), tokenInfo.ContractAddress)
	if err != nil {
		// Not a transfer to our wallet, ignore
		return
	}

	fmt.Printf("Detected BEP20 transfer: %s sent %s %s to wallet\n",
		transfer.From.Hex(), transfer.Amount.String(), tokenInfo.Symbol)

	// Try to extract payment ID from transaction
	// Note: For BSC, memo extraction is optional and might not always work
	// depending on how the payment was sent
	paymentID, err := extractMemoFromTransaction(tx)
	if err != nil {
		fmt.Printf("Warning: No payment ID found in transaction %s: %v\n", txHash, err)
		// For BSC, we might need to match by amount instead
		// For now, skip transactions without memo
		return
	}

	// Convert amount based on token decimals
	divisor := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(tokenInfo.Decimals)))
	amount := decimal.NewFromBigInt(transfer.Amount, 0).Div(divisor)

	// Call the confirmation callback
	err = l.confirmationCallback(
		paymentID,
		txHash,
		amount,
		tokenInfo.Symbol,
	)

	if err != nil {
		fmt.Printf("Payment confirmation callback failed for %s: %v\n", txHash, err)
		return
	}

	// Mark transaction as processed
	l.processedTxsMu.Lock()
	l.processedTxs[txHash] = true
	l.processedTxsMu.Unlock()

	fmt.Printf("Successfully confirmed payment %s for transaction %s\n", paymentID, txHash)
}

// isSupportedTokenContract checks if a contract address is a supported token
func (l *TransactionListener) isSupportedTokenContract(contractAddr common.Address) (TokenContractInfo, bool) {
	for _, tokenInfo := range l.supportedTokenContracts {
		if tokenInfo.ContractAddress == contractAddr {
			return tokenInfo, true
		}
	}
	return TokenContractInfo{}, false
}

// GetWalletAddress returns the wallet address being monitored
func (l *TransactionListener) GetWalletAddress() string {
	return l.wallet.GetAddress()
}

// SubscribeToNewBlocks uses WebSocket subscription to listen for new blocks (optional enhancement)
// For MVP, we use polling. This method can be implemented for better real-time performance
func (l *TransactionListener) SubscribeToNewBlocks() error {
	// This would use ethclient.Client.SubscribeNewHead() for real-time updates
	// For MVP, we stick with polling which is simpler and more reliable
	return fmt.Errorf("WebSocket subscription not implemented yet, using polling")
}

// CleanupProcessedTxs removes old processed transactions from memory
// Should be called periodically to prevent memory bloat
func (l *TransactionListener) CleanupProcessedTxs(maxAge time.Duration) {
	// For MVP, we keep all processed transactions in memory
	// In production, you would implement a cleanup mechanism based on timestamp
	// or use a database to track processed transactions
	fmt.Println("Transaction cleanup not implemented (MVP)")
}
