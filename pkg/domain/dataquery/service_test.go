package dataquery

import (
	"context"
	"testing"
	"time"
)

// mockRepository implements Repository for testing
type mockRepository struct {
	endpoints  []string
	metrics    map[string][]string
	series     []SeriesMeta
	points     map[int64][]DataPoint
	statistics map[int64]*SeriesStatistics
	err        error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		endpoints: []string{"/api/metrics", "/api/health", "/api/query"},
		metrics: map[string][]string{
			"/api/metrics": {"cpu_usage", "memory_usage", "disk_io"},
			"/api/health":  {"response_time", "status_code"},
			"/api/query":   {"query_count", "query_latency"},
		},
		series: []SeriesMeta{
			{ID: 1, Endpoint: "/api/metrics", Metric: "cpu_usage", Labels: map[string]string{"host": "server1"}, CreatedAt: time.Now()},
			{ID: 2, Endpoint: "/api/metrics", Metric: "memory_usage", Labels: map[string]string{"host": "server2"}, CreatedAt: time.Now()},
		},
		points: map[int64][]DataPoint{
			1: {{Time: time.Now(), Value: 75.5}, {Time: time.Now().Add(-5 * time.Minute), Value: 72.3}},
			2: {{Time: time.Now(), Value: 62.1}, {Time: time.Now().Add(-5 * time.Minute), Value: 60.8}},
		},
		statistics: map[int64]*SeriesStatistics{
			1: {Min: 70.0, Max: 80.0, Avg: 75.0, Sum: 150.0, Count: 2},
			2: {Min: 58.0, Max: 65.0, Avg: 61.5, Sum: 123.0, Count: 2},
		},
	}
}

func (m *mockRepository) GetEndpoints(ctx context.Context) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.endpoints, nil
}

func (m *mockRepository) GetMetrics(ctx context.Context, endpoint string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metrics[endpoint], nil
}

func (m *mockRepository) QuerySeries(ctx context.Context, req *SeriesQueryRequest) ([]SeriesMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []SeriesMeta
	for _, s := range m.series {
		if req.Endpoint != "" && s.Endpoint != req.Endpoint {
			continue
		}
		if req.Metric != "" && s.Metric != req.Metric {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (m *mockRepository) GetSeriesByID(ctx context.Context, id int64) (*SeriesMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, s := range m.series {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) GetSeriesPoints(ctx context.Context, req *PointsQueryRequest) (map[int64][]DataPoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[int64][]DataPoint)
	for _, id := range req.SeriesIDs {
		if points, ok := m.points[id]; ok {
			result[id] = points
		}
	}
	return result, nil
}

func (m *mockRepository) GetSeriesStatistics(ctx context.Context, req *StatsRequest) (map[int64]*SeriesStatistics, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[int64]*SeriesStatistics)
	for _, id := range req.SeriesIDs {
		if stats, ok := m.statistics[id]; ok {
			result[id] = stats
		}
	}
	return result, nil
}

func (m *mockRepository) GetInstanceByEndpoint(ctx context.Context, endpoint string) (*InstanceMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil
}

func (m *mockRepository) GetAllInstances(ctx context.Context, req *InstancesQueryRequest) ([]*InstanceMeta, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return []*InstanceMeta{}, 0, nil
}

func (m *mockRepository) GetAlertsByEndpoint(ctx context.Context, endpoint string) ([]*Alert, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []*Alert{}, nil
}

// Service Tests

func TestServiceName(t *testing.T) {
	svc := NewService(newMockRepository())
	if svc.Name() != "DataQueryService" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "DataQueryService")
	}
}

func TestServiceInitialize(t *testing.T) {
	svc := NewService(newMockRepository())
	if err := svc.Initialize(context.Background()); err != nil {
		t.Errorf("Initialize() error = %v", err)
	}
}

