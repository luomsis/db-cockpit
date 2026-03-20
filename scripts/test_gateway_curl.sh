#!/bin/bash

# Manual Gateway Testing Script
# Uses curl to test the complete flow: frontend -> gateway -> dataquery -> db

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GATEWAY_URL=${GATEWAY_URL:-"http://localhost:8080"}
TENANT_ID=${TENANT_ID:-"test-tenant"}
USER_ID=${USER_ID:-"test-user"}

# Auth header (format: Bearer tenantID:userID:roles)
AUTH_HEADER="Bearer ${TENANT_ID}:${USER_ID}:user"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Gateway Manual Testing Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Gateway URL: ${BLUE}${GATEWAY_URL}${NC}"
echo -e "Tenant ID:   ${BLUE}${TENANT_ID}${NC}"
echo -e "User ID:     ${BLUE}${USER_ID}${NC}"
echo ""

# Function to print section header
section() {
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
}

# Function to print curl command and response
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local auth=$4

    echo -e "${BLUE}Request:${NC}"
    if [ -n "$data" ]; then
        if [ "$auth" = "true" ]; then
            echo "curl -s -X $method '${GATEWAY_URL}${endpoint}' \\"
            echo "  -H 'Authorization: ${AUTH_HEADER}' \\"
            echo "  -H 'Content-Type: application/json' \\"
            echo "  -d '${data}'"
        else
            echo "curl -s -X $method '${GATEWAY_URL}${endpoint}' \\"
            echo "  -H 'Content-Type: application/json' \\"
            echo "  -d '${data}'"
        fi
    else
        if [ "$auth" = "true" ]; then
            echo "curl -s -X $method '${GATEWAY_URL}${endpoint}' \\"
            echo "  -H 'Authorization: ${AUTH_HEADER}'"
        else
            echo "curl -s -X $method '${GATEWAY_URL}${endpoint}'"
        fi
    fi

    echo -e "${GREEN}Response:${NC}"
    if [ -n "$data" ]; then
        if [ "$auth" = "true" ]; then
            curl -s -X $method "${GATEWAY_URL}${endpoint}" \
                -H "Authorization: ${AUTH_HEADER}" \
                -H "Content-Type: application/json" \
                -d "${data}" | jq . 2>/dev/null || cat
        else
            curl -s -X $method "${GATEWAY_URL}${endpoint}" \
                -H "Content-Type: application/json" \
                -d "${data}" | jq . 2>/dev/null || cat
        fi
    else
        if [ "$auth" = "true" ]; then
            curl -s -X $method "${GATEWAY_URL}${endpoint}" \
                -H "Authorization: ${AUTH_HEADER}" | jq . 2>/dev/null || cat
        else
            curl -s -X $method "${GATEWAY_URL}${endpoint}" | jq . 2>/dev/null || cat
        fi
    fi
    echo ""
}

# ============================================
# Health Check Tests
# ============================================
section "1. Health Check Tests"

echo -e "${BLUE}GET /health${NC}"
curl -s "${GATEWAY_URL}/health" | jq . 2>/dev/null || curl -s "${GATEWAY_URL}/health"
echo ""

echo -e "${BLUE}GET /api/v1/health${NC}"
curl -s "${GATEWAY_URL}/api/v1/health" | jq . 2>/dev/null || curl -s "${GATEWAY_URL}/api/v1/health"
echo ""

# ============================================
# Authentication Tests
# ============================================
section "2. Authentication Tests"

echo -e "${BLUE}Without Authorization header (should return 401)${NC}"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -X POST "${GATEWAY_URL}/api/v1/sql/review" \
    -H "Content-Type: application/json" \
    -d '{"database_id":"test-db","sql":"SELECT 1"}'
echo ""

echo -e "${BLUE}With valid Authorization header${NC}"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -X POST "${GATEWAY_URL}/api/v1/sql/review" \
    -H "Authorization: ${AUTH_HEADER}" \
    -H "Content-Type: application/json" \
    -d '{"database_id":"test-db","sql":"SELECT 1"}' | jq . 2>/dev/null || cat
echo ""

# ============================================
# SQL Governance API Tests
# ============================================
section "3. SQL Governance API Tests"

echo -e "${BLUE}POST /api/v1/sql/review - Review SQL${NC}"
test_endpoint "POST" "/api/v1/sql/review" \
    '{"database_id":"test-db","sql":"SELECT * FROM users WHERE id = 1","context":{}}' "true"

echo -e "${BLUE}POST /api/v1/sql/execute - Execute SQL (dry run)${NC}"
test_endpoint "POST" "/api/v1/sql/execute" \
    '{"database_id":"test-db","sql":"SELECT * FROM users LIMIT 10","timeout_seconds":30,"max_rows":100,"dry_run":true}' "true"

echo -e "${BLUE}POST /api/v1/sql/execute - Execute SQL (actual)${NC}"
test_endpoint "POST" "/api/v1/sql/execute" \
    '{"database_id":"test-db","sql":"SELECT 1 as test","timeout_seconds":30}' "true"

echo -e "${BLUE}GET /api/v1/sql/audit - Get Audit Trail${NC}"
test_endpoint "GET" "/api/v1/sql/audit" "" "true"

# ============================================
# Performance API Tests
# ============================================
section "4. Performance API Tests"

NOW=$(date +%s)
START=$((NOW - 86400))  # 24 hours ago

