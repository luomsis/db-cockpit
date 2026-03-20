#!/bin/bash

# Demo script showing the complete call path with expected outputs
# This simulates what happens when curl calls the gateway

echo "========================================"
echo "Database Intelligent Cockpit - Demo"
echo "Complete Call Path Demonstration"
echo "========================================"
echo ""

echo "📍 Call Path:"
echo "  ┌─────────────┐"
echo "  │   curl      │ (Front-end UI simulation)"
echo "  └──────┬──────┘"
echo "         │ HTTP POST /graphql"
echo "         ▼"
echo "  ┌─────────────┐"
echo "  │   Gateway   │ (Hertz Framework)"
echo "  │  - Auth     │"
echo "  │  - Routing  │"
echo "  └──────┬──────┘"
echo "         │ GraphQL Query"
echo "         ▼"
echo "  ┌─────────────┐"
echo "  │ Data Query  │ (GraphQL Service)"
echo "  │   Service   │"
echo "  │  - Resolver │"
echo "  │  - Schema   │"
echo "  └──────┬──────┘"
echo "         │ QueryMetrics()"
echo "         ▼"
echo "  ┌─────────────┐"
echo "  │ TimescaleDB │ (Mock Implementation)"
echo "  │   Mock      │"
echo "  │  - cpu_usage│"
echo "  │  - memory   │"
echo "  │  - latency  │"
echo "  └─────────────┘"
echo ""

echo "========================================"
echo "Example Request & Response"
echo "========================================"
echo ""

echo "🔹 Request 1: Get Available Metrics"
echo "----------------------------------------"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'Authorization: Bearer tenant-001:user-001:admin' \\"
echo "  -d '{\"query\": \"query { availableMetrics }\"}'"
echo ""
echo "Response:"
cat << 'EOF'
{
  "data": {
    "availableMetrics": [
      "cpu_usage",
      "memory_usage",
      "query_latency_ms",
      "connection_count",
      "slow_query_count",
      "disk_io_read_mbps"
    ]
  }
}
EOF
echo ""

echo "🔹 Request 2: Query CPU Metrics"
echo "----------------------------------------"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'Authorization: Bearer tenant-001:user-001:admin' \\"
echo "  -d '{\"query\": \"query { queryMetrics(name: \\\"cpu_usage\\\", limit: 3) { name points { value timestamp } } }\"}'"
echo ""
echo "Response:"
cat << 'EOF'
{
  "data": {
    "queryMetrics": {
      "name": "cpu_usage",
      "points": [
        {"value": 35.1, "timestamp": "2024-01-15T10:00:00Z"},
        {"value": 36.2, "timestamp": "2024-01-15T10:01:00Z"},
        {"value": 37.3, "timestamp": "2024-01-15T10:02:00Z"}
      ]
    }
  }
}
EOF
echo ""

echo "🔹 Request 3: Query Multiple Metrics"
echo "----------------------------------------"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'Authorization: Bearer tenant-001:user-001:admin' \\"
echo "  -d '{\"query\": \"query { queryMetricsRange(names: [\\\"cpu_usage\\\", \\\"memory_usage\\\"], limit: 2) { name points { value } } }\"}'"
echo ""
echo "Response:"
cat << 'EOF'
{
  "data": {
    "queryMetricsRange": [
      {
        "name": "cpu_usage",
        "points": [{"value": 35.1}, {"value": 36.2}]
      },
      {
        "name": "memory_usage",
        "points": [{"value": 65.05}, {"value": 66.1}]
      }
    ]
  }
}
EOF
echo ""

echo "🔹 Request 4: Vector Search"
echo "----------------------------------------"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'Authorization: Bearer tenant-001:user-001:admin' \\"
echo "  -d '{\"query\": \"query { vectorSearch(collection: \\\"documents\\\", topK: 2) { id similarity metadata } }\"}'"
echo ""
echo "Response:"
cat << 'EOF'
{
  "data": {
    "vectorSearch": [
      {"id": "doc-001", "similarity": 0.95, "metadata": "{title: SQL Optimization Guide, type: documentation}"},
      {"id": "doc-002", "similarity": 0.87, "metadata": "{title: Index Best Practices, type: documentation}"}
    ]
  }
}
EOF
echo ""

echo "========================================"
echo "To run the actual server:"
echo "========================================"
echo ""
echo "  chmod +x scripts/start.sh"
echo "  ./scripts/start.sh"
echo ""
echo "Then run:"
echo "  chmod +x scripts/verify_graphql.sh"
echo "  ./scripts/verify_graphql.sh"
echo ""