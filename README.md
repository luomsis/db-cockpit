# Database Intelligent Cockpit

A comprehensive database management and optimization platform built with Golang, Hertz framework, and microservices architecture.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      Interaction Layer                           │
│  ┌──────────────────┐  ┌──────────────────┐                    │
│  │   Dashboard UI   │  │  Chat Assistant  │                    │
│  └────────┬─────────┘  └────────┬─────────┘                    │
└───────────┼─────────────────────┼───────────────────────────────┘
            │                     │
┌───────────┼─────────────────────┼───────────────────────────────┐
│           │    Access Layer     │                               │
│  ┌────────▼─────────────────────▼────────┐                     │
│  │           API Gateway (Hertz)          │                    │
│  │  • Authentication & RBAC               │                    │
│  │  • Multi-tenant Support                │                    │
│  │  • Audit Logging                       │                    │
│  └────────────────┬───────────────────────┘                    │
└───────────────────┼─────────────────────────────────────────────┘
                    │
┌───────────────────┼─────────────────────────────────────────────┐
│                   │   Domain Layer                              │
│  ┌────────────────▼────────────────┐                           │
│  │       Domain Services           │                           │
│  │  ┌──────────────┐ ┌────────────┐│                           │
│  │  │ Data Query   │ │SQL Govern- ││                           │
│  │  │  (GraphQL)   │ │   ance     ││                           │
│  │  └──────────────┘ └────────────┘│                           │
│  │  ┌──────────────┐ ┌────────────┐│                           │
│  │  │ Performance  │ │  Dynamic   ││                           │
│  │  │  Diagnosis   │ │ Threshold  ││                           │
│  │  └──────────────┘ └────────────┘│                           │
│  │  ┌──────────────────────────────┐                           │
│  │  │      LLM Orchestrator        │                           │
│  │  └──────────────────────────────┘                           │
│  └─────────────────────────────────┘                           │
└───────────────────┬─────────────────────────────────────────────┘
                    │ RPC
┌───────────────────┼─────────────────────────────────────────────┐
│                   │   Independent Components                    │
│  ┌────────────────▼──────┐  ┌────────────────┐                 │
│  │   Execution Agent     │  │  Task Engine   │                 │
│  │  • SQL Execution      │  │  • Async Tasks │                 │
│  │  • API Calls          │  │  • Scheduling  │                 │
│  │  • Audit Trail        │  │  • Retries     │                 │
│  └───────────────────────┘  └────────────────┘                 │
│  ┌───────────────────────┐                                    │
│  │      Collector        │                                    │
│  │  • Metrics Collection │                                    │
│  └───────────┬───────────┘                                    │
└──────────────┼──────────────────────────────────────────────────┘
               │
