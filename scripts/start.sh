#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Go JWT Auth Service...${NC}"

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Warning: .env file not found!${NC}"
    echo "Creating .env from .env.example..."
    cp .env.example .env
    echo -e "${YELLOW}Please update .env with your configuration before proceeding.${NC}"
    exit 1
fi

# Load environment variables
export $(cat .env | grep -v '^#' | xargs)

# Check if database is reachable
echo "Checking database connection..."
if ! pg_isready -h ${DB_HOST:-localhost} -p ${DB_PORT:-5432} -U ${DB_USER:-postgres} > /dev/null 2>&1; then
    echo -e "${RED}Error: Database is not reachable!${NC}"
    echo "Please ensure PostgreSQL is running and accessible."
    exit 1
fi

# Run migrations
echo "Running database migrations..."
./scripts/migrate.sh up

# Build the application
echo "Building application..."
go build -o bin/api cmd/api/main.go

# Start the application
echo -e "${GREEN}Starting API server on port ${APP_PORT:-8080}...${NC}"
./bin/api