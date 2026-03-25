# DataQuery Service: GraphQL to REST API Migration

**Date:** 2026-03-25
**Status:** Draft

## Summary

Convert the DataQuery service from GraphQL to REST API, removing all GraphQL dependencies and code. The service will expose RESTful endpoints using the Hertz framework for consistency with the Gateway service.

## Goals

- Replace GraphQL API with REST API
- Maintain all existing functionality
- Keep consistent with Gateway service architecture (Hertz framework)
- Update frontend to use REST API calls

## API Design

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/endpoints` | List all distinct endpoints |
| GET | `/api/v1/metrics` | List metrics for an endpoint (query param: `endpoint`) |
| GET | `/api/v1/series` | Query series with filters |
| GET | `/api/v1/series/:id` | Get series by ID with data points |
| POST | `/api/v1/series/query` | Complex multi-series query with aggregation |

### Detailed Specifications

#### GET /api/v1/endpoints

**Response:**
```json
{
  "data": ["/api/metrics", "/api/users", "/api/orders"]
}
```

#### GET /api/v1/metrics?endpoint=/api/metrics

**Query Parameters:**
- `endpoint` (required): The endpoint to get metrics for

**Response:**
```json
{
  "data": ["cpu_usage", "memory_usage", "request_latency"]
}
```

#### GET /api/v1/series

**Query Parameters:**
- `endpoint` (optional): Filter by endpoint
- `metric` (optional): Filter by metric name
- `labels` (optional): Label filter expression (e.g., `job="prometheus",instance="localhost:9090"`)
- `start` (required): Start time (RFC3339 format)
- `end` (required): End time (RFC3339 format)
- `limit` (optional): Maximum number of series to return (default: 10)

**Response:**
```json
{
  "data": [
    {
      "meta": {
        "id": "1",
        "endpoint": "/api/metrics",
        "metric": "cpu_usage",
        "labels": {
          "job": "prometheus",
          "instance": "localhost:9090"
        },
        "labels_hash": "abc123",
        "created_at": "2024-01-01T00:00:00Z"
      },
      "points": [
        {"time": "2024-01-01T00:00:00Z", "value": 75.5},
        {"time": "2024-01-01T00:01:00Z", "value": 78.2}
      ],
      "aggregated_points": null,
      "statistics": null
    }
  ]
}
```

#### GET /api/v1/series/:id

**Path Parameters:**
- `id`: Series ID (int64)

**Query Parameters:**
- `start` (required): Start time (RFC3339 format)
- `end` (required): End time (RFC3339 format)

**Response:**
```json
{
  "data": {
    "meta": {
      "id": "1",
      "endpoint": "/api/metrics",
      "metric": "cpu_usage",
      "labels": {"job": "prometheus"},
      "labels_hash": "abc123",
      "created_at": "2024-01-01T00:00:00Z"
    },
    "points": [
      {"time": "2024-01-01T00:00:00Z", "value": 75.5}
    ],
    "aggregated_points": null,
    "statistics": {
      "min": 50.0,
      "max": 100.0,
      "avg": 75.5,
      "sum": 7550.0,
      "count": 100
    }
  }
}
```

#### POST /api/v1/series/query

**Request Body:**
```json
{
  "endpoints": ["/api/metrics"],
  "metrics": ["cpu_usage", "memory_usage"],
  "labels": "job=\"prometheus\"",
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T01:00:00Z",
  "aggregation": {
    "interval": "5m",
    "function": "AVG"
  }
}
```

**Aggregation Functions:** `AVG`, `MIN`, `MAX`, `SUM`, `COUNT`
**Aggregation Intervals:** `1m`, `5m`, `15m`, `1h`, `1d`

**Response:** Same format as GET /api/v1/series

### Error Response

All endpoints return consistent error format:

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "start time is required"
  }
}
```

**Error Codes:**
- `INVALID_PARAMETER` - Missing or invalid query parameter
- `NOT_FOUND` - Resource not found
- `INTERNAL_ERROR` - Server error

## Code Changes

### Backend (Go)

#### Delete
- `pkg/domain/dataquery/graph/` - entire GraphQL directory
- `pkg/domain/dataquery/generate.go` - gqlgen generate directive
- `pkg/domain/dataquery/gqlgen.yml` - gqlgen configuration file

#### Create
- `pkg/domain/dataquery/handler.go` - REST handlers using Hertz
- `pkg/domain/dataquery/handler_test.go` - handler unit tests

#### Modify
- `cmd/dataquery/main.go` - replace GraphQL server with Hertz REST server

#### Keep Unchanged
- `pkg/domain/dataquery/service.go` - service interface and implementation
- `pkg/domain/dataquery/service_test.go` - service tests
- `pkg/domain/dataquery/repository.go` - repository interface
- `pkg/domain/dataquery/pg_repository.go` - PostgreSQL implementation
- `pkg/domain/dataquery/models.go` - domain models
- `pkg/domain/dataquery/models_test.go` - model tests
- `pkg/domain/dataquery/labels/` - label parser

### Frontend (TypeScript)

#### Delete
- `web/dashboard/lib/graphql-client.ts` - GraphQL client
- `web/dashboard/lib/queries.ts` - GraphQL query definitions

#### Create
- `web/dashboard/lib/api-client.ts` - REST API client

#### Modify
- `web/dashboard/app/page.tsx` - use REST API calls
- `web/dashboard/package.json` - remove `graphql-request` dependency

## Implementation Notes

### Handler Implementation

Use Hertz framework patterns consistent with Gateway:

```go
// pkg/domain/dataquery/handler.go
package dataquery

import (
    "github.com/cloudwego/hertz/pkg/app"
)

type Handler struct {
    service DataQueryService
}

func NewHandler(service DataQueryService) *Handler {
    return &Handler{service: service}
}

// GET /api/v1/endpoints
func (h *Handler) GetEndpoints(ctx context.Context, c *app.RequestContext)

// GET /api/v1/metrics
func (h *Handler) GetMetrics(ctx context.Context, c *app.RequestContext)

// GET /api/v1/series
func (h *Handler) GetSeries(ctx context.Context, c *app.RequestContext)

// GET /api/v1/series/:id
func (h *Handler) GetSeriesByID(ctx context.Context, c *app.RequestContext)

// POST /api/v1/series/query
func (h *Handler) QuerySeries(ctx context.Context, c *app.RequestContext)
```

### Time Format

All timestamps use RFC3339 format (`2006-01-02T15:04:05Z07:00`).

### ID Format

Series IDs are int64 in the database but returned as strings in JSON for compatibility with JavaScript's number precision limits.

## Testing

- Unit tests for handler request/response transformation
- Integration tests for full API flow
- Update `test/integration/query_test.go` - currently tests GraphQL endpoint, needs REST API tests
- Update `test/integration/gateway_test.go` - remove GraphQL references if any

## Migration Path

1. Delete GraphQL code
2. Implement REST handlers
3. Update main.go to use Hertz
4. Update frontend to use REST API
5. Remove unused dependencies from go.mod and package.json