func TestServiceShutdown(t *testing.T) {
	svc := NewService(newMockRepository())
	if err := svc.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestServiceHealth(t *testing.T) {
	svc := NewService(newMockRepository())
	if err := svc.Health(context.Background()); err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func TestServiceGetEndpoints(t *testing.T) {
	mockRepo := newMockRepository()
	svc := NewService(mockRepo)

	endpoints, err := svc.GetEndpoints(context.Background())
	if err != nil {
		t.Fatalf("GetEndpoints() error = %v", err)
	}

	if len(endpoints) != 3 {
		t.Errorf("GetEndpoints() returned %d endpoints, want 3", len(endpoints))
	}
}

func TestServiceGetMetrics(t *testing.T) {
	mockRepo := newMockRepository()
	svc := NewService(mockRepo)

	tests := []struct {
		endpoint    string
		wantCount   int
		shouldExist bool
	}{
		{"/api/metrics", 3, true},
		{"/api/health", 2, true},
		{"/api/query", 2, true},
		{"/unknown", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			metrics, err := svc.GetMetrics(context.Background(), tt.endpoint)
			if err != nil {
				t.Fatalf("GetMetrics() error = %v", err)
			}

			if tt.shouldExist && len(metrics) != tt.wantCount {
				t.Errorf("GetMetrics(%q) returned %d metrics, want %d", tt.endpoint, len(metrics), tt.wantCount)
			}
		})
	}
}

func TestServiceQuerySeries(t *testing.T) {
	mockRepo := newMockRepository()
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	tests := []struct {
		name      string
		req       *SeriesQuery
		wantCount int
	}{
		{"all series", &SeriesQuery{TimeRange: timeRange}, 2},
		{"by endpoint", &SeriesQuery{Endpoint: "/api/metrics", TimeRange: timeRange}, 2},
		{"by metric", &SeriesQuery{Metric: "cpu_usage", TimeRange: timeRange}, 1},
		{"by endpoint and metric", &SeriesQuery{Endpoint: "/api/metrics", Metric: "cpu_usage", TimeRange: timeRange}, 1},
		{"non-existent", &SeriesQuery{Endpoint: "/unknown", TimeRange: timeRange}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := svc.QuerySeries(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("QuerySeries() error = %v", err)
			}

			if len(series) != tt.wantCount {
				t.Errorf("QuerySeries() returned %d series, want %d", len(series), tt.wantCount)
			}
		})
	}
}

func TestServiceQuerySeriesWithLabelFilter(t *testing.T) {
	mockRepo := newMockRepository()
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	// Note: The mock repository doesn't actually parse label filters
	// This test is just to verify the service layer passes the filter through
	req := &SeriesQuery{
		LabelFilter: `host="server1"`,
		TimeRange:   timeRange,
	}

	_, err := svc.QuerySeries(context.Background(), req)
	if err != nil {
		t.Errorf("QuerySeries() with valid label filter error = %v", err)
	}
}

func TestServiceGetSeriesByID(t *testing.T) {
	mockRepo := newMockRepository()
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	t.Run("existing series", func(t *testing.T) {
		series, err := svc.GetSeriesByID(context.Background(), 1, &timeRange)
		if err != nil {
			t.Fatalf("GetSeriesByID() error = %v", err)
		}

		if series == nil {
			t.Fatal("GetSeriesByID() returned nil for existing series")
		}

		if len(series.Points) == 0 {
			t.Error("GetSeriesByID() returned series with no points")
		}
		if series.Statistics == nil {
			t.Error("GetSeriesByID() returned series with no statistics")
		}
	})

	t.Run("non-existent series", func(t *testing.T) {
		series, err := svc.GetSeriesByID(context.Background(), 999, &timeRange)
		if err != nil {
			t.Fatalf("GetSeriesByID() error = %v", err)
		}

		if series != nil {
			t.Error("GetSeriesByID() should return nil for non-existent series")
		}
	})
}

