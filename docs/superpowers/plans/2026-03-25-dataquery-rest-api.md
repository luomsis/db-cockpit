# DataQuery REST API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert DataQuery service from GraphQL to REST API using Hertz framework, removing all GraphQL code.

**Architecture:** Replace gqlgen GraphQL server with Hertz REST handlers. Keep existing service layer and repository unchanged. Update frontend to use REST API calls instead of GraphQL queries.

**Tech Stack:** Go, Hertz framework, PostgreSQL/TimescaleDB, TypeScript, Next.js

---

## File Structure

### Files to Delete
- `pkg/domain/dataquery/graph/` - entire GraphQL directory (generated.go, models_gen.go, resolver.go, schema.resolvers.go)
- `pkg/domain/dataquery/generate.go` - gqlgen generate directive
- `pkg/domain/dataquery/gqlgen.yml` - gqlgen configuration
- `web/dashboard/lib/graphql-client.ts` - GraphQL client
- `web/dashboard/lib/queries.ts` - GraphQL query definitions

### Files to Create
- `pkg/domain/dataquery/handler.go` - REST handlers using Hertz
- `pkg/domain/dataquery/handler_test.go` - handler unit tests
- `web/dashboard/lib/api-client.ts` - REST API client

### Files to Modify
- `cmd/dataquery/main.go` - replace GraphQL server with Hertz REST server
- `web/dashboard/app/page.tsx` - use REST API calls
- `web/dashboard/types/index.ts` - update types for REST response format
- `web/dashboard/package.json` - remove graphql-request dependency
- `cmd/gateway/main.go` - update proxy to use REST instead of GraphQL

---

## Task 1: Create REST Handler (Backend)

**Files:**
- Create: `pkg/domain/dataquery/handler.go`
- Test: `pkg/domain/dataquery/handler_test.go`

- [ ] **Step 1: Write the failing test for GetEndpoints handler**

Create `pkg/domain/dataquery/handler_test.go`:

```go
package dataquery

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

// Mock DataQueryService for testing
type mockDataQueryService struct {
	endpoints []string
	metrics   []string
	series    []*SeriesData
	err       error
}

func (m *mockDataQueryService) Name() string { return "MockDataQueryService" }
func (m *mockDataQueryService) Initialize(ctx context.Context) error { return nil }
func (m *mockDataQueryService) Shutdown(ctx context.Context) error { return nil }
func (m *mockDataQueryService) Health(ctx context.Context) error { return nil }
func (m *mockDataQueryService) GetEndpoints(ctx context.Context) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.endpoints, nil
}
func (m *mockDataQueryService) GetMetrics(ctx context.Context, endpoint string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metrics, nil
}
func (m *mockDataQueryService) QuerySeries(ctx context.Context, req *SeriesQuery) ([]*SeriesData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.series, nil
}
func (m *mockDataQueryService) QuerySeriesMulti(ctx context.Context, req *MultiSeriesQuery) ([]*SeriesData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.series, nil
}
func (m *mockDataQueryService) GetSeriesByID(ctx context.Context, id int64, timeRange *TimeRange) (*SeriesData, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.series) > 0 {
		return m.series[0], nil
	}
	return nil, nil
}

func createTestRequestContext(body string) (context.Context, *app.RequestContext) {
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.SetBody([]byte(body))
	return ctx, reqCtx
}

func TestGetEndpoints(t *testing.T) {
	mockService := &mockDataQueryService{
		endpoints: []string{"/api/metrics", "/api/users"},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	handler.GetEndpoints(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetEndpoints() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp EndpointsResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("GetEndpoints() returned %d endpoints, want 2", len(resp.Data))
	}
	if resp.Data[0] != "/api/metrics" {
		t.Errorf("GetEndpoints() first endpoint = %s, want /api/metrics", resp.Data[0])
	}
}

func TestGetEndpoints_ServiceError(t *testing.T) {
	mockService := &mockDataQueryService{err: context.Canceled}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	handler.GetEndpoints(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetEndpoints() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/domain/dataquery/... -run TestGetEndpoints -v`
Expected: FAIL with "undefined: NewHandler" or similar

- [ ] **Step 3: Write handler implementation**

Create `pkg/domain/dataquery/handler.go`:

