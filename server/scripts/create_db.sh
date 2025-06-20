#!/bin/bash
set -e

DB_ADMIN_USER=${DB_ADMIN_USER:-postgres}
DB_ADMIN_PASSWORD=${DB_ADMIN_PASSWORD:-}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-powhunter}

# Application user configuration
APP_DB_USER=${APP_DB_USER:-powhunter_rw}
APP_DB_PASSWORD=${APP_DB_PASSWORD:-root}

echo "Setting up database '$DB_NAME' with application user '$APP_DB_USER'..."
echo ""

# Check if we can connect to PostgreSQL
echo "Checking PostgreSQL connection..."
if [ -n "$DB_ADMIN_PASSWORD" ]; then
    echo "Using password authentication for admin user '$DB_ADMIN_USER'"
else
    echo "Attempting peer authentication for admin user '$DB_ADMIN_USER' (no password set)"
    echo "If this fails, you may need to:"
    echo "  1. Run as postgres system user: sudo -u postgres $0"
    echo "  2. Or set DB_ADMIN_PASSWORD environment variable"
    echo "  3. Or configure PostgreSQL to allow password authentication"
fi
echo ""

# Function to run SQL commands as admin
run_sql_as_admin() {
    if [ -n "$DB_ADMIN_PASSWORD" ]; then
        PGPASSWORD=$DB_ADMIN_PASSWORD psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d postgres -c "$1"
    else
        # Try without password first (for peer authentication)
        psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d postgres -c "$1" 2>/dev/null || \
        # If that fails, try with empty password
        PGPASSWORD="" psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d postgres -c "$1"
    fi
}

# Function to run SQL commands on target database as admin
run_sql_on_db_as_admin() {
    if [ -n "$DB_ADMIN_PASSWORD" ]; then
        PGPASSWORD=$DB_ADMIN_PASSWORD psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "$1"
    else
        # Try without password first (for peer authentication)
        psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "$1" 2>/dev/null || \
        # If that fails, try with empty password
        PGPASSWORD="" psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "$1"
    fi
}

# Function to run createdb command
run_createdb() {
    if [ -n "$DB_ADMIN_PASSWORD" ]; then
        PGPASSWORD=$DB_ADMIN_PASSWORD createdb -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT $DB_NAME
    else
        # Try without password first (for peer authentication)
        createdb -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT $DB_NAME 2>/dev/null || \
        # If that fails, try with empty password
        PGPASSWORD="" createdb -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT $DB_NAME
    fi
}

# Create database if it doesn't exist
echo "Creating database '$DB_NAME'..."
run_createdb 2>/dev/null || echo "Database '$DB_NAME' already exists"

# Create application user if it doesn't exist
echo "Creating application user '$APP_DB_USER'..."
run_sql_as_admin "DO \$\$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$APP_DB_USER') THEN
        CREATE USER $APP_DB_USER WITH ENCRYPTED PASSWORD '$APP_DB_PASSWORD';
        RAISE NOTICE 'User $APP_DB_USER created successfully';
    ELSE
        RAISE NOTICE 'User $APP_DB_USER already exists';
    END IF;
END
\$\$;" 2>/dev/null || echo "User creation completed"

# Grant privileges to application user
echo "Granting privileges to '$APP_DB_USER'..."

# Database-level privileges
run_sql_as_admin "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $APP_DB_USER;"

# Schema privileges (run on the target database)
run_sql_on_db_as_admin "GRANT USAGE, CREATE ON SCHEMA public TO $APP_DB_USER;"

# Grant privileges on all existing tables, sequences, and functions
run_sql_on_db_as_admin "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $APP_DB_USER;"
run_sql_on_db_as_admin "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $APP_DB_USER;"
run_sql_on_db_as_admin "GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO $APP_DB_USER;"

# Set default privileges for future objects
run_sql_on_db_as_admin "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $APP_DB_USER;"
run_sql_on_db_as_admin "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $APP_DB_USER;"
run_sql_on_db_as_admin "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO $APP_DB_USER;"

# Test connection with application user
echo "Testing connection with application user..."
PGPASSWORD=$APP_DB_PASSWORD psql -U $APP_DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "SELECT 'Connection successful!' as status;" > /dev/null

echo "✅ Database setup complete!"
echo ""
echo "Database Details:"
echo "  Database: $DB_NAME"
echo "  App User: $APP_DB_USER"
echo "  Host: $DB_HOST:$DB_PORT"
echo ""