# List All Instances Endpoint Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /api/v1/instances` endpoint to list all database instances with full metadata.

**Architecture:** Follow existing Data Query Service patterns - add method to Repository interface, implement in PGRepository, add service method, create handler with swagger annotations, register route, update swagger.json, add tests.

**Tech Stack:** Go, Hertz HTTP framework, PostgreSQL/TimescaleDB, Swagger 2.0

---

## File Structure

| File | Responsibility |
|------|-----------------|
| `pkg/domain/dataquery/repository.go` | Repository interface definition |
| `pkg/domain/dataquery/pg_repository.go` | PostgreSQL implementation |
| `pkg/domain/dataquery/service.go` | Service interface and implementation |
| `pkg/domain/dataquery/handler.go` | HTTP handler with swagger annotations |
| `pkg/domain/dataquery/handler_test.go` | Unit tests |
| `cmd/dataquery/main.go` | Route registration |
| `docs/swagger.json` | API documentation |
| `CLAUDE.md` | Workflow documentation |

---

### Task 1: Repository Interface

**Files:**
- Modify: `pkg/domain/dataquery/repository.go`

- [ ] **Step 1: Add GetAllInstances method to Repository interface**

```go
// Add to Repository interface in repository.go (after GetInstanceByEndpoint)

// GetAllInstances retrieves all instance metadata
GetAllInstances(ctx context.Context) ([]*InstanceMeta, error)
```

- [ ] **Step 2: Verify interface compiles**

Run: `go build ./pkg/domain/dataquery/...`
Expected: Build succeeds (interface only, no implementation yet)

---

### Task 2: Repository Implementation

**Files:**
- Modify: `pkg/domain/dataquery/pg_repository.go`

- [ ] **Step 1: Implement GetAllInstances in PGRepository**

Add at end of `pg_repository.go`:

```go
// GetAllInstances retrieves all instance metadata
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
	if err != nil {
		return nil, fmt.Errorf("failed to query all instances: %w", err)
	}
	defer rows.Close()

	var instances []*InstanceMeta
	for rows.Next() {
		var instance InstanceMeta
		err := rows.Scan(
			&instance.ID, &instance.DbType, &instance.EntityName, &instance.ChineseDesc,
			&instance.OrgCode, &instance.ServiceUser, &instance.OprDba, &instance.BusinessOwner,
			&instance.AlertSubscriber, &instance.InfraType, &instance.ReqCPU, &instance.ReqMemoryGB,
			&instance.ReqStorageGB, &instance.CreatedDate, &instance.Environment, &instance.OprDbaII,
			&instance.InsCreatedDate, &instance.InsUpdatedDate, &instance.HostEnvironment1,
			&instance.HostEnvironment2, &instance.LeName, &instance.InstanceEndpoint,
			&instance.SubsysCode, &instance.SourceSys, &instance.AttachDb, &instance.HostNamel,
			&instance.HostName2, &instance.DefaultRole, &instance.Role, &instance.Status,
			&instance.VersionDetail, &instance.InstanceName, &instance.IsCreatedByCloud,
			&instance.CharacterSet, &instance.InstanceVip, &instance.InstancePort, &instance.UserName,
			&instance.HostIP1, &instance.HostInfraType1, &instance.OsName, &instance.HostIP2,
			&instance.HostInfraType2, &instance.HaType, &instance.BackupMethod, &instance.FailoverType,
			&instance.InsUUID, &instance.CcmName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}
		instances = append(instances, &instance)
	}

	if instances == nil {
		instances = []*InstanceMeta{}
	}
	return instances, nil
}
```

- [ ] **Step 2: Verify implementation compiles**

Run: `go build ./pkg/domain/dataquery/...`
Expected: Build succeeds

- [ ] **Step 3: Commit repository changes**

```bash
git add pkg/domain/dataquery/repository.go pkg/domain/dataquery/pg_repository.go
git commit -m "feat(dataquery): add GetAllInstances repository method"
```

---

### Task 3: Service Layer

**Files:**
- Modify: `pkg/domain/dataquery/service.go`

- [ ] **Step 1: Add GetAllInstances to DataQueryService interface**

Add to interface in `service.go` (after GetInstanceByEndpoint):

```go
// GetAllInstances retrieves all instance metadata
GetAllInstances(ctx context.Context) ([]*InstanceMeta, error)
```

