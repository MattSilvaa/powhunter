#!/bin/bash
set -e

# Color codes for better output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database configuration (should match setup script)
DB_ADMIN_USER=${DB_ADMIN_USER:-postgres}
DB_ADMIN_PASSWORD=${DB_ADMIN_PASSWORD:-postgres}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=powhunter

# Application user configuration
APP_DB_USER=${APP_DB_USER:-$USER}

# Options
DROP_USER=${DROP_USER:-true}   # Set to false to preserve the user
FORCE=${FORCE:-false}          # Set to true to skip confirmation
ENVIRONMENT=${ENVIRONMENT:-${NODE_ENV:-development}}

echo -e "${RED}💀 Database Teardown Script${NC}"
echo "Environment: $ENVIRONMENT"
echo "Database: $DB_NAME"
echo "App User: $APP_DB_USER"
echo "Host: $DB_HOST:$DB_PORT"
echo "Drop User: $DROP_USER"
echo ""

# Production safety check
if [ "$ENVIRONMENT" = "production" ]; then
    echo -e "${RED}🚨 PRODUCTION ENVIRONMENT DETECTED!${NC}"
    echo -e "${RED}This script will permanently delete data in production!${NC}"
    echo ""
    echo "If you really want to proceed in production, you must:"
    echo "1. Set FORCE=true"
    echo "2. Type 'DELETE_PRODUCTION_DATA' when prompted"
    echo ""
    
    if [ "$FORCE" != "true" ]; then
        echo -e "${RED}ERROR: FORCE=true required for production${NC}"
        exit 1
    fi
    
    echo -n "Type 'DELETE_PRODUCTION_DATA' to confirm: "
    read confirmation
    
    if [ "$confirmation" != "DELETE_PRODUCTION_DATA" ]; then
        echo -e "${RED}Confirmation failed. Exiting safely.${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}⚠️  Proceeding with production database deletion...${NC}"
    echo ""
elif [ "$ENVIRONMENT" = "development" ]; then
    echo -e "${YELLOW}⚠️  Development environment detected${NC}"
    if [ "$FORCE" != "true" ]; then
        echo ""
        echo -e "${YELLOW}This will permanently delete:${NC}"
        echo "  • Database: $DB_NAME"
        echo "  • All tables and data"
        if [ "$DROP_USER" = "true" ]; then
            echo "  • User: $APP_DB_USER"
        fi
        echo ""
        echo -n "Are you sure? (yes/no): "
        read confirmation
        
        if [ "$confirmation" != "yes" ]; then
            echo "Operation cancelled."
            exit 0
        fi
    fi
fi

# Set up authentication for admin user
if [ -n "$DB_ADMIN_PASSWORD" ]; then
    export PGPASSWORD=$DB_ADMIN_PASSWORD
    echo "Using password authentication for admin user '$DB_ADMIN_USER'"
else
    echo "Using peer authentication for admin user '$DB_ADMIN_USER'"
fi
echo ""

# Test PostgreSQL connectivity
echo "Testing PostgreSQL connectivity..."
if ! pg_isready -h $DB_HOST -p $DB_PORT -U $DB_ADMIN_USER >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT${NC}"
    exit 1
fi
echo -e "${GREEN}✅ PostgreSQL connectivity confirmed${NC}"
echo ""

# Function to run SQL commands as admin
run_sql_as_admin() {
    local sql="$1"
    local database="${2:-postgres}"
    local result
    
    result=$(psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d $database -c "$sql" 2>&1)
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        echo -e "${RED}ERROR: Failed to execute SQL command${NC}"
        echo "Database: $database"
        echo "Command: $sql"
        echo "Details: $result"
        return 1
    fi
    
    echo "$result"
}

# Function to check if database exists
database_exists() {
    local db_name="$1"
    local count
    
    count=$(run_sql_as_admin "SELECT COUNT(*) FROM pg_database WHERE datname='$db_name';" | grep -o '[0-9]*' | head -1)
    
    if [ "$count" = "1" ]; then
        return 0  # Database exists
    else
        return 1  # Database doesn't exist
    fi
}

