package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrAuditLogNotFound is returned when an audit log is not found
	ErrAuditLogNotFound = errors.New("audit log not found")
	// ErrInvalidAuditFilter is returned when an invalid filter is provided
	ErrInvalidAuditFilter = errors.New("invalid audit filter")
)

// AuditRepository handles database operations for audit logs
// IMPORTANT: Audit logs are append-only and immutable
// This repository ONLY supports Create and List operations, NO Update or Delete
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{
		db: db,
	}
}

// AuditFilter represents filter criteria for querying audit logs
type AuditFilter struct {
	// Actor filters
	ActorType      *model.ActorType
	ActorID        *string
	ActorEmail     *string
	ActorIPAddress *string

	// Action filters
	Action         *string
	ActionCategory *model.ActionCategory

	// Resource filters
	ResourceType *string
	ResourceID   *string

	// Status filter
	Status *model.AuditStatus

	// Request tracing
	RequestID     *string
	CorrelationID *string

	// Time range
	StartTime *time.Time
	EndTime   *time.Time

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // field to sort by (created_at, action, etc.)
	SortOrder string // ASC or DESC
}

// Create inserts a new audit log entry into the database
// Audit logs are immutable and cannot be updated or deleted
func (r *AuditRepository) Create(log *model.AuditLog) error {
	if log == nil {
		return errors.New("audit log cannot be nil")
	}

	query := `
		INSERT INTO audit_logs (
			id, actor_type, actor_id, actor_email, actor_ip_address,
			action, action_category,
			resource_type, resource_id,
			status, error_message,
			http_method, http_path, http_status_code, user_agent,
			old_values, new_values,
			metadata, description,
			request_id, correlation_id,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22
		)
		RETURNING id, created_at
	`

	now := time.Now()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}

	err := r.db.QueryRow(
		query,
		log.ID,
		log.ActorType,
		log.ActorID,
		log.ActorEmail,
		log.ActorIPAddress,
		log.Action,
		log.ActionCategory,
		log.ResourceType,
		log.ResourceID,
		log.Status,
		log.ErrorMessage,
		log.HTTPMethod,
		log.HTTPPath,
		log.HTTPStatusCode,
		log.UserAgent,
		log.OldValues,
		log.NewValues,
		log.Metadata,
		log.Description,
		log.RequestID,
		log.CorrelationID,
		log.CreatedAt,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple audit log entries in a single transaction
// This is more efficient for bulk operations
func (r *AuditRepository) CreateBatch(logs []*model.AuditLog) error {
	if len(logs) == 0 {
		return errors.New("logs array cannot be empty")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO audit_logs (
			id, actor_type, actor_id, actor_email, actor_ip_address,
			action, action_category,
			resource_type, resource_id,
			status, error_message,
			http_method, http_path, http_status_code, user_agent,
			old_values, new_values,
			metadata, description,
			request_id, correlation_id,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22
		)
		RETURNING id, created_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, log := range logs {
		if log == nil {
			continue
		}

		if log.CreatedAt.IsZero() {
			log.CreatedAt = now
		}

		err = stmt.QueryRow(
			log.ID,
			log.ActorType,
			log.ActorID,
			log.ActorEmail,
			log.ActorIPAddress,
			log.Action,
			log.ActionCategory,
			log.ResourceType,
			log.ResourceID,
			log.Status,
			log.ErrorMessage,
			log.HTTPMethod,
			log.HTTPPath,
			log.HTTPStatusCode,
			log.UserAgent,
			log.OldValues,
			log.NewValues,
			log.Metadata,
			log.Description,
			log.RequestID,
			log.CorrelationID,
			log.CreatedAt,
		).Scan(&log.ID, &log.CreatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert audit log: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a single audit log by ID
func (r *AuditRepository) GetByID(id string) (*model.AuditLog, error) {
	if id == "" {
		return nil, errors.New("audit log ID cannot be empty")
	}

	query := `
		SELECT
			id, actor_type, actor_id, actor_email, actor_ip_address,
			action, action_category,
			resource_type, resource_id,
			status, error_message,
			http_method, http_path, http_status_code, user_agent,
			old_values, new_values,
			metadata, description,
			request_id, correlation_id,
			created_at
		FROM audit_logs
		WHERE id = $1
	`

	log := &model.AuditLog{}
	err := r.db.QueryRow(query, id).Scan(
		&log.ID,
		&log.ActorType,
		&log.ActorID,
		&log.ActorEmail,
		&log.ActorIPAddress,
		&log.Action,
		&log.ActionCategory,
		&log.ResourceType,
		&log.ResourceID,
		&log.Status,
		&log.ErrorMessage,
		&log.HTTPMethod,
		&log.HTTPPath,
		&log.HTTPStatusCode,
		&log.UserAgent,
		&log.OldValues,
		&log.NewValues,
		&log.Metadata,
		&log.Description,
		&log.RequestID,
		&log.CorrelationID,
		&log.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAuditLogNotFound
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return log, nil
}

// List retrieves audit logs based on filter criteria
func (r *AuditRepository) List(filter AuditFilter) ([]*model.AuditLog, error) {
	// Build dynamic query based on filter
	query := `
		SELECT
			id, actor_type, actor_id, actor_email, actor_ip_address,
			action, action_category,
			resource_type, resource_id,
			status, error_message,
			http_method, http_path, http_status_code, user_agent,
			old_values, new_values,
			metadata, description,
			request_id, correlation_id,
			created_at
		FROM audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Add filter conditions
	if filter.ActorType != nil {
		query += fmt.Sprintf(" AND actor_type = $%d", argPos)
		args = append(args, *filter.ActorType)
		argPos++
	}

	if filter.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id = $%d", argPos)
		args = append(args, *filter.ActorID)
		argPos++
	}

	if filter.ActorEmail != nil {
		query += fmt.Sprintf(" AND actor_email = $%d", argPos)
		args = append(args, *filter.ActorEmail)
		argPos++
	}

	if filter.ActorIPAddress != nil {
		query += fmt.Sprintf(" AND actor_ip_address = $%d", argPos)
		args = append(args, *filter.ActorIPAddress)
		argPos++
	}

	if filter.Action != nil {
		query += fmt.Sprintf(" AND action = $%d", argPos)
		args = append(args, *filter.Action)
		argPos++
	}

	if filter.ActionCategory != nil {
		query += fmt.Sprintf(" AND action_category = $%d", argPos)
		args = append(args, *filter.ActionCategory)
		argPos++
	}

	if filter.ResourceType != nil {
		query += fmt.Sprintf(" AND resource_type = $%d", argPos)
		args = append(args, *filter.ResourceType)
		argPos++
	}

	if filter.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argPos)
		args = append(args, *filter.ResourceID)
		argPos++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filter.Status)
		argPos++
	}

	if filter.RequestID != nil {
		query += fmt.Sprintf(" AND request_id = $%d", argPos)
		args = append(args, *filter.RequestID)
		argPos++
	}

	if filter.CorrelationID != nil {
		query += fmt.Sprintf(" AND correlation_id = $%d", argPos)
		args = append(args, *filter.CorrelationID)
		argPos++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *filter.StartTime)
		argPos++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *filter.EndTime)
		argPos++
	}

	// Add sorting
	sortBy := "created_at"
	if filter.SortBy != "" {
		// Validate sort field to prevent SQL injection
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

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	} else {
		// Default limit to prevent excessive data retrieval
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, 100)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
		argPos++
	}

	// Execute query
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	logs := []*model.AuditLog{}
	for rows.Next() {
		log := &model.AuditLog{}
		err := rows.Scan(
			&log.ID,
			&log.ActorType,
			&log.ActorID,
			&log.ActorEmail,
			&log.ActorIPAddress,
			&log.Action,
			&log.ActionCategory,
			&log.ResourceType,
			&log.ResourceID,
			&log.Status,
			&log.ErrorMessage,
			&log.HTTPMethod,
			&log.HTTPPath,
			&log.HTTPStatusCode,
			&log.UserAgent,
			&log.OldValues,
			&log.NewValues,
			&log.Metadata,
			&log.Description,
			&log.RequestID,
			&log.CorrelationID,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}

// Count returns the total number of audit logs matching the filter
func (r *AuditRepository) Count(filter AuditFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	// Add filter conditions (same as List)
	if filter.ActorType != nil {
		query += fmt.Sprintf(" AND actor_type = $%d", argPos)
		args = append(args, *filter.ActorType)
		argPos++
	}

	if filter.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id = $%d", argPos)
		args = append(args, *filter.ActorID)
		argPos++
	}

	if filter.ActionCategory != nil {
		query += fmt.Sprintf(" AND action_category = $%d", argPos)
		args = append(args, *filter.ActionCategory)
		argPos++
	}

	if filter.ResourceType != nil {
		query += fmt.Sprintf(" AND resource_type = $%d", argPos)
		args = append(args, *filter.ResourceType)
		argPos++
	}

	if filter.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argPos)
		args = append(args, *filter.ResourceID)
		argPos++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filter.Status)
		argPos++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *filter.StartTime)
		argPos++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *filter.EndTime)
		argPos++
	}

	var count int64
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// GetByResource retrieves all audit logs for a specific resource
func (r *AuditRepository) GetByResource(resourceType, resourceID string, limit, offset int) ([]*model.AuditLog, error) {
	filter := AuditFilter{
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Limit:        limit,
		Offset:       offset,
	}
	return r.List(filter)
}

// GetByActor retrieves all audit logs for a specific actor
func (r *AuditRepository) GetByActor(actorType model.ActorType, actorID string, limit, offset int) ([]*model.AuditLog, error) {
	filter := AuditFilter{
		ActorType: &actorType,
		ActorID:   &actorID,
		Limit:     limit,
		Offset:    offset,
	}
	return r.List(filter)
}

// GetByRequestID retrieves all audit logs for a specific request (for tracing)
func (r *AuditRepository) GetByRequestID(requestID string) ([]*model.AuditLog, error) {
	filter := AuditFilter{
		RequestID: &requestID,
	}
	return r.List(filter)
}

// GetRecentFailures retrieves recent failed operations for monitoring
func (r *AuditRepository) GetRecentFailures(hours int, limit int) ([]*model.AuditLog, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	failedStatus := model.AuditStatusFailed

	filter := AuditFilter{
		Status:    &failedStatus,
		StartTime: &startTime,
		Limit:     limit,
	}
	return r.List(filter)
}

// Helper function to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
