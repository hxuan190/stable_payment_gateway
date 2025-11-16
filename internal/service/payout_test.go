package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Test the payout fee calculation logic
func TestCalculatePayoutFee(t *testing.T) {
	tests := []struct {
		name        string
		amount      decimal.Decimal
		expectedFee decimal.Decimal
	}{
		{
			name:        "Small amount uses minimum fee",
			amount:      decimal.NewFromInt(1000000), // 1M VND
			expectedFee: decimal.NewFromInt(MinimumPayoutFeeVND),
		},
		{
			name:        "Large amount uses percentage fee",
			amount:      decimal.NewFromInt(100000000), // 100M VND
			expectedFee: decimal.NewFromInt(500000),    // 0.5% = 500k VND
		},
		{
			name:        "Medium amount",
			amount:      decimal.NewFromInt(10000000), // 10M VND
			expectedFee: decimal.NewFromInt(50000),    // 0.5% = 50k VND
		},
		{
			name:        "Amount exactly at minimum",
			amount:      decimal.NewFromInt(2000000), // 2M VND
			expectedFee: decimal.NewFromInt(MinimumPayoutFeeVND), // 10k VND (minimum)
		},
		{
			name:        "Very large amount",
			amount:      decimal.NewFromInt(500000000), // 500M VND (max allowed)
			expectedFee: decimal.NewFromInt(2500000),   // 0.5% = 2.5M VND
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := CalculatePayoutFee(tt.amount)
			assert.True(t, fee.Equal(tt.expectedFee),
				"Expected fee %s, got %s for amount %s",
				tt.expectedFee, fee, tt.amount)
		})
	}
}

// Test payout request input validation
func TestRequestPayoutInput_Validate(t *testing.T) {
	tests := []struct {
		name        string
		input       RequestPayoutInput
		wantErr     bool
		expectedErr error
	}{
		{
			name: "valid input",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr: false,
		},
		{
			name: "empty merchant ID",
			input: RequestPayoutInput{
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutMerchantNotFound,
		},
		{
			name: "zero amount",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.Zero,
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutInvalidAmount,
		},
		{
			name: "negative amount",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(-1000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutInvalidAmount,
		},
		{
			name: "below minimum amount",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(500000), // Below 1M minimum
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutBelowMinimum,
		},
		{
			name: "above maximum amount",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(600000000), // Above 500M maximum
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr: true,
		},
		{
			name: "invalid bank account name - too short",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "A", // Too short
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutInvalidBankDetails,
		},
		{
			name: "invalid bank account name - empty",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "", // Empty
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutInvalidBankDetails,
		},
		{
			name: "invalid bank account number - too short",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "123", // Too short
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutInvalidBankDetails,
		},
		{
			name: "invalid bank account number - empty",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "", // Empty
				BankName:          "Test Bank",
			},
			wantErr:     true,
			expectedErr: ErrPayoutInvalidBankDetails,
		},
		{
			name: "invalid bank name - empty",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "", // Empty
			},
			wantErr:     true,
			expectedErr: ErrPayoutInvalidBankDetails,
		},
		{
			name: "valid with optional bank branch",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
				BankBranch:        "Main Branch",
			},
			wantErr: false,
		},
		{
			name: "valid with optional notes",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "John Doe",
				BankAccountNumber: "12345678",
				BankName:          "Test Bank",
				Notes:             "Emergency withdrawal",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				assert.Error(t, err, "Expected validation error but got none")
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr,
						"Expected error %v but got %v", tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err, "Expected no validation error but got: %v", err)
			}
		})
	}
}

// Test payout fee calculation edge cases
func TestCalculatePayoutFee_EdgeCases(t *testing.T) {
	t.Run("Fee is always positive", func(t *testing.T) {
		amounts := []decimal.Decimal{
			decimal.NewFromInt(MinimumPayoutAmountVND),
			decimal.NewFromInt(10000000),
			decimal.NewFromInt(100000000),
			decimal.NewFromInt(MaximumPayoutAmountVND),
		}

		for _, amount := range amounts {
			fee := CalculatePayoutFee(amount)
			assert.True(t, fee.GreaterThan(decimal.Zero),
				"Fee should be positive for amount %s, got %s", amount, fee)
		}
	})

	t.Run("Fee respects minimum", func(t *testing.T) {
		// For small amounts, fee should equal minimum
		smallAmount := decimal.NewFromInt(MinimumPayoutAmountVND)
		fee := CalculatePayoutFee(smallAmount)
		minFee := decimal.NewFromInt(MinimumPayoutFeeVND)

		assert.True(t, fee.GreaterThanOrEqual(minFee),
			"Fee %s should be at least minimum fee %s", fee, minFee)
	})

	t.Run("Fee is properly rounded", func(t *testing.T) {
		// Fee should be rounded to whole VND
		amount := decimal.NewFromInt(3333333) // Will produce fractional fee
		fee := CalculatePayoutFee(amount)

		// Check that fee has no decimal places
		truncated := fee.Truncate(0)
		assert.True(t, fee.Equal(truncated),
			"Fee should be rounded to whole VND, got %s", fee)
	})
}

// Test payout constants are sensible
func TestPayoutConstants(t *testing.T) {
	t.Run("Minimum payout amount is reasonable", func(t *testing.T) {
		assert.Equal(t, int64(1000000), int64(MinimumPayoutAmountVND),
			"Minimum payout should be 1M VND")
	})

	t.Run("Maximum payout amount is reasonable", func(t *testing.T) {
		assert.Equal(t, int64(500000000), int64(MaximumPayoutAmountVND),
			"Maximum payout should be 500M VND")
	})

	t.Run("Minimum fee is reasonable", func(t *testing.T) {
		assert.Equal(t, int64(10000), int64(MinimumPayoutFeeVND),
			"Minimum fee should be 10k VND")
	})

	t.Run("Fee percentage is reasonable", func(t *testing.T) {
		assert.Equal(t, 0.005, PayoutFeePercentage,
			"Fee percentage should be 0.5%")
	})

	t.Run("Minimum payout less than maximum", func(t *testing.T) {
		assert.Less(t, int64(MinimumPayoutAmountVND), int64(MaximumPayoutAmountVND),
			"Minimum payout should be less than maximum")
	})
}

// Test payout input validation with Vietnamese bank details
func TestRequestPayoutInput_VietnameseBankDetails(t *testing.T) {
	tests := []struct {
		name    string
		input   RequestPayoutInput
		wantErr bool
	}{
		{
			name: "Vietcombank account",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "Nguyen Van A",
				BankAccountNumber: "0123456789",
				BankName:          "Vietcombank",
				BankBranch:        "Chi nhanh TP. HCM",
			},
			wantErr: false,
		},
		{
			name: "VietinBank account",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(5000000),
				BankAccountName:   "Tran Thi B",
				BankAccountNumber: "987654321011",
				BankName:          "VietinBank",
				BankBranch:        "Chi nhanh Da Nang",
			},
			wantErr: false,
		},
		{
			name: "BIDV account",
			input: RequestPayoutInput{
				MerchantID:        "merchant-123",
				AmountVND:         decimal.NewFromInt(10000000),
				BankAccountName:   "Le Van C",
				BankAccountNumber: "11122233344",
				BankName:          "BIDV",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
