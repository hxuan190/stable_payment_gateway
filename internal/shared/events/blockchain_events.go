package events

import (
	"time"

	"github.com/shopspring/decimal"
)

// BlockchainType represents the type of blockchain
type BlockchainType string

const (
	BlockchainTypeSolana BlockchainType = "solana"
	BlockchainTypeBSC    BlockchainType = "bsc"
	BlockchainTypeTRON   BlockchainType = "tron"
)

// PaymentConfirmedEvent is published when a payment is confirmed on the blockchain
// This event decouples the blockchain listener from the payment service
type PaymentConfirmedEvent struct {
	BaseEvent

	// PaymentID is the unique identifier extracted from the transaction memo
	PaymentID string `json:"payment_id"`

	// TxHash is the blockchain transaction hash/signature
	TxHash string `json:"tx_hash"`

	// Amount is the amount transferred in decimal form
	Amount decimal.Decimal `json:"amount"`

	// TokenSymbol is the token symbol (e.g., "USDT", "USDC", "BUSD")
	TokenSymbol string `json:"token_symbol"`

	// BlockchainType identifies which blockchain this payment came from
	Blockchain BlockchainType `json:"blockchain"`

	// Sender is the sender's wallet address
	Sender string `json:"sender"`

	// Recipient is the recipient's wallet address (our hot wallet)
	Recipient string `json:"recipient"`

	// BlockNumber is the block number where the transaction was confirmed
	BlockNumber uint64 `json:"block_number"`

	// Timestamp is the blockchain timestamp of the transaction
	BlockTimestamp int64 `json:"block_timestamp"`
}

// NewPaymentConfirmedEvent creates a new payment confirmed event
func NewPaymentConfirmedEvent(
	paymentID string,
	txHash string,
	amount decimal.Decimal,
	tokenSymbol string,
	blockchain BlockchainType,
	sender string,
	recipient string,
	blockNumber uint64,
	blockTimestamp int64,
) *PaymentConfirmedEvent {
	return &PaymentConfirmedEvent{
		BaseEvent:      NewBaseEvent("payment.confirmed"),
		PaymentID:      paymentID,
		TxHash:         txHash,
		Amount:         amount,
		TokenSymbol:    tokenSymbol,
		Blockchain:     blockchain,
		Sender:         sender,
		Recipient:      recipient,
		BlockNumber:    blockNumber,
		BlockTimestamp: blockTimestamp,
	}
}

// TransactionDetectedEvent is published when a transaction is first detected (before full confirmation)
type TransactionDetectedEvent struct {
	BaseEvent

	// TxHash is the blockchain transaction hash/signature
	TxHash string `json:"tx_hash"`

	// Blockchain identifies which blockchain this transaction is on
	Blockchain BlockchainType `json:"blockchain"`

	// Confirmations is the current number of confirmations
	Confirmations uint64 `json:"confirmations"`

	// RequiredConfirmations is the required confirmations for finality
	RequiredConfirmations uint64 `json:"required_confirmations"`
}

// NewTransactionDetectedEvent creates a new transaction detected event
func NewTransactionDetectedEvent(
	txHash string,
	blockchain BlockchainType,
	confirmations uint64,
	requiredConfirmations uint64,
) *TransactionDetectedEvent {
	return &TransactionDetectedEvent{
		BaseEvent:             NewBaseEvent("blockchain.transaction_detected"),
		TxHash:                txHash,
		Blockchain:            blockchain,
		Confirmations:         confirmations,
		RequiredConfirmations: requiredConfirmations,
	}
}

// BlockchainHealthEvent is published when blockchain listener health status changes
type BlockchainHealthEvent struct {
	BaseEvent

	// Blockchain identifies which blockchain this event is for
	Blockchain BlockchainType `json:"blockchain"`

	// IsHealthy indicates if the listener is functioning correctly
	IsHealthy bool `json:"is_healthy"`

	// LastProcessedBlock is the last block number processed
	LastProcessedBlock uint64 `json:"last_processed_block"`

	// ErrorMessage contains error details if unhealthy
	ErrorMessage string `json:"error_message,omitempty"`

	// ConnectionStatus indicates the blockchain RPC connection status
	ConnectionStatus string `json:"connection_status"`
}