- [ ] **Step 2: Implement GetAllInstances in Service struct**

Add at end of `service.go`:

```go
// GetAllInstances retrieves all instance metadata
func (s *Service) GetAllInstances(ctx context.Context) ([]*InstanceMeta, error) {
	return s.repo.GetAllInstances(ctx)
}
```

- [ ] **Step 3: Verify service compiles**

Run: `go build ./pkg/domain/dataquery/...`
Expected: Build succeeds

- [ ] **Step 4: Commit service changes**

```bash
git add pkg/domain/dataquery/service.go
git commit -m "feat(dataquery): add GetAllInstances service method"
```

---

### Task 4: Handler Layer

**Files:**
- Modify: `pkg/domain/dataquery/handler.go`

- [ ] **Step 1: Add InstancesListResponse DTO**

Add after `InstanceMetaResponse` in `handler.go`:

```go
// InstancesListResponse is the response for GetInstances
type InstancesListResponse struct {
	Data []*InstanceMeta `json:"data"`
}
```

- [ ] **Step 2: Add GetInstances handler with swagger annotations**

Add after `GetInstance` handler:

```go
// GetInstances handles GET /instances requests
// @Summary Get all instances
// @Description Get all database instances with full metadata
// @Tags instances
// @Produce json
// @Success 200 {object} InstancesListResponse
// @Failure 500 {object} ErrorResponse
// @Router /instances [get]
func (h *Handler) GetInstances(ctx context.Context, c *app.RequestContext) {
	logger.Debug("GetInstances called")

	instances, err := h.service.GetAllInstances(ctx)
	if err != nil {
		logger.Error("GetInstances failed", zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("GetInstances success", zap.Int("count", len(instances)))
	c.JSON(200, InstancesListResponse{Data: instances})
}
```

- [ ] **Step 3: Verify handler compiles**

Run: `go build ./pkg/domain/dataquery/...`
Expected: Build succeeds

- [ ] **Step 4: Commit handler changes**

```bash
git add pkg/domain/dataquery/handler.go
git commit -m "feat(dataquery): add GetInstances handler"
```

---

### Task 5: Route Registration

**Files:**
- Modify: `cmd/dataquery/main.go`

- [ ] **Step 1: Add GET /instances route**

Add to API group in `main.go` (before `/instances/:endpoint`):

```go
api.GET("/instances", func(c context.Context, ctx *app.RequestContext) {
	handler.GetInstances(c, ctx)
})
```

- [ ] **Step 2: Update printEndpoints function**

Add new endpoint to `printEndpoints` function:

```go
fmt.Printf("  GET  http://%s/api/v1/instances\n", addr)
```

Add example curl command:

```go
fmt.Print(`
  # Get all instances
  curl http://localhost:8084/api/v1/instances
`)
```

- [ ] **Step 3: Build and verify**

Run: `go build ./cmd/dataquery`
Expected: Build succeeds

- [ ] **Step 4: Commit route changes**

```bash
git add cmd/dataquery/main.go
git commit -m "feat(dataquery): register GET /instances route"
```

---

### Task 6: Swagger Documentation

**Files:**
- Modify: `docs/swagger.json`

- [ ] **Step 1: Add /instances path to swagger.json**

Add new path entry in `paths` section (before `/instances/{endpoint}`):

```json
"/instances": {
    "get": {
        "description": "Get all database instances with full metadata",
        "produces": [
            "application/json"
        ],
        "tags": [
            "instances"
        ],
        "summary": "Get all instances",
        "responses": {
            "200": {
                "description": "OK",
                "schema": {
                    "$ref": "#/definitions/dataquery.InstancesListResponse"
                }
            },
            "500": {
                "description": "Internal Server Error",
                "schema": {
                    "$ref": "#/definitions/dataquery.ErrorResponse"
                }
            }
        }
    }
}
```

- [ ] **Step 2: Add InstancesListResponse definition**

Add to `definitions` section:

```json
"dataquery.InstancesListResponse": {
    "type": "object",
    "properties": {
        "data": {
            "type": "array",
            "items": {
                "$ref": "#/definitions/dataquery.InstanceMeta"
            }
        }
    }
}
```

- [ ] **Step 3: Verify swagger.json is valid JSON**

