#!/bin/bash
set -e

# Database configuration from environment variables
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-powhunter}

# Export password for PostgreSQL commands
export PGPASSWORD=$DB_PASSWORD

echo "Setting up database '$DB_NAME'..."

# Create database if it doesn't exist (ignoring errors if it does)
createdb -U $DB_USER -h $DB_HOST -p $DB_PORT $DB_NAME 2>/dev/null || echo "Database already exists"

echo "Database setup complete!"