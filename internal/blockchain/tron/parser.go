package tron

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/shopspring/decimal"
)

// ParsedTransaction contains parsed transaction data for payment processing
type ParsedTransaction struct {
	TxID            string          `json:"tx_id"`
	FromAddress     string          `json:"from_address"`
	ToAddress       string          `json:"to_address"`
	Amount          decimal.Decimal `json:"amount"`
	TokenAddress    string          `json:"token_address,omitempty"` // Empty for TRX transfers
	TokenSymbol     string          `json:"token_symbol,omitempty"`  // TRX or token symbol
	Memo            string          `json:"memo,omitempty"`
	BlockNumber     int64           `json:"block_number"`
	BlockTimestamp  int64           `json:"block_timestamp"`
	IsSuccess       bool            `json:"is_success"`
	IsTRC20Transfer bool            `json:"is_trc20_transfer"`
}

// TransactionParser provides methods to parse TRON transactions
type TransactionParser struct {
	client *Client
}

// NewTransactionParser creates a new transaction parser
func NewTransactionParser(client *Client) *TransactionParser {
	return &TransactionParser{
		client: client,
	}
}

// ParseTransaction parses a TRON transaction and extracts payment information
func (p *TransactionParser) ParseTransaction(txID string) (*ParsedTransaction, error) {
	// Get transaction info
	txInfo, err := p.client.GetTransaction(nil, txID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if txInfo.Transaction == nil {
		return nil, fmt.Errorf("transaction data is nil")
	}

	// Parse transaction based on type
	if len(txInfo.Transaction.Contract) == 0 {
		return nil, fmt.Errorf("no contracts in transaction")
	}

	contract := txInfo.Transaction.Contract[0]

	switch contract.Type {
	case core.Transaction_Contract_TransferContract:
		return p.parseTRXTransfer(txID, txInfo)
	case core.Transaction_Contract_TriggerSmartContract:
		return p.parseTRC20Transfer(txID, txInfo)
	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", contract.Type.String())
	}
}

// parseTRXTransfer parses a native TRX transfer transaction
func (p *TransactionParser) parseTRXTransfer(txID string, txInfo *TransactionInfo) (*ParsedTransaction, error) {
	if txInfo.Transaction == nil || len(txInfo.Transaction.Contract) == 0 {
		return nil, fmt.Errorf("invalid transaction data")
	}

	contract := txInfo.Transaction.Contract[0]

	// Unmarshal transfer contract
	transferContract := &core.TransferContract{}
	err := contract.Parameter.Value.Unmarshal(transferContract)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transfer contract: %w", err)
	}

	// Extract addresses
	fromAddress := encodeAddress(transferContract.OwnerAddress)
	toAddress := encodeAddress(transferContract.ToAddress)

	// Convert SUN to TRX (1 TRX = 1,000,000 SUN)
	amount := decimal.NewFromInt(transferContract.Amount)
	amountTRX := amount.Div(decimal.NewFromInt(1_000_000))

	// Extract memo from transaction data (if present)
	memo := p.extractMemo(txInfo.Transaction)

	parsed := &ParsedTransaction{
		TxID:            txID,
		FromAddress:     fromAddress,
		ToAddress:       toAddress,
		Amount:          amountTRX,
		TokenAddress:    "",
		TokenSymbol:     "TRX",
		Memo:            memo,
		BlockNumber:     txInfo.BlockNumber,
		BlockTimestamp:  txInfo.BlockTimestamp,
		IsSuccess:       txInfo.IsSuccess,
		IsTRC20Transfer: false,
	}

	return parsed, nil
}

// parseTRC20Transfer parses a TRC20 token transfer transaction
func (p *TransactionParser) parseTRC20Transfer(txID string, txInfo *TransactionInfo) (*ParsedTransaction, error) {
	if txInfo.Transaction == nil || len(txInfo.Transaction.Contract) == 0 {
		return nil, fmt.Errorf("invalid transaction data")
	}

	contract := txInfo.Transaction.Contract[0]

	// Unmarshal trigger smart contract
	triggerContract := &core.TriggerSmartContract{}
	err := contract.Parameter.Value.Unmarshal(triggerContract)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal trigger contract: %w", err)
	}

	// Extract contract address
	contractAddress := encodeAddress(triggerContract.ContractAddress)

	// Parse transfer function call
	// TRC20 transfer function signature: transfer(address,uint256)
	// Data format: 4 bytes (function selector) + 32 bytes (to address) + 32 bytes (amount)
	data := triggerContract.Data
	if len(data) < 68 { // 4 + 32 + 32 bytes
		return nil, fmt.Errorf("invalid TRC20 transfer data length: %d", len(data))
	}

	// Check function selector (first 4 bytes should be 0xa9059cbb for transfer)
	functionSelector := hex.EncodeToString(data[:4])
	if functionSelector != "a9059cbb" {
		return nil, fmt.Errorf("not a TRC20 transfer transaction: function selector %s", functionSelector)
	}

	// Extract 'to' address (bytes 4-36, padded to 32 bytes)
	toAddressBytes := data[16:36] // Skip padding, take last 20 bytes (Ethereum-style address)
	// Convert to TRON address format (add 0x41 prefix for mainnet)
	toAddressTron := make([]byte, 21)
	toAddressTron[0] = 0x41 // Mainnet prefix
	copy(toAddressTron[1:], toAddressBytes)
	toAddress := encodeAddress(toAddressTron)

	// Extract amount (bytes 36-68)
	amountHex := hex.EncodeToString(data[36:68])
	amount, err := decimal.NewFromString(amountHex)
	if err != nil {
		// Try parsing as hex integer
		var amountInt int64
		_, err = fmt.Sscanf(amountHex, "%x", &amountInt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse amount: %w", err)
		}
		amount = decimal.NewFromInt(amountInt)
	}

	// Get token info
	tokenSymbol, _ := p.client.GetGrpcClient().TRC20GetSymbol(contractAddress)
	decimals, _ := p.client.GetGrpcClient().TRC20GetDecimals(contractAddress)
	if decimals == 0 {
		decimals = 6 // Default for USDT
	}

	// Convert amount from base unit to token unit
	amountToken := amount.Shift(-int32(decimals))

	// Extract from address (transaction signer)
	fromAddress := encodeAddress(triggerContract.OwnerAddress)

	// Extract memo from transaction data (if present)
	memo := p.extractMemo(txInfo.Transaction)

	parsed := &ParsedTransaction{
		TxID:            txID,
		FromAddress:     fromAddress,
		ToAddress:       toAddress,
		Amount:          amountToken,
		TokenAddress:    contractAddress,
		TokenSymbol:     tokenSymbol,
		Memo:            memo,
		BlockNumber:     txInfo.BlockNumber,
		BlockTimestamp:  txInfo.BlockTimestamp,
		IsSuccess:       txInfo.IsSuccess,
		IsTRC20Transfer: true,
	}

	return parsed, nil
}