```go
package dataquery

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// Handler handles REST API requests for data query operations
type Handler struct {
	service DataQueryService
}

// NewHandler creates a new Handler
func NewHandler(service DataQueryService) *Handler {
	return &Handler{service: service}
}

// Response types
type EndpointsResponse struct {
	Data []string `json:"data"`
}

type MetricsResponse struct {
	Data []string `json:"data"`
}

type SeriesResponse struct {
	Data []*SeriesDataResponse `json:"data"`
}

type SeriesSingleResponse struct {
	Data *SeriesDataResponse `json:"data"`
}

type SeriesDataResponse struct {
	Meta             SeriesMetaResponse    `json:"meta"`
	Points           []DataPointResponse   `json:"points"`
	AggregatedPoints []AggregatedPointResp `json:"aggregated_points,omitempty"`
	Statistics       *SeriesStatisticsResp `json:"statistics,omitempty"`
}

type SeriesMetaResponse struct {
	ID         string            `json:"id"`
	Endpoint   string            `json:"endpoint"`
	Metric     string            `json:"metric"`
	Labels     map[string]string `json:"labels"`
	LabelsHash string            `json:"labels_hash"`
	CreatedAt  time.Time         `json:"created_at"`
}

type DataPointResponse struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

type AggregatedPointResp struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
	Count int       `json:"count"`
}

type SeriesStatisticsResp struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Sum   float64 `json:"sum"`
	Count int     `json:"count"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Request types
type SeriesQueryRequest struct {
	Endpoints   []string           `json:"endpoints"`
	Metrics     []string           `json:"metrics"`
	Labels      string             `json:"labels"`
	Start       time.Time          `json:"start"`
	End         time.Time          `json:"end"`
	Aggregation *AggregationInput  `json:"aggregation"`
}

type AggregationInput struct {
	Interval string `json:"interval"`
	Function string `json:"function"`
}

// GetEndpoints handles GET /api/v1/endpoints
func (h *Handler) GetEndpoints(ctx context.Context, c *app.RequestContext) {
	endpoints, err := h.service.GetEndpoints(ctx)
	if err != nil {
		c.JSON(500, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(200, EndpointsResponse{Data: endpoints})
}

// GetMetrics handles GET /api/v1/metrics
func (h *Handler) GetMetrics(ctx context.Context, c *app.RequestContext) {
	endpoint := string(c.Query("endpoint"))
	if endpoint == "" {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "endpoint parameter is required",
			},
		})
		return
	}

	metrics, err := h.service.GetMetrics(ctx, endpoint)
	if err != nil {
		c.JSON(500, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(200, MetricsResponse{Data: metrics})
}

// GetSeries handles GET /api/v1/series
func (h *Handler) GetSeries(ctx context.Context, c *app.RequestContext) {
	// Parse time range (required)
	startStr := string(c.Query("start"))
	endStr := string(c.Query("end"))
	if startStr == "" || endStr == "" {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "start and end time are required",
			},
		})
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "invalid start time format, use RFC3339",
			},
		})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "invalid end time format, use RFC3339",
			},
		})
		return
	}

	// Build query
	req := &SeriesQuery{
		TimeRange: TimeRange{Start: start, End: end},
	}

	if endpoint := string(c.Query("endpoint")); endpoint != "" {
		req.Endpoint = endpoint
	}
	if metric := string(c.Query("metric")); metric != "" {
		req.Metric = metric
	}
	if labels := string(c.Query("labels")); labels != "" {
		req.LabelFilter = labels
	}
	if limitStr := string(c.Query("limit")); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	series, err := h.service.QuerySeries(ctx, req)
	if err != nil {
		c.JSON(500, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(200, SeriesResponse{Data: toSeriesResponses(series)})
}

// GetSeriesByID handles GET /api/v1/series/:id
func (h *Handler) GetSeriesByID(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(string(idStr), 10, 64)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "invalid series id",
			},
		})
		return
	}

	// Parse time range (required)
	startStr := string(c.Query("start"))
	endStr := string(c.Query("end"))
	if startStr == "" || endStr == "" {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "start and end time are required",
			},
		})
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "invalid start time format",
			},
		})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "invalid end time format",
			},
		})
		return
	}

	series, err := h.service.GetSeriesByID(ctx, id, &TimeRange{Start: start, End: end})
	if err != nil {
		c.JSON(500, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	if series == nil {
		c.JSON(404, ErrorResponse{
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: "series not found",
			},
		})
		return
	}

	c.JSON(200, SeriesSingleResponse{Data: toSeriesDataResponse(series)})
}

