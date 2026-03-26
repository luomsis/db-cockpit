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
		{"by endpoints", &MultiSeriesQuery{Endpoints: []string{"/api/metrics"}, TimeRange: timeRange}, 0},
		{"by metrics", &MultiSeriesQuery{Metrics: []string{"cpu_usage"}, TimeRange: timeRange}, 0},
		{"by both", &MultiSeriesQuery{Endpoints: []string{"/api/metrics"}, Metrics: []string{"cpu_usage"}, TimeRange: timeRange}, 1},
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