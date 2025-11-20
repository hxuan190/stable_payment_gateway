package solana

import (
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/memo"
	"github.com/gagliardetto/solana-go/programs/token"
)

// SPLTokenTransfer represents a parsed SPL token transfer
type SPLTokenTransfer struct {
	Sender     solana.PublicKey
	Recipient  solana.PublicKey
	Amount     *big.Int
	TokenMint  solana.PublicKey
}

// extractMemoFromTransaction extracts the memo field from a Solana transaction
// The memo is used to store the payment ID
func extractMemoFromTransaction(tx *solana.Transaction) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("transaction is nil")
	}

	// Iterate through all instructions to find memo instruction
	for _, instruction := range tx.Message.Instructions {
		// Check if this is a memo program instruction
		programID := tx.Message.AccountKeys[instruction.ProgramIDIndex]

		// Memo program ID: MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr
		if programID.Equals(memo.ProgramID) {
			// The memo data is in the instruction data
			if len(instruction.Data) > 0 {
				return string(instruction.Data), nil
			}
		}
	}

	return "", fmt.Errorf("no memo instruction found in transaction")
}

// parseSPLTokenTransfer extracts SPL token transfer details from a transaction
// It looks for Transfer or TransferChecked instructions in the transaction
func parseSPLTokenTransfer(tx *solana.Transaction, expectedRecipient solana.PublicKey) (*SPLTokenTransfer, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction is nil")
	}

	// Iterate through all instructions to find token transfer
	for _, instruction := range tx.Message.Instructions {
		programID := tx.Message.AccountKeys[instruction.ProgramIDIndex]

		// Check if this is a token program instruction
		// Token Program: TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA
		if !programID.Equals(token.ProgramID) {
			continue
		}

		// Try to decode the instruction
		decodedInst, err := decodeTokenInstruction(instruction, tx.Message.AccountKeys)
		if err != nil {
			continue
		}

		// Check if it's a Transfer or TransferChecked instruction
		transfer, ok := decodedInst.(*SPLTokenTransfer)
		if !ok {
			continue
		}

		// Verify recipient matches expected
		if !transfer.Recipient.Equals(expectedRecipient) {
			continue
		}

		return transfer, nil
	}

	return nil, fmt.Errorf("no SPL token transfer found to recipient %s", expectedRecipient.String())
}

// decodeTokenInstruction decodes a token program instruction
func decodeTokenInstruction(instruction solana.CompiledInstruction, accountKeys []solana.PublicKey) (interface{}, error) {
	if len(instruction.Data) == 0 {
		return nil, fmt.Errorf("instruction data is empty")
	}

	// The first byte is the instruction discriminator
	instructionType := instruction.Data[0]

	switch instructionType {
	case 3: // Transfer instruction
		return decodeTransferInstruction(instruction, accountKeys)
	case 12: // TransferChecked instruction
		return decodeTransferCheckedInstruction(instruction, accountKeys)
	default:
		return nil, fmt.Errorf("unsupported token instruction type: %d", instructionType)
	}
}