// QuerySeries handles POST /api/v1/series/query
func (h *Handler) QuerySeries(ctx context.Context, c *app.RequestContext) {
	var req SeriesQueryRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "invalid request body",
			},
		})
		return
	}

	if req.Start.IsZero() || req.End.IsZero() {
		c.JSON(400, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "start and end time are required",
			},
		})
		return
	}

	query := &MultiSeriesQuery{
		Endpoints:   req.Endpoints,
		Metrics:     req.Metrics,
		LabelFilter: req.Labels,
		TimeRange:   TimeRange{Start: req.Start, End: req.End},
	}

	if req.Aggregation != nil {
		query.Aggregation = &Aggregation{
			Interval: req.Aggregation.Interval,
			Function: AggFunction(req.Aggregation.Function),
		}
	}

	series, err := h.service.QuerySeriesMulti(ctx, query)
	if err != nil {
		c.JSON(500, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(200, SeriesResponse{Data: toSeriesResponses(series)})
}

// Helper functions
func toSeriesResponses(series []*SeriesData) []*SeriesDataResponse {
	result := make([]*SeriesDataResponse, len(series))
	for i, s := range series {
		result[i] = toSeriesDataResponse(s)
	}
	return result
}

func toSeriesDataResponse(s *SeriesData) *SeriesDataResponse {
	return &SeriesDataResponse{
		Meta:             toSeriesMetaResponse(s.Meta),
		Points:           toDataPointResponses(s.Points),
		AggregatedPoints: toAggregatedPointResponses(s.AggregatedPoints),
		Statistics:       toStatisticsResponse(s.Statistics),
	}
}

func toSeriesMetaResponse(m SeriesMeta) SeriesMetaResponse {
	return SeriesMetaResponse{
		ID:         strconv.FormatInt(m.ID, 10),
		Endpoint:   m.Endpoint,
		Metric:     m.Metric,
		Labels:     m.Labels,
		LabelsHash: m.LabelsHash,
		CreatedAt:  m.CreatedAt,
	}
}

func toDataPointResponses(points []DataPoint) []DataPointResponse {
	result := make([]DataPointResponse, len(points))
	for i, p := range points {
		result[i] = DataPointResponse{Time: p.Time, Value: p.Value}
	}
	return result
}

func toAggregatedPointResponses(points []AggregatedPoint) []AggregatedPointResp {
	if points == nil {
		return nil
	}
	result := make([]AggregatedPointResp, len(points))
	for i, p := range points {
		result[i] = AggregatedPointResp{Time: p.Time, Value: p.Value, Count: p.Count}
	}
	return result
}

func toStatisticsResponse(stats *SeriesStatistics) *SeriesStatisticsResp {
	if stats == nil {
		return nil
	}
	return &SeriesStatisticsResp{
		Min:   stats.Min,
		Max:   stats.Max,
		Avg:   stats.Avg,
		Sum:   stats.Sum,
		Count: stats.Count,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/domain/dataquery/... -run TestGetEndpoints -v`
Expected: PASS

- [ ] **Step 5: Write additional handler tests**

Add to `pkg/domain/dataquery/handler_test.go`:

```go
func TestGetMetrics(t *testing.T) {
	mockService := &mockDataQueryService{
		metrics: []string{"cpu_usage", "memory_usage"},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/api/v1/metrics?endpoint=/api/test")

	handler.GetMetrics(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetMetrics() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp MetricsResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("GetMetrics() returned %d metrics, want 2", len(resp.Data))
	}
}

func TestGetMetrics_MissingEndpoint(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")

	handler.GetMetrics(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("GetMetrics() status = %d, want 400", reqCtx.Response.StatusCode())
	}
}

func TestGetSeries(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{ID: 1, Endpoint: "/api/test", Metric: "cpu"},
				Points: []DataPoint{
					{Time: time.Now(), Value: 75.5},
				},
			},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/api/v1/series?start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z")

	handler.GetSeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetSeries() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp SeriesResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Errorf("GetSeries() returned %d series, want 1", len(resp.Data))
	}
}

func TestGetSeries_MissingTimeRange(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/api/v1/series")

	handler.GetSeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("GetSeries() status = %d, want 400", reqCtx.Response.StatusCode())
	}
}

func TestGetSeriesByID(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{ID: 1, Endpoint: "/api/test", Metric: "cpu"},
				Points: []DataPoint{{Time: time.Now(), Value: 75.5}},
				Statistics: &SeriesStatistics{Min: 50, Max: 100, Avg: 75},
			},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/api/v1/series/1?start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z")
	reqCtx.Params = app.Params{{Key: "id", Value: "1"}}

	handler.GetSeriesByID(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetSeriesByID() status = %d, want 200", reqCtx.Response.StatusCode())
	}
}

