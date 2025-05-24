#!/bin/bash
set -e

# Database configuration from environment variables
# Try to detect the correct admin user (common on macOS)
DEFAULT_ADMIN_USER=$(whoami)
DB_ADMIN_USER=${DB_ADMIN_USER:-$DEFAULT_ADMIN_USER}
DB_ADMIN_PASSWORD=${DB_ADMIN_PASSWORD:-""}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-powhunter}

# Application user configuration
APP_DB_USER=${APP_DB_USER:-powerhunter_rw}
APP_DB_PASSWORD=${APP_DB_PASSWORD:-$(openssl rand -base64 32)}

echo "Setting up database '$DB_NAME' with application user '$APP_DB_USER'..."

# Function to run SQL commands as admin
run_sql_as_admin() {
    PGPASSWORD=$DB_ADMIN_PASSWORD psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d postgres -c "$1"
}

# Function to run SQL commands on target database as admin
run_sql_on_db_as_admin() {
    PGPASSWORD=$DB_ADMIN_PASSWORD psql -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "$1"
}

# Create database if it doesn't exist
echo "Creating database '$DB_NAME'..."
PGPASSWORD=$DB_ADMIN_PASSWORD createdb -U $DB_ADMIN_USER -h $DB_HOST -p $DB_PORT $DB_NAME 2>/dev/null || echo "Database '$DB_NAME' already exists"

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