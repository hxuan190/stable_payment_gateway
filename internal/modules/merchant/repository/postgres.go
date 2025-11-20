package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrMerchantNotFound is returned when a merchant is not found
	ErrMerchantNotFound = errors.New("merchant not found")
	// ErrMerchantAlreadyExists is returned when a merchant with the same email already exists
	ErrMerchantAlreadyExists = errors.New("merchant with this email already exists")
	// ErrInvalidMerchantID is returned when the merchant ID is invalid
	ErrInvalidMerchantID = errors.New("invalid merchant ID")
)

// MerchantRepository handles database operations for merchants
type MerchantRepository struct {
	db *sql.DB
}

// NewMerchantRepository creates a new merchant repository
func NewMerchantRepository(db *sql.DB) *MerchantRepository {
	return &MerchantRepository{
		db: db,
	}
}

// Create inserts a new merchant into the database
func (r *MerchantRepository) Create(merchant *model.Merchant) error {
	if merchant == nil {
		return errors.New("merchant cannot be nil")
	}

	query := `
		INSERT INTO merchants (
			id, email, business_name, business_tax_id, business_license_number,
			owner_full_name, owner_id_card, phone_number,
			business_address, city, district,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			kyc_status, kyc_submitted_at,
			api_key, api_key_created_at,
			webhook_url, webhook_secret, webhook_events,
			status, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if merchant.CreatedAt.IsZero() {
		merchant.CreatedAt = now
	}
	if merchant.UpdatedAt.IsZero() {
		merchant.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		merchant.ID,
		merchant.Email,
		merchant.BusinessName,
		merchant.BusinessTaxID,
		merchant.BusinessLicenseNumber,
		merchant.OwnerFullName,
		merchant.OwnerIDCard,
		merchant.PhoneNumber,
		merchant.BusinessAddress,
		merchant.City,
		merchant.District,
		merchant.BankAccountName,
		merchant.BankAccountNumber,
		merchant.BankName,
		merchant.BankBranch,
		merchant.KYCStatus,
		merchant.KYCSubmittedAt,
		merchant.APIKey,
		merchant.APIKeyCreatedAt,
		merchant.WebhookURL,
		merchant.WebhookSecret,
		merchant.WebhookEvents,
		merchant.Status,
		merchant.Metadata,
		merchant.CreatedAt,
		merchant.UpdatedAt,
	).Scan(&merchant.ID, &merchant.CreatedAt, &merchant.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if err.Error() == `pq: duplicate key value violates unique constraint "merchants_email_key"` {
			return ErrMerchantAlreadyExists
		}
		return fmt.Errorf("failed to create merchant: %w", err)
	}

	return nil
}

// GetByID retrieves a merchant by ID
func (r *MerchantRepository) GetByID(id string) (*model.Merchant, error) {
	if id == "" {
		return nil, ErrInvalidMerchantID
	}

	query := `
		SELECT
			id, email, business_name, business_tax_id, business_license_number,
			owner_full_name, owner_id_card, phone_number,
			business_address, city, district,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			kyc_status, kyc_submitted_at, kyc_approved_at, kyc_approved_by, kyc_rejection_reason,
			api_key, api_key_created_at,
			webhook_url, webhook_secret, webhook_events,
			status, metadata, created_at, updated_at, deleted_at
		FROM merchants
		WHERE id = $1 AND deleted_at IS NULL
	`

	merchant := &model.Merchant{}
	err := r.db.QueryRow(query, id).Scan(
		&merchant.ID,
		&merchant.Email,
		&merchant.BusinessName,
		&merchant.BusinessTaxID,
		&merchant.BusinessLicenseNumber,
		&merchant.OwnerFullName,
		&merchant.OwnerIDCard,
		&merchant.PhoneNumber,
		&merchant.BusinessAddress,
		&merchant.City,
		&merchant.District,
		&merchant.BankAccountName,
		&merchant.BankAccountNumber,
		&merchant.BankName,
		&merchant.BankBranch,
		&merchant.KYCStatus,
		&merchant.KYCSubmittedAt,
		&merchant.KYCApprovedAt,
		&merchant.KYCApprovedBy,
		&merchant.KYCRejectionReason,
		&merchant.APIKey,
		&merchant.APIKeyCreatedAt,
		&merchant.WebhookURL,
		&merchant.WebhookSecret,
		&merchant.WebhookEvents,
		&merchant.Status,
		&merchant.Metadata,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
		&merchant.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant by ID: %w", err)
	}

	return merchant, nil
}

// GetByEmail retrieves a merchant by email address
func (r *MerchantRepository) GetByEmail(email string) (*model.Merchant, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	query := `
		SELECT
			id, email, business_name, business_tax_id, business_license_number,
			owner_full_name, owner_id_card, phone_number,
			business_address, city, district,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			kyc_status, kyc_submitted_at, kyc_approved_at, kyc_approved_by, kyc_rejection_reason,
			api_key, api_key_created_at,
			webhook_url, webhook_secret, webhook_events,
			status, metadata, created_at, updated_at, deleted_at
		FROM merchants
		WHERE email = $1 AND deleted_at IS NULL
	`

	merchant := &model.Merchant{}
	err := r.db.QueryRow(query, email).Scan(
		&merchant.ID,
		&merchant.Email,
		&merchant.BusinessName,
		&merchant.BusinessTaxID,
		&merchant.BusinessLicenseNumber,
		&merchant.OwnerFullName,
		&merchant.OwnerIDCard,
		&merchant.PhoneNumber,
		&merchant.BusinessAddress,
		&merchant.City,
		&merchant.District,
		&merchant.BankAccountName,
		&merchant.BankAccountNumber,
		&merchant.BankName,
		&merchant.BankBranch,
		&merchant.KYCStatus,
		&merchant.KYCSubmittedAt,
		&merchant.KYCApprovedAt,
		&merchant.KYCApprovedBy,
		&merchant.KYCRejectionReason,
		&merchant.APIKey,
		&merchant.APIKeyCreatedAt,
		&merchant.WebhookURL,
		&merchant.WebhookSecret,
		&merchant.WebhookEvents,
		&merchant.Status,
		&merchant.Metadata,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
		&merchant.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant by email: %w", err)
	}

	return merchant, nil
}

// GetByAPIKey retrieves a merchant by API key
func (r *MerchantRepository) GetByAPIKey(apiKey string) (*model.Merchant, error) {
	if apiKey == "" {
		return nil, errors.New("API key cannot be empty")
	}

	query := `
		SELECT
			id, email, business_name, business_tax_id, business_license_number,
			owner_full_name, owner_id_card, phone_number,
			business_address, city, district,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			kyc_status, kyc_submitted_at, kyc_approved_at, kyc_approved_by, kyc_rejection_reason,
			api_key, api_key_created_at,
			webhook_url, webhook_secret, webhook_events,
			status, metadata, created_at, updated_at, deleted_at
		FROM merchants
		WHERE api_key = $1 AND deleted_at IS NULL
	`

	merchant := &model.Merchant{}
	err := r.db.QueryRow(query, apiKey).Scan(
		&merchant.ID,
		&merchant.Email,
		&merchant.BusinessName,
		&merchant.BusinessTaxID,
		&merchant.BusinessLicenseNumber,
		&merchant.OwnerFullName,
		&merchant.OwnerIDCard,
		&merchant.PhoneNumber,
		&merchant.BusinessAddress,
		&merchant.City,
		&merchant.District,
		&merchant.BankAccountName,
		&merchant.BankAccountNumber,
		&merchant.BankName,
		&merchant.BankBranch,
		&merchant.KYCStatus,
		&merchant.KYCSubmittedAt,
		&merchant.KYCApprovedAt,
		&merchant.KYCApprovedBy,
		&merchant.KYCRejectionReason,
		&merchant.APIKey,
		&merchant.APIKeyCreatedAt,
		&merchant.WebhookURL,
		&merchant.WebhookSecret,
		&merchant.WebhookEvents,
		&merchant.Status,
		&merchant.Metadata,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
		&merchant.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant by API key: %w", err)
	}

	return merchant, nil
}

// Update updates an existing merchant in the database
func (r *MerchantRepository) Update(merchant *model.Merchant) error {
	if merchant == nil {
		return errors.New("merchant cannot be nil")
	}
	if merchant.ID == "" {
		return ErrInvalidMerchantID
	}

	query := `
		UPDATE merchants SET
			email = $2,
			business_name = $3,
			business_tax_id = $4,
			business_license_number = $5,
			owner_full_name = $6,
			owner_id_card = $7,
			phone_number = $8,
			business_address = $9,
			city = $10,
			district = $11,
			bank_account_name = $12,
			bank_account_number = $13,
			bank_name = $14,
			bank_branch = $15,
			kyc_status = $16,
			kyc_submitted_at = $17,
			kyc_approved_at = $18,
			kyc_approved_by = $19,
			kyc_rejection_reason = $20,
			api_key = $21,
			api_key_created_at = $22,
			webhook_url = $23,
			webhook_secret = $24,
			webhook_events = $25,
			status = $26,
			metadata = $27,
			updated_at = $28
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	merchant.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		merchant.ID,
		merchant.Email,
		merchant.BusinessName,
		merchant.BusinessTaxID,
		merchant.BusinessLicenseNumber,
		merchant.OwnerFullName,
		merchant.OwnerIDCard,
		merchant.PhoneNumber,
		merchant.BusinessAddress,
		merchant.City,
		merchant.District,
		merchant.BankAccountName,
		merchant.BankAccountNumber,
		merchant.BankName,
		merchant.BankBranch,
		merchant.KYCStatus,
		merchant.KYCSubmittedAt,
		merchant.KYCApprovedAt,
		merchant.KYCApprovedBy,
		merchant.KYCRejectionReason,
		merchant.APIKey,
		merchant.APIKeyCreatedAt,
		merchant.WebhookURL,
		merchant.WebhookSecret,
		merchant.WebhookEvents,
		merchant.Status,
		merchant.Metadata,
		merchant.UpdatedAt,
	).Scan(&merchant.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrMerchantNotFound
		}
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	return nil
}