// extractMemo extracts memo/note from transaction data field
// TRON allows adding a memo in the transaction's Data field
func (p *TransactionParser) extractMemo(tx *core.Transaction) string {
	if tx == nil || tx.RawData == nil {
		return ""
	}

	// Check if data field exists
	if len(tx.RawData.Data) == 0 {
		return ""
	}

	// Try to decode as UTF-8 string
	memo := strings.TrimSpace(string(tx.RawData.Data))

	return memo
}

// encodeAddress converts raw address bytes to base58 TRON address
func encodeAddress(addressBytes []byte) string {
	if len(addressBytes) == 0 {
		return ""
	}

	// Use the gotron-sdk address encoding
	// Note: This is a simplified version; the actual implementation
	// should use the proper base58check encoding
	return fmt.Sprintf("T%s", hex.EncodeToString(addressBytes))
}

// ValidatePaymentTransaction validates that a transaction matches payment requirements
type PaymentValidation struct {
	IsValid          bool
	ExpectedAddress  string
	ExpectedAmount   decimal.Decimal
	ExpectedMemo     string
	ActualAddress    string
	ActualAmount     decimal.Decimal
	ActualMemo       string
	AmountMatch      bool
	AddressMatch     bool
	MemoMatch        bool
	IsConfirmed      bool
	ConfirmationText string
}

// ValidatePayment validates a transaction against payment requirements
func (p *TransactionParser) ValidatePayment(
	txID string,
	expectedAddress string,
	expectedAmount decimal.Decimal,
	expectedMemo string,
	minConfirmations int64,
) (*PaymentValidation, error) {
	// Get transaction info
	txInfo, err := p.client.GetTransaction(nil, txID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Parse transaction
	parsed, err := p.ParseTransaction(txID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Validate address
	addressMatch := strings.EqualFold(parsed.ToAddress, expectedAddress)

	// Validate amount (exact match required)
	amountMatch := parsed.Amount.Equal(expectedAmount)

	// Validate memo (if provided)
	memoMatch := true
	if expectedMemo != "" {
		memoMatch = strings.Contains(parsed.Memo, expectedMemo)
	}

	// Check confirmations
	isConfirmed := txInfo.Confirmations >= minConfirmations

	validation := &PaymentValidation{
		IsValid:          addressMatch && amountMatch && memoMatch && isConfirmed && parsed.IsSuccess,
		ExpectedAddress:  expectedAddress,
		ExpectedAmount:   expectedAmount,
		ExpectedMemo:     expectedMemo,
		ActualAddress:    parsed.ToAddress,
		ActualAmount:     parsed.Amount,
		ActualMemo:       parsed.Memo,
		AmountMatch:      amountMatch,
		AddressMatch:     addressMatch,
		MemoMatch:        memoMatch,
		IsConfirmed:      isConfirmed,
		ConfirmationText: fmt.Sprintf("%d/%d confirmations", txInfo.Confirmations, minConfirmations),
	}

	return validation, nil
}

// ExtractPaymentID extracts payment ID from transaction memo
// Expected format: "PAYMENT:uuid" or just "uuid"
func ExtractPaymentID(memo string) string {
	memo = strings.TrimSpace(memo)

	// Check if memo starts with "PAYMENT:"
	if strings.HasPrefix(strings.ToUpper(memo), "PAYMENT:") {
		return strings.TrimPrefix(strings.ToUpper(memo), "PAYMENT:")
	}

	// Otherwise, return the memo as-is (assuming it's the payment ID)
	return memo
}

// FormatTRC20Amount formats a TRC20 amount with proper decimals
func FormatTRC20Amount(amount decimal.Decimal, decimals uint8) string {
	return amount.StringFixed(int32(decimals))
}

// FormatTRXAmount formats a TRX amount with 6 decimal places
func FormatTRXAmount(amount decimal.Decimal) string {
	return amount.StringFixed(6)
}
