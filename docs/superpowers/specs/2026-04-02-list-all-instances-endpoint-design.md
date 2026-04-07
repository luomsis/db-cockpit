---
name: list-all-instances-endpoint
description: Add GET /api/v1/instances endpoint to list all database instances with full metadata
type: project
---

# List All Instances Endpoint Design

## Context

The Data Query Service currently has `GET /instances/:endpoint` to retrieve a single instance by endpoint name, but lacks an endpoint to list all instances. This design adds a new endpoint to return all database instance metadata.

**Why**: Frontend needs to display a list of all available instances for users to select and explore metrics.

**How to apply**: When implementing API changes, always sync swagger documentation and run tests.

## Endpoint Specification

- **Path**: `GET /api/v1/instances`
- **Description**: Returns all database instances with full metadata
- **Response**: `InstancesListResponse` containing array of `InstanceMeta`

```json
{
  "data": [
    {
      "id": 1,
      "db_type": "mysql",
      "entity_name": "finance-order",
      "instance_endpoint": "mysql-cn-east-1-finance-order-01",
      "instance_vip": "10.0.1.100",
      "instance_port": 3306,
      "status": "active",
      "environment": "prod",
      ...
    },
    ...
  ]
}
```

## Implementation Plan

### 1. Repository Interface (`repository.go`)

Add method to Repository interface:

```go
type Repository interface {
    // ... existing methods
    GetAllInstances(ctx context.Context) ([]*InstanceMeta, error)
}
```

### 2. Repository Implementation (`pg_repository.go`)

Add method to query all instances:

```go
func (r *PGRepository) GetAllInstances(ctx context.Context) ([]*InstanceMeta, error) {
    rows, err := r.pool.Query(ctx, `
        SELECT id, db_type, entity_name, chinese_desc, org_code, service_user, opr_dba,
               business_owner, alert_subscriber, infra_type, req_cpu, req_memory_gb,
               req_storage_gb, created_date, environment, opr_dba_ii, ins_created_date,
               ins_updated_date, host_environment1, host_environment2, le_name,
               instance_endpoint, subsys_code, source_sys, attach_db, host_namel,
               host_name2, default_role, "role", status, version_detail, instance_name,
               is_created_by_cloud, character_set, instance_vip, instance_port, user_name,
               host_ip1, host_infra_type1, os_name, host_ip2, host_infra_type2,
               ha_type, backup_method, failover_type, ins_uuid, ccm_name
        FROM instance_meta
        ORDER BY instance_endpoint
    `)
    // ... scan logic similar to GetInstanceByEndpoint
}
```

### 3. Service Layer (`service.go`)

Add interface method and implementation:

```go
// Interface addition
type DataQueryService interface {
    // ... existing methods
    GetAllInstances(ctx context.Context) ([]*InstanceMeta, error)
}

// Implementation
func (s *Service) GetAllInstances(ctx context.Context) ([]*InstanceMeta, error) {
    return s.repo.GetAllInstances(ctx)
}
```

### 4. Handler Layer (`handler.go`)

Add response DTO and handler with swagger annotations:

```go
// InstancesListResponse is the response for GetInstances
type InstancesListResponse struct {
    Data []*InstanceMeta `json:"data"`
}

// GetInstances handles GET /instances requests
// @Summary Get all instances
// @Description Get all database instances with full metadata
// @Tags instances
// @Produce json
// @Success 200 {object} InstancesListResponse
// @Failure 500 {object} ErrorResponse
// @Router /instances [get]
func (h *Handler) GetInstances(ctx context.Context, c *app.RequestContext) {
    instances, err := h.service.GetAllInstances(ctx)
    if err != nil {
        c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
        return
    }
    c.JSON(200, InstancesListResponse{Data: instances})
}
```

### 5. Route Registration (`main.go`)

Add route to API group:

```go
api.GET("/instances", func(c context.Context, ctx *app.RequestContext) {
    handler.GetInstances(c, ctx)
})
```

### 6. Swagger Update (`swagger.json`)

Add new path and definition:

```json
{
  "paths": {
    "/instances": {
      "get": {
        "description": "Get all database instances with full metadata",
        "produces": ["application/json"],
        "tags": ["instances"],
        "summary": "Get all instances",
        "responses": {
          "200": {
            "description": "OK",
            "schema": {"$ref": "#/definitions/dataquery.InstancesListResponse"}
          },
          "500": {
            "description": "Internal Server Error",
            "schema": {"$ref": "#/definitions/dataquery.ErrorResponse"}
          }
        }
      }
    }
  },
  "definitions": {
    "dataquery.InstancesListResponse": {
      "type": "object",
      "properties": {
        "data": {
          "type": "array",
          "items": {"$ref": "#/definitions/dataquery.InstanceMeta"}
        }
      }
    }
  }
}
```

### 7. Testing (`handler_test.go`)

Add unit tests:

```go
func TestGetInstances(t *testing.T) {
    mockService := &mockDataQueryService{
        instances: []*InstanceMeta{
            {ID: 1, DbType: "mysql", InstanceEndpoint: "mysql-test-01"},
        },
    }
    handler := NewHandler(mockService)
    ctx, reqCtx := createTestRequestContext("")
    handler.GetInstances(ctx, reqCtx)
    // assert status 200 and response structure
}

func TestGetInstances_ServiceError(t *testing.T) {
    mockService := &mockDataQueryService{err: context.Canceled}
    handler := NewHandler(mockService)
    ctx, reqCtx := createTestRequestContext("")
    handler.GetInstances(ctx, reqCtx)
    // assert status 500
}
```

### 8. Update CLAUDE.md

Add documentation requirement:

```markdown
## API Development Workflow

When adding or modifying API endpoints in the Data Query Service:

1. Update handler with swagger annotations
2. Update `docs/swagger.json` with new endpoint and definitions
3. Run tests: `go test ./pkg/domain/dataquery/... -v`
4. Verify Swagger UI at `/swagger/index.html`
```

## Verification Steps

1. Run unit tests: `go test ./pkg/domain/dataquery/... -v`
2. Build service: `go build ./cmd/dataquery`
3. Start service: `go run cmd/dataquery/main.go`
4. Test endpoint: `curl http://localhost:8084/api/v1/instances`
5. Check Swagger UI: `http://localhost:8084/swagger/index.html`

## Files Modified

| File | Change Type |
|------|-------------|
| pkg/domain/dataquery/repository.go | Add method to interface |
| pkg/domain/dataquery/service.go | Add method |
| pkg/domain/dataquery/pg_repository.go | Add method |
| pkg/domain/dataquery/handler.go | Add DTO and handler |
| cmd/dataquery/main.go | Add route |
| docs/swagger.json | Add path and definition |
| pkg/domain/dataquery/handler_test.go | Add tests |
| CLAUDE.md | Add workflow documentation |