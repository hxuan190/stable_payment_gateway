package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

var (
	// ErrTravelRuleDataNotFound is returned when travel rule data is not found
	ErrTravelRuleDataNotFound = errors.New("travel rule data not found")
	// ErrInvalidTravelRuleData is returned when travel rule data is invalid
	ErrInvalidTravelRuleData = errors.New("invalid travel rule data")
)

// TravelRuleRepository defines the interface for travel rule data access
type TravelRuleRepository interface {
	Create(ctx context.Context, data *model.TravelRuleData) error
	Update(ctx context.Context, data *model.TravelRuleData) error
	GetByID(ctx context.Context, id string) (*model.TravelRuleData, error)
	GetByPaymentID(ctx context.Context, paymentID string) (*model.TravelRuleData, error)
	List(ctx context.Context, filter TravelRuleFilter) ([]*model.TravelRuleData, error)
	Delete(ctx context.Context, id string) error
	UpdateRiskAssessment(ctx context.Context, id string, riskLevel string, riskScore float64) error
	MarkAsReported(ctx context.Context, id string, reportReference string) error
}

// TravelRuleFilter represents filters for querying travel rule data
type TravelRuleFilter struct {
	PayerCountry string
	MinAmount    *float64
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}

type travelRuleRepositoryImpl struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewTravelRuleRepository creates a new travel rule repository
func NewTravelRuleRepository(db *sql.DB, log *logger.Logger) TravelRuleRepository {
	return &travelRuleRepositoryImpl{
		db:     db,
		logger: log,
	}
}

// Create creates a new travel rule data record
// CRITICAL: Used by ComplianceService.StoreTravelRuleData
func (r *travelRuleRepositoryImpl) Create(ctx context.Context, data *model.TravelRuleData) error {
	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}

	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = time.Now()
	}

	// Set defaults for optional fields
	if data.RetentionPolicy == "" {
		data.RetentionPolicy = "standard"
	}

	query := `
		INSERT INTO travel_rule_data (
			id, payment_id,
			payer_full_name, payer_wallet_address, payer_id_document, payer_country,
			payer_date_of_birth, payer_address,
			merchant_full_name, merchant_country, merchant_wallet_address,
			merchant_id_document, merchant_business_registration, merchant_address,
			transaction_amount, transaction_currency, transaction_purpose,
			risk_level, risk_score, screening_status,
			retention_policy, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`

	_, err := r.db.ExecContext(ctx, query,
		data.ID,
		data.PaymentID,
		data.PayerFullName,
		data.PayerWalletAddress,
		data.PayerIDDocument,
		data.PayerCountry,
		data.PayerDateOfBirth,
		data.PayerAddress,
		data.MerchantFullName,
		data.MerchantCountry,
		data.MerchantWalletAddress,
		data.MerchantIDDocument,
		data.MerchantBusinessRegistration,
		data.MerchantAddress,
		data.TransactionAmount,
		data.TransactionCurrency,
		data.TransactionPurpose,
		data.RiskLevel,
		data.RiskScore,
		data.ScreeningStatus,
		data.RetentionPolicy,
		data.CreatedAt,
		data.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create travel rule data: %w", err)
	}

	return nil
}

// Update updates an existing travel rule data record
func (r *travelRuleRepositoryImpl) Update(ctx context.Context, data *model.TravelRuleData) error {
	query := `
		UPDATE travel_rule_data
		SET payer_full_name = $2,
		    payer_wallet_address = $3,
		    payer_id_document = $4,
		    payer_country = $5,
		    payer_date_of_birth = $6,
		    payer_address = $7,
		    merchant_full_name = $8,
		    merchant_country = $9,
		    merchant_wallet_address = $10,
		    merchant_id_document = $11,
		    merchant_business_registration = $12,
		    merchant_address = $13,
		    transaction_amount = $14,
		    transaction_currency = $15,
		    transaction_purpose = $16,
		    risk_level = $17,
		    risk_score = $18,
		    screening_status = $19,
		    screening_completed_at = $20,
		    reported_to_authority = $21,
		    reported_at = $22,
		    report_reference = $23,
		    retention_policy = $24,
		    archived_at = $25,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		data.ID,
		data.PayerFullName,
		data.PayerWalletAddress,
		data.PayerIDDocument,
		data.PayerCountry,
		data.PayerDateOfBirth,
		data.PayerAddress,
		data.MerchantFullName,
		data.MerchantCountry,
		data.MerchantWalletAddress,
		data.MerchantIDDocument,
		data.MerchantBusinessRegistration,
		data.MerchantAddress,
		data.TransactionAmount,
		data.TransactionCurrency,
		data.TransactionPurpose,
		data.RiskLevel,
		data.RiskScore,
		data.ScreeningStatus,
		data.ScreeningCompletedAt,
		data.ReportedToAuthority,
		data.ReportedAt,
		data.ReportReference,
		data.RetentionPolicy,
		data.ArchivedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update travel rule data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTravelRuleDataNotFound
	}

	return nil
}

// UpdateRiskAssessment updates the risk assessment for a travel rule record
func (r *travelRuleRepositoryImpl) UpdateRiskAssessment(ctx context.Context, id string, riskLevel string, riskScore float64) error {
	query := `
		UPDATE travel_rule_data
		SET risk_level = $2,
		    risk_score = $3,
		    screening_status = 'completed',
		    screening_completed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, riskLevel, riskScore)
	if err != nil {
		return fmt.Errorf("failed to update risk assessment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTravelRuleDataNotFound
	}

	return nil
}

