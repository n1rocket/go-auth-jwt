#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"

echo -e "${BLUE}===========================================
JWT Auth Service - Endpoint Test Report
===========================================${NC}\n"

# Function to test endpoint
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local headers=$4
    local expected_status=$5
    local description=$6
    
    echo -e "${YELLOW}Testing: ${description}${NC}"
    echo "Method: $method"
    echo "Endpoint: $endpoint"
    
    if [ -n "$data" ]; then
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" \
                -H "Content-Type: application/json" \
                -H "$headers" \
                -d "$data" 2>/dev/null || echo "000")
        else
            response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" \
                -H "Content-Type: application/json" \
                -d "$data" 2>/dev/null || echo "000")
        fi
    else
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" \
                -H "$headers" 2>/dev/null || echo "000")
        else
            response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" 2>/dev/null || echo "000")
        fi
    fi
    
    status_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$status_code" == "$expected_status" ]; then
        echo -e "Status: ${GREEN}$status_code (Expected: $expected_status) ✓${NC}"
    else
        echo -e "Status: ${RED}$status_code (Expected: $expected_status) ✗${NC}"
    fi
    
    if [ -n "$body" ]; then
        echo "Response: $body"
    fi
    echo "---"
    
    # Return tokens if signup/login successful
    if [[ "$endpoint" == "/api/v1/auth/signup" || "$endpoint" == "/api/v1/auth/login" ]] && [ "$status_code" == "201" -o "$status_code" == "200" ]; then
        echo "$body"
    fi
}

# Generate unique test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="test_${TIMESTAMP}@example.com"
TEST_PASSWORD="TestPassword@123"

echo -e "${BLUE}1. PUBLIC ENDPOINTS${NC}\n"

# Test health endpoint
test_endpoint "GET" "/health" "" "" "200" "Health Check"
sleep 1

# Test ready endpoint
test_endpoint "GET" "/ready" "" "" "200" "Readiness Check"
sleep 1

# Test signup endpoint
echo -e "\n${BLUE}2. AUTHENTICATION ENDPOINTS${NC}\n"
signup_response=$(test_endpoint "POST" "/api/v1/auth/signup" \
    "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    "" "201" "User Signup")
sleep 2

# Extract tokens from signup response
if echo "$signup_response" | grep -q "access_token"; then
    ACCESS_TOKEN=$(echo "$signup_response" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
    REFRESH_TOKEN=$(echo "$signup_response" | grep -o '"refresh_token":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}Tokens extracted successfully${NC}\n"
fi

# Test login endpoint
sleep 2
login_response=$(test_endpoint "POST" "/api/v1/auth/login" \
    "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    "" "200" "User Login")
sleep 2

# Update tokens from login if needed
if echo "$login_response" | grep -q "access_token"; then
    ACCESS_TOKEN=$(echo "$login_response" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
    REFRESH_TOKEN=$(echo "$login_response" | grep -o '"refresh_token":"[^"]*' | cut -d'"' -f4)
fi

# Test protected endpoints
echo -e "\n${BLUE}3. PROTECTED ENDPOINTS (Require JWT)${NC}\n"

# Test get current user
sleep 2
test_endpoint "GET" "/api/v1/auth/me" "" \
    "Authorization: Bearer $ACCESS_TOKEN" "200" "Get Current User"
sleep 2

# Test refresh token
test_endpoint "POST" "/api/v1/auth/refresh" \
    "{\"refresh_token\":\"$REFRESH_TOKEN\"}" \
    "" "200" "Refresh Token"
sleep 2

# Test logout
test_endpoint "POST" "/api/v1/auth/logout" \
    "{\"refresh_token\":\"$REFRESH_TOKEN\"}" \
    "Authorization: Bearer $ACCESS_TOKEN" "200" "Logout"
sleep 2

# Test logout all (need new login first)
echo -e "\n${BLUE}4. LOGOUT ALL DEVICES${NC}\n"
login_response=$(test_endpoint "POST" "/api/v1/auth/login" \
    "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    "" "200" "Re-login for Logout All Test")
sleep 2

if echo "$login_response" | grep -q "access_token"; then
    NEW_ACCESS_TOKEN=$(echo "$login_response" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
    test_endpoint "POST" "/api/v1/auth/logout-all" "" \
        "Authorization: Bearer $NEW_ACCESS_TOKEN" "200" "Logout All Devices"
fi
sleep 2

# Test error scenarios
echo -e "\n${BLUE}5. ERROR SCENARIOS${NC}\n"

# Invalid credentials
test_endpoint "POST" "/api/v1/auth/login" \
    "{\"email\":\"wrong@example.com\",\"password\":\"wrongpass\"}" \
    "" "401" "Invalid Credentials"
sleep 2

# Invalid signup data
test_endpoint "POST" "/api/v1/auth/signup" \
    "{\"email\":\"invalid-email\",\"password\":\"short\"}" \
    "" "400" "Invalid Signup Data"
sleep 2

# Unauthorized access
test_endpoint "GET" "/api/v1/auth/me" "" "" "401" "Unauthorized Access (No Token)"
sleep 2

# Invalid token
test_endpoint "GET" "/api/v1/auth/me" "" \
    "Authorization: Bearer invalid-token" "401" "Invalid Token"
sleep 2

# Duplicate signup
test_endpoint "POST" "/api/v1/auth/signup" \
    "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    "" "409" "Duplicate Email Signup"

# Email verification (will fail with dummy token)
echo -e "\n${BLUE}6. EMAIL VERIFICATION${NC}\n"
test_endpoint "POST" "/api/v1/auth/verify-email" \
    "{\"token\":\"dummy-verification-token\"}" \
    "" "400" "Email Verification (Invalid Token)"

echo -e "\n${BLUE}===========================================
Test Summary
===========================================${NC}"

echo -e "${GREEN}✓ All endpoints tested${NC}"
echo -e "${YELLOW}Note: Some tests may fail due to rate limiting.${NC}"
echo -e "${YELLOW}Wait a few seconds between runs to avoid 429 errors.${NC}"
echo -e "\n${BLUE}Available endpoints:${NC}"
echo "- GET  /health              - Health check"
echo "- GET  /ready               - Readiness check"
echo "- POST /api/v1/auth/signup  - User registration"
echo "- POST /api/v1/auth/login   - User authentication"
echo "- POST /api/v1/auth/refresh - Token refresh"
echo "- POST /api/v1/auth/verify-email - Email verification"
echo "- GET  /api/v1/auth/me      - Get current user (protected)"
echo "- POST /api/v1/auth/logout  - Logout (protected)"
echo "- POST /api/v1/auth/logout-all - Logout all devices (protected)"