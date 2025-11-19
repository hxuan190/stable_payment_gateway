package tron

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// TransactionEvent represents a transaction event detected by the listener
type TransactionEvent struct {
	TxID           string          `json:"tx_id"`
	FromAddress    string          `json:"from_address"`
	ToAddress      string          `json:"to_address"`
	Amount         decimal.Decimal `json:"amount"`
	TokenAddress   string          `json:"token_address,omitempty"`
	TokenSymbol    string          `json:"token_symbol"`
	Memo           string          `json:"memo,omitempty"`
	BlockNumber    int64           `json:"block_number"`
	BlockTimestamp int64           `json:"block_timestamp"`
	Confirmations  int64           `json:"confirmations"`
	IsConfirmed    bool            `json:"is_confirmed"`
}

// TransactionHandler is called when a new transaction is detected
type TransactionHandler func(event *TransactionEvent) error

// ListenerConfig holds configuration for the blockchain listener
type ListenerConfig struct {
	Client           *Client
	WatchAddress     string
	TRC20Contracts   []string // List of TRC20 token contracts to monitor
	PollInterval     time.Duration
	MinConfirmations int64
	StartBlock       int64 // Block number to start listening from (0 = current)
	Logger           *logrus.Logger
}

// Listener monitors the TRON blockchain for transactions to a specific address
type Listener struct {
	config          ListenerConfig
	parser          *TransactionParser
	handlers        []TransactionHandler
	isRunning       bool
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	lastBlockNumber int64
	processedTxs    map[string]bool // Track processed transactions
}

// NewListener creates a new blockchain listener
func NewListener(config ListenerConfig) (*Listener, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	if config.WatchAddress == "" {
		return nil, fmt.Errorf("watch address cannot be empty")
	}

	if config.PollInterval == 0 {
		config.PollInterval = 3 * time.Second // TRON block time
	}

	if config.MinConfirmations == 0 {
		config.MinConfirmations = 19 // TRON solid block confirmation
	}

	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	listener := &Listener{
		config:       config,
		parser:       NewTransactionParser(config.Client),
		handlers:     make([]TransactionHandler, 0),
		stopChan:     make(chan struct{}),
		processedTxs: make(map[string]bool),
	}

	return listener, nil
}

// AddHandler adds a transaction handler
func (l *Listener) AddHandler(handler TransactionHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, handler)
}

// Start starts the blockchain listener
func (l *Listener) Start(ctx context.Context) error {
	l.mu.Lock()
	if l.isRunning {
		l.mu.Unlock()
		return fmt.Errorf("listener is already running")
	}
	l.isRunning = true
	l.mu.Unlock()

	// Get starting block number
	if l.config.StartBlock > 0 {
		l.lastBlockNumber = l.config.StartBlock
	} else {
		// Start from current block
		currentBlock, err := l.config.Client.GetCurrentBlockNumber(ctx)
		if err != nil {
			l.config.Logger.WithError(err).Error("Failed to get current block number")
			return fmt.Errorf("failed to get starting block: %w", err)
		}
		l.lastBlockNumber = currentBlock
	}

	l.config.Logger.WithFields(logrus.Fields{
		"watch_address":  l.config.WatchAddress,
		"start_block":    l.lastBlockNumber,
		"poll_interval":  l.config.PollInterval,
		"trc20_tokens":   len(l.config.TRC20Contracts),
	}).Info("TRON listener started")

	l.wg.Add(1)
	go l.listenLoop(ctx)

	return nil
}

// Stop stops the blockchain listener
func (l *Listener) Stop() {
	l.mu.Lock()
	if !l.isRunning {
		l.mu.Unlock()
		return
	}
	l.isRunning = false
	l.mu.Unlock()

	close(l.stopChan)
	l.wg.Wait()

	l.config.Logger.Info("TRON listener stopped")
}

// IsRunning returns true if the listener is running
func (l *Listener) IsRunning() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.isRunning
}

// listenLoop is the main listening loop
func (l *Listener) listenLoop(ctx context.Context) {
	defer l.wg.Done()

	ticker := time.NewTicker(l.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.config.Logger.Info("Context cancelled, stopping listener")
			return
		case <-l.stopChan:
			l.config.Logger.Info("Stop signal received, stopping listener")
			return
		case <-ticker.C:
			err := l.pollBlocks(ctx)
			if err != nil {
				l.config.Logger.WithError(err).Error("Failed to poll blocks")
			}
		}
	}
}

// pollBlocks polls for new blocks and processes transactions
func (l *Listener) pollBlocks(ctx context.Context) error {
	// Get current block number
	currentBlock, err := l.config.Client.GetCurrentBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block: %w", err)
	}

	// No new blocks
	if currentBlock <= l.lastBlockNumber {
		return nil
	}

	l.config.Logger.WithFields(logrus.Fields{
		"from_block": l.lastBlockNumber + 1,
		"to_block":   currentBlock,
	}).Debug("Processing new blocks")

	// Process blocks one by one
	for blockNum := l.lastBlockNumber + 1; blockNum <= currentBlock; blockNum++ {
		err := l.processBlock(ctx, blockNum)
		if err != nil {
			l.config.Logger.WithError(err).WithField("block", blockNum).Error("Failed to process block")
			// Continue processing other blocks even if one fails
		}

		// Update last processed block
		l.lastBlockNumber = blockNum
	}

	return nil
}

