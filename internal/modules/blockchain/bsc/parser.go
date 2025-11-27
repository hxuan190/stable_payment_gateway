package bsc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// BEP20TokenTransfer represents a parsed BEP20 token transfer
type BEP20TokenTransfer struct {
	From          common.Address
	To            common.Address
	Amount        *big.Int
	TokenContract common.Address
}

// Transfer event signature: Transfer(address indexed from, address indexed to, uint256 value)
var transferEventSignature = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// parseBEP20TokenTransfer extracts BEP20 token transfer details from a transaction
// It looks for Transfer events in the transaction logs
func parseBEP20TokenTransfer(receipt *types.Receipt, expectedRecipient common.Address, tokenContract common.Address) (*BEP20TokenTransfer, error) {
	if receipt == nil {
		return nil, fmt.Errorf("receipt is nil")
	}

	// Iterate through all logs to find token transfer event
	for _, log := range receipt.Logs {
		// Check if this log is from the expected token contract
		if log.Address != tokenContract {
			continue
		}

		// Check if this is a Transfer event (Topic[0] is the event signature)
		if len(log.Topics) < 3 {
			continue
		}

		if log.Topics[0] != transferEventSignature {
			continue
		}

		// Parse Transfer event
		// Topic[0]: Event signature
		// Topic[1]: from address (indexed)
		// Topic[2]: to address (indexed)
		// Data: amount (uint256)

		from := common.BytesToAddress(log.Topics[1].Bytes())
		to := common.BytesToAddress(log.Topics[2].Bytes())

		// Verify recipient matches expected
		if to != expectedRecipient {
			continue
		}

		// Parse amount from data
		if len(log.Data) < 32 {
			return nil, fmt.Errorf("invalid transfer event data length: %d", len(log.Data))
		}

		amount := new(big.Int).SetBytes(log.Data)

		transfer := &BEP20TokenTransfer{
			From:          from,
			To:            to,
			Amount:        amount,
			TokenContract: tokenContract,
		}

		return transfer, nil
	}

	return nil, fmt.Errorf("no BEP20 token transfer found to recipient %s from contract %s",
		expectedRecipient.Hex(), tokenContract.Hex())
}

// extractMemoFromTransaction extracts the memo field from a BSC transaction
// The memo is stored in the input data of the transaction
// For BEP20 transfers, we look for the memo in additional data after the transfer parameters
func extractMemoFromTransaction(tx *types.Transaction) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("transaction is nil")
	}

	data := tx.Data()
	if len(data) == 0 {
		return "", fmt.Errorf("no data in transaction")
	}

	// BEP20 transfer method signature: transfer(address,uint256)
	// Method ID: 0xa9059cbb (first 4 bytes of keccak256("transfer(address,uint256)"))
	transferMethodID := []byte{0xa9, 0x05, 0x9c, 0xbb}

	// Check if this is a standard BEP20 transfer
	if len(data) >= 4 && bytes.Equal(data[:4], transferMethodID) {
		// Standard BEP20 transfer has 68 bytes (4 + 32 + 32)
		// If there's more data, it might be a memo
		if len(data) > 68 {
			memoData := data[68:]
			// Try to decode as string
			memo, err := decodeMemoFromBytes(memoData)
			if err == nil && memo != "" {
				return memo, nil
			}
		}
	}

	// If no memo found in standard location, check for custom memo patterns
	// Some contracts might store memo differently
	return "", fmt.Errorf("no memo found in transaction")
}

