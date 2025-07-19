#!/bin/bash
set -e

# Color codes for better output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database configuration (should match setup script)
DB_USER=${APP_DB_USER:-$USER}
DB_PASSWORD=${APP_DB_PASSWORD:-developer}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=powhunter

# Migration configuration
MIGRATION_DIR=${MIGRATION_DIR:-internal/db/migrations}
MIGRATION_COMMAND=${1:-up}  # Allow passing command as first argument

echo -e "${BLUE}🚀 Database Migration Runner${NC}"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo "Host: $DB_HOST:$DB_PORT"
echo "Migration Directory: $MIGRATION_DIR"
echo "Command: $MIGRATION_COMMAND"
echo ""

# Validate migration directory exists (running from server directory)
if [ ! -d "$MIGRATION_DIR" ]; then
    echo -e "${RED}ERROR: Migration directory '$MIGRATION_DIR' not found${NC}"
    echo "Current directory: $(pwd)"
    echo "Make sure you're running from the server directory"
    exit 1
fi

# Check if goose is installed
if ! command -v goose &> /dev/null; then
    echo -e "${RED}ERROR: goose is not installed or not in PATH${NC}"
    echo "Install it with: go install github.com/pressly/goose/v3/cmd/goose@latest"
    exit 1
fi

# Build connection string
CONNECTION_STRING="host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=disable"

# Test database connectivity first
echo "Testing database connectivity..."
if ! pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT${NC}"
    echo "Please ensure:"
    echo "  1. Database setup script has been run"
    echo "  2. PostgreSQL is running"
    echo "  3. User '$DB_USER' exists and has proper authentication"
    exit 1
fi

# Test actual database connection with credentials
echo "Verifying database access..."
export PGPASSWORD=$DB_PASSWORD
if ! psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Cannot authenticate with database${NC}"
    echo "Please check your credentials or run the database setup script first"
    exit 1
fi
unset PGPASSWORD

echo -e "${GREEN}✅ Database connectivity confirmed${NC}"
echo ""

# Count migration files
MIGRATION_COUNT=$(find "$MIGRATION_DIR" -name "*.sql" | wc -l)
echo "Found $MIGRATION_COUNT migration files in $MIGRATION_DIR"

if [ "$MIGRATION_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No migration files found${NC}"
    exit 0
fi
echo ""

# Show current migration status before running
echo "Current migration status:"
goose -dir "$MIGRATION_DIR" postgres "$CONNECTION_STRING" status 2>/dev/null || echo "No migrations applied yet"
echo ""

# Run migrations
echo -e "${BLUE}Running migrations...${NC}"
if goose -dir "$MIGRATION_DIR" postgres "$CONNECTION_STRING" "$MIGRATION_COMMAND"; then
    echo ""
    echo -e "${GREEN}✅ Migrations completed successfully!${NC}"
    
    # Show final status
    echo ""
    echo "Final migration status:"
    goose -dir "$MIGRATION_DIR" postgres "$CONNECTION_STRING" status
else
    echo ""
    echo -e "${RED}❌ Migration failed!${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}📋 Migration Summary:${NC}"
echo "  Command: $MIGRATION_COMMAND"
echo "  Database: $DB_NAME"
echo "  Migrations: $MIGRATION_COUNT files"
echo ""

# Show available commands
if [ "$MIGRATION_COMMAND" = "up" ]; then
    echo -e "${BLUE}💡 Other available commands:${NC}"
    echo "  ./migrate.sh status    - Show migration status"
    echo "  ./migrate.sh down      - Rollback last migration"
    echo "  ./migrate.sh reset     - Rollback all migrations"
    echo "  ./migrate.sh version   - Show current version"
fi