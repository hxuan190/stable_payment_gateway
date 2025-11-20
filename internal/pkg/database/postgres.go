package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/crypto"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// PostgresDB wraps the database connection pool
type PostgresDB struct {
	*sql.DB
	gormDB *gorm.DB // GORM instance for models with encryption
	config *config.DatabaseConfig
	cipher *crypto.AES256GCM // Encryption cipher for PII data
}

// New creates a new PostgreSQL database connection pool
func New(cfg *config.DatabaseConfig) (*PostgresDB, error) {
	// CRITICAL SECURITY: Initialize encryption cipher from environment variable
	// This MUST be configured before any PII data is stored
	encryptionKey := os.Getenv("ENCRYPTION_MASTER_KEY")
	if encryptionKey == "" {
		log.Fatal("FATAL: ENCRYPTION_MASTER_KEY environment variable is not set. PII encryption is mandatory. Application cannot start.")
	}

	// Validate key length (must be exactly 32 bytes for AES-256)
	keyBytes := []byte(encryptionKey)
	if len(keyBytes) != 32 {
		log.Fatalf("FATAL: ENCRYPTION_MASTER_KEY must be exactly 32 bytes for AES-256, got %d bytes. Application cannot start.", len(keyBytes))
	}

	// Initialize AES-256-GCM cipher
	cipher, err := crypto.NewAES256GCM(keyBytes)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize encryption cipher: %v. Application cannot start.", err)
	}

	logger.Info("Encryption cipher initialized successfully for PII protection", logger.Fields{
		"algorithm": "AES-256-GCM",
		"key_size":  len(keyBytes),
	})

	// Build connection string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 5)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize GORM instance for models with encryption support
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}

	// Initialize and register encryption plugin
	encryptionPlugin := crypto.NewEncryptionPlugin(cipher)
	if err := gormDB.Use(encryptionPlugin); err != nil {
		log.Fatalf("FATAL: Failed to register encryption plugin: %v. Application cannot start.", err)
	}

	logger.Info("Encryption plugin registered successfully for GORM models", logger.Fields{
		"plugin_name": encryptionPlugin.Name(),
	})

	logger.Info("Database connection pool established", logger.Fields{
		"host":           cfg.Host,
		"port":           cfg.Port,
		"database":       cfg.Database,
		"max_open_conns": cfg.MaxOpenConns,
		"max_idle_conns": cfg.MaxIdleConns,
	})

	return &PostgresDB{
		DB:     db,
		gormDB: gormDB,
		config: cfg,
		cipher: cipher,
	}, nil
}

// HealthCheck verifies the database connection is healthy
func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	// Create context with timeout if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	// Ping database
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check if we can execute a simple query
	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("database query check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database returned unexpected result: %d", result)
	}

	return nil
}

// GetStats returns database connection pool statistics
func (db *PostgresDB) GetStats() sql.DBStats {
	return db.Stats()
}

// Close gracefully closes the database connection pool
func (db *PostgresDB) Close() error {
	logger.Info("Closing database connection pool")

	if err := db.DB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	logger.Info("Database connection pool closed successfully")
	return nil
}

// BeginTx starts a new database transaction
func (db *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (db *PostgresDB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Defer rollback in case of panic
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	// Execute function
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CheckMigrationVersion returns the current database migration version
func (db *PostgresDB) CheckMigrationVersion(ctx context.Context) (uint, error) {
	var version uint

	// Check if schema_migrations table exists
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'schema_migrations'
		)
	`).Scan(&exists)

	if err != nil {
		return 0, fmt.Errorf("failed to check migrations table: %w", err)
	}

	if !exists {
		return 0, nil
	}

	// Get current version
	err = db.QueryRowContext(ctx, `
		SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1
	`).Scan(&version)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, nil
}

// WaitForConnection waits for the database to become available
// This is useful during application startup when the database might not be ready yet
func (db *PostgresDB) WaitForConnection(ctx context.Context, maxRetries int) error {
	retryDelay := time.Second

	for i := 0; i < maxRetries; i++ {
		if err := db.HealthCheck(ctx); err == nil {
			return nil
		}

		logger.Warn("Database not ready, retrying...", logger.Fields{
			"attempt":     i + 1,
			"max_retries": maxRetries,
			"retry_in":    retryDelay.String(),
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			// Exponential backoff (max 10 seconds)
			retryDelay *= 2
			if retryDelay > 10*time.Second {
				retryDelay = 10 * time.Second
			}
		}
	}

	return fmt.Errorf("failed to connect to database after %d retries", maxRetries)
}

// LogPoolStats logs database connection pool statistics
// This is useful for monitoring and debugging connection pool issues
func (db *PostgresDB) LogPoolStats() {
	stats := db.GetStats()
	logger.Info("Database connection pool stats", logger.Fields{
		"open_connections":    stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration.String(),
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	})
}

// GetCipher returns the encryption cipher for PII data
// Repositories should use this to encrypt/decrypt sensitive fields
func (db *PostgresDB) GetCipher() *crypto.AES256GCM {
	return db.cipher
}

// GetGORM returns the GORM database instance with encryption plugin registered
// Use this for models tagged with encrypt:"true"
func (db *PostgresDB) GetGORM() *gorm.DB {
	return db.gormDB
}