// decodeMemoFromBytes attempts to decode memo from bytes
// Handles both plain text and ABI-encoded strings
func decodeMemoFromBytes(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty data")
	}

	// Try decoding as ABI-encoded string
	// ABI string format: [offset][length][data]
	if len(data) >= 64 {
		// Skip offset (first 32 bytes)
		lengthStart := 32
		if len(data) > lengthStart+32 {
			length := new(big.Int).SetBytes(data[lengthStart : lengthStart+32])
			if length.IsInt64() {
				l := length.Int64()
				dataStart := lengthStart + 32
				if len(data) >= int(dataStart)+int(l) {
					memoBytes := data[dataStart : int(dataStart)+int(l)]
					memo := string(memoBytes)
					if isPrintable(memo) {
						return memo, nil
					}
				}
			}
		}
	}

	// Try decoding as plain text
	memo := string(data)
	if isPrintable(memo) {
		return memo, nil
	}

	// Try hex decoding
	hexStr := strings.TrimPrefix(hex.EncodeToString(data), "0x")
	if isPrintable(hexStr) {
		return hexStr, nil
	}

	return "", fmt.Errorf("unable to decode memo")
}

// isPrintable checks if a string contains only printable characters
func isPrintable(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			// Allow common whitespace characters
			if r != '\n' && r != '\r' && r != '\t' {
				return false
			}
		}
	}
	return len(s) > 0
}

// ValidatePaymentTransaction performs validation checks on a payment transaction
func ValidatePaymentTransaction(
	receipt *types.Receipt,
	expectedRecipient common.Address,
	expectedAmount *big.Int,
	tokenContract common.Address,
	tolerance *big.Int,
) error {
	if receipt == nil {
		return fmt.Errorf("receipt is nil")
	}

	// Check transaction status
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed")
	}

	// Extract BEP20 token transfer
	transfer, err := parseBEP20TokenTransfer(receipt, expectedRecipient, tokenContract)
	if err != nil {
		return fmt.Errorf("failed to parse transfer: %w", err)
	}

	// Verify recipient
	if transfer.To != expectedRecipient {
		return fmt.Errorf("recipient mismatch: expected %s, got %s",
			expectedRecipient.Hex(), transfer.To.Hex())
	}

	// Verify token contract
	if transfer.TokenContract != tokenContract {
		return fmt.Errorf("token contract mismatch: expected %s, got %s",
			tokenContract.Hex(), transfer.TokenContract.Hex())
	}

	// Verify amount (within tolerance if provided)
	if tolerance != nil && tolerance.Sign() > 0 {
		diff := new(big.Int).Sub(transfer.Amount, expectedAmount)
		diff.Abs(diff)
		if diff.Cmp(tolerance) > 0 {
			return fmt.Errorf("amount mismatch: expected %s ± %s, got %s",
				expectedAmount.String(), tolerance.String(), transfer.Amount.String())
		}
	} else {
		// Exact match required
		if transfer.Amount.Cmp(expectedAmount) != 0 {
			return fmt.Errorf("amount mismatch: expected %s, got %s",
				expectedAmount.String(), transfer.Amount.String())
		}
	}

	return nil
}

// ParseTransferFromLogs finds and parses a BEP20 Transfer event from logs
// This is a more general version that doesn't filter by recipient
func ParseTransferFromLogs(logs []*types.Log, tokenContract common.Address) ([]*BEP20TokenTransfer, error) {
	transfers := []*BEP20TokenTransfer{}

	for _, log := range logs {
		// Check if this log is from the expected token contract
		if log.Address != tokenContract {
			continue
		}

		// Check if this is a Transfer event
		if len(log.Topics) < 3 {
			continue
		}

		if log.Topics[0] != transferEventSignature {
			continue
		}

		// Parse Transfer event
		from := common.BytesToAddress(log.Topics[1].Bytes())
		to := common.BytesToAddress(log.Topics[2].Bytes())

		// Parse amount from data
		if len(log.Data) < 32 {
			continue
		}

		amount := new(big.Int).SetBytes(log.Data)

		transfer := &BEP20TokenTransfer{
			From:          from,
			To:            to,
			Amount:        amount,
			TokenContract: tokenContract,
		}

		transfers = append(transfers, transfer)
	}

	if len(transfers) == 0 {
		return nil, fmt.Errorf("no BEP20 token transfers found from contract %s", tokenContract.Hex())
	}

	return transfers, nil
}

// GetTransferEventSignature returns the Transfer event signature hash
func GetTransferEventSignature() common.Hash {
	return transferEventSignature
}
