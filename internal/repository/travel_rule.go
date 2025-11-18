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
	GetByID(ctx context.Context, id string) (*model.TravelRuleData, error)
	GetByPaymentID(ctx context.Context, paymentID string) (*model.TravelRuleData, error)
	List(ctx context.Context, filter TravelRuleFilter) ([]*model.TravelRuleData, error)
	Delete(ctx context.Context, id string) error
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

	query := `
		INSERT INTO travel_rule_data (
			id, payment_id, payer_full_name, payer_wallet_address, payer_id_document,
			payer_country, merchant_full_name, merchant_country,
			transaction_amount, transaction_currency, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		data.ID,
		data.PaymentID,
		data.PayerFullName,
		data.PayerWalletAddress,
		data.PayerIDDocument,
		data.PayerCountry,
		data.MerchantFullName,
		data.MerchantCountry,
		data.TransactionAmount,
		data.TransactionCurrency,
		data.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create travel rule data: %w", err)
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

// Stub implementations for other methods (can be implemented later as needed)

func (r *travelRuleRepositoryImpl) GetByID(ctx context.Context, id string) (*model.TravelRuleData, error) {
	// TODO: Implement when needed
	r.logger.Warn("TravelRuleRepository.GetByID not yet implemented")
	return nil, errors.New("not implemented")
}

func (r *travelRuleRepositoryImpl) GetByPaymentID(ctx context.Context, paymentID string) (*model.TravelRuleData, error) {
	// TODO: Implement when needed
	r.logger.Warn("TravelRuleRepository.GetByPaymentID not yet implemented")
	return nil, errors.New("not implemented")
}

func (r *travelRuleRepositoryImpl) Delete(ctx context.Context, id string) error {
	// TODO: Implement when needed
	r.logger.Warn("TravelRuleRepository.Delete not yet implemented")
	return errors.New("not implemented")
}