func TestQuerySeries(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{ID: 1, Endpoint: "/api/test", Metric: "cpu"},
				Points: []DataPoint{{Time: time.Now(), Value: 75.5}},
			},
		},
	}
	handler := NewHandler(mockService)

	body := `{"endpoints":["/api/test"],"metrics":["cpu"],"start":"2024-01-01T00:00:00Z","end":"2024-01-01T01:00:00Z"}`
	ctx, reqCtx := createTestRequestContext(body)
	reqCtx.Request.SetMethod("POST")

	handler.QuerySeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("QuerySeries() status = %d, want 200", reqCtx.Response.StatusCode())
	}
}
```

- [ ] **Step 6: Run all handler tests**

Run: `go test ./pkg/domain/dataquery/... -v`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/domain/dataquery/handler.go pkg/domain/dataquery/handler_test.go
git commit -m "feat(dataquery): add REST API handlers

- Add Handler with GetEndpoints, GetMetrics, GetSeries, GetSeriesByID, QuerySeries
- Add response/request DTOs
- Add unit tests for all handlers

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Update DataQuery main.go to use Hertz

**Files:**
- Modify: `cmd/dataquery/main.go`

- [ ] **Step 1: Write the current main.go test**

Run: `go build ./cmd/dataquery/...`
Expected: Build succeeds (current state)

- [ ] **Step 2: Rewrite main.go to use Hertz**

Replace `cmd/dataquery/main.go`:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/db-cockpit/pkg/common/config"
	"github.com/db-cockpit/pkg/common/logger"
	"github.com/db-cockpit/pkg/domain/dataquery"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v, using defaults\n", err)
		cfg = config.DefaultConfig()
	}

	// Initialize logger
	if err := logger.Init(&logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting Data Query Service")

	ctx := context.Background()

	// Initialize PostgreSQL connection pool for TimescaleDB
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.TimescaleDB.Host,
		cfg.Database.TimescaleDB.Port,
		cfg.Database.TimescaleDB.User,
		cfg.Database.TimescaleDB.Password,
		cfg.Database.TimescaleDB.Database,
		cfg.Database.TimescaleDB.SSLMode,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	logger.Info("PostgreSQL connection pool initialized")

	// Initialize Data Query Service
	repo := dataquery.NewPGRepository(pool)
	dataQueryService := dataquery.NewService(repo)
	handler := dataquery.NewHandler(dataQueryService)
	logger.Info("Data Query Service initialized with PostgreSQL")

	// Create Hertz server
	addr := fmt.Sprintf("%s:%d", cfg.Server.DataQuery.Host, cfg.Server.DataQuery.Port)
	h := server.Default(
		server.WithHostPorts(addr),
		server.WithDisablePrintRoute(false),
	)

	// Register routes
	v1 := h.Group("/api/v1")
	{
		v1.GET("/endpoints", handler.GetEndpoints)
		v1.GET("/metrics", handler.GetMetrics)
		v1.GET("/series", handler.GetSeries)
		v1.GET("/series/:id", handler.GetSeriesByID)
		v1.POST("/series/query", handler.QuerySeries)
	}

	// Health check
	h.GET("/health", func(c context.Context, ctx *app.RequestContext) {
		ctx.JSON(200, map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"service":   "dataquery",
		})
	})

	// Start server in goroutine
	go func() {
		logger.Info("Data Query Service started", zap.String("addr", addr))
		printEndpoints(addr)
		if err := h.Run(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Data Query Service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = dataQueryService.Shutdown(shutdownCtx)

	if pool != nil {
		pool.Close()
	}

	h.Shutdown(shutdownCtx)

	logger.Info("Data Query Service stopped")
}

func printEndpoints(addr string) {
	fmt.Println("\n========================================")
	fmt.Println("Data Query Service (REST API)")
	fmt.Println("========================================")
	fmt.Println("\n📡 REST API Endpoints:")
	fmt.Printf("  GET  http://%s/api/v1/endpoints\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/metrics?endpoint=...\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/series?start=...&end=...\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/series/:id?start=...&end=...\n", addr)
	fmt.Printf("  POST http://%s/api/v1/series/query\n", addr)
	fmt.Println("\n📝 Example REST API Calls:")
	fmt.Print(`
  # Get all endpoints
  curl http://localhost:8084/api/v1/endpoints

  # Get metrics for an endpoint
  curl "http://localhost:8084/api/v1/metrics?endpoint=/api/metrics"

  # Query series with time range
  curl "http://localhost:8084/api/v1/series?start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z"

  # Complex query with aggregation
  curl -X POST http://localhost:8084/api/v1/series/query \
    -H "Content-Type: application/json" \
    -d '{"endpoints":["/api/metrics"],"metrics":["cpu"],"start":"2024-01-01T00:00:00Z","end":"2024-01-01T01:00:00Z","aggregation":{"interval":"5m","function":"AVG"}}'
