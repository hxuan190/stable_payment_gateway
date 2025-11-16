package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/shopspring/decimal"
)

// Common errors for merchant service
var (
	ErrMerchantNotFound       = errors.New("merchant not found")
	ErrMerchantAlreadyExists  = errors.New("merchant already exists")
	ErrInvalidKYCStatus       = errors.New("invalid KYC status for this operation")
	ErrMerchantNotApproved    = errors.New("merchant is not approved")
	ErrInvalidEmail           = errors.New("invalid email address")
	ErrInvalidBusinessName    = errors.New("invalid business name")
	ErrAPIKeyGenerationFailed = errors.New("failed to generate API key")
)

// MerchantService handles business logic for merchant management
type MerchantService struct {
	merchantRepo *repository.MerchantRepository
	balanceRepo  *repository.BalanceRepository
	db           *sql.DB
}

// NewMerchantService creates a new merchant service instance
func NewMerchantService(
	merchantRepo *repository.MerchantRepository,
	balanceRepo *repository.BalanceRepository,
	db *sql.DB,
) *MerchantService {
	return &MerchantService{
		merchantRepo: merchantRepo,
		balanceRepo:  balanceRepo,
		db:           db,
	}
}

// RegisterMerchantInput contains the information needed to register a new merchant
type RegisterMerchantInput struct {
	Email                 string
	BusinessName          string
	BusinessTaxID         string
	BusinessLicenseNumber string
	OwnerFullName         string
	OwnerIDCard           string
	PhoneNumber           string
	BusinessAddress       string
	City                  string
	District              string
	BankAccountName       string
	BankAccountNumber     string
	BankName              string
	BankBranch            string
}

// Validate validates the merchant registration input
func (input *RegisterMerchantInput) Validate() error {
	if input.Email == "" {
		return ErrInvalidEmail
	}
	if input.BusinessName == "" || len(input.BusinessName) < 2 {
		return ErrInvalidBusinessName
	}
	if input.OwnerFullName == "" {
		return errors.New("owner full name is required")
	}
	return nil
}