Run: `python3 -c "import json; json.load(open('docs/swagger.json'))"` or use `jq`
Expected: No error (valid JSON)

- [ ] **Step 4: Commit swagger changes**

```bash
git add docs/swagger.json
git commit -m "docs(dataquery): add /instances endpoint to swagger"
```

---

### Task 7: Unit Tests

**Files:**
- Modify: `pkg/domain/dataquery/handler_test.go`

- [ ] **Step 1: Add instances field to mockDataQueryService**

Update mock struct:

```go
type mockDataQueryService struct {
	endpoints  []string
	metrics    []string
	series     []*SeriesData
	instances  []*InstanceMeta
	err        error
}
```

- [ ] **Step 2: Add GetAllInstances mock method**

Add to mock:

```go
func (m *mockDataQueryService) GetAllInstances(ctx context.Context) ([]*InstanceMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.instances, nil
}
```

- [ ] **Step 3: Write test for successful GetInstances**

```go
func TestGetInstances(t *testing.T) {
	mockService := &mockDataQueryService{
		instances: []*InstanceMeta{
			{
				ID:               1,
				DbType:           "mysql",
				EntityName:       "finance-order",
				InstanceEndpoint: "mysql-cn-east-1-finance-order-01",
				InstanceVip:      "10.0.1.100",
				InstancePort:     3306,
				Status:           "active",
			},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	handler.GetInstances(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetInstances() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp InstancesListResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Errorf("GetInstances() returned %d instances, want 1", len(resp.Data))
	}
	if resp.Data[0].InstanceEndpoint != "mysql-cn-east-1-finance-order-01" {
		t.Errorf("GetInstances() endpoint = %s, want mysql-cn-east-1-finance-order-01", resp.Data[0].InstanceEndpoint)
	}
}
```

- [ ] **Step 4: Write test for empty instances list**

```go
func TestGetInstances_EmptyList(t *testing.T) {
	mockService := &mockDataQueryService{
		instances: []*InstanceMeta{},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	handler.GetInstances(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetInstances() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp InstancesListResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("GetInstances() returned %d instances, want 0", len(resp.Data))
	}
}
```

- [ ] **Step 5: Write test for service error**

```go
func TestGetInstances_ServiceError(t *testing.T) {
	mockService := &mockDataQueryService{err: context.Canceled}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	handler.GetInstances(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetInstances() status = %d, want 500", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("GetInstances() error code = %s, want INTERNAL_ERROR", resp.Error.Code)
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/domain/dataquery/... -v`
Expected: All tests PASS

- [ ] **Step 7: Commit test changes**

```bash
git add pkg/domain/dataquery/handler_test.go
git commit -m "test(dataquery): add GetInstances handler tests"
```

---

### Task 8: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Add API Development Workflow section**

Add at end of `CLAUDE.md`:

```markdown
## API Development Workflow

When adding or modifying API endpoints in the Data Query Service:

1. Update handler with swagger annotations (@Summary, @Description, @Tags, @Router, etc.)
2. Update `docs/swagger.json` with new endpoint path and response definitions
3. Run tests: `go test ./pkg/domain/dataquery/... -v`
4. Verify Swagger UI at `/swagger/index.html` displays the new endpoint
```

- [ ] **Step 2: Commit documentation update**

```bash
git add CLAUDE.md
git commit -m "docs: add API development workflow with swagger sync requirement"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Run all tests**

Run: `go test ./pkg/domain/dataquery/... -v`
Expected: All tests PASS

- [ ] **Step 2: Build entire service**

Run: `go build ./cmd/dataquery`
Expected: Build succeeds

- [ ] **Step 3: Start service and test endpoint manually**

Run: `go run cmd/dataquery/main.go`
Test: `curl http://localhost:8084/api/v1/instances`
Expected: JSON response with instance list

- [ ] **Step 4: Verify Swagger UI**

Open: `http://localhost:8084/swagger/index.html`
Expected: `/instances` endpoint visible in documentation

- [ ] **Step 5: Stop service**

Stop the running service (Ctrl+C)

---

## Summary

This plan adds a new `GET /api/v1/instances` endpoint following existing patterns in the Data Query Service. Each task produces a self-contained change with tests. The implementation maintains consistency with existing code and includes swagger documentation updates.