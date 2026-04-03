package dataquery

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
)

// Mock DataQueryService for testing
type mockDataQueryService struct {
	endpoints   []string
	metrics     []string
	series      []*SeriesData
	instances   []*InstanceMeta
	instance    *InstanceMeta // Single instance for GetInstanceByEndpoint
	instanceCount int64
	err         error
	// Track pagination params for testing
	lastPageRequest *InstancesQueryRequest
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
	// Return single instance if set, otherwise nil
	if m.instance != nil {
		return m.instance, nil
	}
	return nil, nil
}
func (m *mockDataQueryService) GetAllInstances(ctx context.Context, req *InstancesQueryRequest) (*InstancesListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Track the request for testing
	m.lastPageRequest = req
	// Calculate total pages based on instance count or default to len
	totalCount := m.instanceCount
	if totalCount == 0 {
		totalCount = int64(len(m.instances))
	}
	totalPages := int(totalCount) / req.Pagination.PageSize
	if int(totalCount) % req.Pagination.PageSize > 0 {
		totalPages++
	}
	return &InstancesListResponse{
		Data:       m.instances,
		Pagination: &PaginationMeta{TotalCount: totalCount, TotalPages: totalPages, CurrentPage: req.Pagination.Page, PageSize: req.Pagination.PageSize},
	}, nil
}
func (m *mockDataQueryService) GetAlertsByEndpoint(ctx context.Context, endpoint string) ([]*Alert, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []*Alert{}, nil
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

func TestGetInstances_DefaultPagination(t *testing.T) {
	mockService := &mockDataQueryService{
		instances: []*InstanceMeta{
			{ID: 1, InstanceEndpoint: "instance-1"},
			{ID: 2, InstanceEndpoint: "instance-2"},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/instances")
	handler.GetInstances(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetInstances() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp InstancesListResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check pagination defaults
	if resp.Pagination == nil {
		t.Fatal("GetInstances() pagination is nil")
	}
	if resp.Pagination.CurrentPage != 1 {
		t.Errorf("GetInstances() current_page = %d, want 1", resp.Pagination.CurrentPage)
	}
	if resp.Pagination.PageSize != 20 {
		t.Errorf("GetInstances() page_size = %d, want 20", resp.Pagination.PageSize)
	}
}

func TestGetInstances_CustomPagination(t *testing.T) {
	mockService := &mockDataQueryService{
		instances: []*InstanceMeta{
			{ID: 1, InstanceEndpoint: "instance-1"},
			{ID: 2, InstanceEndpoint: "instance-2"},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/instances?page=2&page_size=10")
	handler.GetInstances(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetInstances() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp InstancesListResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Pagination == nil {
		t.Fatal("GetInstances() pagination is nil")
	}
	if resp.Pagination.CurrentPage != 2 {
		t.Errorf("GetInstances() current_page = %d, want 2", resp.Pagination.CurrentPage)
	}
	if resp.Pagination.PageSize != 10 {
		t.Errorf("GetInstances() page_size = %d, want 10", resp.Pagination.PageSize)
	}
}

func TestGetInstances_InvalidPage(t *testing.T) {
	mockService := &mockDataQueryService{
		instances: []*InstanceMeta{{ID: 1, InstanceEndpoint: "instance-1"}},
	}
	handler := NewHandler(mockService)

	tests := []struct {
		name  string
		uri   string
		wantPage int
	}{
		{"page=0", "/instances?page=0", 1},
		{"page=-1", "/instances?page=-1", 1},
		{"page=abc", "/instances?page=abc", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, reqCtx := createTestRequestContext("")
			reqCtx.Request.SetRequestURI(tt.uri)
			handler.GetInstances(ctx, reqCtx)

			var resp InstancesListResponse
			if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if resp.Pagination.CurrentPage != tt.wantPage {
				t.Errorf("GetInstances() current_page = %d, want %d", resp.Pagination.CurrentPage, tt.wantPage)
			}
		})
	}
}

func TestGetInstances_InvalidPageSize(t *testing.T) {
	mockService := &mockDataQueryService{
		instances: []*InstanceMeta{{ID: 1, InstanceEndpoint: "instance-1"}},
	}
	handler := NewHandler(mockService)

	tests := []struct {
		name  string
		uri   string
		wantPageSize int
	}{
		{"page_size=0", "/instances?page_size=0", 20},
		{"page_size=-1", "/instances?page_size=-1", 20},
		{"page_size=abc", "/instances?page_size=abc", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, reqCtx := createTestRequestContext("")
			reqCtx.Request.SetRequestURI(tt.uri)
			handler.GetInstances(ctx, reqCtx)

			var resp InstancesListResponse
			if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if resp.Pagination.PageSize != tt.wantPageSize {
				t.Errorf("GetInstances() page_size = %d, want %d", resp.Pagination.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestGetInstances_MaxPageSizeExceeded(t *testing.T) {
	mockService := &mockDataQueryService{
		instances: []*InstanceMeta{{ID: 1, InstanceEndpoint: "instance-1"}},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/instances?page_size=200")
	handler.GetInstances(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetInstances() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp InstancesListResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Pagination.PageSize != 100 {
		t.Errorf("GetInstances() page_size = %d, want 100 (max limit)", resp.Pagination.PageSize)
	}
}

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

func TestGetAlerts(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "endpoint", Value: "pg-cn-north-2-ecom-user-01"}}

	handler.GetAlerts(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetAlerts() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp AlertsResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Data == nil {
		t.Error("GetAlerts() data should not be nil")
	}
}

func TestGetAlerts_MissingEndpoint(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	// No endpoint parameter provided
	reqCtx.Params = param.Params{}

	handler.GetAlerts(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("GetAlerts() status = %d, want 400", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INVALID_PARAMETER" {
		t.Errorf("GetAlerts() error code = %s, want INVALID_PARAMETER", resp.Error.Code)
	}
}

func TestGetAlerts_ServiceError(t *testing.T) {
	mockService := &mockDataQueryService{err: context.Canceled}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "endpoint", Value: "pg-cn-north-2-ecom-user-01"}}

	handler.GetAlerts(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetAlerts() status = %d, want 500", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("GetAlerts() error code = %s, want INTERNAL_ERROR", resp.Error.Code)
	}
}

// Step parsing tests

func TestParseStep(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      time.Duration
		wantError bool
	}{
		{"empty", "", 0, false},
		{"1 minute", "1m", time.Minute, false},
		{"5 minutes", "5m", 5 * time.Minute, false},
		{"30 minutes", "30m", 30 * time.Minute, false},
		{"1 hour", "1h", time.Hour, false},
		{"invalid format", "abc", 0, true},
		{"below minimum - 30 seconds", "30s", 0, true},
		{"below minimum - 500ms", "500ms", 0, true},
		{"above maximum - 2 hours", "2h", 0, true},
		{"above maximum - 24 hours", "24h", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStep(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("parseStep(%s) expected error, got nil", tt.input)
				}
				// Check it's an InvalidStepError
				if _, ok := err.(*InvalidStepError); !ok {
					t.Errorf("parseStep(%s) error type = %T, want *InvalidStepError", tt.input, err)
				}
			} else {
				if err != nil {
					t.Errorf("parseStep(%s) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("parseStep(%s) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestGetSeries_WithStep(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{
					ID:       1,
					Endpoint: "/api/metrics",
					Metric:   "cpu_usage",
				},
				Points: []DataPoint{
					{Time: parseTestTime("2024-01-01T00:00:00Z"), Value: 55.0}, // averaged value
				},
			},
		},
	}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/series?endpoint=/api/metrics&metric=cpu_usage&start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z&step=5m")

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

func TestGetSeries_InvalidStep_BelowMinimum(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/series?endpoint=/api/metrics&start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z&step=30s")

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
	// Check error message contains "minimum step"
	if !containsString(resp.Error.Message, "minimum step") {
		t.Errorf("GetSeries() error message should contain 'minimum step', got: %s", resp.Error.Message)
	}
}

func TestGetSeries_InvalidStep_AboveMaximum(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/series?endpoint=/api/metrics&start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z&step=2h")

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
	// Check error message contains "maximum step"
	if !containsString(resp.Error.Message, "maximum step") {
		t.Errorf("GetSeries() error message should contain 'maximum step', got: %s", resp.Error.Message)
	}
}

func TestGetSeries_InvalidStep_Format(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/series?endpoint=/api/metrics&start=2024-01-01T00:00:00Z&end=2024-01-01T01:00:00Z&step=abc")

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
	// Check error message contains "invalid format"
	if !containsString(resp.Error.Message, "invalid format") {
		t.Errorf("GetSeries() error message should contain 'invalid format', got: %s", resp.Error.Message)
	}
}

func TestGetSeries_NoStep(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{
					ID:       1,
					Endpoint: "/api/metrics",
					Metric:   "cpu_usage",
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
	// No step parameter - should return raw data
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
	// Without step, we expect raw data points
	if len(resp.Data[0].Points) != 2 {
		t.Errorf("GetSeries() points count = %d, want 2 (raw data)", len(resp.Data[0].Points))
	}
}

func TestInvalidStepError(t *testing.T) {
	err := &InvalidStepError{Value: "30s", Reason: "minimum step is 1m"}
	expected := "invalid step '30s': minimum step is 1m"
	if err.Error() != expected {
		t.Errorf("InvalidStepError.Error() = %s, want %s", err.Error(), expected)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================
// Phase 2: Handler Layer Unit Tests
// ============================================================

func TestGetInstance_Success(t *testing.T) {
	mockService := &mockDataQueryService{
		instance: &InstanceMeta{
			ID:               1,
			DbType:           "mysql",
			EntityName:       "finance-order",
			InstanceEndpoint: "mysql-cn-east-1-finance-order-01",
			InstanceVip:      "10.0.1.100",
			InstancePort:     3306,
			Status:           "active",
		},
	}

	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "endpoint", Value: "mysql-cn-east-1-finance-order-01"}}

	handler.GetInstance(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetInstance() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp InstanceMetaResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("GetInstance() data is nil")
	}

	if resp.Data.InstanceEndpoint != "mysql-cn-east-1-finance-order-01" {
		t.Errorf("GetInstance() endpoint = %s, want mysql-cn-east-1-finance-order-01", resp.Data.InstanceEndpoint)
	}

	if resp.Data.DbType != "mysql" {
		t.Errorf("GetInstance() db_type = %s, want mysql", resp.Data.DbType)
	}
}

func TestGetInstance_NotFound(t *testing.T) {
	mockService := &mockDataQueryService{
		instance: nil, // No instance found
	}

	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "endpoint", Value: "nonexistent-endpoint"}}

	handler.GetInstance(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 404 {
		t.Errorf("GetInstance() status = %d, want 404", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("GetInstance() error code = %s, want NOT_FOUND", resp.Error.Code)
	}
}

func TestGetInstance_ServiceError(t *testing.T) {
	mockService := &mockDataQueryService{err: context.Canceled}

	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "endpoint", Value: "test-endpoint"}}

	handler.GetInstance(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetInstance() status = %d, want 500", reqCtx.Response.StatusCode())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("GetInstance() error code = %s, want INTERNAL_ERROR", resp.Error.Code)
	}
}

func TestGetSeries_UnixTimestampFormat(t *testing.T) {
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

	// Use Unix timestamps
	now := time.Now().UTC()
	startUnix := now.Add(-1 * time.Hour).Unix()
	endUnix := now.Unix()

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI(fmt.Sprintf("/series?start=%d&end=%d", startUnix, endUnix))

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

func TestGetSeriesByID_UnixTimestampFormat(t *testing.T) {
	mockService := &mockDataQueryService{
		series: []*SeriesData{
			{
				Meta: SeriesMeta{
					ID:       1,
					Endpoint: "/api/metrics",
					Metric:   "cpu_usage",
				},
				Points: []DataPoint{
					{Time: time.Now(), Value: 50.0},
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

	// Use Unix timestamps for time range
	now := time.Now().UTC()
	startUnix := now.Add(-1 * time.Hour).Unix()
	endUnix := now.Unix()

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Params = param.Params{{Key: "id", Value: "1"}}
	reqCtx.Request.SetRequestURI(fmt.Sprintf("/series/1?start=%d&end=%d", startUnix, endUnix))

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
}

func TestGetSeries_OnlyStartProvided(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/series?start=2024-01-01T00:00:00Z")

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

	if !containsString(resp.Error.Message, "start and end") {
		t.Errorf("GetSeries() error message should mention 'start and end', got: %s", resp.Error.Message)
	}
}

func TestGetSeries_OnlyEndProvided(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	reqCtx.Request.SetRequestURI("/series?end=2024-01-01T01:00:00Z")

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

func TestErrorResponse_Format(t *testing.T) {
	mockService := &mockDataQueryService{}
	handler := NewHandler(mockService)

	ctx, reqCtx := createTestRequestContext("")
	// Missing endpoint parameter
	reqCtx.Request.SetRequestURI("/metrics")

	handler.GetMetrics(ctx, reqCtx)

	var resp ErrorResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	// Verify error response structure
	if resp.Error.Code == "" {
		t.Error("ErrorResponse.Code is empty")
	}
	if resp.Error.Message == "" {
		t.Error("ErrorResponse.Message is empty")
	}

	// Verify JSON structure
	var raw map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &raw); err != nil {
		t.Fatalf("Failed to parse raw JSON: %v", err)
	}

	errorObj, ok := raw["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Error response should have 'error' object")
	}

	if _, ok := errorObj["code"].(string); !ok {
		t.Error("Error object should have 'code' string field")
	}
	if _, ok := errorObj["message"].(string); !ok {
		t.Error("Error object should have 'message' string field")
	}
}

func TestToSeriesDataDTO_NilStatistics(t *testing.T) {
	seriesData := &SeriesData{
		Meta: SeriesMeta{
			ID:        1,
			Endpoint:  "/api/metrics",
			Metric:    "cpu_usage",
			Labels:    map[string]string{"host": "server1"},
			CreatedAt: time.Now(),
		},
		Points:     []DataPoint{},
		Statistics: nil, // nil statistics
	}

	dto := toSeriesDataDTO(seriesData)

	if dto.ID != "1" {
		t.Errorf("toSeriesDataDTO() ID = %s, want 1", dto.ID)
	}

	if dto.Statistics != nil {
		t.Errorf("toSeriesDataDTO() Statistics should be nil when input is nil, got %+v", dto.Statistics)
	}
}

func TestToSeriesDataDTO_WithStatistics(t *testing.T) {
	seriesData := &SeriesData{
		Meta: SeriesMeta{
			ID:        1,
			Endpoint:  "/api/metrics",
			Metric:    "cpu_usage",
			Labels:    map[string]string{"host": "server1"},
			CreatedAt: time.Now(),
		},
		Points: []DataPoint{},
		Statistics: &SeriesStatistics{
			Min:   10.0,
			Max:   100.0,
			Avg:   55.0,
			Sum:   550.0,
			Count: 10,
		},
	}

	dto := toSeriesDataDTO(seriesData)

	if dto.Statistics == nil {
		t.Fatal("toSeriesDataDTO() Statistics is nil")
	}

	if dto.Statistics.Min != 10.0 {
		t.Errorf("toSeriesDataDTO() Statistics.Min = %f, want 10.0", dto.Statistics.Min)
	}
	if dto.Statistics.Max != 100.0 {
		t.Errorf("toSeriesDataDTO() Statistics.Max = %f, want 100.0", dto.Statistics.Max)
	}
	if dto.Statistics.Count != 10 {
		t.Errorf("toSeriesDataDTO() Statistics.Count = %d, want 10", dto.Statistics.Count)
	}
}

func TestToDataPointDTOs_EmptySlice(t *testing.T) {
	points := []DataPoint{}

	dto := toDataPointDTOs(points)

	// Empty slice should return nil
	if dto != nil {
		t.Errorf("toDataPointDTOs() with empty slice should return nil, got %+v", dto)
	}
}

func TestToDataPointDTOs_NilSlice(t *testing.T) {
	var points []DataPoint = nil

	dto := toDataPointDTOs(points)

	// Nil slice should return nil
	if dto != nil {
		t.Errorf("toDataPointDTOs() with nil slice should return nil, got %+v", dto)
	}
}

func TestToDataPointDTOs_WithData(t *testing.T) {
	now := time.Now()
	points := []DataPoint{
		{Time: now, Value: 50.0},
		{Time: now.Add(1 * time.Minute), Value: 60.0},
	}

	dto := toDataPointDTOs(points)

	if len(dto) != 2 {
		t.Fatalf("toDataPointDTOs() returned %d points, want 2", len(dto))
	}

	if dto[0].Value != 50.0 {
		t.Errorf("toDataPointDTOs() first point value = %f, want 50.0", dto[0].Value)
	}
	if dto[1].Value != 60.0 {
		t.Errorf("toDataPointDTOs() second point value = %f, want 60.0", dto[1].Value)
	}
}

func TestToSeriesDataDTOs_EmptySlice(t *testing.T) {
	series := []*SeriesData{}

	dto := toSeriesDataDTOs(series)

	// Empty slice should return empty slice, not nil
	if dto == nil {
		t.Error("toSeriesDataDTOs() with empty slice should return empty slice, not nil")
	}
	if len(dto) != 0 {
		t.Errorf("toSeriesDataDTOs() returned %d items, want 0", len(dto))
	}
}

func TestToSeriesStatisticsDTO_Nil(t *testing.T) {
	dto := toSeriesStatisticsDTO(nil)

	if dto != nil {
		t.Errorf("toSeriesStatisticsDTO(nil) should return nil, got %+v", dto)
	}
}

func TestToSeriesStatisticsDTO_WithValue(t *testing.T) {
	stats := &SeriesStatistics{
		Min:   10.0,
		Max:   100.0,
		Avg:   55.0,
		Sum:   550.0,
		Count: 10,
	}

	dto := toSeriesStatisticsDTO(stats)

	if dto == nil {
		t.Fatal("toSeriesStatisticsDTO() returned nil")
	}

	if dto.Min != 10.0 {
		t.Errorf("toSeriesStatisticsDTO() Min = %f, want 10.0", dto.Min)
	}
	if dto.Count != 10 {
		t.Errorf("toSeriesStatisticsDTO() Count = %d, want 10", dto.Count)
	}
}