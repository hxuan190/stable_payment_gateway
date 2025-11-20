package bsc

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// Client wraps the Ethereum/BSC RPC client with additional functionality
// for blockchain operations and monitoring
type Client struct {
	ethClient *ethclient.Client
	rpcURL    string
	timeout   time.Duration
	chainID   *big.Int
}

// TransactionInfo contains detailed information about a BSC transaction
type TransactionInfo struct {
	Hash          common.Hash
	BlockNumber   *big.Int
	BlockTime     uint64
	From          common.Address
	To            *common.Address
	Value         *big.Int
	GasUsed       uint64
	Receipt       *types.Receipt
	Transaction   *types.Transaction
	Confirmations uint64
	IsFinalized   bool
	Error         error
}

// ClientConfig holds configuration for the BSC RPC client
type ClientConfig struct {
	RPCURL  string
	ChainID *big.Int
	Timeout time.Duration
}

// NewClient creates a new BSC RPC client with the given configuration
func NewClient(config ClientConfig) (*Client, error) {
	if config.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL cannot be empty")
	}

	// Set default timeout if not specified
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Create Ethereum client
	ethClient, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to BSC RPC: %w", err)
	}

	// Get chain ID if not provided
	chainID := config.ChainID
	if chainID == nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		chainID, err = ethClient.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get chain ID: %w", err)
		}
	}

	client := &Client{
		ethClient: ethClient,
		rpcURL:    config.RPCURL,
		timeout:   timeout,
		chainID:   chainID,
	}

	return client, nil
}

// NewClientWithURL creates a new BSC RPC client with just the RPC URL
// Uses default timeout of 30 seconds and auto-detects chain ID
func NewClientWithURL(rpcURL string) (*Client, error) {
	return NewClient(ClientConfig{
		RPCURL:  rpcURL,
		Timeout: 30 * time.Second,
	})
}

