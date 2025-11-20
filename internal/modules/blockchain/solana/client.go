package solana

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/shopspring/decimal"
)

// Client wraps the Solana RPC client with additional functionality
// for blockchain operations and monitoring
type Client struct {
	rpcClient *rpc.Client
	rpcURL    string
	timeout   time.Duration
}

// TransactionInfo contains detailed information about a Solana transaction
type TransactionInfo struct {
	Signature       solana.Signature      `json:"signature"`
	Slot            uint64                `json:"slot"`
	BlockTime       *int64                `json:"block_time"`
	Meta            *rpc.TransactionMeta  `json:"meta"`
	Transaction     *solana.Transaction   `json:"transaction"`
	Confirmations   uint64                `json:"confirmations"`
	IsFinalized     bool                  `json:"is_finalized"`
	Error           error                 `json:"error,omitempty"`
}

// ClientConfig holds configuration for the Solana RPC client
type ClientConfig struct {
	RPCURL  string
	Timeout time.Duration
}

// NewClient creates a new Solana RPC client with the given configuration
func NewClient(config ClientConfig) (*Client, error) {
	if config.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL cannot be empty")
	}

	// Set default timeout if not specified
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Create RPC client
	rpcClient := rpc.New(config.RPCURL)

	client := &Client{
		rpcClient: rpcClient,
		rpcURL:    config.RPCURL,
		timeout:   timeout,
	}

	return client, nil
}

// NewClientWithURL creates a new Solana RPC client with just the RPC URL
// Uses default timeout of 30 seconds
func NewClientWithURL(rpcURL string) (*Client, error) {
	return NewClient(ClientConfig{
		RPCURL:  rpcURL,
		Timeout: 30 * time.Second,
	})
}

// GetTransaction retrieves detailed information about a transaction by signature
// Returns TransactionInfo with transaction details and metadata
func (c *Client) GetTransaction(ctx context.Context, signature solana.Signature) (*TransactionInfo, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Get transaction with full details
	out, err := c.rpcClient.GetTransaction(
		timeoutCtx,
		signature,
		&rpc.GetTransactionOpts{
			Encoding:   solana.EncodingBase64,
			Commitment: rpc.CommitmentFinalized,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if out == nil {
		return nil, fmt.Errorf("transaction not found: %s", signature.String())
	}

	// Parse the transaction
	tx, err := out.Transaction.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Check if transaction is finalized
	isFinalized := false
	if out.Meta != nil && out.Meta.Err == nil {
		// Transaction succeeded, check confirmation status
		status, err := c.GetSignatureStatus(timeoutCtx, signature)
		if err == nil && status != nil && status.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
			isFinalized = true
		}
	}

	// Convert block time if present
	var blockTimePtr *int64
	if out.BlockTime != nil {
		blockTime := int64(*out.BlockTime)
		blockTimePtr = &blockTime
	}

	info := &TransactionInfo{
		Signature:     signature,
		Slot:          out.Slot,
		BlockTime:     blockTimePtr,
		Meta:          out.Meta,
		Transaction:   tx,
		Confirmations: 0, // Solana doesn't use confirmations like Bitcoin/Ethereum
		IsFinalized:   isFinalized,
	}

	// Check for transaction errors
	if out.Meta != nil && out.Meta.Err != nil {
		info.Error = fmt.Errorf("transaction failed: %v", out.Meta.Err)
	}

	return info, nil
}

// SignatureStatusResult represents the status of a transaction signature
type SignatureStatusResult struct {
	Slot               uint64
	Confirmations      *uint64
	Err                interface{}
	ConfirmationStatus rpc.ConfirmationStatusType
}

// GetSignatureStatus retrieves the status of a transaction signature
func (c *Client) GetSignatureStatus(ctx context.Context, signature solana.Signature) (*SignatureStatusResult, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	statuses, err := c.rpcClient.GetSignatureStatuses(timeoutCtx, true, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature status: %w", err)
	}

	if statuses == nil || len(statuses.Value) == 0 {
		return nil, fmt.Errorf("no status found for signature: %s", signature.String())
	}

	status := statuses.Value[0]
	if status == nil {
		return nil, fmt.Errorf("signature status is nil: %s", signature.String())
	}

	result := &SignatureStatusResult{
		Slot:               status.Slot,
		Confirmations:      status.Confirmations,
		Err:                status.Err,
		ConfirmationStatus: status.ConfirmationStatus,
	}

	return result, nil
}

// GetConfirmations returns the number of confirmations for a transaction
// Note: Solana uses a different finality model than Bitcoin/Ethereum
// This method returns:
// - 0 if transaction is not found or not confirmed
// - 1 if transaction is confirmed
// - 32+ if transaction is finalized (approximate, based on slot difference)
func (c *Client) GetConfirmations(ctx context.Context, signature solana.Signature) (uint64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Get transaction status
	status, err := c.GetSignatureStatus(timeoutCtx, signature)
	if err != nil {
		return 0, err
	}

	// If transaction is finalized, return high confirmation count
	if status.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
		// Get current slot to calculate approximate confirmations
		currentSlot, err := c.GetSlot(timeoutCtx, rpc.CommitmentFinalized)
		if err != nil {
			return 32, nil // Return minimum finalized confirmations
		}

		if status.Slot > 0 {
			slotDiff := currentSlot - status.Slot
			return slotDiff, nil
		}

		return 32, nil
	}

	// If transaction is confirmed but not finalized
	if status.ConfirmationStatus == rpc.ConfirmationStatusConfirmed {
		return 1, nil
	}

	// Transaction is processed but not yet confirmed
	if status.ConfirmationStatus == rpc.ConfirmationStatusProcessed {
		return 0, nil
	}

	return 0, nil
}