// MarkAsReported marks a travel rule record as reported to regulatory authority
func (r *travelRuleRepositoryImpl) MarkAsReported(ctx context.Context, id string, reportReference string) error {
	query := `
		UPDATE travel_rule_data
		SET reported_to_authority = TRUE,
		    reported_at = NOW(),
		    report_reference = $2,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, reportReference)
	if err != nil {
		return fmt.Errorf("failed to mark as reported: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTravelRuleDataNotFound
	}

	return nil
}

// List retrieves travel rule data with filters
// CRITICAL: Used by ComplianceService.GetTravelRuleReport
func (r *travelRuleRepositoryImpl) List(ctx context.Context, filter TravelRuleFilter) ([]*model.TravelRuleData, error) {
	query := `
		SELECT id, payment_id, payer_full_name, payer_wallet_address, payer_id_document,
		       payer_country, merchant_full_name, merchant_country,
		       transaction_amount, transaction_currency, created_at
		FROM travel_rule_data
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, filter.StartDate)
		argCount++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, filter.EndDate)
		argCount++
	}

	if filter.MinAmount != nil {
		query += fmt.Sprintf(" AND transaction_amount >= $%d", argCount)
		args = append(args, filter.MinAmount)
		argCount++
	}

	if filter.PayerCountry != "" {
		query += fmt.Sprintf(" AND payer_country = $%d", argCount)
		args = append(args, filter.PayerCountry)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list travel rule data: %w", err)
	}
	defer rows.Close()

	var results []*model.TravelRuleData
	for rows.Next() {
		var data model.TravelRuleData
		err := rows.Scan(
			&data.ID,
			&data.PaymentID,
			&data.PayerFullName,
			&data.PayerWalletAddress,
			&data.PayerIDDocument,
			&data.PayerCountry,
			&data.MerchantFullName,
			&data.MerchantCountry,
			&data.TransactionAmount,
			&data.TransactionCurrency,
			&data.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan travel rule data: %w", err)
		}
		results = append(results, &data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating travel rule data: %w", err)
	}

	return results, nil
}

// GetByID retrieves a travel rule data record by ID
func (r *travelRuleRepositoryImpl) GetByID(ctx context.Context, id string) (*model.TravelRuleData, error) {
	query := `
		SELECT id, payment_id, payer_full_name, payer_wallet_address, payer_id_document,
		       payer_country, payer_date_of_birth, payer_address,
		       merchant_full_name, merchant_country, merchant_wallet_address,
		       merchant_id_document, merchant_business_registration, merchant_address,
		       transaction_amount, transaction_currency, transaction_purpose,
		       risk_level, risk_score, screening_status, screening_completed_at,
		       reported_to_authority, reported_at, report_reference,
		       retention_policy, archived_at, created_at, updated_at
		FROM travel_rule_data
		WHERE id = $1
	`

	var data model.TravelRuleData
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&data.ID,
		&data.PaymentID,
		&data.PayerFullName,
		&data.PayerWalletAddress,
		&data.PayerIDDocument,
		&data.PayerCountry,
		&data.PayerDateOfBirth,
		&data.PayerAddress,
		&data.MerchantFullName,
		&data.MerchantCountry,
		&data.MerchantWalletAddress,
		&data.MerchantIDDocument,
		&data.MerchantBusinessRegistration,
		&data.MerchantAddress,
		&data.TransactionAmount,
		&data.TransactionCurrency,
		&data.TransactionPurpose,
		&data.RiskLevel,
		&data.RiskScore,
		&data.ScreeningStatus,
		&data.ScreeningCompletedAt,
		&data.ReportedToAuthority,
		&data.ReportedAt,
		&data.ReportReference,
		&data.RetentionPolicy,
		&data.ArchivedAt,
		&data.CreatedAt,
		&data.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTravelRuleDataNotFound
		}
		return nil, fmt.Errorf("failed to get travel rule data by ID: %w", err)
	}

	return &data, nil
}

