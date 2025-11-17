package solana

import (
	"math/big"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/memo"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMemoFromTransaction(t *testing.T) {
	tests := []struct {
		name        string
		setupTx     func() *solana.Transaction
		expectedMemo string
		expectError bool
	}{
		{
			name: "valid memo instruction",
			setupTx: func() *solana.Transaction {
				tx := &solana.Transaction{
					Message: solana.Message{
						AccountKeys: []solana.PublicKey{
							solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
							memo.ProgramID,
						},
						Instructions: []solana.CompiledInstruction{
							{
								ProgramIDIndex: 1,
								Data:           []byte("payment-123"),
							},
						},
					},
				}
				return tx
			},
			expectedMemo: "payment-123",
			expectError:  false,
		},
		{
			name: "no memo instruction",
			setupTx: func() *solana.Transaction {
				tx := &solana.Transaction{
					Message: solana.Message{
						AccountKeys: []solana.PublicKey{
							solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
							solana.MustPublicKeyFromBase58("22222222222222222222222222222222"),
						},
						Instructions: []solana.CompiledInstruction{
							{
								ProgramIDIndex: 1,
								Data:           []byte("some-data"),
							},
						},
					},
				}
				return tx
			},
			expectError: true,
		},
		{
			name: "empty memo data",
			setupTx: func() *solana.Transaction {
				tx := &solana.Transaction{
					Message: solana.Message{
						AccountKeys: []solana.PublicKey{
							solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
							memo.ProgramID,
						},
						Instructions: []solana.CompiledInstruction{
							{
								ProgramIDIndex: 1,
								Data:           []byte{},
							},
						},
					},
				}
				return tx
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := tt.setupTx()
			memo, err := extractMemoFromTransaction(tx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMemo, memo)
			}
		})
	}
}

func TestReverseBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "simple bytes",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: []byte{0x04, 0x03, 0x02, 0x01},
		},
		{
			name:     "single byte",
			input:    []byte{0x42},
			expected: []byte{0x42},
		},
		{
			name:     "empty bytes",
			input:    []byte{},
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reverseBytes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeTransferInstruction(t *testing.T) {
	// Test data: Transfer 1000000 tokens (1 USDT with 6 decimals)
	amount := uint64(1000000)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	// Create instruction data
	instructionData := append([]byte{3}, amountBytes...) // 3 = Transfer discriminator

	sender := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	sourceAccount := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	destAccount := solana.MustPublicKeyFromBase58("33333333333333333333333333333333")
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	accountKeys := []solana.PublicKey{
		sender,
		sourceAccount,
		destAccount,
		authority,
	}

	instruction := solana.CompiledInstruction{
		ProgramIDIndex: 0,
		Accounts:       []uint16{1, 2, 3},
		Data:           instructionData,
	}

	transfer, err := decodeTransferInstruction(instruction, accountKeys)
	require.NoError(t, err)
	require.NotNil(t, transfer)

	assert.Equal(t, authority, transfer.Sender)
	assert.Equal(t, destAccount, transfer.Recipient)
	assert.Equal(t, big.NewInt(int64(amount)), transfer.Amount)
}

func TestDecodeTransferCheckedInstruction(t *testing.T) {
	// Test data: Transfer 1000000 tokens (1 USDT with 6 decimals)
	amount := uint64(1000000)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	// Create instruction data: discriminator + amount + decimals
	instructionData := append([]byte{12}, amountBytes...) // 12 = TransferChecked discriminator
	instructionData = append(instructionData, 6)          // 6 decimals

	sender := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	sourceAccount := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	mintAccount := solana.MustPublicKeyFromBase58("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB") // USDT mint
	destAccount := solana.MustPublicKeyFromBase58("33333333333333333333333333333333")
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	accountKeys := []solana.PublicKey{
		sender,
		sourceAccount,
		mintAccount,
		destAccount,
		authority,
	}

	instruction := solana.CompiledInstruction{
		ProgramIDIndex: 0,
		Accounts:       []uint16{1, 2, 3, 4},
		Data:           instructionData,
	}

	transfer, err := decodeTransferCheckedInstruction(instruction, accountKeys)
	require.NoError(t, err)
	require.NotNil(t, transfer)

	assert.Equal(t, authority, transfer.Sender)
	assert.Equal(t, destAccount, transfer.Recipient)
	assert.Equal(t, big.NewInt(int64(amount)), transfer.Amount)
	assert.Equal(t, mintAccount, transfer.TokenMint)
}

func TestParseSPLTokenTransfer(t *testing.T) {
	// Create a mock transaction with a transfer instruction
	amount := uint64(1000000)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	instructionData := append([]byte{3}, amountBytes...)

	walletAddress := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	sourceAccount := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	destAccount := walletAddress
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	accountKeys := []solana.PublicKey{
		token.ProgramID,
		sourceAccount,
		destAccount,
		authority,
	}

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys: accountKeys,
			Instructions: []solana.CompiledInstruction{
				{
					ProgramIDIndex: 0,
					Accounts:       []uint16{1, 2, 3},
					Data:           instructionData,
				},
			},
		},
	}

	transfer, err := parseSPLTokenTransfer(tx, walletAddress)
	require.NoError(t, err)
	require.NotNil(t, transfer)

	assert.Equal(t, authority, transfer.Sender)
	assert.Equal(t, destAccount, transfer.Recipient)
	assert.Equal(t, big.NewInt(int64(amount)), transfer.Amount)
}

