# Database Migrations

This directory contains database migration files for the Stablecoin Payment Gateway.

## Overview

We use [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations. Each migration has two files:
- `.up.sql` - Applies the migration
- `.down.sql` - Reverts the migration

## Migration Files

| Migration | Description |
|-----------|-------------|
| 001 | Create merchants table |
| 002 | Create payments table |
| 003 | Create payouts table |
| 004 | Create ledger_entries table (double-entry accounting) |
| 005 | Create merchant_balances table |
| 006 | Create audit_logs table |
| 007 | Create blockchain_transactions table |
| 008 | Create views and helper functions |

## Prerequisites

1. **Install golang-migrate CLI**:
   ```bash
   # macOS
   brew install golang-migrate

   # Linux
   curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.0/migrate.linux-amd64.tar.gz | tar xvz
   sudo mv migrate /usr/local/bin/

   # Or via Go
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

2. **Set up database connection**:
   ```bash
   export DATABASE_URL="postgres://user:password@localhost:5432/payment_gateway?sslmode=disable"
   ```

## Usage

### Using Makefile (Recommended)

The project includes a Makefile with migration commands:

```bash
# Run all migrations
make migrate-up

# Rollback all migrations
make migrate-down

# Rollback one migration
make migrate-down-one

# Create a new migration
make migrate-create NAME=add_new_table

# Check migration version
make migrate-version

# Force set version (use with caution!)
make migrate-force VERSION=1
```

### Using migrate CLI directly

```bash
# Apply all pending migrations
migrate -path migrations -database "postgres://user:password@localhost:5432/payment_gateway?sslmode=disable" up

# Rollback last migration
migrate -path migrations -database "postgres://user:password@localhost:5432/payment_gateway?sslmode=disable" down 1

# Rollback all migrations
migrate -path migrations -database "postgres://user:password@localhost:5432/payment_gateway?sslmode=disable" down

# Check current version
migrate -path migrations -database "postgres://user:password@localhost:5432/payment_gateway?sslmode=disable" version

# Create new migration
migrate create -ext sql -dir migrations -seq create_new_table
```

### Using Go code

```go
import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(databaseURL string) error {
    m, err := migrate.New(
        "file://migrations",
        databaseURL,
    )
    if err != nil {
        return err
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }

    return nil
}
```

## Best Practices

### Creating Migrations

1. **Always create paired migrations**: Every `.up.sql` must have a corresponding `.down.sql`
2. **Make migrations reversible**: Ensure down migrations properly undo the up migration
3. **Test both directions**: Test both `up` and `down` migrations before committing
4. **Keep migrations small**: One logical change per migration
5. **Use transactions**: Wrap migration in BEGIN/COMMIT when possible
6. **Add comments**: Document what each migration does and why

### Naming Conventions

- Use sequential numbers: `001`, `002`, `003`, etc.
- Use descriptive names: `create_merchants_table`, not `merchants`
- Use snake_case for file names

### Testing Migrations

Before committing a migration:

```bash
# 1. Apply the migration
make migrate-up

# 2. Verify it worked
psql $DATABASE_URL -c "\dt"  # List tables
psql $DATABASE_URL -c "\d merchants"  # Describe table

# 3. Rollback the migration
make migrate-down-one

# 4. Verify rollback worked
psql $DATABASE_URL -c "\dt"

# 5. Re-apply migration
make migrate-up
```

## Troubleshooting

### Migration is stuck / dirty

If a migration fails halfway through, the database might be in a "dirty" state:

```bash
# Check current version and dirty state
migrate -path migrations -database $DATABASE_URL version

# Force to a specific version (use with caution!)
migrate -path migrations -database $DATABASE_URL force VERSION

# Then fix the issue and re-run
migrate -path migrations -database $DATABASE_URL up
```

### Cannot rollback migration

If a down migration fails:

1. Manually inspect the database state
2. Manually fix any issues
3. Update the `.down.sql` file if needed
4. Test the rollback again

### Connection issues

Common connection string formats:

```bash
# Standard PostgreSQL
postgres://user:password@localhost:5432/dbname?sslmode=disable

# With SSL
postgres://user:password@localhost:5432/dbname?sslmode=require

# Unix socket
postgres://user:password@/dbname?host=/var/run/postgresql
```

## Schema Documentation

### Core Tables

**merchants**: Stores merchant information, KYC status, and API credentials
- Primary key: `id` (UUID)
- Unique constraints: `email`, `api_key`

**payments**: Tracks all payment requests and their lifecycle
- Primary key: `id` (UUID)
- Foreign key: `merchant_id` → merchants
- Unique constraint: `payment_reference`

**payouts**: Records merchant withdrawal requests
- Primary key: `id` (UUID)
- Foreign key: `merchant_id` → merchants

**ledger_entries**: Double-entry accounting ledger (append-only)
- Primary key: `id` (UUID)
- Foreign key: `merchant_id` → merchants (nullable)
- **IMPORTANT**: Immutable - no updates or deletes

**merchant_balances**: Cached balance state for quick lookups
- Primary key: `id` (UUID)
- Foreign key: `merchant_id` → merchants
- Unique constraint: `merchant_id`

**audit_logs**: Comprehensive audit trail (append-only)
- Primary key: `id` (UUID)
- **IMPORTANT**: Immutable - no updates or deletes

**blockchain_transactions**: Blockchain transaction details
- Primary key: `id` (UUID)
- Foreign key: `payment_id` → payments (nullable)
- Unique constraint: `tx_hash`

### Views

**active_merchants_with_balance**: Active merchants with current balance
**recent_payments_summary**: Recent payments with merchant details
**pending_payouts_for_review**: Pending payouts for admin review
**system_statistics**: System-wide statistics for dashboard
**unmatched_blockchain_transactions**: Unmatched blockchain transactions

### Functions

**calculate_merchant_balance_from_ledger**: Calculate balance from ledger (for reconciliation)
**get_merchant_payment_stats**: Get payment statistics for a merchant

## Production Considerations

### Before Running Migrations in Production

1. **Backup the database**:
   ```bash
   pg_dump -h localhost -U postgres payment_gateway > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **Test on staging first**: Always test migrations on staging environment

3. **Plan for downtime**: Some migrations may require brief downtime

4. **Have rollback plan**: Test down migrations beforehand

5. **Monitor during migration**: Watch for errors and performance issues

### Running Migrations in Production

```bash
# 1. Backup database
make db-backup

# 2. Check current version
make migrate-version

# 3. Run migrations
make migrate-up

# 4. Verify success
make migrate-version
psql $DATABASE_URL -c "SELECT COUNT(*) FROM merchants;"

# 5. If issues, rollback
make migrate-down-one
```

## Maintenance

### Adding a New Migration

```bash
# Create new migration files
make migrate-create NAME=add_new_feature

# Edit the generated files:
# migrations/00X_add_new_feature.up.sql
# migrations/00X_add_new_feature.down.sql

# Test locally
make migrate-up
make migrate-down-one
make migrate-up

# Commit to git
git add migrations/
git commit -m "Add migration: add_new_feature"
```

### Migration Checklist

- [ ] Migration file names follow convention
- [ ] Both `.up.sql` and `.down.sql` created
- [ ] Down migration properly reverses up migration
- [ ] Tested both up and down migrations locally
- [ ] Indexes added for foreign keys
- [ ] Constraints added where needed
- [ ] Comments added for documentation
- [ ] Migration is idempotent (can run multiple times safely)
- [ ] No sensitive data in migration files
- [ ] Reviewed by team member

## References

- [golang-migrate documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)
- [Project ARCHITECTURE.md](../ARCHITECTURE.md)
- [Project TECH_STACK_GOLANG.md](../TECH_STACK_GOLANG.md)