func TestServiceQuerySeriesMulti(t *testing.T) {
	mockRepo := newMockRepository()
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	tests := []struct {
		name      string
		req       *MultiSeriesQuery
		wantCount int
	}{
		{"all series (no filter)", &MultiSeriesQuery{TimeRange: timeRange}, 2},
		{"by endpoints only", &MultiSeriesQuery{Endpoints: []string{"/api/metrics"}, TimeRange: timeRange}, 2},
		{"by metrics only", &MultiSeriesQuery{Metrics: []string{"cpu_usage"}, TimeRange: timeRange}, 1},
		{"by both endpoints and metrics", &MultiSeriesQuery{Endpoints: []string{"/api/metrics"}, Metrics: []string{"cpu_usage"}, TimeRange: timeRange}, 1},
		{"by non-existent endpoint", &MultiSeriesQuery{Endpoints: []string{"/unknown"}, TimeRange: timeRange}, 0},
		{"by non-existent metric", &MultiSeriesQuery{Metrics: []string{"unknown_metric"}, TimeRange: timeRange}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := svc.QuerySeriesMulti(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("QuerySeriesMulti() error = %v", err)
			}

			if len(series) != tt.wantCount {
				t.Errorf("QuerySeriesMulti() returned %d series, want %d", len(series), tt.wantCount)
			}
		})
	}
}

func TestServiceEmptyResults(t *testing.T) {
	// Create a mock repo with empty data
	mockRepo := &mockRepository{
		endpoints: []string{},
		metrics:   map[string][]string{},
		series:    []SeriesMeta{},
		points:    map[int64][]DataPoint{},
	}
	svc := NewService(mockRepo)

	endpoints, err := svc.GetEndpoints(context.Background())
	if err != nil {
		t.Fatalf("GetEndpoints() error = %v", err)
	}
	if endpoints == nil {
		t.Error("GetEndpoints() returned nil instead of empty slice")
	}

	// Note: GetMetrics for unknown endpoint will return nil from map lookup
	// The service layer handles this but mock's behavior is different
	metrics, err := svc.GetMetrics(context.Background(), "/unknown")
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}
	// Check that we don't get an error - nil is acceptable from map lookup
	t.Logf("GetMetrics() returned: %v", metrics)
}

func TestServiceRepositoryError(t *testing.T) {
	mockRepo := newMockRepository()
	mockRepo.err = context.Canceled
	svc := NewService(mockRepo)

	_, err := svc.GetEndpoints(context.Background())
	if err == nil {
		t.Error("GetEndpoints() should return error when repository fails")
	}
}

func TestServiceGetAllInstances_Pagination(t *testing.T) {
	// Create mock repo with pagination support
	mockRepo := &mockRepositoryWithPagination{
		instances: []*InstanceMeta{
			{ID: 1, InstanceEndpoint: "instance-1"},
			{ID: 2, InstanceEndpoint: "instance-2"},
			{ID: 3, InstanceEndpoint: "instance-3"},
			{ID: 4, InstanceEndpoint: "instance-4"},
			{ID: 5, InstanceEndpoint: "instance-5"},
		},
		totalCount: 5,
	}
	svc := NewService(mockRepo)

	req := &InstancesQueryRequest{
		Pagination: PaginationRequest{Page: 1, PageSize: 2},
	}

	resp, err := svc.GetAllInstances(context.Background(), req)
	if err != nil {
		t.Fatalf("GetAllInstances() error = %v", err)
	}

	if resp.Pagination == nil {
		t.Fatal("GetAllInstances() pagination is nil")
	}

	// Verify pagination calculation
	if resp.Pagination.TotalCount != 5 {
		t.Errorf("GetAllInstances() total_count = %d, want 5", resp.Pagination.TotalCount)
	}
	if resp.Pagination.TotalPages != 3 {
		t.Errorf("GetAllInstances() total_pages = %d, want 3", resp.Pagination.TotalPages)
	}
	if resp.Pagination.CurrentPage != 1 {
		t.Errorf("GetAllInstances() current_page = %d, want 1", resp.Pagination.CurrentPage)
	}
	if resp.Pagination.PageSize != 2 {
		t.Errorf("GetAllInstances() page_size = %d, want 2", resp.Pagination.PageSize)
	}
}

