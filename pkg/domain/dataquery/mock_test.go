package dataquery

import (
	"testing"
	"time"
)

func TestMockServiceName(t *testing.T) {
	svc := NewMockService()
	if svc.Name() != "MockDataQueryService" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "MockDataQueryService")
	}
}

func TestMockServiceLifecycle(t *testing.T) {
	svc := NewMockService()

	if err := svc.Initialize(nil); err != nil {
		t.Errorf("Initialize() error = %v", err)
	}

	if err := svc.Shutdown(nil); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if err := svc.Health(nil); err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func TestMockServiceGetEndpoints(t *testing.T) {
	svc := NewMockService()

	endpoints, err := svc.GetEndpoints(nil)
	if err != nil {
		t.Fatalf("GetEndpoints() error = %v", err)
	}

	expected := []string{"/api/metrics", "/api/health", "/api/query"}
	if len(endpoints) != len(expected) {
		t.Errorf("GetEndpoints() returned %d endpoints, want %d", len(endpoints), len(expected))
	}
}

func TestMockServiceGetMetrics(t *testing.T) {
	svc := NewMockService()

	tests := []struct {
		endpoint  string
		wantCount int
	}{
		{"/api/metrics", 4},
		{"/api/health", 2},
		{"/api/query", 2},
		{"/unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			metrics, err := svc.GetMetrics(nil, tt.endpoint)
			if err != nil {
				t.Fatalf("GetMetrics() error = %v", err)
			}

			if len(metrics) != tt.wantCount {
				t.Errorf("GetMetrics(%q) returned %d metrics, want %d", tt.endpoint, len(metrics), tt.wantCount)
			}
		})
	}
}

func TestMockServiceQuerySeries(t *testing.T) {
	svc := NewMockService()

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	// Query all series
	series, err := svc.QuerySeries(nil, &SeriesQuery{TimeRange: timeRange})
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}

	if len(series) == 0 {
		t.Error("QuerySeries() returned no series")
	}

	// Verify each series has points
	for _, s := range series {
		if len(s.Points) == 0 {
			t.Errorf("Series %d has no points", s.Meta.ID)
		}
	}
}

func TestMockServiceQuerySeriesWithFilters(t *testing.T) {
	svc := NewMockService()

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	tests := []struct {
		name      string
		req       *SeriesQuery
		wantCount int
	}{
		{"by endpoint", &SeriesQuery{Endpoint: "/api/metrics", TimeRange: timeRange}, 3},
		{"by metric", &SeriesQuery{Metric: "cpu_usage", TimeRange: timeRange}, 3},
		{"non-existent endpoint", &SeriesQuery{Endpoint: "/unknown", TimeRange: timeRange}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := svc.QuerySeries(nil, tt.req)
			if err != nil {
				t.Fatalf("QuerySeries() error = %v", err)
			}

			if len(series) != tt.wantCount {
				t.Errorf("QuerySeries() returned %d series, want %d", len(series), tt.wantCount)
			}
		})
	}
}

func TestMockServiceGetSeriesByID(t *testing.T) {
	svc := NewMockService()

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	// Test existing series
	series, err := svc.GetSeriesByID(nil, 1, &timeRange)
	if err != nil {
		t.Fatalf("GetSeriesByID() error = %v", err)
	}

	if series == nil {
		t.Fatal("GetSeriesByID() returned nil for existing series")
	}

	if len(series.Points) == 0 {
		t.Error("GetSeriesByID() returned series with no points")
	}

	// Test non-existent series
	series, err = svc.GetSeriesByID(nil, 999, &timeRange)
	if err != nil {
		t.Fatalf("GetSeriesByID() error = %v", err)
	}

	if series != nil {
		t.Error("GetSeriesByID() should return nil for non-existent series")
	}
}

func TestMockServiceQuerySeriesMulti(t *testing.T) {
	svc := NewMockService()

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	tests := []struct {
		name      string
		req       *MultiSeriesQuery
		wantCount int
	}{
		{"all series", &MultiSeriesQuery{TimeRange: timeRange}, 3},
		{"by endpoints", &MultiSeriesQuery{Endpoints: []string{"/api/metrics"}, TimeRange: timeRange}, 3},
		{"by metrics", &MultiSeriesQuery{Metrics: []string{"cpu_usage"}, TimeRange: timeRange}, 3},
		{"with aggregation", &MultiSeriesQuery{TimeRange: timeRange, Aggregation: &Aggregation{Interval: "5m", Function: AggAvg}}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := svc.QuerySeriesMulti(nil, tt.req)
			if err != nil {
				t.Fatalf("QuerySeriesMulti() error = %v", err)
			}

			if len(series) != tt.wantCount {
				t.Errorf("QuerySeriesMulti() returned %d series, want %d", len(series), tt.wantCount)
			}
		})
	}
}

func TestMockServiceTimeRangeFiltering(t *testing.T) {
	svc := NewMockService()

	// Create a time range outside the data range
	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),    // Only last hour
		End:   now.Add(-30 * time.Minute), // Ending 30 mins ago
	}

	series, err := svc.QuerySeries(nil, &SeriesQuery{TimeRange: timeRange})
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}

	// Points should be filtered by time range
	for _, s := range series {
		for _, p := range s.Points {
			if p.Time.Before(timeRange.Start) || p.Time.After(timeRange.End) {
				t.Errorf("Point time %v is outside range [%v, %v]", p.Time, timeRange.Start, timeRange.End)
			}
		}
	}
}