# Function to check if user exists
user_exists() {
    local username="$1"
    local count
    
    count=$(run_sql_as_admin "SELECT COUNT(*) FROM pg_catalog.pg_roles WHERE rolname='$username';" | grep -o '[0-9]*' | head -1)
    
    if [ "$count" = "1" ]; then
        return 0  # User exists
    else
        return 1  # User doesn't exist
    fi
}

# Function to terminate active connections to database
terminate_connections() {
    local db_name="$1"
    echo "Terminating active connections to database '$db_name'..."
    
    # PostgreSQL 9.2+ syntax
    run_sql_as_admin "
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = '$db_name'
        AND pid <> pg_backend_pid();
    " >/dev/null 2>&1 || true
    
    echo -e "${GREEN}    ✅ Connections terminated${NC}"
}

# Check if database exists before attempting to drop
echo "Checking if database '$DB_NAME' exists..."
if ! database_exists "$DB_NAME"; then
    echo -e "${YELLOW}ℹ️  Database '$DB_NAME' does not exist${NC}"
    DB_EXISTS=false
else
    echo -e "${YELLOW}⚠️  Database '$DB_NAME' exists and will be dropped${NC}"
    DB_EXISTS=true
fi

# Check if user exists before attempting to drop
echo "Checking if user '$APP_DB_USER' exists..."
if ! user_exists "$APP_DB_USER"; then
    echo -e "${YELLOW}ℹ️  User '$APP_DB_USER' does not exist${NC}"
    USER_EXISTS=false
else
    if [ "$DROP_USER" = "true" ]; then
        echo -e "${YELLOW}⚠️  User '$APP_DB_USER' exists and will be dropped${NC}"
    else
        echo -e "${YELLOW}ℹ️  User '$APP_DB_USER' exists but will be preserved${NC}"
    fi
    USER_EXISTS=true
fi
echo ""

# Drop database if it exists
if [ "$DB_EXISTS" = "true" ]; then
    echo -e "${RED}🗑️  Dropping database '$DB_NAME'...${NC}"
    
    # Terminate active connections first
    terminate_connections "$DB_NAME"
    
    # Drop the database
    if dropdb -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT $DB_NAME; then
        echo -e "${GREEN}✅ Database '$DB_NAME' dropped successfully${NC}"
    else
        echo -e "${RED}ERROR: Failed to drop database '$DB_NAME'${NC}"
        echo "This might be due to active connections. Try again in a moment."
        exit 1
    fi
else
    echo -e "${BLUE}ℹ️  No database to drop${NC}"
fi

# Drop user if requested and exists
if [ "$DROP_USER" = "true" ] && [ "$USER_EXISTS" = "true" ]; then
    echo ""
    echo -e "${RED}🗑️  Dropping user '$APP_DB_USER'...${NC}"
    
    if run_sql_as_admin "DROP USER $APP_DB_USER;" >/dev/null; then
        echo -e "${GREEN}✅ User '$APP_DB_USER' dropped successfully${NC}"
    else
        echo -e "${RED}ERROR: Failed to drop user '$APP_DB_USER'${NC}"
        echo "The user might own objects in other databases."
        exit 1
    fi
elif [ "$DROP_USER" = "true" ]; then
    echo -e "${BLUE}ℹ️  No user to drop${NC}"
fi

# Final summary
echo ""
echo -e "${GREEN}🧹 Teardown complete!${NC}"
echo ""
echo -e "${BLUE}📋 Summary:${NC}"
if [ "$DB_EXISTS" = "true" ]; then
    echo "  ✅ Database '$DB_NAME' dropped"
else
    echo "  ➖ Database '$DB_NAME' was not present"
fi

if [ "$DROP_USER" = "true" ]; then
    if [ "$USER_EXISTS" = "true" ]; then
        echo "  ✅ User '$APP_DB_USER' dropped"
    else
        echo "  ➖ User '$APP_DB_USER' was not present"
    fi
else
    echo "  ➖ User '$APP_DB_USER' preserved"
fi

echo ""
echo -e "${BLUE}🔄 Next Steps:${NC}"
echo "  1. Run ./setup_db.sh to recreate the database"
echo "  2. Run ./migrate.sh to apply migrations"
echo "  3. Restore any backup data if needed"
echo ""

# Clean up environment
unset PGPASSWORD