// GetTransaction retrieves detailed information about a transaction by hash
// Returns TransactionInfo with transaction details and receipt
func (c *Client) GetTransaction(ctx context.Context, txHash common.Hash) (*TransactionInfo, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Get transaction
	tx, isPending, err := c.ethClient.TransactionByHash(timeoutCtx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if isPending {
		return &TransactionInfo{
			Hash:        txHash,
			Transaction: tx,
			IsFinalized: false,
		}, nil
	}

	// Get transaction receipt
	receipt, err := c.ethClient.TransactionReceipt(timeoutCtx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Get block to extract timestamp
	block, err := c.ethClient.BlockByNumber(timeoutCtx, receipt.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	// Calculate confirmations
	currentBlock, err := c.ethClient.BlockNumber(timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block number: %w", err)
	}

	confirmations := uint64(0)
	if currentBlock >= receipt.BlockNumber.Uint64() {
		confirmations = currentBlock - receipt.BlockNumber.Uint64() + 1
	}

	// BSC considers 15+ confirmations as finalized
	isFinalized := confirmations >= 15

	// Extract from address
	from, err := types.Sender(types.LatestSignerForChainID(c.chainID), tx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract sender: %w", err)
	}

	info := &TransactionInfo{
		Hash:          txHash,
		BlockNumber:   receipt.BlockNumber,
		BlockTime:     block.Time(),
		From:          from,
		To:            tx.To(),
		Value:         tx.Value(),
		GasUsed:       receipt.GasUsed,
		Receipt:       receipt,
		Transaction:   tx,
		Confirmations: confirmations,
		IsFinalized:   isFinalized,
	}

	// Check for transaction errors
	if receipt.Status == types.ReceiptStatusFailed {
		info.Error = fmt.Errorf("transaction failed")
	}

	return info, nil
}

// GetConfirmations returns the number of confirmations for a transaction
func (c *Client) GetConfirmations(ctx context.Context, txHash common.Hash) (uint64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Get transaction receipt
	receipt, err := c.ethClient.TransactionReceipt(timeoutCtx, txHash)
	if err != nil {
		return 0, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Get current block number
	currentBlock, err := c.ethClient.BlockNumber(timeoutCtx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current block number: %w", err)
	}

	if currentBlock < receipt.BlockNumber.Uint64() {
		return 0, nil
	}

	return currentBlock - receipt.BlockNumber.Uint64() + 1, nil
}

// GetBalance retrieves the BNB balance for a given address
func (c *Client) GetBalance(ctx context.Context, address common.Address) (decimal.Decimal, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	balance, err := c.ethClient.BalanceAt(timeoutCtx, address, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get balance: %w", err)
	}

	// Convert wei to BNB
	weiDecimal := decimal.NewFromBigInt(balance, 0)
	bnbBalance := weiDecimal.Div(decimal.NewFromInt(1e18))

	return bnbBalance, nil
}

// WaitForConfirmation waits for a transaction to reach the specified number of confirmations
// Returns error if transaction fails or timeout is reached
func (c *Client) WaitForConfirmation(ctx context.Context, txHash common.Hash, requiredConfirmations uint64, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 60 // Default to 60 retries (about 3 minutes with 3-second block time)
	}

	retryDelay := 3 * time.Second // BSC block time is ~3 seconds

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check confirmations
		confirmations, err := c.GetConfirmations(ctx, txHash)
		if err != nil {
			// Retry on error
			time.Sleep(retryDelay)
			continue
		}

		if confirmations >= requiredConfirmations {
			// Check if transaction succeeded
			receipt, err := c.ethClient.TransactionReceipt(ctx, txHash)
			if err != nil {
				return fmt.Errorf("failed to get receipt: %w", err)
			}

			if receipt.Status == types.ReceiptStatusFailed {
				return fmt.Errorf("transaction failed")
			}

			return nil
		}

		time.Sleep(retryDelay)
	}

	return fmt.Errorf("transaction not confirmed after %d retries", maxRetries)
}

// HealthCheck verifies the RPC client can connect to the BSC network
func (c *Client) HealthCheck(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Try to get current block number
	_, err := c.ethClient.BlockNumber(timeoutCtx)
	if err != nil {
		return fmt.Errorf("RPC health check failed: %w", err)
	}

	// Verify chain ID
	chainID, err := c.ethClient.ChainID(timeoutCtx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	if chainID.Cmp(c.chainID) != 0 {
		return fmt.Errorf("chain ID mismatch: expected %s, got %s", c.chainID.String(), chainID.String())
	}

	return nil
}

// HealthCheckWithRetry performs health check with retry logic
func (c *Client) HealthCheckWithRetry(ctx context.Context, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	retryDelay := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := c.HealthCheck(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't sleep on last retry
		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				// Exponential backoff
				retryDelay *= 2
			}
		}
	}

	return fmt.Errorf("health check failed after %d retries: %w", maxRetries, lastErr)
}

// GetBlockNumber returns the current block number
func (c *Client) GetBlockNumber(ctx context.Context) (uint64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	blockNumber, err := c.ethClient.BlockNumber(timeoutCtx)
	if err != nil {
		return 0, fmt.Errorf("failed to get block number: %w", err)
	}

	return blockNumber, nil
}

// GetChainID returns the chain ID
func (c *Client) GetChainID() *big.Int {
	return c.chainID
}

// GetEthClient returns the underlying Ethereum client for advanced operations
func (c *Client) GetEthClient() *ethclient.Client {
	return c.ethClient
}

// GetRPCURL returns the RPC URL being used
func (c *Client) GetRPCURL() string {
	return c.rpcURL
}

// GetTimeout returns the configured timeout
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// SetTimeout updates the timeout for RPC calls
func (c *Client) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		c.timeout = timeout
	}
}

// Close closes the underlying Ethereum client connection
func (c *Client) Close() {
	c.ethClient.Close()
}
