package service

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	ledgerDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	// ErrLedgerInvalidAmount is returned when amount is invalid (negative or zero)
	ErrLedgerInvalidAmount = errors.New("amount must be greater than zero")
	// ErrLedgerInvalidMerchantID is returned when merchant ID is empty
	ErrLedgerInvalidMerchantID = errors.New("merchant ID cannot be empty")
	// ErrLedgerInvalidReferenceID is returned when reference ID is empty
	ErrLedgerInvalidReferenceID = errors.New("reference ID cannot be empty")
	// ErrLedgerInvalidCurrency is returned when currency is empty or invalid
	ErrLedgerInvalidCurrency = errors.New("currency cannot be empty")
	// ErrLedgerTransactionFailed is returned when transaction cannot be completed
	ErrLedgerTransactionFailed = errors.New("transaction failed")
)

// Account names used in the double-entry ledger system
const (
	// Asset accounts (debit increases, credit decreases)
	AccountCryptoPool   = "crypto_pool"   // Hot wallet crypto holdings
	AccountVNDPool      = "vnd_pool"      // VND holdings from OTC conversion
	AccountReceivables  = "receivables"   // Outstanding amounts owed to us
	AccountPlatformBank = "platform_bank" // Platform's bank account

	// Liability accounts (credit increases, debit decreases)
	AccountMerchantPendingPrefix   = "merchant_pending:"   // Prefix for merchant pending balances
	AccountMerchantAvailablePrefix = "merchant_available:" // Prefix for merchant available balances
	AccountMerchantReservedPrefix  = "merchant_reserved:"  // Prefix for merchant reserved balances
	AccountPayoutLiability         = "payout_liability"    // Pending payouts owed to merchants

	// Revenue accounts (credit increases, debit decreases)
	AccountFeeRevenue   = "fee_revenue"   // Transaction and payout fees
	AccountOTCSpread    = "otc_spread"    // Revenue from OTC exchange rate spread
	AccountOtherRevenue = "other_revenue" // Other revenue sources

	// Expense accounts (debit increases, credit decreases)
	AccountOTCExpense    = "otc_expense"    // Costs paid to OTC partner
	AccountPayoutExpense = "payout_expense" // Bank transfer fees for payouts
	AccountOperatingExp  = "operating_exp"  // Other operating expenses
)

// LedgerService provides business logic for double-entry accounting
type LedgerService struct {
	ledgerRepo  *repository.LedgerRepository
	balanceRepo *repository.BalanceRepository
	db          *gorm.DB
}

// NewLedgerService creates a new ledger service
func NewLedgerService(
	ledgerRepo *repository.LedgerRepository,
	balanceRepo *repository.BalanceRepository,
	db *gorm.DB,
) *LedgerService {
	return &LedgerService{
		ledgerRepo:  ledgerRepo,
		balanceRepo: balanceRepo,
		db:          db,
	}
}