┌──────────────┼──────────────────────────────────────────────────┐
│              │   Data Layer                                    │
│  ┌───────────▼───────────┐  ┌──────────────┐                  │
│  │     TimescaleDB       │  │    Redis     │                  │
│  │   (Time-series DB)    │  │   (Cache)    │                  │
│  └───────────────────────┘  └──────────────┘                  │
│  ┌───────────────────────┐  ┌──────────────┐                  │
│  │       Neo4j           │  │   pgvector   │                  │
│  │    (Graph Store)      │  │(Vector Store)│                  │
│  └───────────────────────┘  └──────────────┘                  │
│  ┌───────────────────────┐                                    │
│  │        PGMQ           │                                    │
│  │   (Message Queue)     │                                    │
│  └───────────────────────┘                                    │
└─────────────────────────────────────────────────────────────────┘
```

## Communication Protocols

| Path | Protocol |
|------|----------|
| Frontend ↔ Gateway | RESTful API |
| Frontend ↔ Data Query | GraphQL (`/graphql`) |
| Domain Layer ↔ Agent | RPC (gRPC) |
| Domain Layer ↔ Task Engine | RPC (gRPC) |
| Domain Layer internal | Direct function calls |

## Project Structure

```
db-cockpit/
├── api/proto/              # Protobuf definitions (Agent, Task only)
│   ├── agent/             # Execution Agent RPC
│   └── task/              # Task Engine RPC
├── cmd/                    # Entry points
│   ├── gateway/           # API Gateway main
│   ├── dataquery/         # Data Query Service main
│   ├── collector/         # Collector main
│   ├── agent/             # Execution Agent main
│   └── taskengine/        # Task Engine main
├── pkg/
│   ├── api/               # API Gateway layer
│   │   ├── handler/       # Request handlers
│   │   ├── middleware/    # Auth, RBAC, Audit middleware
│   │   └── router/        # Route registration
│   ├── domain/            # Domain Layer
│   │   ├── dataquery/     # Data Query (GraphQL)
│   │   │   ├── service.go       # Service interface
│   │   │   ├── impl.go          # Service implementation
│   │   │   ├── repository.go    # Repository interface
│   │   │   ├── pg_repository.go # PostgreSQL/TimescaleDB impl
│   │   │   ├── models.go        # Domain models
│   │   │   ├── labels/          # Label expression parser
│   │   │   │   ├── ast.go       # AST types
│   │   │   │   ├── parser.go    # Parser implementation
│   │   │   │   └── sql.go       # SQL translation
│   │   │   └── graph/           # GraphQL schema & resolvers
│   │   ├── sqlgovernance/ # SQL Governance
│   │   ├── performance/   # Performance Diagnosis
│   │   ├── threshold/     # Dynamic Threshold
│   │   └── llm/          # LLM Orchestrator
│   ├── agent/             # Execution Agent
│   ├── task/              # Task Engine
│   ├── collector/         # Collector
│   ├── rpc/               # RPC clients
│   │   ├── agent/         # Agent RPC client
│   │   ├── task/          # Task RPC client
│   │   └── adapter/       # Domain interface adapters
│   ├── data/              # Data Layer access
│   └── common/            # Common utilities
├── configs/               # Configuration files
├── scripts/               # Build and test scripts
│   ├── build/             # Build and code generation
│   │   ├── build.sh       # Build all services
│   │   └── generate_proto.sh # Generate protobuf code
│   ├── db/                # Database management
│   │   ├── db-data.sh     # Database data management (seed/clear/reset)
│   │   ├── init-extensions.sql # Initialize TimescaleDB/pgvector/PGMQ
│   │   └── insert_test_data.go # Test data insertion script
│   └── dev/               # Development utilities
│       └── services.sh    # Service management (start/stop/status)
├── test/                  # Tests
│   └── integration/       # Integration tests
│       ├── query_test.go  # Data query tests
│       └── gateway_test.go # Gateway E2E tests
├── web/                   # Frontend
│   └── dashboard/         # Next.js Dashboard
│       ├── app/           # Pages and layouts
│       ├── components/    # React components
│       ├── lib/           # Utilities and API clients
│       └── types/         # TypeScript types
└── docs/                  # Documentation
```

## Technology Stack

- **Framework**: Hertz (CloudWeGo)
- **RPC**: Kitex with Protobuf
- **GraphQL**: gqlgen
- **Databases**:
  - TimescaleDB (PostgreSQL extension for time-series)
  - Neo4j (Graph database)
  - pgvector (Vector similarity search)
- **Cache**: Redis
- **Message Queue**: PGMQ (PostgreSQL-based)
- **Testing**: Testify, GoMock

## Database Schema

### Time-Series Tables (TimescaleDB)

```sql
-- Series metadata
CREATE TABLE series_meta (
    id BIGSERIAL PRIMARY KEY,
    endpoint TEXT NOT NULL,        -- e.g., "/api/metrics"
    metric TEXT NOT NULL,          -- e.g., "cpu_usage"
    labels JSONB NOT NULL,         -- e.g., {"host": "server1", "region": "us-east"}
    labels_hash TEXT NOT NULL,     -- MD5 hash of labels
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(endpoint, metric, labels_hash)
);

-- Series data points (TimescaleDB hypertable)
CREATE TABLE series_points (
    "time" TIMESTAMPTZ NOT NULL,
    series_id BIGINT NOT NULL REFERENCES series_meta(id),
    value DOUBLE PRECISION NOT NULL
);
-- Automatically converted to hypertable by TimescaleDB
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Node.js 18+ (for frontend)
- Protobuf compiler (protoc)

### Using Service Management Scripts

The project includes convenient scripts for managing all services:

#### Start/Stop All Services

```bash
# Start all services (Data Query, Gateway, Frontend)
# Note: Ensure PostgreSQL/TimescaleDB is running first
./scripts/dev/services.sh start

# Stop all services
./scripts/dev/services.sh stop

# Restart all services
./scripts/dev/services.sh restart

# Check service status
./scripts/dev/services.sh status

# View logs for a specific service
./scripts/dev/services.sh logs gateway
./scripts/dev/services.sh logs dataquery
./scripts/dev/services.sh logs frontend
```

#### Manage Individual Services

```bash
# Data Query Service
./scripts/dev/services.sh start-dataquery
./scripts/dev/services.sh stop-dataquery

# Gateway Service
./scripts/dev/services.sh start-gateway
./scripts/dev/services.sh stop-gateway

# Frontend (Next.js)
./scripts/dev/services.sh start-frontend
./scripts/dev/services.sh stop-frontend
```

> **Note**: PostgreSQL/TimescaleDB should be running before starting services. Use `docker-compose up -d` or your preferred method to start the database.

#### Manage Test Data

