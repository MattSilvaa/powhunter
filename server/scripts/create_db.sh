#!/bin/bash
set -e

# Database configuration from environment variables or defaults
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-powhunter}

# Print information
echo "Creating database '$DB_NAME' if it doesn't exist..."

# Check if database exists
if psql -U $DB_USER -h $DB_HOST -p $DB_PORT -lqt | cut -d \| -f 1 | grep -qw $DB_NAME; then
    echo "Database '$DB_NAME' already exists"
else
    echo "Creating database '$DB_NAME'..."
    createdb -U $DB_USER -h $DB_HOST -p $DB_PORT $DB_NAME
    echo "Database created successfully"
fi

echo "Database setup complete!"