`)
	fmt.Println("\n========================================")
}
```

- [ ] **Step 3: Run build to verify it compiles**

Run: `go build ./cmd/dataquery/...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add cmd/dataquery/main.go
git commit -m "feat(dataquery): replace GraphQL server with Hertz REST server

- Remove gqlgen dependencies
- Add REST routes for endpoints, metrics, series
- Update startup messages

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Delete GraphQL Code

**Files:**
- Delete: `pkg/domain/dataquery/graph/` directory
- Delete: `pkg/domain/dataquery/generate.go`
- Delete: `pkg/domain/dataquery/gqlgen.yml`

- [ ] **Step 1: Delete GraphQL directory and files**

```bash
rm -rf pkg/domain/dataquery/graph/
rm pkg/domain/dataquery/generate.go
rm pkg/domain/dataquery/gqlgen.yml
```

- [ ] **Step 2: Verify build still works**

Run: `go build ./...`
Expected: Build succeeds

- [ ] **Step 3: Verify tests still pass**

Run: `go test ./pkg/domain/dataquery/... -v`
Expected: All tests PASS

- [ ] **Step 3: Clean up Go modules**

Run: `go mod tidy`
Expected: Unused GraphQL dependencies removed

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor(dataquery): remove GraphQL implementation

- Delete graph/ directory (gqlgen generated code)
- Delete generate.go and gqlgen.yml
- GraphQL replaced by REST API

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Update Frontend REST Client

**Files:**
- Create: `web/dashboard/lib/api-client.ts`
- Delete: `web/dashboard/lib/graphql-client.ts`
- Delete: `web/dashboard/lib/queries.ts`

- [ ] **Step 1: Create REST API client**

Create `web/dashboard/lib/api-client.ts`:

```typescript
const API_BASE_URL = '/api'

// 开发环境使用模拟token
const DEV_TOKEN = 'dev_tenant:dev_user:admin'

async function fetchAPI<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${DEV_TOKEN}`,
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: { message: 'Unknown error' } }))
    throw new Error(error.error?.message || `HTTP ${response.status}`)
  }

  return response.json()
}

// Types
export interface EndpointsResponse {
  data: string[]
}

export interface MetricsResponse {
  data: string[]
}

export interface SeriesResponse {
  data: SeriesData[]
}

export interface SeriesData {
  meta: SeriesMeta
  points: DataPoint[]
  aggregated_points?: AggregatedPoint[]
  statistics?: SeriesStatistics
}

export interface SeriesMeta {
  id: string
  endpoint: string
  metric: string
  labels: Record<string, string>
  labels_hash: string
  created_at: string
}

export interface DataPoint {
  time: string
  value: number
}

export interface AggregatedPoint {
  time: string
  value: number
  count: number
}

export interface SeriesStatistics {
  min: number
  max: number
  avg: number
  sum: number
  count: number
}

export interface SeriesQueryRequest {
  endpoints?: string[]
  metrics?: string[]
  labels?: string
  start: string
  end: string
  aggregation?: {
    interval: string
    function: 'AVG' | 'MIN' | 'MAX' | 'SUM' | 'COUNT'
  }
}

// API functions
export async function getEndpoints(): Promise<string[]> {
  const response = await fetchAPI<EndpointsResponse>('/v1/endpoints')
  return response.data || []
}

export async function getMetrics(endpoint: string): Promise<string[]> {
  const response = await fetchAPI<MetricsResponse>(
    `/v1/metrics?endpoint=${encodeURIComponent(endpoint)}`
  )
  return response.data || []
}

export async function getSeries(params: {
  endpoint?: string
  metric?: string
  labels?: string
  start: string
  end: string
  limit?: number
}): Promise<SeriesData[]> {
  const queryParams = new URLSearchParams()
  queryParams.set('start', params.start)
  queryParams.set('end', params.end)
  if (params.endpoint) queryParams.set('endpoint', params.endpoint)
  if (params.metric) queryParams.set('metric', params.metric)
  if (params.labels) queryParams.set('labels', params.labels)
  if (params.limit) queryParams.set('limit', params.limit.toString())

  const response = await fetchAPI<SeriesResponse>(`/v1/series?${queryParams.toString()}`)
  return response.data || []
}

export async function querySeries(request: SeriesQueryRequest): Promise<SeriesData[]> {
  const response = await fetchAPI<SeriesResponse>('/v1/series/query', {
    method: 'POST',
    body: JSON.stringify(request),
  })
  return response.data || []
}
```

