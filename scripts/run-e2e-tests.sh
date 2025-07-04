#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting E2E Test Suite for JWT Auth Service${NC}"
echo "=============================================="

# Function to cleanup
cleanup() {
    echo -e "\n${YELLOW}Cleaning up test environment...${NC}"
    docker compose -f docker-compose.test.yml down -v
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Build the application
echo -e "\n${YELLOW}Building application...${NC}"
docker compose -f docker-compose.test.yml build

# Start services in detached mode
echo -e "\n${YELLOW}Starting test environment...${NC}"
docker compose -f docker-compose.test.yml up -d postgres mailhog

# Wait for PostgreSQL to be ready
echo -e "\n${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
for i in {1..30}; do
    if docker compose -f docker-compose.test.yml exec postgres pg_isready -U auth -d authsvc_test > /dev/null 2>&1; then
        echo -e "${GREEN}PostgreSQL is ready${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Run migrations
echo -e "\n${YELLOW}Running database migrations...${NC}"
docker compose -f docker-compose.test.yml run --rm migrate

# Start API service
echo -e "\n${YELLOW}Starting API service...${NC}"
docker compose -f docker-compose.test.yml up -d api

# Wait for API to be ready
echo -e "\n${YELLOW}Waiting for API to be ready...${NC}"
for i in {1..30}; do
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}API is ready${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Show service status
echo -e "\n${YELLOW}Service Status:${NC}"
docker compose -f docker-compose.test.yml ps

# Run k6 tests
echo -e "\n${YELLOW}Running k6 E2E tests...${NC}"
echo "================================"

if docker compose -f docker-compose.test.yml run --rm k6; then
    echo -e "\n${GREEN}✓ E2E tests passed successfully!${NC}"
    EXIT_CODE=0
else
    echo -e "\n${RED}✗ E2E tests failed${NC}"
    EXIT_CODE=1
fi

# Show logs if tests failed
if [ $EXIT_CODE -ne 0 ]; then
    echo -e "\n${YELLOW}API Logs:${NC}"
    docker compose -f docker-compose.test.yml logs api --tail=50
fi

# Optional: Show MailHog UI URL
echo -e "\n${YELLOW}MailHog UI available at: http://localhost:8025${NC}"

exit $EXIT_CODE