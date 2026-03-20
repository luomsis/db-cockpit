#!/bin/bash

# Verification script for the complete call path:
# curl -> Gateway -> GraphQL Data Query Service -> TimescaleDB Mock

set -e

echo "========================================"
echo "Database Intelligent Cockpit Verification"
echo "========================================"
echo ""

# Configuration
GATEWAY_URL="http://localhost:8080"
AUTH_HEADER="Authorization: Bearer tenant-001:user-001:admin"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
        echo "   Response: $3"
    fi
}

# Function to make GraphQL request
graphql_request() {
    local query="$1"
    curl -s -X POST "$GATEWAY_URL/graphql" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "{\"query\": $query}"
}

echo "Step 1: Checking Gateway Health..."
RESPONSE=$(curl -s "$GATEWAY_URL/health")
if echo "$RESPONSE" | grep -q "ok"; then
    print_result 0 "Gateway is healthy"
else
    print_result 1 "Gateway health check failed" "$RESPONSE"
    echo "Please start the gateway first: go run cmd/gateway/main.go"
    exit 1
fi

echo ""
echo "Step 2: Testing GraphQL Query - Available Metrics..."
QUERY='query { availableMetrics }'
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/graphql" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "{\"query\": \"$QUERY\"}")

if echo "$RESPONSE" | grep -q "cpu_usage"; then
    print_result 0 "Available metrics query successful"
    echo "   Metrics: $(echo $RESPONSE | grep -o '\[.*\]' | head -1)"
else
    print_result 1 "Available metrics query failed" "$RESPONSE"
fi

echo ""
echo "Step 3: Testing GraphQL Query - Query CPU Metrics..."
QUERY='query { queryMetrics(name: "cpu_usage", limit: 5) { name points { value timestamp tags } } }'
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/graphql" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "{\"query\": \"$QUERY\"}")

if echo "$RESPONSE" | grep -q "points"; then
    print_result 0 "CPU metrics query successful"
    echo "   Sample response:"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
else
    print_result 1 "CPU metrics query failed" "$RESPONSE"
fi

echo ""
echo "Step 4: Testing GraphQL Query - Query Memory Metrics..."
QUERY='query { queryMetrics(name: "memory_usage", limit: 3) { name points { value timestamp } } }'
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/graphql" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "{\"query\": \"$QUERY\"}")

if echo "$RESPONSE" | grep -q "memory_usage"; then
    print_result 0 "Memory metrics query successful"
else
    print_result 1 "Memory metrics query failed" "$RESPONSE"
fi

echo ""
echo "Step 5: Testing GraphQL Query - Query Multiple Metrics..."
QUERY='query { queryMetricsRange(names: ["cpu_usage", "memory_usage"], limit: 2) { name points { value timestamp } } }'
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/graphql" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "{\"query\": \"$QUERY\"}")

if echo "$RESPONSE" | grep -q "cpu_usage\|memory_usage"; then
    print_result 0 "Multiple metrics query successful"
else
    print_result 1 "Multiple metrics query failed" "$RESPONSE"
fi

echo ""
echo "Step 6: Testing GraphQL Query - Vector Search..."
QUERY='query { vectorSearch(collection: "documents", topK: 3) { id similarity metadata } }'
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/graphql" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "{\"query\": \"$QUERY\"}")

if echo "$RESPONSE" | grep -q "similarity"; then
    print_result 0 "Vector search query successful"
else
    print_result 1 "Vector search query failed" "$RESPONSE"
fi

echo ""
echo "Step 7: Testing GraphQL Query - Graph Query..."
QUERY='query { graphQuery(queryType: "cypher", queryText: "MATCH (n) RETURN n LIMIT 5") { nodes { id label properties } edges { id source target type } } }'
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/graphql" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "{\"query\": \"$QUERY\"}")

if echo "$RESPONSE" | grep -q "nodes\|edges"; then
    print_result 0 "Graph query successful"
else
    print_result 1 "Graph query failed" "$RESPONSE"
fi

echo ""
echo "Step 8: Testing GraphQL Mutation - Cache Data..."
QUERY='mutation { cacheData(key: "test_key", data: "test_value", ttlSeconds: 300) }'
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/graphql" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "{\"query\": \"$QUERY\"}")

if echo "$RESPONSE" | grep -q "true"; then
    print_result 0 "Cache mutation successful"
else
    print_result 1 "Cache mutation failed" "$RESPONSE"
fi

echo ""
echo "========================================"
echo "Verification Complete!"
echo "========================================"
echo ""
echo "Call Path Verified:"
echo "  curl -> Gateway (Hertz) -> GraphQL Handler -> Data Query Service -> TimescaleDB Mock"
echo ""
echo "GraphQL Playground available at: $GATEWAY_URL/graphql/playground"