// UpdateKYCStatus updates only the KYC status of a merchant
func (r *MerchantRepository) UpdateKYCStatus(id string, status string) error {
	if id == "" {
		return ErrInvalidMerchantID
	}
	if status == "" {
		return errors.New("KYC status cannot be empty")
	}

	query := `
		UPDATE merchants
		SET kyc_status = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update KYC status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrMerchantNotFound
	}

	return nil
}

// List retrieves merchants with pagination
func (r *MerchantRepository) List(limit, offset int) ([]*model.Merchant, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, email, business_name, business_tax_id, business_license_number,
			owner_full_name, owner_id_card, phone_number,
			business_address, city, district,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			kyc_status, kyc_submitted_at, kyc_approved_at, kyc_approved_by, kyc_rejection_reason,
			api_key, api_key_created_at,
			webhook_url, webhook_secret, webhook_events,
			status, metadata, created_at, updated_at, deleted_at
		FROM merchants
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list merchants: %w", err)
	}
	defer rows.Close()

	merchants := make([]*model.Merchant, 0)
	for rows.Next() {
		merchant := &model.Merchant{}
		err := rows.Scan(
			&merchant.ID,
			&merchant.Email,
			&merchant.BusinessName,
			&merchant.BusinessTaxID,
			&merchant.BusinessLicenseNumber,
			&merchant.OwnerFullName,
			&merchant.OwnerIDCard,
			&merchant.PhoneNumber,
			&merchant.BusinessAddress,
			&merchant.City,
			&merchant.District,
			&merchant.BankAccountName,
			&merchant.BankAccountNumber,
			&merchant.BankName,
			&merchant.BankBranch,
			&merchant.KYCStatus,
			&merchant.KYCSubmittedAt,
			&merchant.KYCApprovedAt,
			&merchant.KYCApprovedBy,
			&merchant.KYCRejectionReason,
			&merchant.APIKey,
			&merchant.APIKeyCreatedAt,
			&merchant.WebhookURL,
			&merchant.WebhookSecret,
			&merchant.WebhookEvents,
			&merchant.Status,
			&merchant.Metadata,
			&merchant.CreatedAt,
			&merchant.UpdatedAt,
			&merchant.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}
		merchants = append(merchants, merchant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchants: %w", err)
	}

	return merchants, nil
}

// ListByKYCStatus retrieves merchants by KYC status with pagination
func (r *MerchantRepository) ListByKYCStatus(status string, limit, offset int) ([]*model.Merchant, error) {
	if status == "" {
		return nil, errors.New("KYC status cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, email, business_name, business_tax_id, business_license_number,
			owner_full_name, owner_id_card, phone_number,
			business_address, city, district,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			kyc_status, kyc_submitted_at, kyc_approved_at, kyc_approved_by, kyc_rejection_reason,
			api_key, api_key_created_at,
			webhook_url, webhook_secret, webhook_events,
			status, metadata, created_at, updated_at, deleted_at
		FROM merchants
		WHERE kyc_status = $1 AND deleted_at IS NULL
		ORDER BY kyc_submitted_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list merchants by KYC status: %w", err)
	}
	defer rows.Close()

	merchants := make([]*model.Merchant, 0)
	for rows.Next() {
		merchant := &model.Merchant{}
		err := rows.Scan(
			&merchant.ID,
			&merchant.Email,
			&merchant.BusinessName,
			&merchant.BusinessTaxID,
			&merchant.BusinessLicenseNumber,
			&merchant.OwnerFullName,
			&merchant.OwnerIDCard,
			&merchant.PhoneNumber,
			&merchant.BusinessAddress,
			&merchant.City,
			&merchant.District,
			&merchant.BankAccountName,
			&merchant.BankAccountNumber,
			&merchant.BankName,
			&merchant.BankBranch,
			&merchant.KYCStatus,
			&merchant.KYCSubmittedAt,
			&merchant.KYCApprovedAt,
			&merchant.KYCApprovedBy,
			&merchant.KYCRejectionReason,
			&merchant.APIKey,
			&merchant.APIKeyCreatedAt,
			&merchant.WebhookURL,
			&merchant.WebhookSecret,
			&merchant.WebhookEvents,
			&merchant.Status,
			&merchant.Metadata,
			&merchant.CreatedAt,
			&merchant.UpdatedAt,
			&merchant.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}
		merchants = append(merchants, merchant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchants: %w", err)
	}

	return merchants, nil
}

// Delete soft-deletes a merchant by setting the deleted_at timestamp
func (r *MerchantRepository) Delete(id string) error {
	if id == "" {
		return ErrInvalidMerchantID
	}

	query := `
		UPDATE merchants
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete merchant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrMerchantNotFound
	}

	return nil
}

// Count returns the total number of merchants (excluding deleted)
func (r *MerchantRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM merchants WHERE deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count merchants: %w", err)
	}

	return count, nil
}

// CountByKYCStatus returns the count of merchants by KYC status
func (r *MerchantRepository) CountByKYCStatus(status string) (int64, error) {
	if status == "" {
		return 0, errors.New("KYC status cannot be empty")
	}

	query := `SELECT COUNT(*) FROM merchants WHERE kyc_status = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count merchants by KYC status: %w", err)
	}

	return count, nil
}
