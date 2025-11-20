package tron

import (
	"context"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/shopspring/decimal"
)

// Client wraps the TRON gRPC client with additional functionality
// for blockchain operations and monitoring
type Client struct {
	grpcClient *client.GrpcClient
	rpcURL     string
	timeout    time.Duration
	isMainnet  bool
}

// TransactionInfo contains detailed information about a TRON transaction
type TransactionInfo struct {
	TxID            string                    `json:"tx_id"`
	BlockNumber     int64                     `json:"block_number"`
	BlockTimestamp  int64                     `json:"block_timestamp"`
	Transaction     *core.Transaction         `json:"transaction"`
	TransactionInfo *core.TransactionInfo     `json:"transaction_info"`
	Confirmations   int64                     `json:"confirmations"`
	IsConfirmed     bool                      `json:"is_confirmed"`
	IsSuccess       bool                      `json:"is_success"`
	Error           error                     `json:"error,omitempty"`
}

// ClientConfig holds configuration for the TRON gRPC client
type ClientConfig struct {
	RPCURL    string
	Timeout   time.Duration
	IsMainnet bool
}

// NewClient creates a new TRON gRPC client with the given configuration
func NewClient(config ClientConfig) (*Client, error) {
	if config.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL cannot be empty")
	}

	// Set default timeout if not specified
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Create gRPC client
	grpcClient := client.NewGrpcClient(config.RPCURL)
	if grpcClient == nil {
		return nil, fmt.Errorf("failed to create TRON gRPC client")
	}

	err := grpcClient.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start TRON gRPC client: %w", err)
	}

	tronClient := &Client{
		grpcClient: grpcClient,
		rpcURL:     config.RPCURL,
		timeout:    timeout,
		isMainnet:  config.IsMainnet,
	}

	return tronClient, nil
}

// NewClientWithURL creates a new TRON gRPC client with just the RPC URL
// Uses default timeout of 30 seconds
// isMainnet should be true for mainnet, false for testnet (Shasta, Nile)
func NewClientWithURL(rpcURL string, isMainnet bool) (*Client, error) {
	return NewClient(ClientConfig{
		RPCURL:    rpcURL,
		Timeout:   30 * time.Second,
		IsMainnet: isMainnet,
	})
}

// GetTransaction retrieves detailed information about a transaction by transaction ID
// Returns TransactionInfo with transaction details and metadata
func (c *Client) GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Get transaction by ID
	tx, err := c.grpcClient.GetTransactionByID(txID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx == nil {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}

	// Get transaction info (receipt)
	txInfo, err := c.grpcClient.GetTransactionInfoByID(txID)
	if err != nil {
		// Transaction might not be confirmed yet
		txInfo = nil
	}

	// Determine if transaction is confirmed and successful
	isConfirmed := txInfo != nil
	isSuccess := false
	var txError error

	if isConfirmed {
		// Check transaction result
		if txInfo.Result == core.TransactionInfo_SUCCESS {
			isSuccess = true
		} else {
			txError = fmt.Errorf("transaction failed with result: %s", txInfo.Result.String())
		}
	}

	// Get current block number for confirmations
	var confirmations int64
	if isConfirmed && txInfo.BlockNumber > 0 {
		currentBlock, err := c.GetCurrentBlockNumber(timeoutCtx)
		if err == nil {
			confirmations = currentBlock - txInfo.BlockNumber
		}
	}

	info := &TransactionInfo{
		TxID:            txID,
		BlockNumber:     txInfo.GetBlockNumber(),
		BlockTimestamp:  txInfo.GetBlockTimeStamp(),
		Transaction:     tx.GetRawData(),
		TransactionInfo: txInfo,
		Confirmations:   confirmations,
		IsConfirmed:     isConfirmed,
		IsSuccess:       isSuccess,
		Error:           txError,
	}

	return info, nil
}

// GetCurrentBlockNumber retrieves the current block number
func (c *Client) GetCurrentBlockNumber(ctx context.Context) (int64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_ = timeoutCtx // Use context for timeout control

	block, err := c.grpcClient.GetNowBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to get current block: %w", err)
	}

	if block == nil || block.BlockHeader == nil || block.BlockHeader.RawData == nil {
		return 0, fmt.Errorf("invalid block data")
	}

	return block.BlockHeader.RawData.Number, nil
}

// GetBalance retrieves the TRX balance for a given address in TRX (not SUN)
// 1 TRX = 1,000,000 SUN
func (c *Client) GetBalance(ctx context.Context, address string) (decimal.Decimal, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_ = timeoutCtx // Use context for timeout control

	// Get account info
	account, err := c.grpcClient.GetAccount(address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		// Account doesn't exist yet, return zero balance
		return decimal.Zero, nil
	}

	// Convert SUN to TRX
	sunBalance := decimal.NewFromInt(account.Balance)
	trxBalance := sunBalance.Div(decimal.NewFromInt(1_000_000))

	return trxBalance, nil
}

