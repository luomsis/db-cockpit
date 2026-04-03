# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 工作原则

以第一性原理从原始需求和问题本质出发，不从惯例或模板出发。

1. **不要假设我清楚自己想要什么。** 动机或目标不清晰时，停下来讨论。
2. **目标清晰但路径不是最短的，直接告诉我并建议更好的办法。**
3. **遇到问题追根因，不打补丁。** 每个决策都要能回答"为什么"。
4. **输出说重点，砍掉一切不改变决策的信息。**

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
| `/api/v1/instances` | GET | Get all instances with pagination |
| `/api/v1/instances/:endpoint` | GET | Get instance by endpoint |
| `/api/v1/alerts/:endpoint` | GET | Get alerts for an endpoint |
| `/swagger/index.html` | GET | Swagger UI documentation |

**Handler**: `pkg/domain/dataquery/handler.go`
**Service**: `pkg/domain/dataquery/service.go`
**Repository**: `pkg/domain/dataquery/pg_repository.go`

### Label Filter Syntax

Supports PromQL-style label expressions for `/series` endpoint:

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Exact match | `host="server1"` |
| `!=` | Not equal | `host!="localhost"` |
| `=~` | Regex match | `region=~"us-.*"` |
| `!~` | Regex not match | `region!~"eu-.*"` |
| `AND` | Logical AND | `host="server1" AND region="us-east"` |
| `OR` | Logical OR | `host="server1" OR host="server2"` |
| `()` | Grouping | `(host="s1" OR host="s2") AND env="prod"` |

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
    ├── dataquery_http_test.go  # Data Query HTTP integration tests
    ├── query_test.go            # Data Query Service tests
    └── gateway_test.go          # Gateway E2E tests (curl-based)

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

# Run a single test
go test ./pkg/domain/dataquery/... -v -run TestServiceQuerySeries
go test ./test/integration/... -v -run TestDataQueryHTTP_GetSeries

# Integration tests (requires running services)
go test ./test/integration/... -v
```

### Integration Test Database

**IMPORTANT**: Integration tests use the existing `postgres` database, NOT a separate `cockpit` database.

Database connection for integration tests:
- Host: `localhost:5432`
- User: `postgres`
- Password: `postgres`
- Database: `postgres`

The `postgres` database already contains the required tables (`series_meta`, `series_points`, etc.) and test data. Do NOT create a new database for integration tests.

Connection command:
```bash
PGPASSWORD=postgres psql -h localhost -U postgres -d postgres
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
└── dev/                   # Development utilities
    └── services.sh       # Start/stop/restart services
```

### Service Management

```bash
# Start/stop all services
./scripts/dev/services.sh start
./scripts/dev/services.sh stop
./scripts/dev/services.sh status

# Manage individual services
./scripts/dev/services.sh start dataquery
./scripts/dev/services.sh start gateway
```

## API Development Workflow

When adding or modifying API endpoints in the Data Query Service:

1. Update handler with swagger annotations (@Summary, @Description, @Tags, @Router, etc.)
2. Update `docs/swagger.json` with new endpoint path and response definitions
3. Run tests: `go test ./pkg/domain/dataquery/... -v`
4. Verify Swagger UI at `/swagger/index.html` displays the new endpoint

## Frontend

Next.js dashboard located at `web/dashboard/`:

```
web/dashboard/
├── app/           # Pages and layouts
├── components/    # React components
├── lib/           # Utilities and API clients
└── types/         # TypeScript types
```

Start frontend: `./scripts/dev/services.sh start frontend` or `cd web/dashboard && npm run dev`