// decodeTransferInstruction decodes a Token Program Transfer instruction
// Transfer instruction format:
// - [0] discriminator (3)
// - [1..8] amount (u64, little-endian)
func decodeTransferInstruction(instruction solana.CompiledInstruction, accountKeys []solana.PublicKey) (*SPLTokenTransfer, error) {
	if len(instruction.Data) < 9 {
		return nil, fmt.Errorf("invalid transfer instruction data length: %d", len(instruction.Data))
	}

	// Parse amount (u64, little-endian)
	amountBytes := instruction.Data[1:9]
	amount := new(big.Int).SetBytes(reverseBytes(amountBytes))

	// Get accounts
	// Transfer instruction accounts:
	// 0: source account
	// 1: destination account
	// 2: authority
	if len(instruction.Accounts) < 3 {
		return nil, fmt.Errorf("invalid transfer instruction accounts length: %d", len(instruction.Accounts))
	}

	sourceAccountIdx := instruction.Accounts[0]
	destAccountIdx := instruction.Accounts[1]
	authorityIdx := instruction.Accounts[2]

	if int(sourceAccountIdx) >= len(accountKeys) ||
		int(destAccountIdx) >= len(accountKeys) ||
		int(authorityIdx) >= len(accountKeys) {
		return nil, fmt.Errorf("account index out of bounds")
	}

	// sourceAccount := accountKeys[sourceAccountIdx]  // Not used in MVP
	destAccount := accountKeys[destAccountIdx]
	authority := accountKeys[authorityIdx]

	// For SPL tokens, we need to derive the actual wallet addresses from token accounts
	// For MVP, we'll use the authority as sender and dest account as recipient
	// In production, you'd query the account data to get the actual owner

	transfer := &SPLTokenTransfer{
		Sender:    authority,
		Recipient: destAccount,
		Amount:    amount,
		// TokenMint will be resolved separately
	}

	return transfer, nil
}

// decodeTransferCheckedInstruction decodes a Token Program TransferChecked instruction
// TransferChecked instruction format:
// - [0] discriminator (12)
// - [1..8] amount (u64, little-endian)
// - [9] decimals (u8)
func decodeTransferCheckedInstruction(instruction solana.CompiledInstruction, accountKeys []solana.PublicKey) (*SPLTokenTransfer, error) {
	if len(instruction.Data) < 10 {
		return nil, fmt.Errorf("invalid transfer checked instruction data length: %d", len(instruction.Data))
	}

	// Parse amount (u64, little-endian)
	amountBytes := instruction.Data[1:9]
	amount := new(big.Int).SetBytes(reverseBytes(amountBytes))

	// Get accounts
	// TransferChecked instruction accounts:
	// 0: source account
	// 1: mint
	// 2: destination account
	// 3: authority
	if len(instruction.Accounts) < 4 {
		return nil, fmt.Errorf("invalid transfer checked instruction accounts length: %d", len(instruction.Accounts))
	}

	sourceAccountIdx := instruction.Accounts[0]
	mintIdx := instruction.Accounts[1]
	destAccountIdx := instruction.Accounts[2]
	authorityIdx := instruction.Accounts[3]

	if int(sourceAccountIdx) >= len(accountKeys) ||
		int(mintIdx) >= len(accountKeys) ||
		int(destAccountIdx) >= len(accountKeys) ||
		int(authorityIdx) >= len(accountKeys) {
		return nil, fmt.Errorf("account index out of bounds")
	}

	// sourceAccount := accountKeys[sourceAccountIdx]  // Not used in MVP
	mint := accountKeys[mintIdx]
	destAccount := accountKeys[destAccountIdx]
	authority := accountKeys[authorityIdx]

	// Use authority as sender
	transfer := &SPLTokenTransfer{
		Sender:    authority,
		Recipient: destAccount,
		Amount:    amount,
		TokenMint: mint,
	}

	return transfer, nil
}

// reverseBytes reverses a byte slice (for little-endian to big-endian conversion)
func reverseBytes(b []byte) []byte {
	result := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		result[i] = b[len(b)-1-i]
	}
	return result
}

// Note: ParseTransactionForPayment is not needed as parsing is done in listener.go
// The Amount field in PaymentDetails uses decimal.Decimal which is converted in the listener

// ValidatePaymentTransaction performs validation checks on a payment transaction
func ValidatePaymentTransaction(tx *solana.Transaction, expectedRecipient solana.PublicKey, expectedAmount *big.Int, tolerance *big.Int) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	// Extract SPL token transfer
	transfer, err := parseSPLTokenTransfer(tx, expectedRecipient)
	if err != nil {
		return fmt.Errorf("failed to parse transfer: %w", err)
	}

	// Verify recipient
	if !transfer.Recipient.Equals(expectedRecipient) {
		return fmt.Errorf("recipient mismatch: expected %s, got %s",
			expectedRecipient.String(), transfer.Recipient.String())
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