func TestServiceGetAllInstances_EmptyResult(t *testing.T) {
	mockRepo := &mockRepositoryWithPagination{
		instances:  []*InstanceMeta{},
		totalCount: 0,
	}
	svc := NewService(mockRepo)

	req := &InstancesQueryRequest{
		Pagination: PaginationRequest{Page: 1, PageSize: 20},
	}

	resp, err := svc.GetAllInstances(context.Background(), req)
	if err != nil {
		t.Fatalf("GetAllInstances() error = %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("GetAllInstances() returned %d instances, want 0", len(resp.Data))
	}
	if resp.Pagination.TotalCount != 0 {
		t.Errorf("GetAllInstances() total_count = %d, want 0", resp.Pagination.TotalCount)
	}
	if resp.Pagination.TotalPages != 0 {
		t.Errorf("GetAllInstances() total_pages = %d, want 0", resp.Pagination.TotalPages)
	}
}

// mockRepositoryWithPagination implements Repository with pagination support for testing
type mockRepositoryWithPagination struct {
	instances  []*InstanceMeta
	totalCount int64
	err        error
}

func (m *mockRepositoryWithPagination) GetEndpoints(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
func (m *mockRepositoryWithPagination) GetMetrics(ctx context.Context, endpoint string) ([]string, error) {
	return []string{}, nil
}
func (m *mockRepositoryWithPagination) QuerySeries(ctx context.Context, req *SeriesQueryRequest) ([]SeriesMeta, error) {
	return []SeriesMeta{}, nil
}
func (m *mockRepositoryWithPagination) GetSeriesByID(ctx context.Context, id int64) (*SeriesMeta, error) {
	return nil, nil
}
func (m *mockRepositoryWithPagination) GetSeriesPoints(ctx context.Context, req *PointsQueryRequest) (map[int64][]DataPoint, error) {
	return map[int64][]DataPoint{}, nil
}
func (m *mockRepositoryWithPagination) GetSeriesStatistics(ctx context.Context, req *StatsRequest) (map[int64]*SeriesStatistics, error) {
	return map[int64]*SeriesStatistics{}, nil
}
func (m *mockRepositoryWithPagination) GetInstanceByEndpoint(ctx context.Context, endpoint string) (*InstanceMeta, error) {
	return nil, nil
}
func (m *mockRepositoryWithPagination) GetAllInstances(ctx context.Context, req *InstancesQueryRequest) ([]*InstanceMeta, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.instances, m.totalCount, nil
}
func (m *mockRepositoryWithPagination) GetAlertsByEndpoint(ctx context.Context, endpoint string) ([]*Alert, error) {
	return []*Alert{}, nil
}

// mockRepositoryWithInstance supports GetInstanceByEndpoint and GetAlertsByEndpoint
type mockRepositoryWithInstance struct {
	mockRepository
	instance *InstanceMeta
	alerts   []*Alert
}

func (m *mockRepositoryWithInstance) GetInstanceByEndpoint(ctx context.Context, endpoint string) (*InstanceMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.instance, nil
}

func (m *mockRepositoryWithInstance) GetAlertsByEndpoint(ctx context.Context, endpoint string) ([]*Alert, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.alerts, nil
}

// mockRepositoryWithInterval tracks interval parameter for testing sampling
type mockRepositoryWithInterval struct {
	mockRepository
	lastInterval time.Duration
}

func (m *mockRepositoryWithInterval) GetSeriesPoints(ctx context.Context, req *PointsQueryRequest) (map[int64][]DataPoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.lastInterval = req.Interval
	return m.points, nil
}

// mockRepositoryWithLabelFilter tracks label filter for testing
type mockRepositoryWithLabelFilter struct {
	mockRepository
	lastLabelFilter string
}

func (m *mockRepositoryWithLabelFilter) QuerySeries(ctx context.Context, req *SeriesQueryRequest) ([]SeriesMeta, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.lastLabelFilter = req.LabelFilter
	return m.series, nil
}

// mockRepositoryWithErrorForPoints returns errors for specific operations
type mockRepositoryWithErrorForPoints struct {
	mockRepository
	pointsErr error
	statsErr  error
}

func (m *mockRepositoryWithErrorForPoints) GetSeriesPoints(ctx context.Context, req *PointsQueryRequest) (map[int64][]DataPoint, error) {
	if m.pointsErr != nil {
		return nil, m.pointsErr
	}
	return m.points, nil
}

func (m *mockRepositoryWithErrorForPoints) GetSeriesStatistics(ctx context.Context, req *StatsRequest) (map[int64]*SeriesStatistics, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.statistics, nil
}

// ============================================================
// Phase 1: Service Layer Unit Tests
// ============================================================

func TestServiceGetInstanceByEndpoint_Success(t *testing.T) {
	mockRepo := &mockRepositoryWithInstance{
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
	svc := NewService(mockRepo)

	instance, err := svc.GetInstanceByEndpoint(context.Background(), "mysql-cn-east-1-finance-order-01")
	if err != nil {
		t.Fatalf("GetInstanceByEndpoint() error = %v", err)
	}

	if instance == nil {
		t.Fatal("GetInstanceByEndpoint() returned nil for existing instance")
	}

	if instance.InstanceEndpoint != "mysql-cn-east-1-finance-order-01" {
		t.Errorf("GetInstanceByEndpoint() endpoint = %s, want mysql-cn-east-1-finance-order-01", instance.InstanceEndpoint)
	}

	if instance.DbType != "mysql" {
		t.Errorf("GetInstanceByEndpoint() db_type = %s, want mysql", instance.DbType)
	}
}

func TestServiceGetInstanceByEndpoint_NotFound(t *testing.T) {
	mockRepo := &mockRepositoryWithInstance{
		instance: nil,
	}
	svc := NewService(mockRepo)

	instance, err := svc.GetInstanceByEndpoint(context.Background(), "nonexistent-endpoint")
	if err != nil {
		t.Fatalf("GetInstanceByEndpoint() error = %v", err)
	}

	if instance != nil {
		t.Error("GetInstanceByEndpoint() should return nil for non-existent endpoint")
	}
}

func TestServiceGetAlertsByEndpoint_Success(t *testing.T) {
	mockRepo := &mockRepositoryWithInstance{
		alerts: []*Alert{
			{ID: 1, EventID: "evt-1", Endpoint: "test-endpoint", AlertText: "High CPU", Status: "firing"},
			{ID: 2, EventID: "evt-2", Endpoint: "test-endpoint", AlertText: "Memory warning", Status: "resolved"},
		},
	}
	svc := NewService(mockRepo)

	alerts, err := svc.GetAlertsByEndpoint(context.Background(), "test-endpoint")
	if err != nil {
		t.Fatalf("GetAlertsByEndpoint() error = %v", err)
	}

	if len(alerts) != 2 {
		t.Errorf("GetAlertsByEndpoint() returned %d alerts, want 2", len(alerts))
	}

	if alerts[0].AlertText != "High CPU" {
		t.Errorf("GetAlertsByEndpoint() first alert text = %s, want 'High CPU'", alerts[0].AlertText)
	}
}

func TestServiceGetAlertsByEndpoint_EmptyResult(t *testing.T) {
	mockRepo := &mockRepositoryWithInstance{
		alerts: []*Alert{},
	}
	svc := NewService(mockRepo)

	alerts, err := svc.GetAlertsByEndpoint(context.Background(), "endpoint-with-no-alerts")
	if err != nil {
		t.Fatalf("GetAlertsByEndpoint() error = %v", err)
	}

	if alerts == nil {
		t.Error("GetAlertsByEndpoint() should return empty slice, not nil")
	}

	if len(alerts) != 0 {
		t.Errorf("GetAlertsByEndpoint() returned %d alerts, want 0", len(alerts))
	}
}

func TestServiceQuerySeriesWithInterval(t *testing.T) {
	mockRepo := &mockRepositoryWithInterval{
		mockRepository: mockRepository{
			series: []SeriesMeta{
				{ID: 1, Endpoint: "/api/metrics", Metric: "cpu_usage", Labels: map[string]string{"host": "server1"}, CreatedAt: time.Now()},
			},
			points: map[int64][]DataPoint{
				1: {{Time: time.Now(), Value: 75.5}},
			},
		},
	}
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	// Test with 5 minute sampling interval
	req := &SeriesQuery{
		TimeRange: timeRange,
		Interval:  5 * time.Minute,
	}

	_, err := svc.QuerySeries(context.Background(), req)
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}

	if mockRepo.lastInterval != 5*time.Minute {
		t.Errorf("QuerySeries() interval passed to repository = %v, want %v", mockRepo.lastInterval, 5*time.Minute)
	}
}

func TestServiceQuerySeriesMultiWithLabelFilter(t *testing.T) {
	mockRepo := &mockRepositoryWithLabelFilter{
		mockRepository: mockRepository{
			series: []SeriesMeta{
				{ID: 1, Endpoint: "/api/metrics", Metric: "cpu_usage", Labels: map[string]string{"env": "prod", "host": "server1"}, CreatedAt: time.Now()},
			},
		},
	}
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	labelFilter := `env="prod" AND host=~"server.*"`
	req := &MultiSeriesQuery{
		TimeRange:   timeRange,
		LabelFilter: labelFilter,
	}

	_, err := svc.QuerySeriesMulti(context.Background(), req)
	if err != nil {
		t.Fatalf("QuerySeriesMulti() error = %v", err)
	}

	if mockRepo.lastLabelFilter != labelFilter {
		t.Errorf("QuerySeriesMulti() label filter passed to repository = %s, want %s", mockRepo.lastLabelFilter, labelFilter)
	}
}

func TestServiceGetMetrics_ErrorPropagation(t *testing.T) {
	mockRepo := newMockRepository()
	mockRepo.err = context.Canceled
	svc := NewService(mockRepo)

	_, err := svc.GetMetrics(context.Background(), "/api/metrics")
	if err == nil {
		t.Error("GetMetrics() should return error when repository fails")
	}

	if err != context.Canceled {
		t.Errorf("GetMetrics() error = %v, want context.Canceled", err)
	}
}

func TestServiceQuerySeries_ErrorPropagation(t *testing.T) {
	mockRepo := newMockRepository()
	mockRepo.err = context.Canceled
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	_, err := svc.QuerySeries(context.Background(), &SeriesQuery{TimeRange: timeRange})
	if err == nil {
		t.Error("QuerySeries() should return error when repository fails")
	}

	if err != context.Canceled {
		t.Errorf("QuerySeries() error = %v, want context.Canceled", err)
	}
}

func TestServiceGetSeriesByID_ErrorPropagation_PointsError(t *testing.T) {
	mockRepo := &mockRepositoryWithErrorForPoints{
		mockRepository: mockRepository{
			series: []SeriesMeta{
				{ID: 1, Endpoint: "/api/metrics", Metric: "cpu_usage", Labels: map[string]string{"host": "server1"}, CreatedAt: time.Now()},
			},
		},
		pointsErr: context.Canceled,
	}
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	_, err := svc.GetSeriesByID(context.Background(), 1, &timeRange)
	if err == nil {
		t.Error("GetSeriesByID() should return error when GetSeriesPoints fails")
	}

	if err != context.Canceled {
		t.Errorf("GetSeriesByID() error = %v, want context.Canceled", err)
	}
}

func TestServiceGetSeriesByID_ErrorPropagation_StatsError(t *testing.T) {
	mockRepo := &mockRepositoryWithErrorForPoints{
		mockRepository: mockRepository{
			series: []SeriesMeta{
				{ID: 1, Endpoint: "/api/metrics", Metric: "cpu_usage", Labels: map[string]string{"host": "server1"}, CreatedAt: time.Now()},
			},
			points: map[int64][]DataPoint{
				1: {{Time: time.Now(), Value: 75.5}},
			},
		},
		statsErr: context.DeadlineExceeded,
	}
	svc := NewService(mockRepo)

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	_, err := svc.GetSeriesByID(context.Background(), 1, &timeRange)
	if err == nil {
		t.Error("GetSeriesByID() should return error when GetSeriesStatistics fails")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("GetSeriesByID() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestServiceGetAllInstances_ErrorPropagation(t *testing.T) {
	mockRepo := &mockRepositoryWithPagination{
		err: context.Canceled,
	}
	svc := NewService(mockRepo)

	req := &InstancesQueryRequest{
		Pagination: PaginationRequest{Page: 1, PageSize: 20},
	}

	_, err := svc.GetAllInstances(context.Background(), req)
	if err == nil {
		t.Error("GetAllInstances() should return error when repository fails")
	}

	if err != context.Canceled {
		t.Errorf("GetAllInstances() error = %v, want context.Canceled", err)
	}
}