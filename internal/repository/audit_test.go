package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupAuditLogs removes all audit logs from the test database
func cleanupAuditLogs(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM audit_logs")
	require.NoError(t, err, "Failed to cleanup audit_logs table")
}

// createTestAuditLog creates a test audit log with default values
func createTestAuditLog() *model.AuditLog {
	return &model.AuditLog{
		ID:             uuid.New().String(),
		ActorType:      model.ActorTypeSystem,
		Action:         "test_action",
		ActionCategory: model.ActionCategorySystem,
		ResourceType:   "payment",
		ResourceID:     uuid.New().String(),
		Status:         model.AuditStatusSuccess,
		Metadata:       model.JSONBMap{},
	}
}

func TestAuditRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	t.Run("Success - Create audit log", func(t *testing.T) {
		log := createTestAuditLog()

		err := repo.Create(log)
		require.NoError(t, err)

		// Verify the audit log was created
		assert.NotEmpty(t, log.ID)
		assert.False(t, log.CreatedAt.IsZero())
	})

	t.Run("Success - Create with actor details", func(t *testing.T) {
		log := createTestAuditLog()
		log.ActorType = model.ActorTypeMerchant
		log.ActorID = sql.NullString{String: uuid.New().String(), Valid: true}
		log.ActorEmail = sql.NullString{String: "merchant@example.com", Valid: true}
		log.ActorIPAddress = sql.NullString{String: "192.168.1.1", Valid: true}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotEmpty(t, log.ID)

		// Verify we can retrieve it
		retrieved, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.Equal(t, log.ActorEmail.String, retrieved.ActorEmail.String)
		assert.Equal(t, log.ActorIPAddress.String, retrieved.ActorIPAddress.String)
	})

	t.Run("Success - Create with HTTP details", func(t *testing.T) {
		log := createTestAuditLog()
		log.HTTPMethod = sql.NullString{String: "POST", Valid: true}
		log.HTTPPath = sql.NullString{String: "/api/v1/payments", Valid: true}
		log.HTTPStatusCode = sql.NullInt32{Int32: 201, Valid: true}
		log.UserAgent = sql.NullString{String: "Mozilla/5.0", Valid: true}

		err := repo.Create(log)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.Equal(t, "POST", retrieved.HTTPMethod.String)
		assert.Equal(t, "/api/v1/payments", retrieved.HTTPPath.String)
		assert.Equal(t, int32(201), retrieved.HTTPStatusCode.Int32)
	})

	t.Run("Success - Create with metadata and description", func(t *testing.T) {
		log := createTestAuditLog()
		log.Metadata = model.JSONBMap{
			"key1": "value1",
			"key2": 123,
		}
		log.Description = sql.NullString{String: "Test description", Valid: true}

		err := repo.Create(log)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test description", retrieved.Description.String)
		assert.NotNil(t, retrieved.Metadata)
	})

	t.Run("Success - Create with old and new values", func(t *testing.T) {
		log := createTestAuditLog()
		log.OldValues = model.JSONBMap{"status": "pending"}
		log.NewValues = model.JSONBMap{"status": "completed"}

		err := repo.Create(log)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.OldValues)
		assert.NotNil(t, retrieved.NewValues)
	})

	t.Run("Success - Create with request and correlation IDs", func(t *testing.T) {
		log := createTestAuditLog()
		requestID := uuid.New().String()
		correlationID := uuid.New().String()
		log.RequestID = sql.NullString{String: requestID, Valid: true}
		log.CorrelationID = sql.NullString{String: correlationID, Valid: true}

		err := repo.Create(log)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.Equal(t, requestID, retrieved.RequestID.String)
		assert.Equal(t, correlationID, retrieved.CorrelationID.String)
	})

	t.Run("Success - Create failed audit log", func(t *testing.T) {
		log := createTestAuditLog()
		log.Status = model.AuditStatusFailed
		log.ErrorMessage = sql.NullString{String: "Test error", Valid: true}

		err := repo.Create(log)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.Equal(t, model.AuditStatusFailed, retrieved.Status)
		assert.Equal(t, "Test error", retrieved.ErrorMessage.String)
	})

	t.Run("Error - Nil audit log", func(t *testing.T) {
		err := repo.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}

func TestAuditRepository_CreateBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	t.Run("Success - Create multiple audit logs", func(t *testing.T) {
		logs := []*model.AuditLog{
			createTestAuditLog(),
			createTestAuditLog(),
			createTestAuditLog(),
		}

		err := repo.CreateBatch(logs)
		require.NoError(t, err)

		// Verify all logs were created
		for _, log := range logs {
			assert.NotEmpty(t, log.ID)
			assert.False(t, log.CreatedAt.IsZero())
		}

		// Verify we can retrieve them
		for _, log := range logs {
			retrieved, err := repo.GetByID(log.ID)
			require.NoError(t, err)
			assert.Equal(t, log.ID, retrieved.ID)
		}
	})

	t.Run("Error - Empty logs array", func(t *testing.T) {
		err := repo.CreateBatch([]*model.AuditLog{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestAuditRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	t.Run("Success - Get existing audit log", func(t *testing.T) {
		// Create an audit log first
		log := createTestAuditLog()
		err := repo.Create(log)
		require.NoError(t, err)

		// Retrieve the audit log
		retrieved, err := repo.GetByID(log.ID)
		require.NoError(t, err)

		assert.Equal(t, log.ID, retrieved.ID)
		assert.Equal(t, log.ActorType, retrieved.ActorType)
		assert.Equal(t, log.Action, retrieved.Action)
		assert.Equal(t, log.ActionCategory, retrieved.ActionCategory)
		assert.Equal(t, log.ResourceType, retrieved.ResourceType)
		assert.Equal(t, log.ResourceID, retrieved.ResourceID)
		assert.Equal(t, log.Status, retrieved.Status)
	})

	t.Run("Error - Audit log not found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		_, err := repo.GetByID(nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAuditLogNotFound)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		_, err := repo.GetByID("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestAuditRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	// Create test data
	merchantID := uuid.New().String()
	paymentID := uuid.New().String()

	logs := []*model.AuditLog{
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeMerchant,
			ActorID:        sql.NullString{String: merchantID, Valid: true},
			Action:         "payment_created",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     paymentID,
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeSystem,
			Action:         "payment_confirmed",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     paymentID,
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeAdmin,
			Action:         "kyc_approved",
			ActionCategory: model.ActionCategoryKYC,
			ResourceType:   "merchant",
			ResourceID:     merchantID,
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeAPI,
			Action:         "payout_requested",
			ActionCategory: model.ActionCategoryPayout,
			ResourceType:   "payout",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusFailed,
			ErrorMessage:   sql.NullString{String: "Insufficient balance", Valid: true},
			Metadata:       model.JSONBMap{},
		},
	}

	// Insert test data with small delays to ensure different timestamps
	for i, log := range logs {
		err := repo.Create(log)
		require.NoError(t, err)
		if i < len(logs)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	t.Run("Success - List all audit logs", func(t *testing.T) {
		filter := AuditFilter{
			Limit: 100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)
	})

	t.Run("Success - Filter by actor type", func(t *testing.T) {
		actorType := model.ActorTypeMerchant
		filter := AuditFilter{
			ActorType: &actorType,
			Limit:     100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 1)
		for _, log := range retrieved {
			assert.Equal(t, model.ActorTypeMerchant, log.ActorType)
		}
	})

	t.Run("Success - Filter by actor ID", func(t *testing.T) {
		filter := AuditFilter{
			ActorID: &merchantID,
			Limit:   100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 1)
		for _, log := range retrieved {
			assert.Equal(t, merchantID, log.ActorID.String)
		}
	})

	t.Run("Success - Filter by action category", func(t *testing.T) {
		category := model.ActionCategoryPayment
		filter := AuditFilter{
			ActionCategory: &category,
			Limit:          100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 2)
		for _, log := range retrieved {
			assert.Equal(t, model.ActionCategoryPayment, log.ActionCategory)
		}
	})

	t.Run("Success - Filter by resource", func(t *testing.T) {
		resourceType := "payment"
		filter := AuditFilter{
			ResourceType: &resourceType,
			ResourceID:   &paymentID,
			Limit:        100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 2)
		for _, log := range retrieved {
			assert.Equal(t, "payment", log.ResourceType)
			assert.Equal(t, paymentID, log.ResourceID)
		}
	})

	t.Run("Success - Filter by status", func(t *testing.T) {
		status := model.AuditStatusFailed
		filter := AuditFilter{
			Status: &status,
			Limit:  100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 1)
		for _, log := range retrieved {
			assert.Equal(t, model.AuditStatusFailed, log.Status)
		}
	})

	t.Run("Success - Filter by time range", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now().Add(1 * time.Hour)
		filter := AuditFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
			Limit:     100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)
	})

	t.Run("Success - Pagination", func(t *testing.T) {
		// First page
		filter := AuditFilter{
			Limit:  2,
			Offset: 0,
		}
		page1, err := repo.List(filter)
		require.NoError(t, err)
		assert.Len(t, page1, 2)

		// Second page
		filter.Offset = 2
		page2, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(page2), 2)

		// Verify different results
		if len(page2) > 0 {
			assert.NotEqual(t, page1[0].ID, page2[0].ID)
		}
	})

	t.Run("Success - Sort by created_at ASC", func(t *testing.T) {
		filter := AuditFilter{
			Limit:     100,
			SortBy:    "created_at",
			SortOrder: "ASC",
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)

		// Verify ascending order
		for i := 1; i < len(retrieved); i++ {
			assert.True(t, retrieved[i].CreatedAt.After(retrieved[i-1].CreatedAt) ||
				retrieved[i].CreatedAt.Equal(retrieved[i-1].CreatedAt))
		}
	})

	t.Run("Success - Sort by created_at DESC (default)", func(t *testing.T) {
		filter := AuditFilter{
			Limit: 100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)

		// Verify descending order (default)
		for i := 1; i < len(retrieved); i++ {
			assert.True(t, retrieved[i].CreatedAt.Before(retrieved[i-1].CreatedAt) ||
				retrieved[i].CreatedAt.Equal(retrieved[i-1].CreatedAt))
		}
	})

	t.Run("Success - Combined filters", func(t *testing.T) {
		category := model.ActionCategoryPayment
		status := model.AuditStatusSuccess
		filter := AuditFilter{
			ActionCategory: &category,
			Status:         &status,
			Limit:          100,
		}
		retrieved, err := repo.List(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 2)
		for _, log := range retrieved {
			assert.Equal(t, model.ActionCategoryPayment, log.ActionCategory)
			assert.Equal(t, model.AuditStatusSuccess, log.Status)
		}
	})
}

func TestAuditRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	// Create test data
	logs := []*model.AuditLog{
		createTestAuditLog(),
		createTestAuditLog(),
		createTestAuditLog(),
	}
	logs[0].Status = model.AuditStatusSuccess
	logs[1].Status = model.AuditStatusFailed
	logs[2].Status = model.AuditStatusSuccess

	for _, log := range logs {
		err := repo.Create(log)
		require.NoError(t, err)
	}

	t.Run("Success - Count all", func(t *testing.T) {
		count, err := repo.Count(AuditFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
	})

	t.Run("Success - Count by status", func(t *testing.T) {
		status := model.AuditStatusSuccess
		filter := AuditFilter{
			Status: &status,
		}
		count, err := repo.Count(filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
	})
}

func TestAuditRepository_GetByResource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	resourceID := uuid.New().String()
	logs := []*model.AuditLog{
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeSystem,
			Action:         "payment_created",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     resourceID,
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeSystem,
			Action:         "payment_confirmed",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     resourceID,
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
	}

	for _, log := range logs {
		err := repo.Create(log)
		require.NoError(t, err)
	}

	t.Run("Success - Get by resource", func(t *testing.T) {
		retrieved, err := repo.GetByResource("payment", resourceID, 100, 0)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
		for _, log := range retrieved {
			assert.Equal(t, "payment", log.ResourceType)
			assert.Equal(t, resourceID, log.ResourceID)
		}
	})
}

func TestAuditRepository_GetByActor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	actorID := uuid.New().String()
	logs := []*model.AuditLog{
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeMerchant,
			ActorID:        sql.NullString{String: actorID, Valid: true},
			Action:         "payment_created",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeMerchant,
			ActorID:        sql.NullString{String: actorID, Valid: true},
			Action:         "payout_requested",
			ActionCategory: model.ActionCategoryPayout,
			ResourceType:   "payout",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
	}

	for _, log := range logs {
		err := repo.Create(log)
		require.NoError(t, err)
	}

	t.Run("Success - Get by actor", func(t *testing.T) {
		retrieved, err := repo.GetByActor(model.ActorTypeMerchant, actorID, 100, 0)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
		for _, log := range retrieved {
			assert.Equal(t, model.ActorTypeMerchant, log.ActorType)
			assert.Equal(t, actorID, log.ActorID.String)
		}
	})
}

func TestAuditRepository_GetByRequestID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	requestID := uuid.New().String()
	logs := []*model.AuditLog{
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeAPI,
			Action:         "payment_created",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusSuccess,
			RequestID:      sql.NullString{String: requestID, Valid: true},
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeAPI,
			Action:         "validation_check",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusSuccess,
			RequestID:      sql.NullString{String: requestID, Valid: true},
			Metadata:       model.JSONBMap{},
		},
	}

	for _, log := range logs {
		err := repo.Create(log)
		require.NoError(t, err)
	}

	t.Run("Success - Get by request ID", func(t *testing.T) {
		retrieved, err := repo.GetByRequestID(requestID)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
		for _, log := range retrieved {
			assert.Equal(t, requestID, log.RequestID.String)
		}
	})
}