// NewBlockchainHealthEvent creates a new blockchain health event
func NewBlockchainHealthEvent(
	blockchain BlockchainType,
	isHealthy bool,
	lastProcessedBlock uint64,
	errorMessage string,
	connectionStatus string,
) *BlockchainHealthEvent {
	return &BlockchainHealthEvent{
		BaseEvent:          NewBaseEvent("blockchain.health_changed"),
		Blockchain:         blockchain,
		IsHealthy:          isHealthy,
		LastProcessedBlock: lastProcessedBlock,
		ErrorMessage:       errorMessage,
		ConnectionStatus:   connectionStatus,
	}
}

// Settlement Events

// SettlementInitiatedEvent is published when a settlement is initiated
type SettlementInitiatedEvent struct {
	BaseEvent

	// SettlementID is the unique identifier for this settlement
	SettlementID string `json:"settlement_id"`

	// MerchantID is the merchant requesting the settlement
	MerchantID string `json:"merchant_id"`

	// AmountCrypto is the amount in cryptocurrency to settle
	AmountCrypto decimal.Decimal `json:"amount_crypto"`

	// CryptoSymbol is the cryptocurrency symbol
	CryptoSymbol string `json:"crypto_symbol"`

	// AmountVND is the expected VND amount to receive
	AmountVND decimal.Decimal `json:"amount_vnd"`

	// ProviderType identifies the settlement provider
	ProviderType string `json:"provider_type"`

	// ProviderReferenceID is the settlement reference from the provider's system
	ProviderReferenceID string `json:"provider_reference_id"`
}

// NewSettlementInitiatedEvent creates a new settlement initiated event
func NewSettlementInitiatedEvent(
	settlementID string,
	merchantID string,
	amountCrypto decimal.Decimal,
	cryptoSymbol string,
	amountVND decimal.Decimal,
	providerType string,
	providerReferenceID string,
) *SettlementInitiatedEvent {
	return &SettlementInitiatedEvent{
		BaseEvent:           NewBaseEvent("settlement.initiated"),
		SettlementID:        settlementID,
		MerchantID:          merchantID,
		AmountCrypto:        amountCrypto,
		CryptoSymbol:        cryptoSymbol,
		AmountVND:           amountVND,
		ProviderType:        providerType,
		ProviderReferenceID: providerReferenceID,
	}
}

// SettlementCompletedEvent is published when a settlement is completed
type SettlementCompletedEvent struct {
	BaseEvent

	// SettlementID is the unique identifier for this settlement
	SettlementID string `json:"settlement_id"`

	// MerchantID is the merchant who requested the settlement
	MerchantID string `json:"merchant_id"`

	// AmountVND is the actual VND amount transferred
	AmountVND decimal.Decimal `json:"amount_vnd"`

	// Fee is the settlement fee charged
	Fee decimal.Decimal `json:"fee"`

	// ProviderType identifies the settlement provider
	ProviderType string `json:"provider_type"`

	// CompletedAt is when the settlement was completed
	CompletedAt time.Time `json:"completed_at"`
}

// NewSettlementCompletedEvent creates a new settlement completed event
func NewSettlementCompletedEvent(
	settlementID string,
	merchantID string,
	amountVND decimal.Decimal,
	fee decimal.Decimal,
	providerType string,
	completedAt time.Time,
) *SettlementCompletedEvent {
	return &SettlementCompletedEvent{
		BaseEvent:    NewBaseEvent("settlement.completed"),
		SettlementID: settlementID,
		MerchantID:   merchantID,
		AmountVND:    amountVND,
		Fee:          fee,
		ProviderType: providerType,
		CompletedAt:  completedAt,
	}
}

// SettlementFailedEvent is published when a settlement fails
type SettlementFailedEvent struct {
	BaseEvent

	// SettlementID is the unique identifier for this settlement
	SettlementID string `json:"settlement_id"`

	// MerchantID is the merchant who requested the settlement
	MerchantID string `json:"merchant_id"`

	// ProviderType identifies the settlement provider
	ProviderType string `json:"provider_type"`

	// ErrorMessage contains error details
	ErrorMessage string `json:"error_message"`

	// Retryable indicates if the settlement can be retried
	Retryable bool `json:"retryable"`
}

// NewSettlementFailedEvent creates a new settlement failed event
func NewSettlementFailedEvent(
	settlementID string,
	merchantID string,
	providerType string,
	errorMessage string,
	retryable bool,
) *SettlementFailedEvent {
	return &SettlementFailedEvent{
		BaseEvent:    NewBaseEvent("settlement.failed"),
		SettlementID: settlementID,
		MerchantID:   merchantID,
		ProviderType: providerType,
		ErrorMessage: errorMessage,
		Retryable:    retryable,
	}
}
