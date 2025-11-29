package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/audit/domain"
)

var (
	ErrAuditLogNotFound   = errors.New("audit log not found")
	ErrInvalidAuditFilter = errors.New("invalid audit filter")
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{
		db: db,
	}
}

type AuditFilter struct {
	ActorType      *domain.ActorType
	ActorID        *string
	ActorEmail     *string
	ActorIPAddress *string
	Action         *string
	ActionCategory *domain.ActionCategory
	ResourceType   *string
	ResourceID     *string
	Status         *domain.AuditStatus
	RequestID      *string
	CorrelationID  *string
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	Offset         int
	SortBy         string
	SortOrder      string
}

func (r *AuditRepository) Create(log *domain.AuditLog) error {
	if log == nil {
		return errors.New("audit log cannot be nil")
	}

	now := time.Now()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}

	if err := r.db.Create(log).Error; err != nil {
		return err
	}

	return nil
}

func (r *AuditRepository) CreateBatch(logs []*domain.AuditLog) error {
	if len(logs) == 0 {
		return errors.New("logs array cannot be empty")
	}

	now := time.Now()
	for _, log := range logs {
		if log != nil && log.CreatedAt.IsZero() {
			log.CreatedAt = now
		}
	}

	if err := r.db.CreateInBatches(logs, 100).Error; err != nil {
		return err
	}

	return nil
}

func (r *AuditRepository) GetByID(id string) (*domain.AuditLog, error) {
	if id == "" {
		return nil, errors.New("audit log ID cannot be empty")
	}

	log := &domain.AuditLog{}
	if err := r.db.Where("id = ?", id).First(log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAuditLogNotFound
		}
		return nil, err
	}

	return log, nil
}

func (r *AuditRepository) List(filter AuditFilter) ([]*domain.AuditLog, error) {
	query := r.db.Model(&domain.AuditLog{})

	if filter.ActorType != nil {
		query = query.Where("actor_type = ?", *filter.ActorType)
	}
	if filter.ActorID != nil {
		query = query.Where("actor_id = ?", *filter.ActorID)
	}
	if filter.ActorEmail != nil {
		query = query.Where("actor_email = ?", *filter.ActorEmail)
	}
	if filter.ActorIPAddress != nil {
		query = query.Where("actor_ip_address = ?", *filter.ActorIPAddress)
	}
	if filter.Action != nil {
		query = query.Where("action = ?", *filter.Action)
	}
	if filter.ActionCategory != nil {
		query = query.Where("action_category = ?", *filter.ActionCategory)
	}
	if filter.ResourceType != nil {
		query = query.Where("resource_type = ?", *filter.ResourceType)
	}
	if filter.ResourceID != nil {
		query = query.Where("resource_id = ?", *filter.ResourceID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.RequestID != nil {
		query = query.Where("request_id = ?", *filter.RequestID)
	}
	if filter.CorrelationID != nil {
		query = query.Where("correlation_id = ?", *filter.CorrelationID)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	sortBy := "created_at"
	if filter.SortBy != "" {
		allowedSortFields := []string{"created_at", "action", "actor_type", "resource_type", "status"}
		if contains(allowedSortFields, filter.SortBy) {
			sortBy = filter.SortBy
		}
	}

	sortOrder := "DESC"
	if filter.SortOrder != "" {
		upperOrder := strings.ToUpper(filter.SortOrder)
		if upperOrder == "ASC" || upperOrder == "DESC" {
			sortOrder = upperOrder
		}
	}

	query = query.Order(sortBy + " " + sortOrder)

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(100)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var logs []*domain.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *AuditRepository) Count(filter AuditFilter) (int64, error) {
	query := r.db.Model(&domain.AuditLog{})

	if filter.ActorType != nil {
		query = query.Where("actor_type = ?", *filter.ActorType)
	}
	if filter.ActorID != nil {
		query = query.Where("actor_id = ?", *filter.ActorID)
	}
	if filter.ActionCategory != nil {
		query = query.Where("action_category = ?", *filter.ActionCategory)
	}
	if filter.ResourceType != nil {
		query = query.Where("resource_type = ?", *filter.ResourceType)
	}
	if filter.ResourceID != nil {
		query = query.Where("resource_id = ?", *filter.ResourceID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *AuditRepository) GetByResource(resourceType, resourceID string, limit, offset int) ([]*domain.AuditLog, error) {
	filter := AuditFilter{
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Limit:        limit,
		Offset:       offset,
	}
	return r.List(filter)
}

func (r *AuditRepository) GetByActor(actorType domain.ActorType, actorID string, limit, offset int) ([]*domain.AuditLog, error) {
	filter := AuditFilter{
		ActorType: &actorType,
		ActorID:   &actorID,
		Limit:     limit,
		Offset:    offset,
	}
	return r.List(filter)
}

func (r *AuditRepository) GetByRequestID(requestID string) ([]*domain.AuditLog, error) {
	filter := AuditFilter{
		RequestID: &requestID,
	}
	return r.List(filter)
}

func (r *AuditRepository) GetRecentFailures(hours int, limit int) ([]*domain.AuditLog, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	failedStatus := domain.AuditStatusFailed

	filter := AuditFilter{
		Status:    &failedStatus,
		StartTime: &startTime,
		Limit:     limit,
	}
	return r.List(filter)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