```bash
# Check database data status
./scripts/db/db-data.sh status

# Insert test data
./scripts/db/db-data.sh seed

# Clear all data
./scripts/db/db-data.sh clear

# Reset data (clear + seed)
./scripts/db/db-data.sh reset

# Quick test GraphQL endpoints
./scripts/db/db-data.sh test
```

### Service Endpoints

After starting all services, the following endpoints are available:

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | http://localhost:3000 | Dashboard UI |
| Gateway GraphQL | http://localhost:8080/graphql | GraphQL API |
| Gateway Playground | http://localhost:8080/graphql/playground | GraphQL IDE |
| Data Query GraphQL | http://localhost:8084/graphql | Direct GraphQL access |

### Run with Docker

```bash
# Start infrastructure
docker-compose up -d

# Run Data Query Service (must start before Gateway)
go run cmd/dataquery/main.go

# Run API Gateway
go run cmd/gateway/main.go

# Run Collector
go run cmd/collector/main.go

# Run Execution Agent
go run cmd/agent/main.go

# Run Task Engine
go run cmd/taskengine/main.go
```

### Generate Protobuf Code

```bash
chmod +x scripts/build/generate_proto.sh
./scripts/build/generate_proto.sh
```

### Insert Test Data

Insert sample time-series data for testing:

```bash
# With default database URL (postgres://postgres:postgres@localhost:5432/dbcockpit)
go run scripts/db/insert_test_data.go

# With custom database URL
DATABASE_URL="postgres://user:pass@host:port/db" go run scripts/db/insert_test_data.go
```

This inserts:
- 3 endpoints: `/api/metrics`, `/api/health`, `/api/query`
- Multiple metrics per endpoint (cpu_usage, memory_usage, etc.)
- 4 label combinations across different hosts, regions, environments
- ~288 data points per series (24 hours at 5-minute intervals)

## API Endpoints

### GraphQL (Data Query Service)

**Endpoint**: `POST /graphql`

The Data Query Service provides comprehensive time-series query capabilities through GraphQL.

#### Queries

| Query | Description |
|-------|-------------|
| `endpoints` | Get all distinct endpoints |
| `metrics(endpoint: String!)` | Get all metrics for an endpoint |
| `series(...)` | Query time-series data with filters |
| `seriesById(id: ID!, ...)` | Get a single series by ID |
| `seriesMulti(...)` | Query multiple series at once |

#### Example Queries

```graphql
# Get all endpoints
query {
  endpoints
}

# Get metrics for an endpoint
query {
  metrics(endpoint: "/api/metrics")
}

# Query series with time range
query($tr: TimeRangeInput!) {
  series(
    endpoint: "/api/metrics"
    metric: "cpu_usage"
    timeRange: $tr
    limit: 10
  ) {
    meta {
      id
      endpoint
      metric
      labels {
        entries { key value }
      }
    }
    points {
      time
      value
    }
    statistics {
      min max avg sum count
    }
  }
}

# Query with label filter (PromQL-style)
query($tr: TimeRangeInput!) {
  series(
    labels: { expression: "host=\"server1\" AND region=\"us-east\"" }
    timeRange: $tr
  ) {
    meta { id metric }
    points { time value }
  }
}

# Query with aggregation
query($tr: TimeRangeInput!) {
  seriesMulti(
    endpoints: ["/api/metrics"]
    metrics: ["cpu_usage", "memory_usage"]
    timeRange: $tr
    aggregation: { interval: "5m", function: AVG }
  ) {
    meta { id metric }
    aggregatedPoints { time value count }
  }
}
```

#### Label Filter Syntax

Supports PromQL-style label expressions:

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Exact match | `host="server1"` |
| `!=` | Not equal | `host!="localhost"` |
| `=~` | Regex match | `region=~"us-.*"` |
| `!~` | Regex not match | `region!~"eu-.*"` |
| `AND` | Logical AND | `host="server1" AND region="us-east"` |
| `OR` | Logical OR | `host="server1" OR host="server2"` |
| `()` | Grouping | `(host="s1" OR host="s2") AND env="prod"` |

#### Aggregation Functions

- `AVG` - Average value
- `MIN` - Minimum value
- `MAX` - Maximum value
- `SUM` - Sum of values
- `COUNT` - Count of points

#### Time Intervals

Supports PostgreSQL interval format: `1m`, `5m`, `1h`, `1d`, etc.

**Development Tools**:
- `GET /graphql/playground` - GraphQL playground for testing queries

### REST API

#### SQL Governance
- `POST /api/v1/sql/review` - Review SQL before execution
- `POST /api/v1/sql/execute` - Execute SQL with governance
- `GET /api/v1/sql/audit` - Get SQL audit trail

#### Performance
- `POST /api/v1/performance/diagnose` - Run performance diagnosis
- `POST /api/v1/performance/metrics` - Get performance metrics
- `POST /api/v1/performance/slow-queries` - Get slow queries