// GetTRC20Balance retrieves the TRC20 token balance for a given address
// tokenAddress is the contract address of the TRC20 token (e.g., USDT: TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t)
func (c *Client) GetTRC20Balance(ctx context.Context, address string, tokenAddress string) (decimal.Decimal, uint8, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_ = timeoutCtx // Use context for timeout control

	// Call TRC20 balanceOf function
	result, err := c.grpcClient.TRC20GetBalance(address, tokenAddress)
	if err != nil {
		return decimal.Zero, 0, fmt.Errorf("failed to get TRC20 balance: %w", err)
	}

	// Get token decimals
	decimals, err := c.grpcClient.TRC20GetDecimals(tokenAddress)
	if err != nil {
		// Default to 6 decimals (USDT standard)
		decimals = 6
	}

	// Parse balance
	balance := decimal.NewFromBigInt(result, 0)

	return balance, uint8(decimals), nil
}

// WaitForConfirmation waits for a transaction to be confirmed
// Returns error if transaction fails or timeout is reached
// TRON confirmations: 19 blocks (~57 seconds) for solid confirmation
func (c *Client) WaitForConfirmation(ctx context.Context, txID string, minConfirmations int64, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 30
	}

	if minConfirmations <= 0 {
		// TRON standard: 19 confirmations for solid block
		minConfirmations = 19
	}

	retryDelay := 3 * time.Second // TRON block time is ~3 seconds

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		txInfo, err := c.GetTransaction(ctx, txID)
		if err != nil {
			// Retry on error
			time.Sleep(retryDelay)
			continue
		}

		// Check if transaction failed
		if txInfo.Error != nil {
			return txInfo.Error
		}

		// Check if transaction has enough confirmations
		if txInfo.IsConfirmed && txInfo.Confirmations >= minConfirmations {
			return nil
		}

		time.Sleep(retryDelay)
	}

	return fmt.Errorf("transaction not confirmed after %d retries", maxRetries)
}

// HealthCheck verifies the gRPC client can connect to the TRON network
// Returns error if connection fails or network is unhealthy
func (c *Client) HealthCheck(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_ = timeoutCtx // Use context for timeout control

	// Try to get current block
	block, err := c.grpcClient.GetNowBlock()
	if err != nil {
		return fmt.Errorf("gRPC health check failed: %w", err)
	}

	if block == nil {
		return fmt.Errorf("gRPC returned nil block")
	}

	// Additional check: verify block has valid header
	if block.BlockHeader == nil || block.BlockHeader.RawData == nil {
		return fmt.Errorf("invalid block header")
	}

	return nil
}

// HealthCheckWithRetry performs health check with retry logic
// Retries up to maxRetries times with exponential backoff
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
				// Exponential backoff: 1s, 2s, 4s, 8s...
				retryDelay *= 2
			}
		}
	}

	return fmt.Errorf("health check failed after %d retries: %w", maxRetries, lastErr)
}

// ChainInfo represents TRON chain information
type ChainInfo struct {
	BlockNumber int64  `json:"block_number"`
	BlockHash   string `json:"block_hash"`
	NodeVersion string `json:"node_version"`
}

// GetChainInfo retrieves current chain information
func (c *Client) GetChainInfo(ctx context.Context) (*ChainInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_ = timeoutCtx // Use context for timeout control

	// Get current block
	block, err := c.grpcClient.GetNowBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get chain info: %w", err)
	}

	if block == nil || block.BlockHeader == nil || block.BlockHeader.RawData == nil {
		return nil, fmt.Errorf("invalid block data")
	}

	// Get node info
	nodeInfo, err := c.grpcClient.GetNodeInfo()
	version := "unknown"
	if err == nil && nodeInfo != nil {
		version = nodeInfo.ConfigNodeInfo.CodeVersion
	}

	info := &ChainInfo{
		BlockNumber: block.BlockHeader.RawData.Number,
		BlockHash:   fmt.Sprintf("%x", block.BlockID),
		NodeVersion: version,
	}

	return info, nil
}

// GetGrpcClient returns the underlying gRPC client for advanced operations
func (c *Client) GetGrpcClient() *client.GrpcClient {
	return c.grpcClient
}

// GetRPCURL returns the RPC URL being used
func (c *Client) GetRPCURL() string {
	return c.rpcURL
}

// GetTimeout returns the configured timeout
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// SetTimeout updates the timeout for gRPC calls
func (c *Client) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		c.timeout = timeout
	}
}

// IsMainnet returns true if client is configured for mainnet
func (c *Client) IsMainnet() bool {
	return c.isMainnet
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.grpcClient != nil {
		return c.grpcClient.Stop()
	}
	return nil
}

// BroadcastTransaction broadcasts a signed transaction to the network
func (c *Client) BroadcastTransaction(ctx context.Context, signedTx *core.Transaction) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_ = timeoutCtx // Use context for timeout control

	result, err := c.grpcClient.Broadcast(signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	if !result.Result {
		return "", fmt.Errorf("broadcast failed: %s", string(result.Message))
	}

	// Get transaction ID
	txID := fmt.Sprintf("%x", signedTx.GetTxid())

	return txID, nil
}

// ValidateAddress checks if a TRON address is valid
func (c *Client) ValidateAddress(address string) (bool, error) {
	result, err := c.grpcClient.ValidateAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to validate address: %w", err)
	}

	return result.Result, nil
}

// GetAccountResource retrieves account resource information (bandwidth, energy)
func (c *Client) GetAccountResource(ctx context.Context, address string) (*api.AccountResourceMessage, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_ = timeoutCtx // Use context for timeout control

	resource, err := c.grpcClient.GetAccountResource(address)
	if err != nil {
		return nil, fmt.Errorf("failed to get account resource: %w", err)
	}

	return resource, nil
}
