# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Build all services
go build ./cmd/...
./scripts/build/build.sh           # Or use the build script

# Run individual services
go run cmd/gateway/main.go       # API Gateway (default port 8080)
go run cmd/collector/main.go     # Collector (default port 8081)
go run cmd/agent/main.go         # Execution Agent (default port 8082)
go run cmd/taskengine/main.go    # Task Engine (default port 8083)
go run cmd/dataquery/main.go     # Data Query Service (default port 8084)

# Run tests
go test ./...                    # All tests
go test ./pkg/...                # Unit tests only
go test -v ./test/integration/... # Integration tests with verbose output
go test -cover ./...             # With coverage

# Generate protobuf code (requires protoc)
./scripts/build/generate_proto.sh
```

## Architecture: Communication Protocols

The system follows a specific communication pattern:

| Path | Protocol |
|------|----------|
| Frontend ↔ Gateway | RESTful API |
| Frontend ↔ Data Query | **REST API** (`/api/v1/*`) |
| Domain Layer ↔ Agent | **RPC** (gRPC) |
| Domain Layer ↔ Task Engine | **RPC** (gRPC) |
| Domain Layer internal services | Direct function calls (no RPC) |
| Data Query ↔ TimescaleDB | Direct function calls |

**Key insight**: Data Query Service exposes a REST API with Swagger documentation. Other domain services use REST through the Gateway.

## Domain Service Pattern

Every domain service in `pkg/domain/<name>/` follows this structure:

1. **service.go** - Defines interfaces:
   - `<Name>Service` interface extending `domain.DomainService`
   - Repository interface for data access
   - Client interfaces for external dependencies

2. **impl.go** - Implements the service

3. **handler.go** - REST handlers (for services that expose HTTP endpoints)

4. Domain services receive dependencies via constructor injection:
   ```go
   // SQL Governance needs Agent client (via RPC)
   sqlGovernanceService := sqlgovernance.NewService(repo, agentClient)

   // Performance needs Threshold client (direct call, same process)
   performanceService := performance.NewService(repo, thresholdClient)
   ```

## Data Query Service (REST API)

Data Query Service provides a REST API for time-series queries:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/endpoints` | GET | Get all distinct endpoints |
| `/api/v1/metrics?endpoint=<ep>` | GET | Get metrics for an endpoint |
| `/api/v1/series?...` | GET | Query series with filters |
| `/api/v1/series/:id` | GET | Get series by ID |
| `/api/v1/series/query` | POST | Complex query with JSON body |
| `/swagger/index.html` | GET | Swagger UI documentation |

**Handler**: `pkg/domain/dataquery/handler.go`
**Service**: `pkg/domain/dataquery/service.go`
**Repository**: `pkg/domain/dataquery/pg_repository.go`

## RPC Client Pattern

Domain services communicate with Agent and Task Engine via RPC:

1. **pkg/rpc/<service>/client.go** - Low-level gRPC client wrapper
2. **pkg/rpc/adapter/<service>_adapter.go** - Adapts RPC client to domain interface

Example flow for SQL Governance → Agent:
```
SQLGovernanceService
  └── ExecutionAgentClient (interface defined in sqlgovernance/service.go)
        └── AgentClientAdapter (implements interface, calls RPC)
              └── AgentClient (wraps gRPC calls)
```

## Key File Locations

- **Domain interfaces**: `pkg/domain/<name>/service.go`
- **Domain implementations**: `pkg/domain/<name>/impl.go`
- **REST handlers**: `pkg/domain/<name>/handler.go` or `pkg/api/handler/handler.go`
- **Routes**: `pkg/api/router/router.go`
- **Middleware**: `pkg/api/middleware/` (auth, RBAC, audit, multi-tenant)
- **RPC clients**: `pkg/rpc/<agent|task>/client.go`
- **Proto definitions**: `api/proto/<service>/<service>.proto`
- **Generated proto Go**: `api/proto/<service>/<service>.pb.go`
- **Configuration**: `configs/config.yaml`
- **Swagger docs**: `docs/` (auto-generated)

## Service Initialization Flow

In `cmd/<service>/main.go`:
1. Load config → Init logger
2. Initialize database connections (pgxpool for TimescaleDB)
3. Create repository implementations
4. Initialize domain services with their dependencies
5. Create REST handlers
6. Register routes and start Hertz server

## Multi-tenancy Context

All domain operations receive `*domain.DomainContext` containing:
- `TenantID`, `UserID`, `RequestID`, `DatabaseID`
- Underlying `context.Context` via `Ctx` field

Middleware extracts tenant info from JWT and injects into request context.

## Testing

### Test Structure

```
test/
└── integration/
    ├── query_test.go     # Data Query Service tests
    └── gateway_test.go   # Gateway E2E tests (curl-based)

pkg/api/handler/handler_test.go    # Handler layer tests
pkg/api/middleware/middleware_test.go # Middleware tests
pkg/api/router/router_test.go      # Router tests
pkg/domain/dataquery/*_test.go     # Domain service tests
```

### Running Tests

```bash
# Unit tests
go test ./pkg/api/... -v
go test ./pkg/domain/dataquery/... -v

# Integration tests (requires running services)
go test ./test/integration/... -v
```

### Mock Services

Tests use mock implementations of domain services:
- `mockSQLGovernanceService`
- `mockPerformanceService`
- `mockThresholdService`
- `mockLLMService`

These mocks implement the corresponding service interfaces and allow testing the gateway layer in isolation.

## Scripts Organization

```
scripts/
├── build/                 # Build and code generation
│   ├── build.sh          # Build all services
│   └── generate_proto.sh # Generate protobuf code
├── db/                    # Database management
│   ├── db-data.sh        # Manage test data (seed/clear/reset/status)
│   ├── init-extensions.sql # Initialize TimescaleDB/pgvector/PGMQ
│   └── insert_test_data.go # Insert test data
└── dev/                   # Development utilities
    └── services.sh       # Start/stop/restart services
```