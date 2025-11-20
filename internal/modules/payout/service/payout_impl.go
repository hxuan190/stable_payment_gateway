package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payout/repository"
	"github.com/shopspring/decimal"
)

// Payout service errors
var (
	ErrPayoutNotFound            = errors.New("payout not found")
	ErrPayoutInvalidAmount       = errors.New("payout amount must be greater than zero")
	ErrPayoutInsufficientBalance = errors.New("insufficient balance for payout")
	ErrPayoutBelowMinimum        = errors.New("payout amount below minimum")
	ErrPayoutInvalidBankDetails  = errors.New("invalid bank account details")
	ErrPayoutInvalidStatus       = errors.New("payout cannot be processed in current status")
	ErrPayoutMerchantNotFound    = errors.New("merchant not found")
	ErrPayoutMerchantNotApproved = errors.New("merchant is not approved for payouts")
	ErrPayoutAlreadyProcessed    = errors.New("payout has already been processed")
	ErrPayoutCannotBeApproved    = errors.New("payout cannot be approved in current status")
	ErrPayoutCannotBeRejected    = errors.New("payout cannot be rejected in current status")
)

// Payout configuration constants
const (
	// MinimumPayoutAmountVND is the minimum payout amount (1,000,000 VND)
	MinimumPayoutAmountVND = 1000000

	// PayoutFeePercentage is the fee charged per payout (0.5%)
	PayoutFeePercentage = 0.005

	// MinimumPayoutFeeVND is the minimum fee charged (10,000 VND)
	MinimumPayoutFeeVND = 10000

	// MaximumPayoutAmountVND is the maximum single payout amount (500,000,000 VND)
	MaximumPayoutAmountVND = 500000000
)

// PayoutService handles business logic for merchant payouts (withdrawals)
type PayoutService struct {
	payoutRepo repository.PayoutRepository
	db         *sql.DB
}

// NewPayoutService creates a new payout service instance
func NewPayoutService(
	payoutRepo repository.PayoutRepository,
	db *sql.DB,
) *PayoutService {
	return &PayoutService{
		payoutRepo: payoutRepo,
		db:         db,
	}
}

// RequestPayoutInput contains the information needed to request a payout
type RequestPayoutInput struct {
	MerchantID        string
	AmountVND         decimal.Decimal
	BankAccountName   string
	BankAccountNumber string
	BankName          string
	BankBranch        string
	Notes             string
}

// Validate validates the payout request input
func (input *RequestPayoutInput) Validate() error {
	if input.MerchantID == "" {
		return ErrPayoutMerchantNotFound
	}
	if input.AmountVND.LessThanOrEqual(decimal.Zero) {
		return ErrPayoutInvalidAmount
	}
	if input.AmountVND.LessThan(decimal.NewFromInt(MinimumPayoutAmountVND)) {
		return ErrPayoutBelowMinimum
	}
	if input.AmountVND.GreaterThan(decimal.NewFromInt(MaximumPayoutAmountVND)) {
		return fmt.Errorf("payout amount exceeds maximum of %d VND", MaximumPayoutAmountVND)
	}
	if input.BankAccountName == "" || len(input.BankAccountName) < 2 {
		return ErrPayoutInvalidBankDetails
	}
	if input.BankAccountNumber == "" || len(input.BankAccountNumber) < 8 {
		return ErrPayoutInvalidBankDetails
	}
	if input.BankName == "" {
		return ErrPayoutInvalidBankDetails
	}
	return nil
}

// CalculatePayoutFee calculates the fee for a payout
func CalculatePayoutFee(amountVND decimal.Decimal) decimal.Decimal {
	// Calculate percentage-based fee
	fee := amountVND.Mul(decimal.NewFromFloat(PayoutFeePercentage))

	// Apply minimum fee
	minFee := decimal.NewFromInt(MinimumPayoutFeeVND)
	if fee.LessThan(minFee) {
		fee = minFee
	}

	return fee.Round(0) // Round to whole VND
}

