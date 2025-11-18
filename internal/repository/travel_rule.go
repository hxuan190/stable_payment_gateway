package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
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
	PayerCountry  string
	MinAmount     *float64
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int
	Offset        int
}

type travelRuleRepositoryImpl struct {
	db *gorm.DB
}

// NewTravelRuleRepository creates a new travel rule repository
func NewTravelRuleRepository(db *gorm.DB) TravelRuleRepository {
	return &travelRuleRepositoryImpl{
		db: db,
	}
}

// Create creates a new travel rule data record
func (r *travelRuleRepositoryImpl) Create(ctx context.Context, data *model.TravelRuleData) error {
	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}

	result := r.db.WithContext(ctx).Create(data)
	if result.Error != nil {
		return fmt.Errorf("failed to create travel rule data: %w", result.Error)
	}

	return nil
}

// GetByID retrieves travel rule data by ID
func (r *travelRuleRepositoryImpl) GetByID(ctx context.Context, id string) (*model.TravelRuleData, error) {
	var data model.TravelRuleData
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&data)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrTravelRuleDataNotFound
		}
		return nil, fmt.Errorf("failed to get travel rule data: %w", result.Error)
	}

	return &data, nil
}

// GetByPaymentID retrieves travel rule data by payment ID
func (r *travelRuleRepositoryImpl) GetByPaymentID(ctx context.Context, paymentID string) (*model.TravelRuleData, error) {
	var data model.TravelRuleData
	result := r.db.WithContext(ctx).Where("payment_id = ?", paymentID).First(&data)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrTravelRuleDataNotFound
		}
		return nil, fmt.Errorf("failed to get travel rule data by payment ID: %w", result.Error)
	}

	return &data, nil
}

// List retrieves travel rule data with filters
func (r *travelRuleRepositoryImpl) List(ctx context.Context, filter TravelRuleFilter) ([]*model.TravelRuleData, error) {
	query := r.db.WithContext(ctx).Model(&model.TravelRuleData{})

	// Apply filters
	if filter.PayerCountry != "" {
		query = query.Where("payer_country = ?", filter.PayerCountry)
	}

	if filter.MinAmount != nil {
		query = query.Where("transaction_amount >= ?", *filter.MinAmount)
	}

	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	// Apply sorting
	query = query.Order("created_at DESC")

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var dataList []*model.TravelRuleData
	result := query.Find(&dataList)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list travel rule data: %w", result.Error)
	}

	return dataList, nil
}

// Delete deletes travel rule data by ID
// Note: In production, you might want to implement soft delete instead
func (r *travelRuleRepositoryImpl) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.TravelRuleData{}, "id = ?", id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete travel rule data: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTravelRuleDataNotFound
	}

	return nil
}

// GetTravelRuleReport generates a report of travel rule data for regulatory compliance
func (r *travelRuleRepositoryImpl) GetTravelRuleReport(ctx context.Context, startDate, endDate time.Time) ([]*model.TravelRuleData, error) {
	var dataList []*model.TravelRuleData

	result := r.db.WithContext(ctx).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Where("transaction_amount >= ?", 1000). // Travel Rule threshold: $1000 USD
		Order("created_at DESC").
		Find(&dataList)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to generate travel rule report: %w", result.Error)
	}

	return dataList, nil
}

// GetStatsByCountry returns statistics of travel rule data grouped by country
func (r *travelRuleRepositoryImpl) GetStatsByCountry(ctx context.Context, startDate, endDate time.Time) (map[string]int64, error) {
	type CountryStats struct {
		PayerCountry string
		Count        int64
	}

	var stats []CountryStats
	result := r.db.WithContext(ctx).
		Model(&model.TravelRuleData{}).
		Select("payer_country, COUNT(*) as count").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("payer_country").
		Order("count DESC").
		Find(&stats)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get stats by country: %w", result.Error)
	}

	statsMap := make(map[string]int64)
	for _, stat := range stats {
		statsMap[stat.PayerCountry] = stat.Count
	}

	return statsMap, nil
}

// Ensure model.TravelRuleData implements the correct table name
func init() {
	// This ensures GORM uses the correct table name
	sql.Register("travel_rule_data", &model.TravelRuleData{})
}