// RegisterMerchant registers a new merchant with pending KYC status
func (s *MerchantService) RegisterMerchant(input RegisterMerchantInput) (*model.Merchant, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if merchant with this email already exists
	existingMerchant, err := s.merchantRepo.GetByEmail(input.Email)
	if err != nil && err != repository.ErrMerchantNotFound {
		return nil, fmt.Errorf("failed to check existing merchant: %w", err)
	}
	if existingMerchant != nil {
		return nil, ErrMerchantAlreadyExists
	}

	// Create merchant
	merchant := &model.Merchant{
		ID:           uuid.New().String(),
		Email:        input.Email,
		BusinessName: input.BusinessName,
		OwnerFullName: input.OwnerFullName,
		KYCStatus:    model.KYCStatusPending,
		Status:       model.MerchantStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Set optional fields
	if input.BusinessTaxID != "" {
		merchant.BusinessTaxID = sql.NullString{String: input.BusinessTaxID, Valid: true}
	}
	if input.BusinessLicenseNumber != "" {
		merchant.BusinessLicenseNumber = sql.NullString{String: input.BusinessLicenseNumber, Valid: true}
	}
	if input.OwnerIDCard != "" {
		merchant.OwnerIDCard = sql.NullString{String: input.OwnerIDCard, Valid: true}
	}
	if input.PhoneNumber != "" {
		merchant.PhoneNumber = sql.NullString{String: input.PhoneNumber, Valid: true}
	}
	if input.BusinessAddress != "" {
		merchant.BusinessAddress = sql.NullString{String: input.BusinessAddress, Valid: true}
	}
	if input.City != "" {
		merchant.City = sql.NullString{String: input.City, Valid: true}
	}
	if input.District != "" {
		merchant.District = sql.NullString{String: input.District, Valid: true}
	}
	if input.BankAccountName != "" {
		merchant.BankAccountName = sql.NullString{String: input.BankAccountName, Valid: true}
	}
	if input.BankAccountNumber != "" {
		merchant.BankAccountNumber = sql.NullString{String: input.BankAccountNumber, Valid: true}
	}
	if input.BankName != "" {
		merchant.BankName = sql.NullString{String: input.BankName, Valid: true}
	}
	if input.BankBranch != "" {
		merchant.BankBranch = sql.NullString{String: input.BankBranch, Valid: true}
	}

	// Set KYC submission timestamp
	merchant.KYCSubmittedAt = sql.NullTime{Time: time.Now(), Valid: true}

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save merchant to database
	if err := s.merchantRepo.Create(merchant); err != nil {
		return nil, fmt.Errorf("failed to create merchant: %w", err)
	}

	// Create initial balance for merchant
	balance := &model.MerchantBalance{
		ID:                 uuid.New().String(),
		MerchantID:         merchant.ID,
		PendingVND:         decimal.Zero,
		AvailableVND:       decimal.Zero,
		TotalVND:           decimal.Zero,
		ReservedVND:        decimal.Zero,
		TotalReceivedVND:   decimal.Zero,
		TotalPaidOutVND:    decimal.Zero,
		TotalFeesVND:       decimal.Zero,
		TotalPaymentsCount: 0,
		TotalPayoutsCount:  0,
		Version:            1,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.balanceRepo.Create(balance); err != nil {
		return nil, fmt.Errorf("failed to create merchant balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return merchant, nil
}

// GetMerchantBalance retrieves a merchant's current balance
func (s *MerchantService) GetMerchantBalance(merchantID string) (*model.MerchantBalance, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}

	// Verify merchant exists
	_, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	// Get balance
	balance, err := s.balanceRepo.GetByMerchantID(merchantID)
	if err != nil {
		if err == repository.ErrBalanceNotFound {
			return nil, errors.New("merchant balance not found")
		}
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// ApproveKYC approves a merchant's KYC and generates an API key
func (s *MerchantService) ApproveKYC(merchantID string, approvedBy string) (*model.Merchant, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}
	if approvedBy == "" {
		return nil, errors.New("approved by cannot be empty")
	}

	// Get merchant
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	// Check if merchant is in pending status
	if merchant.KYCStatus != model.KYCStatusPending {
		return nil, fmt.Errorf("%w: merchant KYC status is %s, expected pending", ErrInvalidKYCStatus, merchant.KYCStatus)
	}

	// Generate API key
	apiKey, err := s.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Update merchant
	now := time.Now()
	merchant.KYCStatus = model.KYCStatusApproved
	merchant.KYCApprovedAt = sql.NullTime{Time: now, Valid: true}
	merchant.KYCApprovedBy = sql.NullString{String: approvedBy, Valid: true}
	merchant.APIKey = sql.NullString{String: apiKey, Valid: true}
	merchant.APIKeyCreatedAt = sql.NullTime{Time: now, Valid: true}
	merchant.UpdatedAt = now

	// Save to database
	if err := s.merchantRepo.Update(merchant); err != nil {
		return nil, fmt.Errorf("failed to update merchant: %w", err)
	}

	return merchant, nil
}

// RejectKYC rejects a merchant's KYC with a reason
func (s *MerchantService) RejectKYC(merchantID string, reason string) (*model.Merchant, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}
	if reason == "" {
		return nil, errors.New("rejection reason cannot be empty")
	}

	// Get merchant
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	// Check if merchant is in pending status
	if merchant.KYCStatus != model.KYCStatusPending {
		return nil, fmt.Errorf("%w: merchant KYC status is %s, expected pending", ErrInvalidKYCStatus, merchant.KYCStatus)
	}

	// Update merchant
	merchant.KYCStatus = model.KYCStatusRejected
	merchant.KYCRejectionReason = sql.NullString{String: reason, Valid: true}
	merchant.UpdatedAt = time.Now()

	// Save to database
	if err := s.merchantRepo.Update(merchant); err != nil {
		return nil, fmt.Errorf("failed to update merchant: %w", err)
	}

	return merchant, nil
}

// GenerateAPIKey generates a secure random API key
// Format: spg_live_<64 random base64 characters>
func (s *MerchantService) GenerateAPIKey() (string, error) {
	// Generate 48 random bytes (will become 64 base64 characters)
	randomBytes := make([]byte, 48)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAPIKeyGenerationFailed, err)
	}

	// Encode to base64
	encoded := base64.URLEncoding.EncodeToString(randomBytes)

	// Format: spg_live_<random>
	apiKey := fmt.Sprintf("spg_live_%s", encoded)

	return apiKey, nil
}

// RotateAPIKey generates a new API key for a merchant
func (s *MerchantService) RotateAPIKey(merchantID string) (string, error) {
	if merchantID == "" {
		return "", errors.New("merchant ID cannot be empty")
	}

	// Get merchant
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return "", ErrMerchantNotFound
		}
		return "", fmt.Errorf("failed to get merchant: %w", err)
	}

	// Check if merchant is approved
	if merchant.KYCStatus != model.KYCStatusApproved {
		return "", ErrMerchantNotApproved
	}

	// Generate new API key
	newAPIKey, err := s.GenerateAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate new API key: %w", err)
	}

	// Update merchant
	merchant.APIKey = sql.NullString{String: newAPIKey, Valid: true}
	merchant.APIKeyCreatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	merchant.UpdatedAt = time.Now()

	// Save to database
	if err := s.merchantRepo.Update(merchant); err != nil {
		return "", fmt.Errorf("failed to update merchant: %w", err)
	}

	return newAPIKey, nil
}

