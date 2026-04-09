# Project Cleanup and Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean up unused code, optimize project structure, and update documentation to reflect current architecture.

**Architecture:** Systematic removal of outdated GraphQL references, unused data layer components, stub domain services, and consolidation of documentation. The Data Query Service is the only fully implemented service with REST API and database connectivity.

**Tech Stack:** Go, Hertz framework, PostgreSQL/TimescaleDB, Swagger/OpenAPI

---

## Analysis Summary

### What's Actually Working
- **Data Query Service** (`cmd/dataquery/`): Fully functional REST API with TimescaleDB connectivity
- **Gateway** (`cmd/gateway/`): Proxies to Data Query Service, stub implementations for other services

### What's Unused/Outdated
1. **GraphQL Documentation** (`docs/graphql-api-guide.md`): Documents GraphQL API that was replaced by REST
2. **Data Layer Clients** (`pkg/data/`): Neo4j, Redis, pgvector, PGMQ clients initialized but never used
3. **Stub Domain Services**: sqlgovernance, performance, threshold, llm have interfaces but nil repositories
4. **Task Client** (`pkg/domain/taskclient/`): Interface defined but never instantiated
5. **Unused Service Entry Points**: collector, agent, taskengine have main.go but no active implementation

---

## File Structure Changes

### Files to Delete
- `docs/graphql-api-guide.md` (outdated GraphQL documentation)
- `test/test_gateway_curl.sh` (contains GraphQL endpoints no longer valid)
- `pkg/domain/taskclient/interface.go` (unused interface)

### Files to Update
- `README.md` (remove GraphQL references, update architecture diagram)
- `CLAUDE.md` (update data layer description, remove GraphQL references)
- `pkg/data/data_layer.go` (remove unused client initialization)
- `pkg/api/handler/handler.go` (update comment about GraphQL)

### Files to Keep (for future development)
- `pkg/domain/sqlgovernance/`, `pkg/domain/performance/`, `pkg/domain/threshold/`, `pkg/domain/llm/` - Keep interfaces for future implementation
- `pkg/data/neo4j/`, `pkg/data/pgvector/`, `pkg/data/pgmq/`, `pkg/data/redis/` - Keep client definitions for future use
- `cmd/collector/`, `cmd/agent/`, `cmd/taskengine/` - Keep entry points for future development

---

## Task List

### Task 1: Delete Outdated GraphQL Documentation

**Files:**
- Delete: `docs/graphql-api-guide.md`
- Delete: `test/test_gateway_curl.sh`

- [ ] **Step 1: Remove GraphQL API guide**

```bash
rm docs/graphql-api-guide.md
```

- [ ] **Step 2: Remove outdated gateway curl test script**

```bash
rm test/test_gateway_curl.sh
```

- [ ] **Step 3: Commit removal**

```bash
git add docs/graphql-api-guide.md test/test_gateway_curl.sh
git commit -m "docs: remove outdated GraphQL API documentation

GraphQL API was replaced by REST API in Data Query Service.
- Remove docs/graphql-api-guide.md (outdated)
- Remove test/test_gateway_curl.sh (GraphQL endpoints)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Update README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Remove GraphQL references**

Find and remove sections mentioning GraphQL:
- Remove GraphQL API section from documentation table
- Update architecture diagram prompt reference
- Remove GraphQL playground references

- [ ] **Step 2: Update Quick Start section**

Remove GraphQL-related commands and examples.

- [ ] **Step 3: Update API Documentation table**

```markdown
### API Reference