// GetSlot returns the current slot number
func (c *Client) GetSlot(ctx context.Context, commitment rpc.CommitmentType) (uint64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	slot, err := c.rpcClient.GetSlot(timeoutCtx, commitment)
	if err != nil {
		return 0, fmt.Errorf("failed to get slot: %w", err)
	}

	return slot, nil
}

// GetBalance retrieves the SOL balance for a given address
func (c *Client) GetBalance(ctx context.Context, address solana.PublicKey) (decimal.Decimal, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	balance, err := c.rpcClient.GetBalance(
		timeoutCtx,
		address,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get balance: %w", err)
	}

	// Convert lamports to SOL
	lamportsDecimal := decimal.NewFromUint64(balance.Value)
	solBalance := lamportsDecimal.Div(decimal.NewFromInt(1_000_000_000))

	return solBalance, nil
}

// WaitForConfirmation waits for a transaction to reach the specified commitment level
// Returns error if transaction fails or timeout is reached
func (c *Client) WaitForConfirmation(ctx context.Context, signature solana.Signature, commitment rpc.CommitmentType, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 30
	}

	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := c.GetSignatureStatus(ctx, signature)
		if err != nil {
			// Retry on error
			time.Sleep(retryDelay)
			continue
		}

		// Check if transaction failed
		if status.Err != nil {
			return fmt.Errorf("transaction failed: %v", status.Err)
		}

		// Check if desired commitment level is reached
		switch commitment {
		case rpc.CommitmentFinalized:
			if status.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				return nil
			}
		case rpc.CommitmentConfirmed:
			if status.ConfirmationStatus == rpc.ConfirmationStatusConfirmed ||
				status.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				return nil
			}
		case rpc.CommitmentProcessed:
			if status.ConfirmationStatus != "" {
				return nil
			}
		}

		time.Sleep(retryDelay)
	}

	return fmt.Errorf("transaction not confirmed after %d retries", maxRetries)
}

// HealthCheck verifies the RPC client can connect to the Solana network
// Returns error if connection fails or network is unhealthy
func (c *Client) HealthCheck(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Try to get cluster health
	health, err := c.rpcClient.GetHealth(timeoutCtx)
	if err != nil {
		return fmt.Errorf("RPC health check failed: %w", err)
	}

	if health != rpc.HealthOk {
		return fmt.Errorf("RPC cluster is unhealthy: %s", health)
	}

	// Additional check: try to get current slot
	_, err = c.GetSlot(timeoutCtx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to get current slot: %w", err)
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

// VersionInfo represents Solana node version information
type VersionInfo struct {
	SolanaCore string
	FeatureSet int64
}

// GetVersion retrieves the Solana node version
func (c *Client) GetVersion(ctx context.Context) (*VersionInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	version, err := c.rpcClient.GetVersion(timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	info := &VersionInfo{
		SolanaCore: version.SolanaCore,
		FeatureSet: version.FeatureSet,
	}

	return info, nil
}

// GetRecentBlockhash retrieves the most recent blockhash
func (c *Client) GetRecentBlockhash(ctx context.Context) (solana.Hash, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	recent, err := c.rpcClient.GetRecentBlockhash(timeoutCtx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Hash{}, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	return recent.Value.Blockhash, nil
}

// GetRPCClient returns the underlying RPC client for advanced operations
func (c *Client) GetRPCClient() *rpc.Client {
	return c.rpcClient
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
