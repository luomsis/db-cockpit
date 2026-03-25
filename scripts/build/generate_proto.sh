#!/bin/bash

# Database Intelligent Cockpit - Protobuf Code Generator
# Usage: ./scripts/build/generate_proto.sh
# Note: Run from project root directory

set -e

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${PROJECT_ROOT}"

echo "Generating Protobuf code..."

# Generate Go code for RPC services (Agent and Task Engine)
# Note: Domain layer services use REST/GraphQL, not RPC
PROTO_FILES=(
    "api/proto/agent/agent.proto"
    "api/proto/task/task.proto"
)

for proto in "${PROTO_FILES[@]}"; do
    echo "Processing $proto..."
    protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        "$proto"
done

echo "Protobuf code generation completed!"