// RequestPayout creates a new payout request for a merchant
// This will:
// 1. Validate the merchant and their balance
// 2. Calculate the payout fee
// 3. Reserve the balance (lock it from being used)
// 4. Create a payout record with "requested" status
func (s *PayoutService) RequestPayout(input RequestPayoutInput) (*model.Payout, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify merchant exists and is approved
	// Merchant validation moved to merchant module
	// Balance check moved to ledger module
	// Calculate fee
	feeVND := CalculatePayoutFee(input.AmountVND)

	// Create payout record
	payout := &model.Payout{
		ID:                uuid.New().String(),
		MerchantID:        input.MerchantID,
		AmountVND:         input.AmountVND,
		FeeVND:            feeVND,
		NetAmountVND:      input.AmountVND.Sub(feeVND), // Net amount merchant receives
		BankAccountName:   input.BankAccountName,
		BankAccountNumber: input.BankAccountNumber,
		BankName:          input.BankName,
		BankBranch:        sql.NullString{String: input.BankBranch, Valid: input.BankBranch != ""},
		Status:            model.PayoutStatusRequested,
		RequestedBy:       input.MerchantID, // Merchant requesting their own payout
		RetryCount:        0,
		Notes:             sql.NullString{String: input.Notes, Valid: input.Notes != ""},
		Metadata:          model.JSONBMap{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Save payout to database
	if err := s.payoutRepo.Create(payout); err != nil {
		return nil, fmt.Errorf("failed to create payout: %w", err)
	}

	// Balance reservation and ledger recording moved to ledger module

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit payout request: %w", err)
	}

	return payout, nil
}

// GetPayoutByID retrieves a payout by its ID
func (s *PayoutService) GetPayoutByID(payoutID string) (*model.Payout, error) {
	if payoutID == "" {
		return nil, ErrPayoutNotFound
	}

	payout, err := s.payoutRepo.GetByID(payoutID)
	if err != nil {
		if err == repository.ErrPayoutNotFound {
			return nil, ErrPayoutNotFound
		}
		return nil, fmt.Errorf("failed to get payout: %w", err)
	}

	return payout, nil
}

// ListPayoutsByMerchant retrieves all payouts for a merchant with pagination
func (s *PayoutService) ListPayoutsByMerchant(merchantID string, limit, offset int) ([]*model.Payout, error) {
	if merchantID == "" {
		return nil, ErrPayoutMerchantNotFound
	}

	payouts, err := s.payoutRepo.ListByMerchant(merchantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list payouts: %w", err)
	}

	return payouts, nil
}

// ListPendingPayouts retrieves all payouts awaiting approval
func (s *PayoutService) ListPendingPayouts() ([]*model.Payout, error) {
	payouts, err := s.payoutRepo.ListPending()
	if err != nil {
		return nil, fmt.Errorf("failed to list pending payouts: %w", err)
	}

	return payouts, nil
}

// ApprovePayout approves a payout request (admin only)
// This transitions the payout from "requested" to "approved" status
// The balance remains reserved until the payout is completed
func (s *PayoutService) ApprovePayout(payoutID, approvedBy string) error {
	if payoutID == "" {
		return ErrPayoutNotFound
	}
	if approvedBy == "" {
		return errors.New("approver ID cannot be empty")
	}

	// Get payout
	payout, err := s.payoutRepo.GetByID(payoutID)
	if err != nil {
		if err == repository.ErrPayoutNotFound {
			return ErrPayoutNotFound
		}
		return fmt.Errorf("failed to get payout: %w", err)
	}

	// Check if payout can be approved
	if !payout.CanBeApproved() {
		return fmt.Errorf("%w: current status is %s", ErrPayoutCannotBeApproved, payout.Status)
	}

	// Approve payout in repository
	if err := s.payoutRepo.Approve(payoutID, approvedBy); err != nil {
		return fmt.Errorf("failed to approve payout: %w", err)
	}

	return nil
}

// RejectPayout rejects a payout request (admin only)
// This releases the reserved balance back to the merchant
func (s *PayoutService) RejectPayout(payoutID, reason string) error {
	if payoutID == "" {
		return ErrPayoutNotFound
	}
	if reason == "" {
		return errors.New("rejection reason cannot be empty")
	}

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get payout
	payout, err := s.payoutRepo.GetByID(payoutID)
	if err != nil {
		if err == repository.ErrPayoutNotFound {
			return ErrPayoutNotFound
		}
		return fmt.Errorf("failed to get payout: %w", err)
	}

	// Check if payout can be rejected
	if !payout.CanBeRejected() {
		return fmt.Errorf("%w: current status is %s", ErrPayoutCannotBeRejected, payout.Status)
	}

	// Balance release moved to ledger module

	// Reject payout in repository
	if err := s.payoutRepo.Reject(payoutID, reason); err != nil {
		return fmt.Errorf("failed to reject payout: %w", err)
	}

	// Ledger recording moved to ledger module

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit payout rejection: %w", err)
	}

	return nil
}