// GetByPaymentID retrieves travel rule data for a specific payment
func (r *travelRuleRepositoryImpl) GetByPaymentID(ctx context.Context, paymentID string) (*model.TravelRuleData, error) {
	query := `
		SELECT id, payment_id, payer_full_name, payer_wallet_address, payer_id_document,
		       payer_country, payer_date_of_birth, payer_address,
		       merchant_full_name, merchant_country, merchant_wallet_address,
		       merchant_id_document, merchant_business_registration, merchant_address,
		       transaction_amount, transaction_currency, transaction_purpose,
		       risk_level, risk_score, screening_status, screening_completed_at,
		       reported_to_authority, reported_at, report_reference,
		       retention_policy, archived_at, created_at, updated_at
		FROM travel_rule_data
		WHERE payment_id = $1
	`

	var data model.TravelRuleData
	err := r.db.QueryRowContext(ctx, query, paymentID).Scan(
		&data.ID,
		&data.PaymentID,
		&data.PayerFullName,
		&data.PayerWalletAddress,
		&data.PayerIDDocument,
		&data.PayerCountry,
		&data.PayerDateOfBirth,
		&data.PayerAddress,
		&data.MerchantFullName,
		&data.MerchantCountry,
		&data.MerchantWalletAddress,
		&data.MerchantIDDocument,
		&data.MerchantBusinessRegistration,
		&data.MerchantAddress,
		&data.TransactionAmount,
		&data.TransactionCurrency,
		&data.TransactionPurpose,
		&data.RiskLevel,
		&data.RiskScore,
		&data.ScreeningStatus,
		&data.ScreeningCompletedAt,
		&data.ReportedToAuthority,
		&data.ReportedAt,
		&data.ReportReference,
		&data.RetentionPolicy,
		&data.ArchivedAt,
		&data.CreatedAt,
		&data.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTravelRuleDataNotFound
		}
		return nil, fmt.Errorf("failed to get travel rule data by payment ID: %w", err)
	}

	return &data, nil
}

// Delete soft-deletes a travel rule data record
// IMPORTANT: Travel Rule data should NEVER be hard-deleted for compliance reasons
// This method is kept for interface compliance but should NOT be used in production
func (r *travelRuleRepositoryImpl) Delete(ctx context.Context, id string) error {
	r.logger.Warn("TravelRuleRepository.Delete called - this should NOT be used in production", map[string]interface{}{
		"id": id,
	})
	// Return error to prevent accidental deletion
	return errors.New("deletion of Travel Rule data is not allowed for compliance reasons - use archival instead")
}

// GetByDateRange retrieves all travel rule data within a date range for SBV reporting
// Used by SBVReportService for regulatory reporting
func (r *travelRuleRepositoryImpl) GetByDateRange(startDate, endDate time.Time) ([]*model.TravelRuleData, error) {
	query := `
		SELECT id, payment_id,
		       payer_full_name, payer_wallet_address, payer_id_document,
		       payer_country, payer_date_of_birth, payer_address,
		       merchant_full_name, merchant_country, merchant_wallet_address,
		       merchant_id_document, merchant_business_registration, merchant_address,
		       transaction_amount, transaction_currency, transaction_purpose,
		       risk_level, risk_score, screening_status, screening_completed_at,
		       reported_to_authority, reported_at, report_reference,
		       retention_policy, archived_at, created_at, updated_at
		FROM travel_rule_data
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get travel rule data by date range: %w", err)
	}
	defer rows.Close()

	var results []*model.TravelRuleData
	for rows.Next() {
		var data model.TravelRuleData
		err := rows.Scan(
			&data.ID,
			&data.PaymentID,
			&data.PayerFullName,
			&data.PayerWalletAddress,
			&data.PayerIDDocument,
			&data.PayerCountry,
			&data.PayerDateOfBirth,
			&data.PayerAddress,
			&data.MerchantFullName,
			&data.MerchantCountry,
			&data.MerchantWalletAddress,
			&data.MerchantIDDocument,
			&data.MerchantBusinessRegistration,
			&data.MerchantAddress,
			&data.TransactionAmount,
			&data.TransactionCurrency,
			&data.TransactionPurpose,
			&data.RiskLevel,
			&data.RiskScore,
			&data.ScreeningStatus,
			&data.ScreeningCompletedAt,
			&data.ReportedToAuthority,
			&data.ReportedAt,
			&data.ReportReference,
			&data.RetentionPolicy,
			&data.ArchivedAt,
			&data.CreatedAt,
			&data.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan travel rule data: %w", err)
		}
		results = append(results, &data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating travel rule data: %w", err)
	}

	return results, nil
}