#### Thresholds
- `GET /api/v1/thresholds` - Get thresholds
- `PUT /api/v1/thresholds` - Update threshold

#### LLM
- `POST /api/v1/llm/chat` - Chat with AI assistant
- `POST /api/v1/llm/generate-sql` - Generate SQL from natural language
- `GET /api/v1/llm/recommendations` - Get intelligent recommendations

## Domain Layer Flow

### Domain → Execution Agent
```
Domain Layer (SQL Governance)
    │
    ├── Review SQL
    │
    ├── Call Execution Agent
    │   └── ExecuteSQL(ctx, request)
    │
    └── Return Result with Audit ID
```

### Domain → Task Engine
```
Domain Layer (Performance)
    │
    ├── Submit Async Task
    │   └── SubmitTask(ctx, &Task{
    │       TaskType: TaskTypeDiagnosis,
    │       Payload: {...},
    │   })
    │
    └── Task Engine processes via MQ
        └── Handler executes task
            └── Result stored in TSDB
```

## Development

### Add New Domain Service

1. Create service interface in `pkg/domain/<domain>/service.go`
2. Implement service in `pkg/domain/<domain>/impl.go`
3. Add REST handlers in `pkg/api/handler/handler.go`
4. Register routes in `pkg/api/router/router.go`

### Add New RPC Service (for Agent/Task Engine communication)

1. Create proto file in `api/proto/<service>/`
2. Run `./scripts/generate_proto.sh` to generate Go code
3. Create RPC client in `pkg/rpc/<service>/client.go`
4. Create adapter in `pkg/rpc/adapter/<service>_adapter.go`

### Add New Task Handler

```go
type MyTaskHandler struct{}

func (h *MyTaskHandler) Handle(ctx context.Context, task *task.Task) (*task.TaskResult, error) {
    // Process task
    return &task.TaskResult{Data: result}, nil
}

func (h *MyTaskHandler) TaskType() task.TaskType {
    return task.TaskType("my_task")
}

// Register in main
taskEngine.RegisterHandler(&MyTaskHandler{})
```

## Testing

### Unit Tests

```bash
# Run all unit tests
go test ./...

# Run tests with verbose output
go test -v ./pkg/...

# Run tests for specific package
go test ./pkg/api/handler/...
go test ./pkg/domain/dataquery/...

# Run tests with coverage
go test -cover ./...
```

### Integration Tests

Integration tests verify the complete flow: `frontend → gateway → dataquery → database`.

```bash
# Prerequisites: Start services first
go run cmd/dataquery/main.go &
go run cmd/gateway/main.go &

# Run integration tests
go test -v ./test/integration/...
```

### Test Coverage

| Package | Coverage |
|---------|----------|
| pkg/api/handler | Handler layer tests with mocked domain services |
| pkg/api/middleware | Auth, CORS, RequestID, Audit middleware tests |
| pkg/api/router | Route registration and middleware chain tests |
| pkg/domain/dataquery | Service, repository, label parser tests |
| test/integration | End-to-end integration tests |

## Configuration

See `configs/config.yaml` for all configuration options.

## Documentation

### Architecture & Design

| Document | Description |
|----------|-------------|
| [Architecture Diagram Prompt](docs/architecture-diagram-prompt.md) | Prompts for generating system architecture diagrams |
| [GraphQL API Guide](docs/graphql-api-guide.md) | Comprehensive GraphQL API usage guide |

### Specifications

| Document | Date | Description |
|----------|------|-------------|
| [DataQuery GraphQL Design](docs/superpowers/specs/2024-03-20-dataquery-graphql-design.md) | 2024-03-20 | GraphQL query interface design |
| [Monitoring Dashboard Design](docs/superpowers/specs/2026-03-22-monitoring-dashboard-design.md) | 2026-03-22 | Frontend monitoring dashboard design |
| [DataQuery REST API Design](docs/superpowers/specs/2026-03-25-dataquery-rest-api-design.md) | 2026-03-25 | REST API design for DataQuery service |

### Implementation Plans

| Document | Date | Description |
|----------|------|-------------|
| [Monitoring Dashboard Implementation](docs/superpowers/plans/2026-03-22-monitoring-dashboard.md) | 2026-03-22 | Step-by-step dashboard implementation |
| [DataQuery REST API Implementation](docs/superpowers/plans/2026-03-25-dataquery-rest-api.md) | 2026-03-25 | REST API migration implementation |

### API Reference

| Document | Description |
|----------|-------------|
| [Swagger UI](http://localhost:8084/swagger/index.html) | Interactive API documentation (when service running) |
| [swagger.json](docs/swagger.json) | OpenAPI 2.0 specification (JSON) |
| [swagger.yaml](docs/swagger.yaml) | OpenAPI 2.0 specification (YAML) |

## License

MIT License