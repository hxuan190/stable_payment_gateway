package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/config"
)

// TestNewPostgresDB tests creating a new database connection pool
func TestNewPostgresDB(t *testing.T) {
	// This test requires a running PostgreSQL instance
	// Skip if DB_HOST is not set
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Expected database connection, got nil")
	}

	// Verify connection pool configuration
	stats := db.GetStats()
	if stats.MaxOpenConnections != cfg.MaxOpenConns {
		t.Errorf("Expected MaxOpenConnections=%d, got %d", cfg.MaxOpenConns, stats.MaxOpenConnections)
	}
}

// TestHealthCheck tests the database health check
func TestHealthCheck(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.HealthCheck(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

// TestHealthCheckWithTimeout tests health check with context timeout
func TestHealthCheckWithTimeout(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.HealthCheck(ctx); err != nil {
		t.Errorf("Health check with timeout failed: %v", err)
	}
}

// TestWithTransaction tests transaction handling
func TestWithTransaction(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test successful transaction
	err = db.WithTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "SELECT 1")
		return err
	})

	if err != nil {
		t.Errorf("Transaction should succeed, got error: %v", err)
	}
}

// TestWithTransactionRollback tests transaction rollback on error
func TestWithTransactionRollback(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test transaction rollback
	err = db.WithTransaction(ctx, func(tx *sql.Tx) error {
		return sql.ErrConnDone // Simulate error
	})

	if err == nil {
		t.Error("Transaction should fail and rollback")
	}
}

// TestGetStats tests retrieving connection pool statistics
func TestGetStats(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer db.Close()

	stats := db.GetStats()
	if stats.MaxOpenConnections != cfg.MaxOpenConns {
		t.Errorf("Expected MaxOpenConnections=%d, got %d", cfg.MaxOpenConns, stats.MaxOpenConnections)
	}
}

// TestClose tests graceful database connection closure
func TestClose(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Errorf("Close should not return error: %v", err)
	}

	// After closing, health check should fail
	ctx := context.Background()
	if err := db.HealthCheck(ctx); err == nil {
		t.Error("Health check should fail after closing database")
	}
}

// TestWaitForConnection tests waiting for database connection
func TestWaitForConnection(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.WaitForConnection(ctx, 3); err != nil {
		t.Errorf("WaitForConnection should succeed: %v", err)
	}
}

// TestWaitForConnectionTimeout tests wait timeout
func TestWaitForConnectionTimeout(t *testing.T) {
	// Use invalid host to simulate timeout
	cfg := &config.DatabaseConfig{
		Host:         "invalid-host-that-does-not-exist",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		SSLMode:      "disable",
	}

	db, err := New(cfg)
	if err != nil {
		// This is expected to fail, skip test
		t.Skipf("Expected to fail, skipping: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = db.WaitForConnection(ctx, 5)
	if err == nil {
		t.Error("WaitForConnection should timeout with invalid host")
	}
}