- [ ] **Step 2: Delete GraphQL files**

```bash
rm web/dashboard/lib/graphql-client.ts
rm web/dashboard/lib/queries.ts
```

- [ ] **Step 3: Commit**

```bash
git add web/dashboard/lib/api-client.ts
git add -A web/dashboard/lib/
git commit -m "feat(dashboard): replace GraphQL client with REST API client

- Create api-client.ts with getEndpoints, getMetrics, getSeries, querySeries
- Remove graphql-client.ts and queries.ts

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Update Frontend Page to use REST API

**Files:**
- Modify: `web/dashboard/app/page.tsx`
- Modify: `web/dashboard/types/index.ts`

- [ ] **Step 1: Update types/index.ts**

Replace `web/dashboard/types/index.ts`:

```typescript
// REST API响应类型
export interface LabelEntry {
  key: string
  value: string
}

export interface Labels {
  entries: LabelEntry[]
}

export interface SeriesMeta {
  id: string
  endpoint: string
  metric: string
  labels: Record<string, string>
}

export interface DataPoint {
  time: string
  value: number
}

export interface AggregatedPoint {
  time: string
  value: number
  count: number
}

export interface SeriesStatistics {
  min: number
  max: number
  avg: number
  sum: number
  count: number
}

export interface Series {
  meta: SeriesMeta
  points: DataPoint[]
  aggregated_points?: AggregatedPoint[]
  statistics?: SeriesStatistics
}

export interface Statistics {
  min: number
  max: number
  avg: number
  sum: number
  count: number
}

// 告警类型
export interface Alert {
  id: string
  name: string
  severity: 'critical' | 'warning' | 'info'
  endpoint: string
  metric: string
  threshold: number
  currentValue: number
  status: 'firing' | 'resolved'
  startedAt: string
  resolvedAt?: string
  labels: Record<string, string>
}

// UI状态类型
export type TimeRangeOption = '1h' | '6h' | '24h' | '7d'
export type RefreshInterval = 'off' | '30s' | '1m' | '5m'

export interface TimeRangeInput {
  start: string
  end: string
}

// Dashboard状态
export interface DashboardState {
  selectedEndpoint: string
  selectedMetric: string
  timeRange: TimeRangeOption
  refreshInterval: RefreshInterval
}
```

- [ ] **Step 2: Update app/page.tsx to use REST API**

Update imports and data fetching in `web/dashboard/app/page.tsx`:

```typescript
'use client'

import { useState, useEffect, useCallback } from 'react'
import { Header } from '@/components/dashboard/header'
import { FilterBar } from '@/components/dashboard/filter-bar'
import { MetricCards } from '@/components/dashboard/metric-cards'
import { MainChart } from '@/components/dashboard/main-chart'
import { AlertList } from '@/components/dashboard/alert-list'
import { StatsPanel } from '@/components/dashboard/stats-panel'
import { getEndpoints, getMetrics, getSeries } from '@/lib/api-client'
import { toTimeRange } from '@/lib/time-utils'
import { calculateStatistics } from '@/lib/stats-utils'
import { mockAlerts, getFiringAlertsCount } from '@/lib/mock-alerts'
import {
  TimeRangeOption,
  RefreshInterval,
  Series,
  Statistics,
} from '@/types'