// processBlock processes transactions in a specific block
func (l *Listener) processBlock(ctx context.Context, blockNumber int64) error {
	// Get block by number
	block, err := l.config.Client.GetGrpcClient().GetBlockByNum(blockNumber)
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
	}

	if block == nil || len(block.Transactions) == 0 {
		return nil // Empty block
	}

	l.config.Logger.WithFields(logrus.Fields{
		"block":        blockNumber,
		"transactions": len(block.Transactions),
	}).Debug("Processing block")

	// Process each transaction in the block
	for _, tx := range block.Transactions {
		if tx == nil || tx.Transaction == nil {
			continue
		}

		// Get transaction ID
		txID := fmt.Sprintf("%x", tx.Txid)

		// Skip if already processed
		if l.isTransactionProcessed(txID) {
			continue
		}

		// Process transaction
		err := l.processTransaction(ctx, txID, blockNumber)
		if err != nil {
			l.config.Logger.WithError(err).WithField("tx_id", txID).Error("Failed to process transaction")
			continue
		}

		// Mark as processed
		l.markTransactionProcessed(txID)
	}

	return nil
}

// processTransaction processes a single transaction
func (l *Listener) processTransaction(ctx context.Context, txID string, blockNumber int64) error {
	// Parse transaction
	parsed, err := l.parser.ParseTransaction(txID)
	if err != nil {
		// Not a payment transaction, skip silently
		return nil
	}

	// Check if transaction is to our watch address
	if parsed.ToAddress != l.config.WatchAddress {
		return nil // Not for us
	}

	// Filter TRC20 tokens if specified
	if parsed.IsTRC20Transfer && len(l.config.TRC20Contracts) > 0 {
		found := false
		for _, contract := range l.config.TRC20Contracts {
			if parsed.TokenAddress == contract {
				found = true
				break
			}
		}
		if !found {
			return nil // Token not in watch list
		}
	}

	// Get current block for confirmations
	currentBlock, err := l.config.Client.GetCurrentBlockNumber(ctx)
	if err != nil {
		currentBlock = blockNumber
	}

	confirmations := currentBlock - blockNumber
	isConfirmed := confirmations >= l.config.MinConfirmations

	// Create transaction event
	event := &TransactionEvent{
		TxID:           parsed.TxID,
		FromAddress:    parsed.FromAddress,
		ToAddress:      parsed.ToAddress,
		Amount:         parsed.Amount,
		TokenAddress:   parsed.TokenAddress,
		TokenSymbol:    parsed.TokenSymbol,
		Memo:           parsed.Memo,
		BlockNumber:    parsed.BlockNumber,
		BlockTimestamp: parsed.BlockTimestamp,
		Confirmations:  confirmations,
		IsConfirmed:    isConfirmed,
	}

	l.config.Logger.WithFields(logrus.Fields{
		"tx_id":         event.TxID,
		"from":          event.FromAddress,
		"to":            event.ToAddress,
		"amount":        event.Amount,
		"token":         event.TokenSymbol,
		"confirmations": confirmations,
		"memo":          event.Memo,
	}).Info("Transaction detected")

	// Call handlers
	l.mu.RLock()
	handlers := l.handlers
	l.mu.RUnlock()

	for _, handler := range handlers {
		err := handler(event)
		if err != nil {
			l.config.Logger.WithError(err).WithField("tx_id", txID).Error("Handler failed")
		}
	}

	return nil
}

// isTransactionProcessed checks if a transaction has been processed
func (l *Listener) isTransactionProcessed(txID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.processedTxs[txID]
}

// markTransactionProcessed marks a transaction as processed
func (l *Listener) markTransactionProcessed(txID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.processedTxs[txID] = true

	// Cleanup old transactions (keep last 1000)
	if len(l.processedTxs) > 1000 {
		// Simple cleanup: remove random half
		count := 0
		for key := range l.processedTxs {
			delete(l.processedTxs, key)
			count++
			if count >= 500 {
				break
			}
		}
	}
}

// GetLastBlockNumber returns the last processed block number
func (l *Listener) GetLastBlockNumber() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lastBlockNumber
}

// MonitorTransaction monitors a specific transaction until it reaches minimum confirmations
func (l *Listener) MonitorTransaction(ctx context.Context, txID string, callback func(*TransactionEvent)) error {
	ticker := time.NewTicker(l.config.PollInterval)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute) // 5 minute timeout

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("transaction monitoring timeout")
		case <-ticker.C:
			// Get transaction info
			txInfo, err := l.config.Client.GetTransaction(ctx, txID)
			if err != nil {
				l.config.Logger.WithError(err).WithField("tx_id", txID).Error("Failed to get transaction")
				continue
			}

			// Parse transaction
			parsed, err := l.parser.ParseTransaction(txID)
			if err != nil {
				l.config.Logger.WithError(err).WithField("tx_id", txID).Error("Failed to parse transaction")
				continue
			}

			isConfirmed := txInfo.Confirmations >= l.config.MinConfirmations

			event := &TransactionEvent{
				TxID:           parsed.TxID,
				FromAddress:    parsed.FromAddress,
				ToAddress:      parsed.ToAddress,
				Amount:         parsed.Amount,
				TokenAddress:   parsed.TokenAddress,
				TokenSymbol:    parsed.TokenSymbol,
				Memo:           parsed.Memo,
				BlockNumber:    parsed.BlockNumber,
				BlockTimestamp: parsed.BlockTimestamp,
				Confirmations:  txInfo.Confirmations,
				IsConfirmed:    isConfirmed,
			}

			// Call callback
			callback(event)

			// If confirmed, stop monitoring
			if isConfirmed {
				return nil
			}
		}
	}
}