echo -e "${BLUE}POST /api/v1/performance/diagnose - Run Diagnosis${NC}"
test_endpoint "POST" "/api/v1/performance/diagnose" \
    "{\"database_id\":\"test-db\",\"scope\":\"full\",\"start_time\":${START},\"end_time\":${NOW},\"deep_analysis\":false}" "true"

echo -e "${BLUE}POST /api/v1/performance/metrics - Get Metrics${NC}"
test_endpoint "POST" "/api/v1/performance/metrics" \
    "{\"database_id\":\"test-db\",\"metric_names\":[\"cpu_usage\",\"memory_usage\"],\"start_time\":${START},\"end_time\":${NOW}}" "true"

echo -e "${BLUE}POST /api/v1/performance/slow-queries - Get Slow Queries${NC}"
test_endpoint "POST" "/api/v1/performance/slow-queries" \
    "{\"database_id\":\"test-db\",\"start_time\":${START},\"end_time\":${NOW},\"min_duration_ms\":100,\"limit\":10}" "true"

# ============================================
# Threshold API Tests
# ============================================
section "5. Threshold API Tests"

echo -e "${BLUE}GET /api/v1/thresholds - Get Thresholds${NC}"
test_endpoint "GET" "/api/v1/thresholds" \
    '{"database_id":"test-db","metric_names":["cpu_usage","memory_usage"]}' "true"

echo -e "${BLUE}PUT /api/v1/thresholds - Update Threshold${NC}"
test_endpoint "PUT" "/api/v1/thresholds" \
    '{"threshold_id":"thresh-1","value":90.0,"type":"static"}' "true"

# ============================================
# LLM API Tests
# ============================================
section "6. LLM API Tests"

echo -e "${BLUE}POST /api/v1/llm/chat - Chat${NC}"
test_endpoint "POST" "/api/v1/llm/chat" \
    '{"session_id":"session-123","message":"How do I optimize slow queries?"}' "true"

echo -e "${BLUE}POST /api/v1/llm/generate-sql - Generate SQL${NC}"
test_endpoint "POST" "/api/v1/llm/generate-sql" \
    '{"database_id":"test-db","natural_language":"Get all users created in the last 7 days","schema_context":"users(id, name, email, created_at)"}' "true"

echo -e "${BLUE}GET /api/v1/llm/recommendations - Get Recommendations${NC}"
test_endpoint "GET" "/api/v1/llm/recommendations" \
    '{"database_id":"test-db","category":"performance","limit":5}' "true"

# ============================================
# GraphQL Proxy Tests
# ============================================
section "7. GraphQL Proxy Tests (via Gateway)"

echo -e "${BLUE}POST /graphql - Introspection Query${NC}"
test_endpoint "POST" "/graphql" \
    '{"query":"{ __schema { types { name } } }"}' "false"

echo -e "${BLUE}POST /graphql - Get Endpoints${NC}"
test_endpoint "POST" "/graphql" \
    '{"query":"{ endpoints }"}' "false"

echo -e "${BLUE}POST /graphql - Get Metrics for Endpoint${NC}"
test_endpoint "POST" "/graphql" \
    '{"query":"{ metrics(endpoint: \"/api/metrics\") }"}' "false"

echo -e "${BLUE}POST /graphql - Query Series${NC}"
cat <<'EOF'
curl -s -X POST "${GATEWAY_URL}/graphql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(endpoint: \"/api/metrics\", metric: \"cpu_usage\", timeRange: $tr, limit: 5) { meta { id metric labels { entries { key value } } } points { time value } } }",
    "variables": {
      "tr": {
        "start": "'$(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ)'",
        "end": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }
    }
  }' | jq .
EOF

curl -s -X POST "${GATEWAY_URL}/graphql" \
  -H "Content-Type: application/json" \
  -d "{
    \"query\": \"query(\$tr: TimeRangeInput!) { series(endpoint: \\\"/api/metrics\\\", metric: \\\"cpu_usage\\\", timeRange: \$tr, limit: 5) { meta { id metric labels { entries { key value } } } points { time value } } }\",
    \"variables\": {
      \"tr\": {
        \"start\": \"$(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)\",
        \"end\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
      }
    }
  }" | jq . 2>/dev/null || echo "Query executed"
echo ""

# ============================================
# Error Handling Tests
# ============================================
section "8. Error Handling Tests"

echo -e "${BLUE}Invalid JSON (should return 400)${NC}"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -X POST "${GATEWAY_URL}/api/v1/sql/review" \
    -H "Authorization: ${AUTH_HEADER}" \
    -H "Content-Type: application/json" \
    -d 'invalid json'
echo ""

echo -e "${BLUE}Non-existent endpoint (should return 404)${NC}"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    "${GATEWAY_URL}/api/v1/nonexistent"
echo ""

# ============================================
# CORS Tests
# ============================================
section "9. CORS Tests"

echo -e "${BLUE}Preflight (OPTIONS) Request${NC}"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -X OPTIONS "${GATEWAY_URL}/api/v1/sql/review" \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: POST" \
    -H "Access-Control-Request-Headers: Content-Type, Authorization" \
    -D - -o /dev/null
echo ""

# ============================================
# Summary
# ============================================
section "Test Summary"

echo -e "${GREEN}All tests completed!${NC}"
echo ""
echo "To run these tests:"
echo "  1. Start the gateway: go run cmd/gateway/main.go"
echo "  2. Start dataquery: go run cmd/dataquery/main.go"
echo "  3. Run this script: ./scripts/test_gateway_curl.sh"
echo ""
echo "To run automated tests:"
echo "  ./scripts/run_integration_tests.sh"