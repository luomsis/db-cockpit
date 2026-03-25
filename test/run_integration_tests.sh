#!/bin/bash

# Gateway Integration Test Runner
# This script starts the required services and runs integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GATEWAY_PORT=${GATEWAY_PORT:-8080}
DATAQUERY_PORT=${DATAQUERY_PORT:-8084}
TEST_TIMEOUT=30

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${PROJECT_ROOT}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Gateway Integration Test Runner${NC}"
echo -e "${GREEN}========================================${NC}"

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1 ; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Function to wait for a service to be ready
wait_for_service() {
    local port=$1
    local name=$2
    local count=0

    echo -e "${YELLOW}Waiting for $name on port $port...${NC}"

    while ! curl -s "http://localhost:$port/health" > /dev/null 2>&1; do
        sleep 1
        count=$((count + 1))
        if [ $count -ge $TEST_TIMEOUT ]; then
            echo -e "${RED}Timeout waiting for $name${NC}"
            return 1
        fi
        echo -n "."
    done
    echo -e "${GREEN}OK${NC}"
    return 0
}

# Function to start gateway service
start_gateway() {
    echo -e "${YELLOW}Starting Gateway service on port $GATEWAY_PORT...${NC}"

    if check_port $GATEWAY_PORT; then
        echo -e "${YELLOW}Gateway already running on port $GATEWAY_PORT${NC}"
        return 0
    fi

    # Start gateway in background
    go run cmd/gateway/main.go > /tmp/gateway.log 2>&1 &
    GATEWAY_PID=$!
    echo "Gateway PID: $GATEWAY_PID"

    # Wait for gateway to be ready
    if wait_for_service $GATEWAY_PORT "Gateway"; then
        return 0
    else
        return 1
    fi
}

# Function to start dataquery service
start_dataquery() {
    echo -e "${YELLOW}Starting Data Query service on port $DATAQUERY_PORT...${NC}"

    if check_port $DATAQUERY_PORT; then
        echo -e "${YELLOW}Data Query already running on port $DATAQUERY_PORT${NC}"
        return 0
    fi

    # Start dataquery in background
    go run cmd/dataquery/main.go > /tmp/dataquery.log 2>&1 &
    DATAQUERY_PID=$!
    echo "Data Query PID: $DATAQUERY_PID"

    # Wait for dataquery to be ready (check graphql endpoint)
    local count=0
    echo -n "Waiting for Data Query service"
    while ! curl -s "http://localhost:$DATAQUERY_PORT/graphql" -d '{"query":"{__typename}"}' > /dev/null 2>&1; do
        sleep 1
        count=$((count + 1))
        if [ $count -ge $TEST_TIMEOUT ]; then
            echo -e "${RED}Timeout waiting for Data Query service${NC}"
            return 1
        fi
        echo -n "."
    done
    echo -e "${GREEN}OK${NC}"
    return 0
}

# Function to run tests
run_tests() {
    echo -e "${YELLOW}Running integration tests...${NC}"
    echo ""

    # Export gateway URL for tests
    export GATEWAY_URL="http://localhost:$GATEWAY_PORT"

    # Run tests
    go test -v ./test/integration/... -run TestGateway

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
    else
        echo -e "${RED}Some tests failed${NC}"
    fi
}

# Function to cleanup
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"

    if [ ! -z "$GATEWAY_PID" ]; then
        echo "Stopping Gateway (PID: $GATEWAY_PID)"
        kill $GATEWAY_PID 2>/dev/null || true
    fi

    if [ ! -z "$DATAQUERY_PID" ]; then
        echo "Stopping Data Query (PID: $DATAQUERY_PID)"
        kill $DATAQUERY_PID 2>/dev/null || true
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Main execution
echo ""
echo -e "${YELLOW}Step 1: Checking dependencies...${NC}"

# Check if curl is available
if ! command -v curl &> /dev/null; then
    echo -e "${RED}curl is required but not installed${NC}"
    exit 1
fi

# Check if lsof is available
if ! command -v lsof &> /dev/null; then
    echo -e "${YELLOW}Warning: lsof not found, port checking may not work${NC}"
fi

echo -e "${GREEN}Dependencies OK${NC}"
echo ""

echo -e "${YELLOW}Step 2: Starting services...${NC}"

# Start gateway
if ! start_gateway; then
    echo -e "${RED}Failed to start Gateway${NC}"
    echo "Check logs: /tmp/gateway.log"
    exit 1
fi

# Start dataquery
if ! start_dataquery; then
    echo -e "${YELLOW}Data Query service not available, some tests may be skipped${NC}"
fi

echo ""
echo -e "${YELLOW}Step 3: Running integration tests...${NC}"
echo ""

# Run tests
run_tests

echo ""
echo -e "${GREEN}Done!${NC}"