# Database Package

This package provides PostgreSQL database connection pool management with health checks, graceful shutdown, and transaction support.

## Features

- ✅ Connection pool management with configurable limits
- ✅ Automatic health checks
- ✅ Graceful shutdown handling
- ✅ Transaction support with automatic rollback
- ✅ Connection retry logic for Docker environments
- ✅ Connection pool statistics and monitoring
- ✅ Migration version checking

## Usage

### Basic Connection

```go
package main

import (
    "context"
    "github.com/hxuan190/stable_payment_gateway/internal/config"
    "github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }

    // Create database connection
    db, err := database.New(&cfg.Database)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Use the database
    // ...
}
```

### Health Check

```go
ctx := context.Background()
if err := db.HealthCheck(ctx); err != nil {
    log.Printf("Database health check failed: %v", err)
}
```

### Wait for Connection (Docker)

Useful when starting services in Docker where database might not be ready immediately:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := db.WaitForConnection(ctx, 5); err != nil {
    log.Fatalf("Database connection failed: %v", err)
}
```

### Transactions

#### Manual Transaction

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback() // Rollback if not committed

// Execute queries
_, err = tx.ExecContext(ctx, "INSERT INTO ...")
if err != nil {
    return err
}

// Commit transaction
return tx.Commit()
```

#### Automatic Transaction (Recommended)

```go
err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
    // Execute queries - automatic rollback on error
    _, err := tx.ExecContext(ctx, "INSERT INTO merchants ...")
    if err != nil {
        return err
    }

    _, err = tx.ExecContext(ctx, "INSERT INTO balances ...")
    if err != nil {
        return err
    }

    // Automatic commit if no error
    return nil
})
```

### Connection Pool Statistics

```go
stats := db.GetStats()
log.Printf("Open connections: %d", stats.OpenConnections)
log.Printf("In use: %d", stats.InUse)
log.Printf("Idle: %d", stats.Idle)

// Or use the built-in logger
db.LogPoolStats()
```

### Check Migration Version

```go
version, err := db.CheckMigrationVersion(ctx)
if err != nil {
    log.Printf("Failed to check migration version: %v", err)
}
log.Printf("Current migration version: %d", version)
```

## Configuration

The database package uses the `DatabaseConfig` struct from the config package:

```go
type DatabaseConfig struct {
    Host         string  // Database host
    Port         int     // Database port
    User         string  // Database user
    Password     string  // Database password
    Database     string  // Database name
    MaxOpenConns int     // Maximum open connections
    MaxIdleConns int     // Maximum idle connections
    SSLMode      string  // SSL mode: disable, require, verify-ca, verify-full
}
```

### Environment Variables

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=payment_gateway
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_SSL_MODE=disable  # Use 'require' or higher in production
```

## Connection Pool Tuning

### Recommended Settings

- **Development**: MaxOpenConns=10, MaxIdleConns=2
- **Staging**: MaxOpenConns=25, MaxIdleConns=5
- **Production**: MaxOpenConns=50-100, MaxIdleConns=10-20

### Formula for MaxOpenConns

```
MaxOpenConns = (Number of CPU cores × 2) + Disk spindles
```

For cloud databases (AWS RDS, etc):
```
MaxOpenConns = min(100, Available database connections / Number of app instances)
```

## Best Practices

### 1. Always Close Resources

```go
db, err := database.New(&cfg.Database)
if err != nil {
    return err
}
defer db.Close()
```

### 2. Use Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := db.QueryRowContext(ctx, "SELECT ...").Scan(&result)
```

### 3. Use Transactions for Multiple Operations

```go
// Good: Use transaction
err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
    // Multiple operations
    return nil
})

// Bad: Multiple separate queries (not atomic)
db.ExecContext(ctx, "INSERT ...")
db.ExecContext(ctx, "UPDATE ...")
```

### 4. Monitor Connection Pool

```go
// Log stats periodically
ticker := time.NewTicker(5 * time.Minute)
go func() {
    for range ticker.C {
        db.LogPoolStats()
    }
}()
```

### 5. Handle Connection Errors Gracefully

```go
var result int
err := db.QueryRowContext(ctx, "SELECT ...").Scan(&result)
if err == sql.ErrNoRows {
    // Handle no results
    return nil, ErrNotFound
} else if err != nil {
    // Handle connection or query error
    return nil, fmt.Errorf("database query failed: %w", err)
}
```

## Testing

Run the database tests (requires PostgreSQL):

```bash
# Start PostgreSQL with Docker
docker-compose up -d postgres

# Run tests
go test ./internal/pkg/database -v

# Run tests with coverage
go test ./internal/pkg/database -cover
```

## Troubleshooting

### Connection Refused

```
Error: failed to ping database: dial tcp [::1]:5432: connect: connection refused
```

**Solution**: Ensure PostgreSQL is running and accessible:

```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Start PostgreSQL
docker-compose up -d postgres

# Check logs
docker-compose logs postgres
```

### Too Many Connections

```
Error: pq: sorry, too many clients already
```

**Solution**: Reduce `MaxOpenConns` or increase PostgreSQL `max_connections`:

```sql
-- Check current connections
SELECT count(*) FROM pg_stat_activity;

-- Check max connections
SHOW max_connections;

-- Update postgresql.conf
max_connections = 200
```

### Slow Queries

```
Context deadline exceeded
```

**Solution**:
1. Add indexes to frequently queried columns
2. Analyze slow query logs
3. Increase context timeout if query is legitimately slow
4. Optimize the query

## Migration Management

Use the migration script to manage database schema:

```bash
# Run all pending migrations
./scripts/migrate.sh up

# Rollback last migration
./scripts/migrate.sh down

# Check current version
./scripts/migrate.sh version

# Create new migration
./scripts/migrate.sh create add_new_table
```

## Security Considerations

1. **Never log passwords**: The connection string is not logged
2. **Use SSL in production**: Set `DB_SSL_MODE=require` or higher
3. **Use prepared statements**: Prevents SQL injection (repository layer handles this)
4. **Limit connection lifetime**: Automatic with `SetConnMaxLifetime()`
5. **Rotate database credentials**: Update environment variables and restart

## Performance Monitoring

### Key Metrics to Monitor

- **OpenConnections**: Should stay below `MaxOpenConns`
- **WaitCount**: High values indicate connection pool exhaustion
- **WaitDuration**: High values indicate slow queries
- **MaxIdleClosed**: Indicates idle connection cleanup

### Example Monitoring Query

```sql
-- Active connections by state
SELECT state, count(*)
FROM pg_stat_activity
WHERE datname = 'payment_gateway'
GROUP BY state;

-- Long-running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 seconds'
ORDER BY duration DESC;
```

## Related Documentation

- [PostgreSQL Connection Pooling](https://www.postgresql.org/docs/current/runtime-config-connection.html)
- [Go database/sql Package](https://pkg.go.dev/database/sql)
- [Migration Guide](../../migrations/README.md)
- [Configuration Guide](../../config/README.md)