export default function DashboardPage() {
  // 筛选状态
  const [timeRange, setTimeRange] = useState<TimeRangeOption>('1h')
  const [selectedEndpoint, setSelectedEndpoint] = useState('')
  const [selectedMetric, setSelectedMetric] = useState('')
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>('off')

  // 数据状态
  const [endpoints, setEndpoints] = useState<string[]>([])
  const [metrics, setMetrics] = useState<string[]>([])
  const [series, setSeries] = useState<Series[]>([])
  const [statistics, setStatistics] = useState<Statistics | null>(null)

  // 加载状态
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingEndpoints, setIsLoadingEndpoints] = useState(false)
  const [isLoadingMetrics, setIsLoadingMetrics] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // 获取endpoints
  const fetchEndpoints = useCallback(async () => {
    setIsLoadingEndpoints(true)
    try {
      const data = await getEndpoints()
      setEndpoints(data)
    } catch (err) {
      console.error('Failed to fetch endpoints:', err)
      setEndpoints([])
    } finally {
      setIsLoadingEndpoints(false)
    }
  }, [])

  // 获取metrics
  const fetchMetrics = useCallback(async (endpoint: string) => {
    if (!endpoint) {
      setMetrics([])
      return
    }
    setIsLoadingMetrics(true)
    try {
      const data = await getMetrics(endpoint)
      setMetrics(data)
    } catch (err) {
      console.error('Failed to fetch metrics:', err)
      setMetrics([])
    } finally {
      setIsLoadingMetrics(false)
    }
  }, [])

  // 获取时序数据
  const fetchSeriesData = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const timeRangeInput = toTimeRange(timeRange)
      const data = await getSeries({
        endpoint: selectedEndpoint || undefined,
        metric: selectedMetric || undefined,
        start: timeRangeInput.start,
        end: timeRangeInput.end,
        limit: 10,
      })

      setSeries(data as Series[])

      // 计算统计信息
      const allPoints = data.flatMap((s) => s.points)
      setStatistics(calculateStatistics(allPoints))
    } catch (err) {
      console.error('Failed to fetch series data:', err)
      setError('获取数据失败，请检查后端服务是否正常运行')
      setSeries([])
      setStatistics(null)
    } finally {
      setIsLoading(false)
    }
  }, [timeRange, selectedEndpoint, selectedMetric])

  // 初始化：获取endpoints
  useEffect(() => {
    fetchEndpoints()
  }, [fetchEndpoints])

  // 当endpoint变化时获取metrics
  useEffect(() => {
    fetchMetrics(selectedEndpoint)
  }, [selectedEndpoint, fetchMetrics])

  // 当筛选条件变化时获取数据
  useEffect(() => {
    fetchSeriesData()
  }, [fetchSeriesData])

  // 自动刷新
  useEffect(() => {
    if (refreshInterval === 'off') return

    const intervals: Record<RefreshInterval, number> = {
      off: 0,
      '30s': 30000,
      '1m': 60000,
      '5m': 300000,
    }

    const interval = setInterval(() => {
      if (document.visibilityState === 'visible') {
        fetchSeriesData()
      }
    }, intervals[refreshInterval])

    return () => clearInterval(interval)
  }, [refreshInterval, fetchSeriesData])

  const alertCount = getFiringAlertsCount(mockAlerts)

  return (
    <main className="min-h-screen">
      <Header timeRange={timeRange} onTimeRangeChange={setTimeRange} />

      <FilterBar
        endpoints={endpoints}
        metrics={metrics}
        selectedEndpoint={selectedEndpoint}
        selectedMetric={selectedMetric}
        refreshInterval={refreshInterval}
        isLoading={isLoading}
        onEndpointChange={setSelectedEndpoint}
        onMetricChange={setSelectedMetric}
        onRefreshIntervalChange={setRefreshInterval}
        onRefresh={fetchSeriesData}
      />

      {error && (
        <div className="mx-6 mt-4 rounded-md border border-red-500/50 bg-red-500/10 p-4 text-red-500">
          {error}
        </div>
      )}

      <MetricCards
        statistics={statistics}
        alertCount={alertCount}
        isLoading={isLoading}
      />

      <MainChart series={series} isLoading={isLoading} />

      <div className="grid grid-cols-3 gap-4 px-6 pb-6">
        <div className="col-span-2">
          <AlertList alerts={mockAlerts} />
        </div>
        <div>
          <StatsPanel statistics={statistics} />
        </div>
      </div>
    </main>
  )
}
```

- [ ] **Step 3: Verify frontend builds**

Run: `cd web/dashboard && npm run build`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add web/dashboard/app/page.tsx web/dashboard/types/index.ts
git commit -m "feat(dashboard): update page to use REST API instead of GraphQL

- Replace graphqlClient with REST API functions
- Update types for REST response format
- Labels now use Record<string, string> instead of { entries: [...] }

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Remove GraphQL Dependencies and Update Gateway

**Files:**
- Modify: `web/dashboard/package.json`
- Modify: `cmd/gateway/main.go`

- [ ] **Step 1: Remove graphql-request from package.json**

In `web/dashboard/package.json`, remove the line:
```json
"graphql-request": "^6.1.0",
```

- [ ] **Step 2: Update Gateway proxy to REST**

Modify `cmd/gateway/main.go` - update the NoRoute handler and printEndpoints:

Find the NoRoute handler and replace:
```go
// Register GraphQL routes - proxy to Data Query Service
h.NoRoute(func(c context.Context, ctx *app.RequestContext) {
	path := string(ctx.URI().Path())
	if path == "/graphql" || path == "/graphql/playground" {
		proxyToDataQuery(ctx, dataQueryAddr)
	} else {
		ctx.AbortWithStatus(404)
	}
})
```

With:
```go
// Register Data Query REST API proxy
h.NoRoute(func(c context.Context, ctx *app.RequestContext) {
	path := string(ctx.URI().Path())
	if strings.HasPrefix(path, "/api/v1/endpoints") ||
		strings.HasPrefix(path, "/api/v1/metrics") ||
		strings.HasPrefix(path, "/api/v1/series") {
		proxyToDataQuery(ctx, dataQueryAddr)
	} else {
		ctx.AbortWithStatus(404)
	}
})
```

Add `"strings"` to the imports at the top of the file (add it to the existing import block), then update `printEndpoints`:

```go
func printEndpoints(dataQueryAddr string) {
	fmt.Println("\n========================================")
	fmt.Println("Database Intelligent Cockpit API Gateway")
	fmt.Println("========================================")
	fmt.Println("\n📡 Data Query Service (proxied):")
	fmt.Println("  GET  /api/v1/endpoints")
	fmt.Println("  GET  /api/v1/metrics?endpoint=...")
	fmt.Println("  GET  /api/v1/series?start=...&end=...")
	fmt.Println("  POST /api/v1/series/query")
	fmt.Printf("  → Proxied to: %s\n", dataQueryAddr)
	fmt.Println("\n🔐 Authentication:")
	fmt.Println("  Authorization: Bearer tenant_id:user_id:role")
	fmt.Println("  Example: Bearer tenant-001:user-001:admin")
	fmt.Println("\n📝 REST API Endpoints:")
	fmt.Println("  SQL Governance: POST /api/v1/sql/*")
	fmt.Println("  Performance:    POST /api/v1/performance/*")
	fmt.Println("  Thresholds:     GET/PUT /api/v1/thresholds")
	fmt.Println("  LLM:            POST /api/v1/llm/*")
	fmt.Println("\n========================================")
}
```

- [ ] **Step 3: Reinstall frontend dependencies**

Run: `cd web/dashboard && npm install`
Expected: Dependencies installed without graphql-request

- [ ] **Step 4: Verify everything builds**

Run: `go build ./...`
Expected: Build succeeds

Run: `cd web/dashboard && npm run build`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add web/dashboard/package.json cmd/gateway/main.go
git add web/dashboard/package-lock.json 2>/dev/null || true
git commit -m "refactor: remove GraphQL and update gateway proxy for REST

- Remove graphql-request from package.json
- Update gateway to proxy REST API paths instead of GraphQL
- Update startup messages

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Run Full Test Suite

**Note:** The spec mentions updating `test/integration/query_test.go` and `test/integration/gateway_test.go`. If these files contain GraphQL tests, they may need updates or should be skipped. Check their content and update or remove GraphQL-related tests as needed.

- [ ] **Step 1: Run all Go tests**

Run: `go test ./...`
Expected: All tests PASS

- [ ] **Step 2: Run frontend type check**

Run: `cd web/dashboard && npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 3: Final verification commit**

```bash
git add -A
git status
```

Verify no uncommitted changes remain.

---

## Summary

This plan converts the DataQuery service from GraphQL to REST API:

1. **Created REST handlers** with full test coverage
2. **Updated main.go** to use Hertz framework
3. **Deleted GraphQL code** (graph/, generate.go, gqlgen.yml)
4. **Created REST API client** for frontend
5. **Updated frontend page** to use REST API
6. **Removed GraphQL dependencies** and updated gateway proxy
7. **Verified** all tests pass and builds succeed