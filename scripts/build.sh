#!/bin/bash

# Build all services

set -e

echo "Building all services..."

# Build API Gateway
echo "Building API Gateway..."
go build -o bin/gateway cmd/gateway/main.go

# Build Collector
echo "Building Collector..."
go build -o bin/collector cmd/collector/main.go

# Build Execution Agent
echo "Building Execution Agent..."
go build -o bin/agent cmd/agent/main.go

# Build Task Engine
echo "Building Task Engine..."
go build -o bin/taskengine cmd/taskengine/main.go

echo "All services built successfully!"
echo "Binaries available in ./bin/"