| Document | Description |
|----------|-------------|
| [Swagger UI](http://localhost:8084/swagger/index.html) | Interactive API documentation (when service running) |
| [swagger.json](docs/swagger.json) | OpenAPI 2.0 specification (JSON) |
```

- [ ] **Step 4: Commit README update**

```bash
git add README.md
git commit -m "docs: update README to reflect REST API architecture

- Remove outdated GraphQL references
- Update API documentation section
- Align with current Data Query Service implementation

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update Architecture section**

Remove GraphQL from communication protocols table. Current state:

```markdown
## Architecture: Communication Protocols

| Path | Protocol |
|------|----------|
| Frontend ↔ Gateway | RESTful API |
| Frontend ↔ Data Query | REST API (`/api/v1/*`) |
| Domain Layer ↔ Agent | RPC (gRPC) |
| Domain Layer ↔ Task Engine | RPC (gRPC) |
| Domain Layer internal | Direct function calls |
```

This is correct. Keep it.

- [ ] **Step 2: Add Data Layer Status Note**

Add note about data layer components status:

```markdown
## Data Layer Status

The following data layer components are defined but not actively used:
- Neo4j (Graph Store) - client defined, not connected
- pgvector (Vector Store) - client defined, not connected
- PGMQ (Message Queue) - client defined, not connected
- Redis (Cache) - client defined, not connected

Only TimescaleDB is actively used by Data Query Service.
```

- [ ] **Step 3: Commit CLAUDE.md update**

```bash
git add CLAUDE.md
git commit -m "docs: add data layer status clarification to CLAUDE.md

- Document which data layer components are active
- Clarify TimescaleDB is the only connected database

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Update Data Layer Implementation

**Files:**
- Modify: `pkg/data/data_layer.go`

- [ ] **Step 1: Simplify NewDataLayer to return empty struct**

The current `NewDataLayer` returns empty struct. The `InitializeDataLayer` creates connections but is never called. Add documentation:

```go
// NewDataLayer creates a new DataLayer instance
// Note: This returns an empty DataLayer. Use InitializeDataLayer for full setup.
// Currently only TimescaleDB connection is actively used by Data Query Service.
func NewDataLayer(cfg *config.Config) (*DataLayer, error) {
	return &DataLayer{}, nil
}
```

- [ ] **Step 2: Add status comments to InitializeDataLayer**

```go
// InitializeDataLayer initializes and connects all data layer components
// Note: This function is defined for future use. Currently only TimescaleDB
// pool is used by Data Query Service. Other components (Neo4j, Redis, PGMQ,
// PgVector) are initialized but not actively used in the current implementation.
func InitializeDataLayer(cfg *config.Config) (*DataLayer, error) {
```

- [ ] **Step 3: Commit data layer update**

```bash
git add pkg/data/data_layer.go
git commit -m "docs: add status comments to data layer components

- Document which components are actively used
- Clarify InitializeDataLayer is for future use

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Update Handler Comments

**Files:**
- Modify: `pkg/api/handler/handler.go`

- [ ] **Step 1: Fix misleading comment**

Change line 17 comment from:
```go
// GatewayHandler handles gateway requests
// Note: Data Query operations are handled via GraphQL, not REST
```

To:
```go
// GatewayHandler handles gateway requests for domain services
// Note: Data Query operations are proxied to Data Query Service REST API
```

- [ ] **Step 2: Commit handler update**

```bash
git add pkg/api/handler/handler.go
git commit -m "fix: correct misleading comment about Data Query API

Data Query uses REST API, not GraphQL

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Remove Unused Task Client Interface

**Files:**
- Delete: `pkg/domain/taskclient/interface.go`

- [ ] **Step 1: Verify taskclient is unused**

```bash
grep -r "taskclient" pkg/ cmd/ --include="*.go" | grep -v "interface.go"
```

Expected: Should show no imports of the interface (only the file itself).

- [ ] **Step 2: Remove the unused interface file**

```bash
rm pkg/domain/taskclient/interface.go
```

- [ ] **Step 3: Commit removal**

```bash
git add pkg/domain/taskclient/interface.go
git commit -m "refactor: remove unused TaskClientInterface

TaskClientInterface was defined but never imported or used.
Task system functionality exists in pkg/task/ but is not yet integrated.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Update Domain Services with Status Documentation

**Files:**
- Modify: `pkg/domain/sqlgovernance/service.go`
- Modify: `pkg/domain/performance/service.go`
- Modify: `pkg/domain/threshold/service.go`
- Modify: `pkg/domain/llm/service.go`

- [ ] **Step 1: Add implementation status comment to sqlgovernance/service.go**

Add at top of file after package declaration:
```go
// SQL Governance Domain Service
// Status: Interface defined, basic implementation exists
// Note: Repository is nil when used in Gateway, making this a stub.
// Full implementation requires database repository and Agent RPC client.
```

- [ ] **Step 2: Add implementation status comment to performance/service.go**

```go
// Performance Diagnosis Domain Service
// Status: Interface defined, basic implementation exists
// Note: Repository is nil when used in Gateway, making this a stub.
// Full implementation requires database repository and Threshold client.
```

- [ ] **Step 3: Add implementation status comment to threshold/service.go**

```go
// Dynamic Threshold Domain Service
// Status: Interface defined, basic implementation exists
// Note: Repository is nil when used in Gateway, making this a stub.
// Full implementation requires database repository.
```

- [ ] **Step 4: Add implementation status comment to llm/service.go**

```go
// LLM Orchestrator Domain Service
// Status: Interface defined, basic implementation exists
// Note: Repository and Provider are nil when used in Gateway, making this a stub.
// Full implementation requires LLM provider client and vector database.
```

- [ ] **Step 5: Commit domain service updates**

```bash
git add pkg/domain/sqlgovernance/service.go pkg/domain/performance/service.go pkg/domain/threshold/service.go pkg/domain/llm/service.go
git commit -m "docs: add implementation status to domain service interfaces

Clarifies which services are fully implemented vs stubs

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 8: Run Tests and Verify

**Files:**
- Test: All packages

- [ ] **Step 1: Run all unit tests**

```bash
go test ./pkg/... -v
```

Expected: All tests pass.

- [ ] **Step 2: Run integration tests**

```bash
# Ensure Data Query Service is running
curl http://localhost:8084/health

# Run integration tests
go test ./test/integration/... -v
```

- [ ] **Step 3: Verify build succeeds**

```bash
go build ./cmd/...
```

Expected: Build succeeds without errors.

---

### Task 9: Final Commit Summary

- [ ] **Step 1: Create summary of all changes**

```bash
git log --oneline -10
```

- [ ] **Step 2: Verify git status is clean**

```bash
git status
```

Expected: No uncommitted changes (except for this plan file).

---

## Summary of Changes

| Category | Action | Files |
|----------|--------|-------|
| Documentation | Delete | `docs/graphql-api-guide.md`, `test/test_gateway_curl.sh` |
| Documentation | Update | `README.md`, `CLAUDE.md` |
| Code Comments | Update | `pkg/data/data_layer.go`, `pkg/api/handler/handler.go` |
| Domain Services | Document Status | `pkg/domain/*/service.go` |
| Unused Code | Delete | `pkg/domain/taskclient/interface.go` |

**No breaking changes** - Only documentation updates and removal of truly unused files.