// GetMerchantByID retrieves a merchant by ID
func (s *MerchantService) GetMerchantByID(merchantID string) (*model.Merchant, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}

	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	return merchant, nil
}

// GetMerchantByEmail retrieves a merchant by email
func (s *MerchantService) GetMerchantByEmail(email string) (*model.Merchant, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}

	merchant, err := s.merchantRepo.GetByEmail(email)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	return merchant, nil
}

// GetMerchantByAPIKey retrieves a merchant by API key
func (s *MerchantService) GetMerchantByAPIKey(apiKey string) (*model.Merchant, error) {
	if apiKey == "" {
		return nil, errors.New("API key cannot be empty")
	}

	merchant, err := s.merchantRepo.GetByAPIKey(apiKey)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	return merchant, nil
}

// ListMerchants retrieves merchants with pagination
func (s *MerchantService) ListMerchants(limit, offset int) ([]*model.Merchant, error) {
	merchants, err := s.merchantRepo.List(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list merchants: %w", err)
	}

	return merchants, nil
}

// ListMerchantsByKYCStatus retrieves merchants by KYC status with pagination
func (s *MerchantService) ListMerchantsByKYCStatus(status string, limit, offset int) ([]*model.Merchant, error) {
	merchants, err := s.merchantRepo.ListByKYCStatus(status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list merchants by KYC status: %w", err)
	}

	return merchants, nil
}

// UpdateWebhookConfig updates a merchant's webhook configuration
func (s *MerchantService) UpdateWebhookConfig(merchantID, webhookURL, webhookSecret string, webhookEvents []string) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	// Get merchant
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return ErrMerchantNotFound
		}
		return fmt.Errorf("failed to get merchant: %w", err)
	}

	// Update webhook config
	if webhookURL != "" {
		merchant.WebhookURL = sql.NullString{String: webhookURL, Valid: true}
	} else {
		merchant.WebhookURL = sql.NullString{Valid: false}
	}

	if webhookSecret != "" {
		merchant.WebhookSecret = sql.NullString{String: webhookSecret, Valid: true}
	} else {
		merchant.WebhookSecret = sql.NullString{Valid: false}
	}

	if len(webhookEvents) > 0 {
		merchant.WebhookEvents = webhookEvents
	} else {
		merchant.WebhookEvents = nil
	}

	merchant.UpdatedAt = time.Now()

	// Save to database
	if err := s.merchantRepo.Update(merchant); err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	return nil
}

// SuspendMerchant suspends a merchant account
func (s *MerchantService) SuspendMerchant(merchantID string, reason string) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	// Get merchant
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return ErrMerchantNotFound
		}
		return fmt.Errorf("failed to get merchant: %w", err)
	}

	// Update status
	merchant.Status = model.MerchantStatusSuspended
	merchant.KYCStatus = model.KYCStatusSuspended
	merchant.UpdatedAt = time.Now()

	// Save to database
	if err := s.merchantRepo.Update(merchant); err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	return nil
}

// ReactivateMerchant reactivates a suspended merchant account
func (s *MerchantService) ReactivateMerchant(merchantID string) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	// Get merchant
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			return ErrMerchantNotFound
		}
		return fmt.Errorf("failed to get merchant: %w", err)
	}

	// Only reactivate if currently suspended
	if merchant.Status != model.MerchantStatusSuspended {
		return errors.New("merchant is not suspended")
	}

	// Update status
	merchant.Status = model.MerchantStatusActive
	merchant.KYCStatus = model.KYCStatusApproved
	merchant.UpdatedAt = time.Now()

	// Save to database
	if err := s.merchantRepo.Update(merchant); err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	return nil
}
