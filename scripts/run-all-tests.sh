#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}===========================================
JWT Auth Service - Complete Test Suite
===========================================${NC}\n"

# Show usage
show_usage() {
    echo -e "${YELLOW}Usage: $0 [option]${NC}"
    echo -e "\nOptions:"
    echo -e "  ${GREEN}unit${NC}        - Run unit tests"
    echo -e "  ${GREEN}integration${NC} - Run integration tests"
    echo -e "  ${GREEN}e2e${NC}         - Run E2E tests with k6"
    echo -e "  ${GREEN}manual${NC}      - Run manual endpoint tests"
    echo -e "  ${GREEN}all${NC}         - Run all tests"
    echo -e "  ${GREEN}docker${NC}      - Start Docker environment only"
    echo -e "  ${GREEN}stop${NC}        - Stop Docker environment"
    echo -e "\nExamples:"
    echo -e "  $0 unit"
    echo -e "  $0 e2e"
    echo -e "  $0 all"
}

# Check if Docker environment is running
check_docker() {
    if ! docker compose -f docker-compose.dev.yml ps | grep -q "api.*running"; then
        echo -e "${YELLOW}Starting Docker environment...${NC}"
        make dev-up
        echo -e "${GREEN}Waiting for services to be ready...${NC}"
        sleep 10
    else
        echo -e "${GREEN}Docker environment is already running${NC}"
    fi
}

# Run unit tests
run_unit_tests() {
    echo -e "\n${BLUE}Running Unit Tests${NC}"
    echo "================================"
    make test
}

# Run integration tests
run_integration_tests() {
    echo -e "\n${BLUE}Running Integration Tests${NC}"
    echo "================================"
    make test-integration
}

# Run E2E tests
run_e2e_tests() {
    echo -e "\n${BLUE}Running E2E Tests with k6${NC}"
    echo "================================"
    check_docker
    
    echo -e "\n${YELLOW}1. Single User Test (avoids rate limiting)${NC}"
    docker run --rm -v ./scripts/k6:/scripts --network go-auth-jwt_default \
        grafana/k6:latest run /scripts/single_user_test.js \
        --env BASE_URL=http://api:8080 || true
    
    echo -e "\n${YELLOW}2. Comprehensive Test Suite${NC}"
    docker run --rm -v ./scripts/k6:/scripts --network go-auth-jwt_default \
        grafana/k6:latest run /scripts/comprehensive_test.js \
        --env BASE_URL=http://api:8080 --duration 1m || true
    
    echo -e "\n${YELLOW}3. Rate Limit Test${NC}"
    docker run --rm -v ./scripts/k6:/scripts --network go-auth-jwt_default \
        grafana/k6:latest run /scripts/rate_limit_test.js \
        --env BASE_URL=http://api:8080 --duration 30s || true
}

# Run manual endpoint tests
run_manual_tests() {
    echo -e "\n${BLUE}Running Manual Endpoint Tests${NC}"
    echo "================================"
    check_docker
    ./scripts/test-all-endpoints.sh
}

# Main script logic
case "$1" in
    unit)
        run_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    e2e)
        run_e2e_tests
        ;;
    manual)
        run_manual_tests
        ;;
    all)
        run_unit_tests
        run_integration_tests
        run_e2e_tests
        run_manual_tests
        ;;
    docker)
        check_docker
        echo -e "\n${GREEN}Docker environment is ready!${NC}"
        echo -e "API: http://localhost:8080"
        echo -e "MailHog: http://localhost:8025"
        ;;
    stop)
        echo -e "${YELLOW}Stopping Docker environment...${NC}"
        make dev-down
        ;;
    *)
        show_usage
        exit 1
        ;;
esac

echo -e "\n${CYAN}===========================================
Test Execution Complete
===========================================${NC}"

# Show summary
echo -e "\n${BLUE}Summary:${NC}"
echo -e "- API URL: http://localhost:8080"
echo -e "- MailHog UI: http://localhost:8025"
echo -e "- Health Check: http://localhost:8080/health"
echo -e "- Ready Check: http://localhost:8080/ready"

echo -e "\n${YELLOW}Notes:${NC}"
echo -e "- Rate limiting is configured to 10 requests/minute for auth endpoints"
echo -e "- Use single user tests to avoid rate limiting"
echo -e "- Check MailHog for email verification tokens"
echo -e "- JWT tokens expire after 15 minutes (access) and 7 days (refresh)"

echo -e "\n${GREEN}✓ All requested tests completed${NC}"