// CompletePayout marks a payout as completed after the bank transfer is done (ops team)
// This:
// 1. Updates payout status to "completed"
// 2. Deducts the amount from merchant's balance
// 3. Records the transaction in the ledger
func (s *PayoutService) CompletePayout(payoutID, bankReferenceNumber, processedBy string) error {
	if payoutID == "" {
		return ErrPayoutNotFound
	}
	if bankReferenceNumber == "" {
		return errors.New("bank reference number cannot be empty")
	}
	if processedBy == "" {
		return errors.New("processor ID cannot be empty")
	}

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get payout
	payout, err := s.payoutRepo.GetByID(payoutID)
	if err != nil {
		if err == repository.ErrPayoutNotFound {
			return ErrPayoutNotFound
		}
		return fmt.Errorf("failed to get payout: %w", err)
	}

	// Check payout status - must be approved or processing
	if payout.Status != model.PayoutStatusApproved && payout.Status != model.PayoutStatusProcessing {
		return fmt.Errorf("%w: current status is %s, expected approved or processing",
			ErrPayoutInvalidStatus, payout.Status)
	}

	// Update payout record
	payout.Status = model.PayoutStatusCompleted
	payout.BankReferenceNumber = sql.NullString{String: bankReferenceNumber, Valid: true}
	payout.ProcessedBy = sql.NullString{String: processedBy, Valid: true}
	now := time.Now()
	payout.ProcessedAt = sql.NullTime{Time: now, Valid: true}
	payout.CompletionDate = sql.NullTime{Time: now, Valid: true}
	payout.UpdatedAt = now

	if err := s.payoutRepo.Update(payout); err != nil {
		return fmt.Errorf("failed to update payout: %w", err)
	}

	// Balance deduction and ledger recording moved to ledger module

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit payout completion: %w", err)
	}

	return nil
}

// FailPayout marks a payout as failed (e.g., bank transfer failed)
// This releases the reserved balance back to the merchant
func (s *PayoutService) FailPayout(payoutID, failureReason, processedBy string) error {
	if payoutID == "" {
		return ErrPayoutNotFound
	}
	if failureReason == "" {
		return errors.New("failure reason cannot be empty")
	}

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get payout
	payout, err := s.payoutRepo.GetByID(payoutID)
	if err != nil {
		if err == repository.ErrPayoutNotFound {
			return ErrPayoutNotFound
		}
		return fmt.Errorf("failed to get payout: %w", err)
	}

	// Check payout status
	if payout.Status == model.PayoutStatusCompleted {
		return ErrPayoutAlreadyProcessed
	}
	if payout.Status == model.PayoutStatusFailed || payout.Status == model.PayoutStatusRejected {
		return fmt.Errorf("payout already in terminal status: %s", payout.Status)
	}

	// Update payout record
	payout.Status = model.PayoutStatusFailed
	payout.FailureReason = sql.NullString{String: failureReason, Valid: true}
	payout.ProcessedBy = sql.NullString{String: processedBy, Valid: processedBy != ""}
	payout.RetryCount++
	payout.UpdatedAt = time.Now()

	if err := s.payoutRepo.Update(payout); err != nil {
		return fmt.Errorf("failed to update payout: %w", err)
	}

	// Balance release and ledger recording moved to ledger module

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit payout failure: %w", err)
	}

	return nil
}

// GetPayoutStats retrieves statistics about payouts
func (s *PayoutService) GetPayoutStats() (*PayoutStats, error) {
	// Get counts by status
	requestedCount, err := s.payoutRepo.CountByStatus(model.PayoutStatusRequested)
	if err != nil {
		return nil, fmt.Errorf("failed to count requested payouts: %w", err)
	}

	approvedCount, err := s.payoutRepo.CountByStatus(model.PayoutStatusApproved)
	if err != nil {
		return nil, fmt.Errorf("failed to count approved payouts: %w", err)
	}

	processingCount, err := s.payoutRepo.CountByStatus(model.PayoutStatusProcessing)
	if err != nil {
		return nil, fmt.Errorf("failed to count processing payouts: %w", err)
	}

	completedCount, err := s.payoutRepo.CountByStatus(model.PayoutStatusCompleted)
	if err != nil {
		return nil, fmt.Errorf("failed to count completed payouts: %w", err)
	}

	rejectedCount, err := s.payoutRepo.CountByStatus(model.PayoutStatusRejected)
	if err != nil {
		return nil, fmt.Errorf("failed to count rejected payouts: %w", err)
	}

	failedCount, err := s.payoutRepo.CountByStatus(model.PayoutStatusFailed)
	if err != nil {
		return nil, fmt.Errorf("failed to count failed payouts: %w", err)
	}

	totalCount, err := s.payoutRepo.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to count total payouts: %w", err)
	}

	return &PayoutStats{
		TotalPayouts:      totalCount,
		RequestedPayouts:  requestedCount,
		ApprovedPayouts:   approvedCount,
		ProcessingPayouts: processingCount,
		CompletedPayouts:  completedCount,
		RejectedPayouts:   rejectedCount,
		FailedPayouts:     failedCount,
	}, nil
}

// PayoutStats represents statistics about payouts
type PayoutStats struct {
	TotalPayouts      int64 `json:"total_payouts"`
	RequestedPayouts  int64 `json:"requested_payouts"`
	ApprovedPayouts   int64 `json:"approved_payouts"`
	ProcessingPayouts int64 `json:"processing_payouts"`
	CompletedPayouts  int64 `json:"completed_payouts"`
	RejectedPayouts   int64 `json:"rejected_payouts"`
	FailedPayouts     int64 `json:"failed_payouts"`
}
