package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

// BlockchainType represents the type of blockchain
type BlockchainType string

const (
	BlockchainTypeSolana BlockchainType = "solana"
	BlockchainTypeBSC    BlockchainType = "bsc"
	BlockchainTypeTRON   BlockchainType = "tron"
)

// PaymentConfirmation contains details about a confirmed blockchain payment
type PaymentConfirmation struct {
	// PaymentID is the unique identifier extracted from the transaction memo
	PaymentID string

	// TxHash is the blockchain transaction hash/signature
	TxHash string

	// Amount is the amount transferred in decimal form
	Amount decimal.Decimal

	// TokenSymbol is the token symbol (e.g., "USDT", "USDC", "BUSD")
	TokenSymbol string

	// BlockchainType identifies which blockchain this payment came from
	BlockchainType BlockchainType

	// Sender is the sender's wallet address
	Sender string

	// Recipient is the recipient's wallet address (our hot wallet)
	Recipient string

	// BlockNumber is the block number where the transaction was confirmed
	BlockNumber uint64

	// Timestamp is the blockchain timestamp of the transaction
	Timestamp int64
}

// PaymentConfirmationHandler is called when a payment is confirmed on the blockchain
type PaymentConfirmationHandler func(ctx context.Context, confirmation PaymentConfirmation) error

// BlockchainListener defines the interface for monitoring blockchain transactions
// This is the PORT in the Hexagonal Architecture (Ports & Adapters pattern)
// Implementations (Adapters) include: SolanaListener, BSCListener, TRONListener
type BlockchainListener interface {
	// Start begins listening for transactions on the blockchain
	Start(ctx context.Context) error

	// Stop gracefully stops the listener and cleans up resources
	Stop(ctx context.Context) error

	// IsRunning returns whether the listener is currently active
	IsRunning() bool

	// GetBlockchainType returns the type of blockchain this listener monitors
	GetBlockchainType() BlockchainType

	// GetWalletAddress returns the wallet address being monitored
	GetWalletAddress() string

	// SetConfirmationHandler sets the callback for when payments are confirmed
	// This allows decoupling the listener from business logic
	SetConfirmationHandler(handler PaymentConfirmationHandler)

	// GetSupportedTokens returns a list of token symbols supported by this listener
	GetSupportedTokens() []string

	// GetListenerHealth returns health status and metrics
	GetListenerHealth() ListenerHealth
}

// ListenerHealth contains health and performance metrics for a blockchain listener
type ListenerHealth struct {
	// IsHealthy indicates if the listener is functioning correctly
	IsHealthy bool

	// LastProcessedBlock is the last block number processed
	LastProcessedBlock uint64

	// LastActivityTimestamp is when the listener last detected activity
	LastActivityTimestamp int64

	// ErrorCount is the number of errors encountered since start
	ErrorCount uint64

	// SuccessfulConfirmations is the count of successfully confirmed payments
	SuccessfulConfirmations uint64

	// ConnectionStatus indicates the blockchain RPC connection status
	ConnectionStatus string
}

// BlockchainListenerConfig holds common configuration for blockchain listeners
type BlockchainListenerConfig struct {
	// BlockchainType identifies the blockchain
	BlockchainType BlockchainType

	// RPCURL is the blockchain RPC endpoint
	RPCURL string

	// WalletAddress is the hot wallet address to monitor
	WalletAddress string

	// WalletPrivateKey is the private key (for transaction signing if needed)
	WalletPrivateKey string

	// SupportedTokens maps token symbols to their contract addresses or mint addresses
	SupportedTokens map[string]string

	// ConfirmationHandler is the callback for confirmed payments
	ConfirmationHandler PaymentConfirmationHandler

	// PollInterval is how often to poll for new transactions (for polling-based listeners)
	PollIntervalSeconds int

	// RequiredConfirmations is the number of block confirmations required
	RequiredConfirmations uint64

	// MaxRetries is the maximum number of retries for failed RPC calls
	MaxRetries int
}
