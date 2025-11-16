package service

import (
	"errors"
	"testing"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Note: Tests that require database transactions are better suited for integration tests
// These unit tests focus on validation logic

func TestRecordPaymentReceived_InvalidInputs(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	tests := []struct {
		name           string
		paymentID      string
		merchantID     string
		amountCrypto   decimal.Decimal
		cryptoCurrency string
		amountVND      decimal.Decimal
		expectedError  error
	}{
		{
			name:           "Empty payment ID",
			paymentID:      "",
			merchantID:     "merchant-123",
			amountCrypto:   decimal.NewFromFloat(100),
			cryptoCurrency: "USDT",
			amountVND:      decimal.NewFromFloat(2300000),
			expectedError:  ErrLedgerInvalidReferenceID,
		},
		{
			name:           "Empty merchant ID",
			paymentID:      "payment-123",
			merchantID:     "",
			amountCrypto:   decimal.NewFromFloat(100),
			cryptoCurrency: "USDT",
			amountVND:      decimal.NewFromFloat(2300000),
			expectedError:  ErrLedgerInvalidMerchantID,
		},
		{
			name:           "Zero VND amount",
			paymentID:      "payment-123",
			merchantID:     "merchant-123",
			amountCrypto:   decimal.NewFromFloat(100),
			cryptoCurrency: "USDT",
			amountVND:      decimal.Zero,
			expectedError:  ErrLedgerInvalidAmount,
		},
		{
			name:           "Negative VND amount",
			paymentID:      "payment-123",
			merchantID:     "merchant-123",
			amountCrypto:   decimal.NewFromFloat(100),
			cryptoCurrency: "USDT",
			amountVND:      decimal.NewFromFloat(-1000),
			expectedError:  ErrLedgerInvalidAmount,
		},
		{
			name:           "Zero crypto amount",
			paymentID:      "payment-123",
			merchantID:     "merchant-123",
			amountCrypto:   decimal.Zero,
			cryptoCurrency: "USDT",
			amountVND:      decimal.NewFromFloat(2300000),
			expectedError:  ErrLedgerInvalidAmount,
		},
		{
			name:           "Empty crypto currency",
			paymentID:      "payment-123",
			merchantID:     "merchant-123",
			amountCrypto:   decimal.NewFromFloat(100),
			cryptoCurrency: "",
			amountVND:      decimal.NewFromFloat(2300000),
			expectedError:  ErrLedgerInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RecordPaymentReceived(tt.paymentID, tt.merchantID, tt.amountCrypto, tt.cryptoCurrency, tt.amountVND)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func TestRecordPaymentConfirmed_InvalidInputs(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	tests := []struct {
		name          string
		paymentID     string
		merchantID    string
		amountVND     decimal.Decimal
		feeVND        decimal.Decimal
		expectedError error
	}{
		{
			name:          "Empty payment ID",
			paymentID:     "",
			merchantID:    "merchant-123",
			amountVND:     decimal.NewFromFloat(2300000),
			feeVND:        decimal.NewFromFloat(23000),
			expectedError: ErrLedgerInvalidReferenceID,
		},
		{
			name:          "Empty merchant ID",
			paymentID:     "payment-123",
			merchantID:    "",
			amountVND:     decimal.NewFromFloat(2300000),
			feeVND:        decimal.NewFromFloat(23000),
			expectedError: ErrLedgerInvalidMerchantID,
		},
		{
			name:          "Zero amount",
			paymentID:     "payment-123",
			merchantID:    "merchant-123",
			amountVND:     decimal.Zero,
			feeVND:        decimal.NewFromFloat(23000),
			expectedError: ErrLedgerInvalidAmount,
		},
		{
			name:          "Negative fee",
			paymentID:     "payment-123",
			merchantID:    "merchant-123",
			amountVND:     decimal.NewFromFloat(2300000),
			feeVND:        decimal.NewFromFloat(-1000),
			expectedError: errors.New("fee cannot be negative"),
		},
		{
			name:          "Fee greater than amount",
			paymentID:     "payment-123",
			merchantID:    "merchant-123",
			amountVND:     decimal.NewFromFloat(2300000),
			feeVND:        decimal.NewFromFloat(3000000),
			expectedError: errors.New("fee cannot be greater than or equal to amount"),
		},
		{
			name:          "Fee equal to amount",
			paymentID:     "payment-123",
			merchantID:    "merchant-123",
			amountVND:     decimal.NewFromFloat(2300000),
			feeVND:        decimal.NewFromFloat(2300000),
			expectedError: errors.New("fee cannot be greater than or equal to amount"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RecordPaymentConfirmed(tt.paymentID, tt.merchantID, tt.amountVND, tt.feeVND)
			if tt.expectedError != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestRecordPayoutRequested_InvalidInputs(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	tests := []struct {
		name          string
		payoutID      string
		merchantID    string
		amount        decimal.Decimal
		expectedError error
	}{
		{
			name:          "Empty payout ID",
			payoutID:      "",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(1000000),
			expectedError: ErrLedgerInvalidReferenceID,
		},
		{
			name:          "Empty merchant ID",
			payoutID:      "payout-123",
			merchantID:    "",
			amount:        decimal.NewFromFloat(1000000),
			expectedError: ErrLedgerInvalidMerchantID,
		},
		{
			name:          "Zero amount",
			payoutID:      "payout-123",
			merchantID:    "merchant-123",
			amount:        decimal.Zero,
			expectedError: ErrLedgerInvalidAmount,
		},
		{
			name:          "Negative amount",
			payoutID:      "payout-123",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(-1000),
			expectedError: ErrLedgerInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RecordPayoutRequested(tt.payoutID, tt.merchantID, tt.amount)
			if tt.expectedError != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestRecordPayoutCompleted_InvalidInputs(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	tests := []struct {
		name          string
		payoutID      string
		merchantID    string
		amount        decimal.Decimal
		feeVND        decimal.Decimal
		expectedError error
	}{
		{
			name:          "Empty payout ID",
			payoutID:      "",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(1000000),
			feeVND:        decimal.NewFromFloat(10000),
			expectedError: ErrLedgerInvalidReferenceID,
		},
		{
			name:          "Empty merchant ID",
			payoutID:      "payout-123",
			merchantID:    "",
			amount:        decimal.NewFromFloat(1000000),
			feeVND:        decimal.NewFromFloat(10000),
			expectedError: ErrLedgerInvalidMerchantID,
		},
		{
			name:          "Zero amount",
			payoutID:      "payout-123",
			merchantID:    "merchant-123",
			amount:        decimal.Zero,
			feeVND:        decimal.NewFromFloat(10000),
			expectedError: ErrLedgerInvalidAmount,
		},
		{
			name:          "Negative fee",
			payoutID:      "payout-123",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(1000000),
			feeVND:        decimal.NewFromFloat(-1000),
			expectedError: errors.New("fee cannot be negative"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RecordPayoutCompleted(tt.payoutID, tt.merchantID, tt.amount, tt.feeVND)
			if tt.expectedError != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestRecordPayoutCancelled_InvalidInputs(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	tests := []struct {
		name          string
		payoutID      string
		merchantID    string
		amount        decimal.Decimal
		expectedError error
	}{
		{
			name:          "Empty payout ID",
			payoutID:      "",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(1000000),
			expectedError: ErrLedgerInvalidReferenceID,
		},
		{
			name:          "Empty merchant ID",
			payoutID:      "payout-123",
			merchantID:    "",
			amount:        decimal.NewFromFloat(1000000),
			expectedError: ErrLedgerInvalidMerchantID,
		},
		{
			name:          "Zero amount",
			payoutID:      "payout-123",
			merchantID:    "merchant-123",
			amount:        decimal.Zero,
			expectedError: ErrLedgerInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RecordPayoutCancelled(tt.payoutID, tt.merchantID, tt.amount)
			if tt.expectedError != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestRecordOTCConversion_InvalidInputs(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	tests := []struct {
		name           string
		referenceID    string
		cryptoAmount   decimal.Decimal
		cryptoCurrency string
		vndAmount      decimal.Decimal
		spreadAmount   decimal.Decimal
		expectedError  error
	}{
		{
			name:           "Empty reference ID",
			referenceID:    "",
			cryptoAmount:   decimal.NewFromFloat(1000),
			cryptoCurrency: "USDT",
			vndAmount:      decimal.NewFromFloat(23000000),
			spreadAmount:   decimal.Zero,
			expectedError:  ErrLedgerInvalidReferenceID,
		},
		{
			name:           "Zero crypto amount",
			referenceID:    "otc-123",
			cryptoAmount:   decimal.Zero,
			cryptoCurrency: "USDT",
			vndAmount:      decimal.NewFromFloat(23000000),
			spreadAmount:   decimal.Zero,
			expectedError:  ErrLedgerInvalidAmount,
		},
		{
			name:           "Zero VND amount",
			referenceID:    "otc-123",
			cryptoAmount:   decimal.NewFromFloat(1000),
			cryptoCurrency: "USDT",
			vndAmount:      decimal.Zero,
			spreadAmount:   decimal.Zero,
			expectedError:  ErrLedgerInvalidAmount,
		},
		{
			name:           "Empty crypto currency",
			referenceID:    "otc-123",
			cryptoAmount:   decimal.NewFromFloat(1000),
			cryptoCurrency: "",
			vndAmount:      decimal.NewFromFloat(23000000),
			spreadAmount:   decimal.Zero,
			expectedError:  ErrLedgerInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RecordOTCConversion(tt.referenceID, tt.cryptoAmount, tt.cryptoCurrency, tt.vndAmount, tt.spreadAmount)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func TestGetMerchantLedgerEntries_EmptyMerchantID(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	_, err := service.GetMerchantLedgerEntries("", 10, 0)

	assert.ErrorIs(t, err, ErrLedgerInvalidMerchantID)
}

func TestGetTransactionDetails_EmptyTransactionGroup(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	_, err := service.GetTransactionDetails("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction group cannot be empty")
}

func TestGetAccountBalance_EmptyAccountName(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	_, err := service.GetAccountBalance("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account name cannot be empty")
}

func TestHelperFunctions(t *testing.T) {
	// For helper function tests, we don't need actual repository implementations
	service := &LedgerService{}

	merchantID := "merchant-123"

	t.Run("getMerchantPendingAccount", func(t *testing.T) {
		account := service.getMerchantPendingAccount(merchantID)
		expected := "merchant_pending:merchant-123"
		assert.Equal(t, expected, account)
	})

	t.Run("getMerchantAvailableAccount", func(t *testing.T) {
		account := service.getMerchantAvailableAccount(merchantID)
		expected := "merchant_available:merchant-123"
		assert.Equal(t, expected, account)
	})

	t.Run("getMerchantReservedAccount", func(t *testing.T) {
		account := service.getMerchantReservedAccount(merchantID)
		expected := "merchant_reserved:merchant-123"
		assert.Equal(t, expected, account)
	})
}

func TestValidateBasicInputs(t *testing.T) {
	// For validation tests, we don't need actual repository implementations
	service := &LedgerService{}

	tests := []struct {
		name          string
		referenceID   string
		merchantID    string
		amount        decimal.Decimal
		currency      string
		expectedError error
	}{
		{
			name:          "Valid inputs",
			referenceID:   "ref-123",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(1000),
			currency:      "VND",
			expectedError: nil,
		},
		{
			name:          "Empty reference ID",
			referenceID:   "",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(1000),
			currency:      "VND",
			expectedError: ErrLedgerInvalidReferenceID,
		},
		{
			name:          "Empty merchant ID",
			referenceID:   "ref-123",
			merchantID:    "",
			amount:        decimal.NewFromFloat(1000),
			currency:      "VND",
			expectedError: ErrLedgerInvalidMerchantID,
		},
		{
			name:          "Zero amount",
			referenceID:   "ref-123",
			merchantID:    "merchant-123",
			amount:        decimal.Zero,
			currency:      "VND",
			expectedError: ErrLedgerInvalidAmount,
		},
		{
			name:          "Negative amount",
			referenceID:   "ref-123",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(-100),
			currency:      "VND",
			expectedError: ErrLedgerInvalidAmount,
		},
		{
			name:          "Empty currency",
			referenceID:   "ref-123",
			merchantID:    "merchant-123",
			amount:        decimal.NewFromFloat(1000),
			currency:      "",
			expectedError: ErrLedgerInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateBasicInputs(tt.referenceID, tt.merchantID, tt.amount, tt.currency)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateSpreadEntry(t *testing.T) {
	// For helper function tests, we don't need actual repository implementations
	service := &LedgerService{}

	referenceID := "otc-123"
	transactionGroup := "txn-group-123"

	t.Run("Positive spread (revenue)", func(t *testing.T) {
		spreadAmount := decimal.NewFromFloat(100000)
		entries := service.createSpreadEntry(referenceID, spreadAmount, transactionGroup)

		assert.Len(t, entries, 1)
		assert.Equal(t, AccountOTCSpread, entries[0].CreditAccount)
		assert.Equal(t, model.EntryTypeCredit, entries[0].EntryType)
		assert.True(t, entries[0].Amount.Equal(spreadAmount))
	})

	t.Run("Negative spread (loss)", func(t *testing.T) {
		spreadAmount := decimal.NewFromFloat(-100000)
		entries := service.createSpreadEntry(referenceID, spreadAmount, transactionGroup)

		assert.Len(t, entries, 1)
		assert.Equal(t, AccountOTCExpense, entries[0].DebitAccount)
		assert.Equal(t, model.EntryTypeDebit, entries[0].EntryType)
		assert.True(t, entries[0].Amount.Equal(spreadAmount.Abs()))
	})

	t.Run("Zero spread", func(t *testing.T) {
		spreadAmount := decimal.Zero
		entries := service.createSpreadEntry(referenceID, spreadAmount, transactionGroup)

		assert.Len(t, entries, 0)
	})
}
