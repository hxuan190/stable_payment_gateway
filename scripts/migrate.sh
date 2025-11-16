#!/bin/bash

# Database Migration Script
# Usage: ./scripts/migrate.sh [up|down|version|force|create]

set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | xargs)
fi

# Default values
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-payment_gateway}
DB_SSL_MODE=${DB_SSL_MODE:-disable}

# Build database URL
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSL_MODE}"

# Migration directory
MIGRATIONS_DIR="./migrations"

# Check if migrate tool is installed
if ! command -v migrate &> /dev/null; then
    echo "Error: 'migrate' tool not found"
    echo "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    exit 1
fi

# Get command
COMMAND=${1:-help}

case $COMMAND in
    up)
        echo "Running all pending migrations..."
        migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" up
        echo "✓ Migrations completed successfully"
        ;;

    down)
        echo "Rolling back last migration..."
        migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" down 1
        echo "✓ Rollback completed successfully"
        ;;

    version)
        echo "Current migration version:"
        migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" version
        ;;

    force)
        if [ -z "$2" ]; then
            echo "Error: Version number required"
            echo "Usage: ./scripts/migrate.sh force VERSION"
            exit 1
        fi
        echo "Forcing version to $2..."
        migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" force $2
        echo "✓ Version forced successfully"
        ;;

    create)
        if [ -z "$2" ]; then
            echo "Error: Migration name required"
            echo "Usage: ./scripts/migrate.sh create MIGRATION_NAME"
            exit 1
        fi
        TIMESTAMP=$(date +%s)
        MIGRATION_NAME=$(echo "$2" | tr '[:upper:]' '[:lower:]' | tr ' ' '_')

        # Find next migration number
        LAST_MIGRATION=$(ls -1 $MIGRATIONS_DIR | grep -E '^[0-9]+' | tail -1 | cut -d_ -f1)
        if [ -z "$LAST_MIGRATION" ]; then
            NEXT_NUM="001"
        else
            NEXT_NUM=$(printf "%03d" $((10#$LAST_MIGRATION + 1)))
        fi

        UP_FILE="${MIGRATIONS_DIR}/${NEXT_NUM}_${MIGRATION_NAME}.up.sql"
        DOWN_FILE="${MIGRATIONS_DIR}/${NEXT_NUM}_${MIGRATION_NAME}.down.sql"

        touch "$UP_FILE"
        touch "$DOWN_FILE"

        echo "-- Add your migration SQL here" > "$UP_FILE"
        echo "-- Add your rollback SQL here" > "$DOWN_FILE"

        echo "✓ Created migration files:"
        echo "  - $UP_FILE"
        echo "  - $DOWN_FILE"
        ;;

    drop)
        echo "WARNING: This will drop the entire database!"
        read -p "Are you sure? (yes/no): " CONFIRM
        if [ "$CONFIRM" = "yes" ]; then
            migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" drop
            echo "✓ Database dropped successfully"
        else
            echo "Aborted"
        fi
        ;;

    reset)
        echo "WARNING: This will reset the database (down all migrations then up again)!"
        read -p "Are you sure? (yes/no): " CONFIRM
        if [ "$CONFIRM" = "yes" ]; then
            echo "Rolling back all migrations..."
            migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" down -all || true
            echo "Running all migrations..."
            migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" up
            echo "✓ Database reset successfully"
        else
            echo "Aborted"
        fi
        ;;

    help|*)
        echo "Database Migration Helper"
        echo ""
        echo "Usage: ./scripts/migrate.sh COMMAND [OPTIONS]"
        echo ""
        echo "Commands:"
        echo "  up              Run all pending migrations"
        echo "  down            Rollback last migration"
        echo "  version         Show current migration version"
        echo "  force VERSION   Force set migration version (use with caution)"
        echo "  create NAME     Create new migration files"
        echo "  drop            Drop entire database (requires confirmation)"
        echo "  reset           Reset database (down all, then up all)"
        echo "  help            Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  DB_HOST         Database host (default: localhost)"
        echo "  DB_PORT         Database port (default: 5432)"
        echo "  DB_USER         Database user (default: postgres)"
        echo "  DB_PASSWORD     Database password (default: postgres)"
        echo "  DB_NAME         Database name (default: payment_gateway)"
        echo "  DB_SSL_MODE     SSL mode (default: disable)"
        echo ""
        echo "Examples:"
        echo "  ./scripts/migrate.sh up"
        echo "  ./scripts/migrate.sh down"
        echo "  ./scripts/migrate.sh create add_new_table"
        echo "  ./scripts/migrate.sh version"
        ;;
esac
