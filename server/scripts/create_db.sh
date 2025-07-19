#!/bin/bash
set -e

# Color codes for better output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database configuration
DB_ADMIN_USER=${DB_ADMIN_USER:-postgres}
DB_ADMIN_PASSWORD=${DB_ADMIN_PASSWORD:-postgres}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=powhunter

# Application user configuration
APP_DB_USER=${APP_DB_USER:-$USER}
APP_DB_PASSWORD=${APP_DB_PASSWORD:-developer}

# Detect environment
ENVIRONMENT=${ENVIRONMENT:-${NODE_ENV:-development}}

echo -e "${BLUE}🔧 PostgreSQL Database Setup for '$DB_NAME'${NC}"
echo "Environment: $ENVIRONMENT"
echo "Database: $DB_NAME"
echo "App User: $APP_DB_USER"
echo "Host: $DB_HOST:$DB_PORT"
echo ""

# Security warnings for production
if [ "$ENVIRONMENT" = "production" ]; then
    echo -e "${RED}🚨 PRODUCTION ENVIRONMENT DETECTED${NC}"
    
    if [ "$DB_ADMIN_PASSWORD" = "postgres" ]; then
        echo -e "${RED}ERROR: Default admin password detected in production!${NC}"
        echo "Set DB_ADMIN_PASSWORD environment variable to a secure value."
        exit 1
    fi
    
    if [ "$APP_DB_PASSWORD" = "developer" ]; then
        echo -e "${RED}ERROR: Default app password detected in production!${NC}"
        echo "Set APP_DB_PASSWORD environment variable to a secure value."
        exit 1
    fi
    
    echo -e "${GREEN}✅ Production security checks passed${NC}"
    echo ""
elif [ "$ENVIRONMENT" = "development" ]; then
    echo -e "${YELLOW}⚠️  Development environment - using default passwords${NC}"
    echo ""
fi

# Set up authentication
if [ -n "$DB_ADMIN_PASSWORD" ]; then
    export PGPASSWORD=$DB_ADMIN_PASSWORD
    echo "Using password authentication for admin user '$DB_ADMIN_USER'"
else
    echo "Attempting peer authentication for admin user '$DB_ADMIN_USER'"
    echo "If this fails, set DB_ADMIN_PASSWORD environment variable"
fi
echo ""

# Test PostgreSQL connectivity
echo "Testing PostgreSQL connectivity..."
if ! pg_isready -h $DB_HOST -p $DB_PORT -U $DB_ADMIN_USER >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT${NC}"
    echo "Please ensure:"
    echo "  1. PostgreSQL is running"
    echo "  2. Host and port are correct"
    echo "  3. User '$DB_ADMIN_USER' exists and has proper authentication"
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

# Create database if it doesn't exist
echo "Checking database '$DB_NAME'..."
if database_exists "$DB_NAME"; then
    echo -e "${YELLOW}ℹ️  Database '$DB_NAME' already exists${NC}"
else
    echo "Creating database '$DB_NAME'..."
    if createdb -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT $DB_NAME; then
        echo -e "${GREEN}✅ Database '$DB_NAME' created successfully${NC}"
    else
        echo -e "${RED}ERROR: Failed to create database '$DB_NAME'${NC}"
        exit 1
    fi
fi
echo ""

# Create application user if it doesn't exist
echo "Checking application user '$APP_DB_USER'..."
if user_exists "$APP_DB_USER"; then
    echo -e "${YELLOW}ℹ️  User '$APP_DB_USER' already exists${NC}"
    
    # Update password for existing user
    echo "Updating password for existing user..."
    if run_sql_as_admin "ALTER USER $APP_DB_USER WITH ENCRYPTED PASSWORD '$APP_DB_PASSWORD';" >/dev/null; then
        echo -e "${GREEN}✅ Password updated for user '$APP_DB_USER'${NC}"
    else
        echo -e "${RED}ERROR: Failed to update password for user '$APP_DB_USER'${NC}"
        exit 1
    fi
else
    echo "Creating application user '$APP_DB_USER'..."
    if run_sql_as_admin "CREATE USER $APP_DB_USER WITH ENCRYPTED PASSWORD '$APP_DB_PASSWORD';" >/dev/null; then
        echo -e "${GREEN}✅ User '$APP_DB_USER' created successfully${NC}"
    else
        echo -e "${RED}ERROR: Failed to create user '$APP_DB_USER'${NC}"
        exit 1
    fi
fi
echo ""

# Grant privileges to application user
echo "Granting privileges to '$APP_DB_USER'..."

# Database-level privileges
echo "  → Database-level privileges..."
if run_sql_as_admin "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $APP_DB_USER;" >/dev/null; then
    echo -e "${GREEN}    ✅ Database privileges granted${NC}"
else
    echo -e "${RED}    ERROR: Failed to grant database privileges${NC}"
    exit 1
fi

# Schema privileges (run on the target database)
echo "  → Schema privileges..."
if run_sql_as_admin "GRANT USAGE, CREATE ON SCHEMA public TO $APP_DB_USER;" "$DB_NAME" >/dev/null; then
    echo -e "${GREEN}    ✅ Schema privileges granted${NC}"
else
    echo -e "${RED}    ERROR: Failed to grant schema privileges${NC}"
    exit 1
fi

# Grant privileges on all existing tables, sequences, and functions
echo "  → Table, sequence, and function privileges..."
run_sql_as_admin "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $APP_DB_USER;" "$DB_NAME" >/dev/null
run_sql_as_admin "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $APP_DB_USER;" "$DB_NAME" >/dev/null
run_sql_as_admin "GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO $APP_DB_USER;" "$DB_NAME" >/dev/null
echo -e "${GREEN}    ✅ Object privileges granted${NC}"

# Set default privileges for future objects
echo "  → Default privileges for future objects..."
run_sql_as_admin "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $APP_DB_USER;" "$DB_NAME" >/dev/null
run_sql_as_admin "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $APP_DB_USER;" "$DB_NAME" >/dev/null
run_sql_as_admin "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO $APP_DB_USER;" "$DB_NAME" >/dev/null
echo -e "${GREEN}    ✅ Default privileges set${NC}"
echo ""

# Test connection with application user
echo "Testing application user connection..."
unset PGPASSWORD  # Clear admin password
export PGPASSWORD=$APP_DB_PASSWORD

if psql -U $APP_DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "SELECT 'Connection successful!' as status, current_user, current_database();" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Application user connection test passed${NC}"
else
    echo -e "${RED}ERROR: Application user connection test failed${NC}"
    echo "Credentials: $APP_DB_USER / [password hidden]"
    exit 1
fi

# Final summary
echo ""
echo -e "${GREEN}🎉 Database setup complete!${NC}"
echo ""
echo -e "${BLUE}📋 Setup Summary:${NC}"
echo "  Database: $DB_NAME"
echo "  App User: $APP_DB_USER"
echo "  Host: $DB_HOST:$DB_PORT"
echo "  Environment: $ENVIRONMENT"
echo ""
echo -e "${BLUE}🔄 Next Steps:${NC}"
echo "  1. Run your database migrations"
echo "  2. Start your application"
echo "  3. Connect using: psql -U $APP_DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME"
echo ""

# Clean up environment
unset PGPASSWORD