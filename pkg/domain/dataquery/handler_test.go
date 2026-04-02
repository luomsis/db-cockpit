package dataquery

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
)

// Mock DataQueryService for testing
type mockDataQueryService struct {
	endpoints []string
	metrics   []string
	series    []*SeriesData
	instances []*InstanceMeta
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
func (m *mockDataQueryService) GetInstanceByEndpoint(ctx context.Context, endpoint string) (*InstanceMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil
}
func (m *mockDataQueryService) GetAllInstances(ctx context.Context) ([]*InstanceMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.instances, nil
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

func TestGetMetrics(t *testing.T) {
	mockService := &mockDataQueryService{
		metrics: []string{"cpu_usage", "memory_usage"},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	// Use query parameter instead of path parameter
	reqCtx.Request.SetRequestURI("/metrics?endpoint=/api/metrics")

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
	if resp.Data[0] != "cpu_usage" {
		t.Errorf("GetMetrics() first metric = %s, want cpu_usage", resp.Data[0])
	}
}

func TestGetMetrics_MissingEndpoint(t *testing.T) {
	mockService := &mockDataQueryService{
		metrics: []string{"cpu_usage", "memory_usage"},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	// No endpoint query parameter provided
	reqCtx.Request.SetRequestURI("/metrics")

	handler.GetMetrics(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("GetMetrics() status = %d, want 400", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INVALID_PARAMETER" {
		t.Errorf("GetMetrics() error code = %s, want INVALID_PARAMETER", resp.Error.Code)
	}
}

func TestGetMetrics_ServiceError(t *testing.T) {
	mockService := &mockDataQueryService{err: context.Canceled}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/metrics?endpoint=/api/metrics")

	handler.GetMetrics(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetMetrics() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}

func TestGetSeries(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{
					ID:       1,
					Endpoint: "/api/metrics",
					Metric:   "cpu_usage",
					Labels:   map[string]string{"host": "server1"},
				},
				Points: []DataPoint{
					{Time: parseTestTime("2024-01-01T00:00:00Z"), Value: 50.0},
					{Time: parseTestTime("2024-01-01T00:01:00Z"), Value: 60.0},
				},
			},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	// Set query parameters via URI including required start and end time
	reqCtx.Request.SetRequestURI("/series?endpoint=/api/metrics&metric=cpu_usage&start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z")

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
	if resp.Data[0].Endpoint != "/api/metrics" {
		t.Errorf("GetSeries() endpoint = %s, want /api/metrics", resp.Data[0].Endpoint)
	}
	if len(resp.Data[0].Points) != 2 {
		t.Errorf("GetSeries() points count = %d, want 2", len(resp.Data[0].Points))
	}
	// Check ID is string
	if resp.Data[0].ID != "1" {
		t.Errorf("GetSeries() ID = %s, want 1 (string)", resp.Data[0].ID)
	}
}

func TestGetSeries_MissingTimeRange(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	// Missing start and end time parameters
	reqCtx.Request.SetRequestURI("/series?endpoint=/api/metrics&metric=cpu_usage")

	handler.GetSeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("GetSeries() status = %d, want 400", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INVALID_PARAMETER" {
		t.Errorf("GetSeries() error code = %s, want INVALID_PARAMETER", resp.Error.Code)
	}
}

func TestGetSeries_ServiceError(t *testing.T) {
	mockService := &mockDataQueryService{err: context.Canceled}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/series?start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z")

	handler.GetSeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetSeries() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}

func TestGetSeriesByID(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{
					ID:       1,
					Endpoint: "/api/metrics",
					Metric:   "cpu_usage",
					Labels:   map[string]string{"host": "server1"},
				},
				Points: []DataPoint{
					{Time: parseTestTime("2024-01-01T00:00:00Z"), Value: 50.0},
				},
				Statistics: &SeriesStatistics{
					Min:   40.0,
					Max:   60.0,
					Avg:   50.0,
					Sum:   100.0,
					Count: 2,
				},
			},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "id", Value: "1"}}

	handler.GetSeriesByID(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetSeriesByID() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp SeriesSingleResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("GetSeriesByID() data is nil")
	}
	if resp.Data.ID != "1" {
		t.Errorf("GetSeriesByID() ID = %s, want 1 (string)", resp.Data.ID)
	}
	if resp.Data.Statistics == nil {
		t.Error("GetSeriesByID() statistics should not be nil")
	}
}

func TestGetSeriesByID_NotFound(t *testing.T) {
	mockService := &mockDataQueryService{series: []*SeriesData{}}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "id", Value: "999"}}

	handler.GetSeriesByID(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 404 {
		t.Errorf("GetSeriesByID() status = %d, want 404", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("GetSeriesByID() error code = %s, want NOT_FOUND", resp.Error.Code)
	}
}

func TestGetSeriesByID_InvalidID(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "id", Value: "invalid"}}

	handler.GetSeriesByID(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("GetSeriesByID() status = %d, want 400", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INVALID_PARAMETER" {
		t.Errorf("GetSeriesByID() error code = %s, want INVALID_PARAMETER", resp.Error.Code)
	}
}

func TestQuerySeries(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{
					ID:       1,
					Endpoint: "/api/metrics",
					Metric:   "cpu_usage",
				},
				Points: []DataPoint{},
			},
		},
	}
	handler := NewHandler(mockService)

	// Use new request format with start/end time
	body := `{"endpoints": ["/api/metrics"], "metrics": ["cpu_usage"], "start": "2024-01-01T00:00:00Z", "end": "2024-01-01T01:00:00Z"}`
	ctx, reqCtx := createTestRequestContext(body)

	handler.QuerySeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("QuerySeries() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp SeriesResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Errorf("QuerySeries() returned %d series, want 1", len(resp.Data))
	}
}

func TestQuerySeries_MissingTimeRange(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	// Missing start/end time
	body := `{"endpoints": ["/api/metrics"], "metrics": ["cpu_usage"]}`
	ctx, reqCtx := createTestRequestContext(body)

	handler.QuerySeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("QuerySeries() status = %d, want 400", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INVALID_PARAMETER" {
		t.Errorf("QuerySeries() error code = %s, want INVALID_PARAMETER", resp.Error.Code)
	}
}

func TestQuerySeries_ServiceError(t *testing.T) {
	mockService := &mockDataQueryService{err: context.Canceled}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext(`{"start": "2024-01-01T00:00:00Z", "end": "2024-01-01T01:00:00Z"}`)

	handler.QuerySeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("QuerySeries() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}

func TestQuerySeries_InvalidJSON(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("invalid json")

	handler.QuerySeries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("QuerySeries() status = %d, want 400", reqCtx.Response.StatusCode())
	}
}

// Helper function for parsing time in tests
func parseTestTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}