#!/bin/bash

# Quick start script for the Database Intelligent Cockpit

echo "========================================"
echo "Database Intelligent Cockpit - Quick Start"
echo "========================================"
echo ""

# Check if go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

echo "Building Gateway..."
go build -o bin/gateway cmd/gateway/main.go

echo ""
echo "Starting Gateway on port 8080..."
echo ""
echo "========================================"
echo "Gateway is running!"
echo "========================================"
echo ""
echo "Endpoints:"
echo "  - Health:         GET  http://localhost:8080/health"
echo "  - GraphQL:        POST http://localhost:8080/graphql"
echo "  - GraphQL UI:     GET  http://localhost:8080/graphql/playground"
echo ""
echo "Authentication: Bearer tenant_id:user_id:role"
echo "Example: Bearer tenant-001:user-001:admin"
echo ""
echo "Example curl commands:"
echo ""
echo "# Get available metrics"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'Authorization: Bearer tenant-001:user-001:admin' \\"
echo "  -d '{\"query\": \"query { availableMetrics }\"}'"
echo ""
echo "# Query CPU metrics"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'Authorization: Bearer tenant-001:user-001:admin' \\"
echo "  -d '{\"query\": \"query { queryMetrics(name: \\\"cpu_usage\\\", limit: 5) { name points { value timestamp } } }\"}'"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Run the gateway
./bin/gateway