func TestAuditRepository_GetRecentFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupAuditLogs(t, db)

	repo := NewAuditRepository(db)

	logs := []*model.AuditLog{
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeSystem,
			Action:         "payment_failed",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusFailed,
			ErrorMessage:   sql.NullString{String: "Transaction timeout", Valid: true},
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeSystem,
			Action:         "webhook_error",
			ActionCategory: model.ActionCategoryWebhook,
			ResourceType:   "webhook",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusError,
			ErrorMessage:   sql.NullString{String: "Connection refused", Valid: true},
			Metadata:       model.JSONBMap{},
		},
		{
			ID:             uuid.New().String(),
			ActorType:      model.ActorTypeSystem,
			Action:         "payment_confirmed",
			ActionCategory: model.ActionCategoryPayment,
			ResourceType:   "payment",
			ResourceID:     uuid.New().String(),
			Status:         model.AuditStatusSuccess,
			Metadata:       model.JSONBMap{},
		},
	}

	for _, log := range logs {
		err := repo.Create(log)
		require.NoError(t, err)
	}

	t.Run("Success - Get recent failures", func(t *testing.T) {
		retrieved, err := repo.GetRecentFailures(24, 100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 1)
		for _, log := range retrieved {
			assert.True(t, log.IsFailed())
		}
	})
}