// RecordPaymentReceived records when a crypto payment is received from a user
// This creates a ledger entry moving crypto to our pool and crediting merchant's pending balance
//
// Accounting entry:
//
//	DEBIT:  crypto_pool (+X USDT)
//	CREDIT: merchant_pending_balance (+Y VND equivalent)
func (s *LedgerService) RecordPaymentReceived(
	paymentID, merchantID string,
	amountCrypto decimal.Decimal,
	cryptoCurrency string,
	amountVND decimal.Decimal,
) error {
	// Validate inputs
	if err := s.validateBasicInputs(paymentID, merchantID, amountVND, "VND"); err != nil {
		return err
	}
	if amountCrypto.LessThanOrEqual(decimal.Zero) {
		return ErrLedgerInvalidAmount
	}
	if cryptoCurrency == "" {
		return ErrLedgerInvalidCurrency
	}

	// Generate transaction group ID
	transactionGroup := uuid.New().String()

	// Create ledger entries
	entries := []*ledgerDomain.LedgerEntry{
		// Debit: Increase crypto pool
		{
			DebitAccount:     AccountCryptoPool,
			CreditAccount:    AccountCryptoPool, // Placeholder, will be replaced
			Amount:           amountCrypto,
			Currency:         cryptoCurrency,
			ReferenceType:    ledgerDomain.ReferenceTypePayment,
			ReferenceID:      paymentID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payment %s received: %s %s", paymentID, amountCrypto.String(), cryptoCurrency),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"crypto_amount": amountCrypto.String(), "crypto_currency": cryptoCurrency},
		},
		// Credit: This is the corresponding credit (crypto pool from user)
		{
			DebitAccount:     AccountCryptoPool,
			CreditAccount:    fmt.Sprintf("user_payment:%s", paymentID),
			Amount:           amountCrypto,
			Currency:         cryptoCurrency,
			ReferenceType:    ledgerDomain.ReferenceTypePayment,
			ReferenceID:      paymentID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("User payment %s: %s %s", paymentID, amountCrypto.String(), cryptoCurrency),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"crypto_amount": amountCrypto.String(), "crypto_currency": cryptoCurrency},
		},
	}

	// Start database transaction
	tx := s.db.Begin()
	defer tx.Rollback()

	// Create ledger entries
	if err := s.ledgerRepo.CreateEntries(entries); err != nil {
		return fmt.Errorf("failed to create ledger entries: %w", err)
	}

	// Update merchant balance - add to pending
	if err := s.balanceRepo.IncrementPendingTx(tx, merchantID, amountVND); err != nil {
		return fmt.Errorf("failed to update merchant balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RecordPaymentConfirmed records when a payment is confirmed and ready for merchant
// This moves balance from pending to available (after deducting platform fee)
//
// Accounting entry:
//
//	DEBIT:  merchant_pending_balance (-Y VND)
//	CREDIT: merchant_available_balance (+Y * 0.99 VND)
//	CREDIT: fee_revenue (+Y * 0.01 VND)
func (s *LedgerService) RecordPaymentConfirmed(
	paymentID, merchantID string,
	amountVND, feeVND decimal.Decimal,
) error {
	// Validate inputs
	if err := s.validateBasicInputs(paymentID, merchantID, amountVND, "VND"); err != nil {
		return err
	}
	if feeVND.LessThan(decimal.Zero) {
		return errors.New("fee cannot be negative")
	}
	if feeVND.GreaterThanOrEqual(amountVND) {
		return errors.New("fee cannot be greater than or equal to amount")
	}

	netAmount := amountVND.Sub(feeVND)
	transactionGroup := uuid.New().String()

	// Create ledger entries for VND movement
	entries := []*ledgerDomain.LedgerEntry{
		// Debit: Decrease merchant pending balance
		{
			DebitAccount:     s.getMerchantPendingAccount(merchantID),
			CreditAccount:    s.getMerchantPendingAccount(merchantID), // Placeholder
			Amount:           amountVND,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayment,
			ReferenceID:      paymentID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payment %s confirmed: converting pending to available", paymentID),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"gross_amount": amountVND.String(), "fee": feeVND.String()},
		},
		// Credit: Increase merchant available balance (net of fee)
		{
			DebitAccount:     s.getMerchantPendingAccount(merchantID),
			CreditAccount:    s.getMerchantAvailableAccount(merchantID),
			Amount:           netAmount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayment,
			ReferenceID:      paymentID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payment %s: available balance after fee", paymentID),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"net_amount": netAmount.String(), "fee_rate": "0.01"},
		},
		// Credit: Record fee revenue
		{
			DebitAccount:     s.getMerchantPendingAccount(merchantID),
			CreditAccount:    AccountFeeRevenue,
			Amount:           feeVND,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypeFee,
			ReferenceID:      paymentID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Transaction fee for payment %s", paymentID),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"fee_type": "transaction_fee", "fee_rate": "0.01"},
		},
	}

	// Start database transaction
	tx := s.db.Begin()
	defer tx.Rollback()

	// Create ledger entries
	if err := s.ledgerRepo.CreateEntries(entries); err != nil {
		return fmt.Errorf("failed to create ledger entries: %w", err)
	}

	// Update merchant balance - move from pending to available
	if err := s.balanceRepo.ConvertPendingToAvailableTx(tx, merchantID, amountVND, feeVND); err != nil {
		return fmt.Errorf("failed to update merchant balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RecordPayoutRequested records when a merchant requests a payout
// This reserves the requested amount from their available balance
//
// Accounting entry:
//
//	DEBIT:  merchant_available_balance (-Z VND)
//	CREDIT: merchant_reserved_balance (+Z VND)
func (s *LedgerService) RecordPayoutRequested(
	payoutID, merchantID string,
	amount decimal.Decimal,
) error {
	// Validate inputs
	if err := s.validateBasicInputs(payoutID, merchantID, amount, "VND"); err != nil {
		return err
	}

	// Start database transaction
	tx := s.db.Begin()
	defer tx.Rollback()

	// Reserve balance (this will check if sufficient balance exists)
	if err := s.balanceRepo.ReserveBalanceTx(tx, merchantID, amount); err != nil {
		return fmt.Errorf("failed to reserve balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RecordPayoutCompleted records when a payout is completed (bank transfer done)
// This deducts from available balance and records the payout
//
// Accounting entry:
//
//	DEBIT:  merchant_available_balance (-(amount + fee) VND)
//	CREDIT: vnd_pool (+amount VND)
//	CREDIT: fee_revenue (+fee VND)
func (s *LedgerService) RecordPayoutCompleted(
	payoutID, merchantID string,
	amount, feeVND decimal.Decimal,
) error {
	// Validate inputs
	if err := s.validateBasicInputs(payoutID, merchantID, amount, "VND"); err != nil {
		return err
	}
	if feeVND.LessThan(decimal.Zero) {
		return errors.New("fee cannot be negative")
	}

	totalDeduction := amount.Add(feeVND)
	transactionGroup := uuid.New().String()

	// Create ledger entries
	entries := []*ledgerDomain.LedgerEntry{
		// Debit: Decrease merchant available balance
		{
			DebitAccount:     s.getMerchantAvailableAccount(merchantID),
			CreditAccount:    s.getMerchantAvailableAccount(merchantID), // Placeholder
			Amount:           totalDeduction,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayout,
			ReferenceID:      payoutID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payout %s: deducting %s VND (amount: %s, fee: %s)", payoutID, totalDeduction.String(), amount.String(), feeVND.String()),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"payout_amount": amount.String(), "payout_fee": feeVND.String()},
		},
		// Credit: Decrease VND pool (money going out to merchant's bank)
		{
			DebitAccount:     s.getMerchantAvailableAccount(merchantID),
			CreditAccount:    AccountVNDPool,
			Amount:           amount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayout,
			ReferenceID:      payoutID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payout %s: transferred to merchant bank", payoutID),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"payout_amount": amount.String()},
		},
		// Credit: Record payout fee revenue
		{
			DebitAccount:     s.getMerchantAvailableAccount(merchantID),
			CreditAccount:    AccountFeeRevenue,
			Amount:           feeVND,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypeFee,
			ReferenceID:      payoutID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payout fee for payout %s", payoutID),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"fee_type": "payout_fee"},
		},
	}

	// Start database transaction
	tx := s.db.Begin()
	defer tx.Rollback()

	// Create ledger entries
	if err := s.ledgerRepo.CreateEntries(entries); err != nil {
		return fmt.Errorf("failed to create ledger entries: %w", err)
	}

	// Update merchant balance - deduct from available and reserved
	if err := s.balanceRepo.DeductBalanceTx(tx, merchantID, amount, feeVND); err != nil {
		return fmt.Errorf("failed to update merchant balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RecordPayoutCancelled records when a payout request is cancelled
// This releases the reserved balance back to available
func (s *LedgerService) RecordPayoutCancelled(
	payoutID, merchantID string,
	amount decimal.Decimal,
) error {
	// Validate inputs
	if err := s.validateBasicInputs(payoutID, merchantID, amount, "VND"); err != nil {
		return err
	}

	// Start database transaction
	tx := s.db.Begin()
	defer tx.Rollback()

	// Release reserved balance
	if err := s.balanceRepo.ReleaseReservedBalanceTx(tx, merchantID, amount); err != nil {
		return fmt.Errorf("failed to release reserved balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RecordPayoutRejected records when a payout request is rejected by admin
// This releases the reserved balance back to available
func (s *LedgerService) RecordPayoutRejected(
	payoutID, merchantID string,
	amount decimal.Decimal,
) error {
	// Validate inputs
	if err := s.validateBasicInputs(payoutID, merchantID, amount, "VND"); err != nil {
		return err
	}

	transactionGroup := uuid.New().String()

	// Create ledger entries documenting the rejection
	entries := []*ledgerDomain.LedgerEntry{
		// Debit: Merchant reserved balance (unreserve)
		{
			DebitAccount:     s.getMerchantReservedAccount(merchantID),
			CreditAccount:    s.getMerchantAvailableAccount(merchantID),
			Amount:           amount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayout,
			ReferenceID:      payoutID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payout %s rejected: releasing %s VND from reserve", payoutID, amount.String()),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"payout_status": "rejected"},
		},
		// Credit: Merchant available balance (funds returned)
		{
			DebitAccount:     s.getMerchantReservedAccount(merchantID),
			CreditAccount:    s.getMerchantAvailableAccount(merchantID),
			Amount:           amount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayout,
			ReferenceID:      payoutID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payout %s rejected: funds returned to available balance", payoutID),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"payout_status": "rejected"},
		},
	}

	// Create ledger entries
	if err := s.ledgerRepo.CreateEntries(entries); err != nil {
		return fmt.Errorf("failed to create ledger entries: %w", err)
	}

	return nil
}

// RecordPayoutFailed records when a payout fails (e.g., bank transfer failed)
// This releases the reserved balance back to available
func (s *LedgerService) RecordPayoutFailed(
	payoutID, merchantID string,
	amount decimal.Decimal,
	reason string,
) error {
	// Validate inputs
	if err := s.validateBasicInputs(payoutID, merchantID, amount, "VND"); err != nil {
		return err
	}

	transactionGroup := uuid.New().String()

	// Create ledger entries documenting the failure
	entries := []*ledgerDomain.LedgerEntry{
		// Debit: Merchant reserved balance (unreserve)
		{
			DebitAccount:     s.getMerchantReservedAccount(merchantID),
			CreditAccount:    s.getMerchantAvailableAccount(merchantID),
			Amount:           amount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayout,
			ReferenceID:      payoutID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payout %s failed: releasing %s VND from reserve - %s", payoutID, amount.String(), reason),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"payout_status": "failed", "failure_reason": reason},
		},
		// Credit: Merchant available balance (funds returned)
		{
			DebitAccount:     s.getMerchantReservedAccount(merchantID),
			CreditAccount:    s.getMerchantAvailableAccount(merchantID),
			Amount:           amount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypePayout,
			ReferenceID:      payoutID,
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      fmt.Sprintf("Payout %s failed: funds returned to available balance", payoutID),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"payout_status": "failed", "failure_reason": reason},
		},
	}

	// Create ledger entries
	if err := s.ledgerRepo.CreateEntries(entries); err != nil {
		return fmt.Errorf("failed to create ledger entries: %w", err)
	}

	return nil
}

// RecordOTCConversion records when we convert crypto to VND via OTC partner
// This decreases crypto pool and increases VND pool
//
// Accounting entry:
//
//	DEBIT:  vnd_pool (+Y VND received)
//	CREDIT: crypto_pool (-X USDT sent)
//
// If there's a spread (difference between market rate and OTC rate):
//
//	DEBIT/CREDIT: otc_spread (difference)
func (s *LedgerService) RecordOTCConversion(
	referenceID string,
	cryptoAmount decimal.Decimal,
	cryptoCurrency string,
	vndAmount decimal.Decimal,
	spreadAmount decimal.Decimal,
) error {
	// Validate inputs
	if referenceID == "" {
		return ErrLedgerInvalidReferenceID
	}
	if cryptoAmount.LessThanOrEqual(decimal.Zero) {
		return ErrLedgerInvalidAmount
	}
	if vndAmount.LessThanOrEqual(decimal.Zero) {
		return ErrLedgerInvalidAmount
	}
	if cryptoCurrency == "" {
		return ErrLedgerInvalidCurrency
	}

	transactionGroupCrypto := uuid.New().String()
	transactionGroupVND := uuid.New().String()

	// Create ledger entries for crypto decrease
	cryptoEntries := []*ledgerDomain.LedgerEntry{
		// Debit: OTC partner receives crypto
		{
			DebitAccount:     "otc_partner",
			CreditAccount:    AccountCryptoPool,
			Amount:           cryptoAmount,
			Currency:         cryptoCurrency,
			ReferenceType:    ledgerDomain.ReferenceTypeOTCConversion,
			ReferenceID:      referenceID,
			Description:      fmt.Sprintf("OTC conversion %s: sent %s %s to OTC partner", referenceID, cryptoAmount.String(), cryptoCurrency),
			TransactionGroup: transactionGroupCrypto,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"crypto_amount": cryptoAmount.String(), "crypto_currency": cryptoCurrency},
		},
		// Credit: Decrease our crypto pool
		{
			DebitAccount:     "otc_partner",
			CreditAccount:    AccountCryptoPool,
			Amount:           cryptoAmount,
			Currency:         cryptoCurrency,
			ReferenceType:    ledgerDomain.ReferenceTypeOTCConversion,
			ReferenceID:      referenceID,
			Description:      fmt.Sprintf("OTC conversion %s: crypto pool decrease", referenceID),
			TransactionGroup: transactionGroupCrypto,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"crypto_amount": cryptoAmount.String(), "crypto_currency": cryptoCurrency},
		},
	}

	// Create ledger entries for VND increase
	vndEntries := []*ledgerDomain.LedgerEntry{
		// Debit: Increase our VND pool
		{
			DebitAccount:     AccountVNDPool,
			CreditAccount:    "otc_partner",
			Amount:           vndAmount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypeOTCConversion,
			ReferenceID:      referenceID,
			Description:      fmt.Sprintf("OTC conversion %s: received %s VND from OTC partner", referenceID, vndAmount.String()),
			TransactionGroup: transactionGroupVND,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"vnd_amount": vndAmount.String(), "spread": spreadAmount.String()},
		},
		// Credit: OTC partner provides VND
		{
			DebitAccount:     AccountVNDPool,
			CreditAccount:    "otc_partner",
			Amount:           vndAmount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypeOTCConversion,
			ReferenceID:      referenceID,
			Description:      fmt.Sprintf("OTC conversion %s: VND from OTC partner", referenceID),
			TransactionGroup: transactionGroupVND,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"vnd_amount": vndAmount.String(), "spread": spreadAmount.String()},
		},
	}

	// If there's a spread (profit or loss from exchange rate difference)
	if !spreadAmount.IsZero() {
		spreadEntry := s.createSpreadEntry(referenceID, spreadAmount, transactionGroupVND)
		vndEntries = append(vndEntries, spreadEntry...)
	}

	// Create ledger entries (this validates double-entry)
	// Note: We allow cross-currency transactions for OTC
	if err := s.ledgerRepo.CreateEntries(cryptoEntries); err != nil {
		return fmt.Errorf("failed to create crypto ledger entries: %w", err)
	}

	if err := s.ledgerRepo.CreateEntries(vndEntries); err != nil {
		return fmt.Errorf("failed to create VND ledger entries: %w", err)
	}

	return nil
}

// GetMerchantLedgerEntries retrieves ledger entries for a merchant
func (s *LedgerService) GetMerchantLedgerEntries(merchantID string, limit, offset int) ([]*ledgerDomain.LedgerEntry, error) {
	if merchantID == "" {
		return nil, ErrLedgerInvalidMerchantID
	}

	return s.ledgerRepo.GetByMerchant(merchantID, limit, offset)
}

// GetTransactionDetails retrieves all ledger entries for a specific transaction
func (s *LedgerService) GetTransactionDetails(transactionGroup string) ([]*ledgerDomain.LedgerEntry, error) {
	if transactionGroup == "" {
		return nil, errors.New("transaction group cannot be empty")
	}

	return s.ledgerRepo.GetByTransactionGroup(transactionGroup)
}

// GetAccountBalance retrieves the current balance for an account
func (s *LedgerService) GetAccountBalance(accountName string) (decimal.Decimal, error) {
	if accountName == "" {
		return decimal.Zero, errors.New("account name cannot be empty")
	}

	return s.ledgerRepo.GetAccountBalance(accountName)
}

// ValidateLedgerIntegrity validates that all transactions in the ledger balance
func (s *LedgerService) ValidateLedgerIntegrity() error {
	return s.ledgerRepo.ValidateDoubleEntry()
}

// Helper functions

func (s *LedgerService) getMerchantPendingAccount(merchantID string) string {
	return fmt.Sprintf("%s%s", AccountMerchantPendingPrefix, merchantID)
}

func (s *LedgerService) getMerchantAvailableAccount(merchantID string) string {
	return fmt.Sprintf("%s%s", AccountMerchantAvailablePrefix, merchantID)
}

func (s *LedgerService) getMerchantReservedAccount(merchantID string) string {
	return fmt.Sprintf("%s%s", AccountMerchantReservedPrefix, merchantID)
}

func (s *LedgerService) validateBasicInputs(referenceID, merchantID string, amount decimal.Decimal, currency string) error {
	if referenceID == "" {
		return ErrLedgerInvalidReferenceID
	}
	if merchantID == "" {
		return ErrLedgerInvalidMerchantID
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrLedgerInvalidAmount
	}
	if currency == "" {
		return ErrLedgerInvalidCurrency
	}
	return nil
}

func (s *LedgerService) createSpreadEntry(referenceID string, spreadAmount decimal.Decimal, transactionGroup string) []*ledgerDomain.LedgerEntry {
	var entries []*ledgerDomain.LedgerEntry

	if spreadAmount.GreaterThan(decimal.Zero) {
		// Positive spread = revenue (we got more VND than expected)
		entries = append(entries, &ledgerDomain.LedgerEntry{
			DebitAccount:     AccountVNDPool,
			CreditAccount:    AccountOTCSpread,
			Amount:           spreadAmount,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypeOTCConversion,
			ReferenceID:      referenceID,
			Description:      fmt.Sprintf("OTC spread revenue: %s VND", spreadAmount.String()),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeCredit,
			Metadata:         database.JSONBMap{"spread_type": "revenue"},
		})
	} else if spreadAmount.LessThan(decimal.Zero) {
		// Negative spread = loss (we got less VND than expected)
		absSpread := spreadAmount.Abs()
		entries = append(entries, &ledgerDomain.LedgerEntry{
			DebitAccount:     AccountOTCExpense,
			CreditAccount:    AccountVNDPool,
			Amount:           absSpread,
			Currency:         "VND",
			ReferenceType:    ledgerDomain.ReferenceTypeOTCConversion,
			ReferenceID:      referenceID,
			Description:      fmt.Sprintf("OTC spread loss: %s VND", absSpread.String()),
			TransactionGroup: transactionGroup,
			EntryType:        ledgerDomain.EntryTypeDebit,
			Metadata:         database.JSONBMap{"spread_type": "loss"},
		})
	}

	return entries
}