func TestParseSPLTokenTransfer_NoTokenInstruction(t *testing.T) {
	walletAddress := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys: []solana.PublicKey{
				solana.MustPublicKeyFromBase58("22222222222222222222222222222222"),
			},
			Instructions: []solana.CompiledInstruction{
				{
					ProgramIDIndex: 0,
					Data:           []byte{0x00},
				},
			},
		},
	}

	_, err := parseSPLTokenTransfer(tx, walletAddress)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no SPL token transfer found")
}

func TestValidatePaymentTransaction(t *testing.T) {
	amount := uint64(1000000)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	instructionData := append([]byte{3}, amountBytes...)

	expectedRecipient := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	sourceAccount := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	accountKeys := []solana.PublicKey{
		token.ProgramID,
		sourceAccount,
		expectedRecipient,
		authority,
	}

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys: accountKeys,
			Instructions: []solana.CompiledInstruction{
				{
					ProgramIDIndex: 0,
					Accounts:       []uint16{1, 2, 3},
					Data:           instructionData,
				},
			},
		},
	}

	tests := []struct {
		name           string
		expectedAmount *big.Int
		tolerance      *big.Int
		expectError    bool
	}{
		{
			name:           "exact match",
			expectedAmount: big.NewInt(int64(amount)),
			tolerance:      nil,
			expectError:    false,
		},
		{
			name:           "within tolerance",
			expectedAmount: big.NewInt(int64(amount - 100)),
			tolerance:      big.NewInt(1000),
			expectError:    false,
		},
		{
			name:           "outside tolerance",
			expectedAmount: big.NewInt(int64(amount - 2000)),
			tolerance:      big.NewInt(1000),
			expectError:    true,
		},
		{
			name:           "amount mismatch without tolerance",
			expectedAmount: big.NewInt(int64(amount + 1)),
			tolerance:      nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePaymentTransaction(tx, expectedRecipient, tt.expectedAmount, tt.tolerance)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePaymentTransaction_RecipientMismatch(t *testing.T) {
	amount := uint64(1000000)
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[i] = byte(amount >> (i * 8))
	}

	instructionData := append([]byte{3}, amountBytes...)

	actualRecipient := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	expectedRecipient := solana.MustPublicKeyFromBase58("99999999999999999999999999999999")
	sourceAccount := solana.MustPublicKeyFromBase58("22222222222222222222222222222222")
	authority := solana.MustPublicKeyFromBase58("44444444444444444444444444444444")

	accountKeys := []solana.PublicKey{
		token.ProgramID,
		sourceAccount,
		actualRecipient,
		authority,
	}

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys: accountKeys,
			Instructions: []solana.CompiledInstruction{
				{
					ProgramIDIndex: 0,
					Accounts:       []uint16{1, 2, 3},
					Data:           instructionData,
				},
			},
		},
	}

	err := ValidatePaymentTransaction(tx, expectedRecipient, big.NewInt(int64(amount)), nil)
	assert.Error(t, err)
	// The error will be "no SPL token transfer found to recipient" since we're looking for wrong recipient
}
