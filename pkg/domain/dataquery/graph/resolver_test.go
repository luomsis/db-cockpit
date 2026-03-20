package graph

import (
	"context"
	"testing"
	"time"

	"github.com/db-cockpit/pkg/domain/dataquery"
)

// mockService implements dataquery.DataQueryService for testing
type mockService struct {
	endpoints []string
	metrics   map[string][]string
	series    []*dataquery.SeriesData
}

func newMockService() *mockService {
	now := time.Now()
	return &mockService{
		endpoints: []string{"/api/metrics", "/api/health"},
		metrics: map[string][]string{
			"/api/metrics": {"cpu_usage", "memory_usage"},
			"/api/health":  {"response_time"},
		},
		series: []*dataquery.SeriesData{
			{
				Meta: dataquery.SeriesMeta{
					ID:        1,
					Endpoint:  "/api/metrics",
					Metric:    "cpu_usage",
					Labels:    map[string]string{"host": "server1", "region": "us-east"},
					CreatedAt: now,
				},
				Points: []dataquery.DataPoint{
					{Time: now, Value: 75.0},
					{Time: now.Add(-5 * time.Minute), Value: 72.0},
				},
				Statistics: &dataquery.SeriesStatistics{
					Min: 70.0, Max: 80.0, Avg: 75.0, Sum: 150.0, Count: 2,
				},
			},
		},
	}
}

func (m *mockService) Name() string                                       { return "MockService" }
func (m *mockService) Initialize(ctx context.Context) error               { return nil }
func (m *mockService) Shutdown(ctx context.Context) error                 { return nil }
func (m *mockService) Health(ctx context.Context) error                   { return nil }
func (m *mockService) GetEndpoints(ctx context.Context) ([]string, error) { return m.endpoints, nil }
func (m *mockService) GetMetrics(ctx context.Context, ep string) ([]string, error) {
	return m.metrics[ep], nil
}
func (m *mockService) QuerySeries(ctx context.Context, req *dataquery.SeriesQuery) ([]*dataquery.SeriesData, error) {
	return m.series, nil
}
func (m *mockService) QuerySeriesMulti(ctx context.Context, req *dataquery.MultiSeriesQuery) ([]*dataquery.SeriesData, error) {
	return m.series, nil
}
func (m *mockService) GetSeriesByID(ctx context.Context, id int64, tr *dataquery.TimeRange) (*dataquery.SeriesData, error) {
	for _, s := range m.series {
		if s.Meta.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

// Tests

func TestToSeriesMeta(t *testing.T) {
	now := time.Now()
	meta := dataquery.SeriesMeta{
		ID:         1,
		Endpoint:   "/api/metrics",
		Metric:     "cpu_usage",
		Labels:     map[string]string{"host": "server1", "region": "us-east"},
		LabelsHash: "abc123",
		CreatedAt:  now,
	}

	result := toSeriesMeta(meta)

	if result.ID != "1" {
		t.Errorf("ID = %q, want %q", result.ID, "1")
	}
	if result.Endpoint != "/api/metrics" {
		t.Errorf("Endpoint = %q, want %q", result.Endpoint, "/api/metrics")
	}
	if result.Metric != "cpu_usage" {
		t.Errorf("Metric = %q, want %q", result.Metric, "cpu_usage")
	}
	if result.Labels == nil {
		t.Error("Labels should not be nil")
	}
}

func TestToLabels(t *testing.T) {
	labels := map[string]string{
		"host":   "server1",
		"region": "us-east",
	}

	result := toLabels(labels)

	if len(result.Keys) != 2 {
		t.Errorf("Keys length = %d, want 2", len(result.Keys))
	}
	if len(result.Entries) != 2 {
		t.Errorf("Entries length = %d, want 2", len(result.Entries))
	}
}

func TestToDataPoints(t *testing.T) {
	now := time.Now()
	points := []dataquery.DataPoint{
		{Time: now, Value: 75.0},
		{Time: now.Add(-5 * time.Minute), Value: 72.0},
	}

	result := toDataPoints(points)

	if len(result) != 2 {
		t.Errorf("Points length = %d, want 2", len(result))
	}
	if result[0].Value != 75.0 {
		t.Errorf("Point[0].Value = %f, want 75.0", result[0].Value)
	}
}

func TestToAggregatedPoints(t *testing.T) {
	now := time.Now()
	points := []dataquery.AggregatedPoint{
		{Time: now, Value: 75.0, Count: 10},
		{Time: now.Add(-1 * time.Hour), Value: 72.0, Count: 12},
	}

	result := toAggregatedPoints(points)

	if len(result) != 2 {
		t.Errorf("Points length = %d, want 2", len(result))
	}
	if result[0].Count != 10 {
		t.Errorf("Point[0].Count = %d, want 10", result[0].Count)
	}
}

func TestToSeriesStatistics(t *testing.T) {
	stats := &dataquery.SeriesStatistics{
		Min:   70.0,
		Max:   80.0,
		Avg:   75.0,
		Sum:   150.0,
		Count: 2,
	}

	result := toSeriesStatistics(stats)

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.Min != 70.0 {
		t.Errorf("Min = %f, want 70.0", result.Min)
	}
	if result.Max != 80.0 {
		t.Errorf("Max = %f, want 80.0", result.Max)
	}
}

func TestToSeriesStatisticsNil(t *testing.T) {
	result := toSeriesStatistics(nil)
	if result != nil {
		t.Error("Result should be nil for nil input")
	}
}

func TestToSeries(t *testing.T) {
	now := time.Now()
	sd := &dataquery.SeriesData{
		Meta: dataquery.SeriesMeta{
			ID:       1,
			Endpoint: "/api/metrics",
			Metric:   "cpu_usage",
			Labels:   map[string]string{"host": "server1"},
		},
		Points: []dataquery.DataPoint{
			{Time: now, Value: 75.0},
		},
		Statistics: &dataquery.SeriesStatistics{
			Min: 70.0, Max: 80.0, Avg: 75.0, Sum: 150.0, Count: 2,
		},
	}

	result := toSeries(sd)

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.Meta.ID != "1" {
		t.Errorf("Meta.ID = %q, want %q", result.Meta.ID, "1")
	}
	if len(result.Points) != 1 {
		t.Errorf("Points length = %d, want 1", len(result.Points))
	}
	if result.Statistics == nil {
		t.Error("Statistics should not be nil")
	}
}

func TestToSeriesNil(t *testing.T) {
	result := toSeries(nil)
	if result != nil {
		t.Error("Result should be nil for nil input")
	}
}

func TestToSeriesList(t *testing.T) {
	now := time.Now()
	series := []*dataquery.SeriesData{
		{
			Meta:   dataquery.SeriesMeta{ID: 1},
			Points: []dataquery.DataPoint{{Time: now, Value: 75.0}},
		},
		{
			Meta:   dataquery.SeriesMeta{ID: 2},
			Points: []dataquery.DataPoint{{Time: now, Value: 62.0}},
		},
	}

	result := toSeriesList(series)

	if len(result) != 2 {
		t.Errorf("Result length = %d, want 2", len(result))
	}
}

func TestParseTimeRange(t *testing.T) {
	now := time.Now()
	start := now.Add(-1 * time.Hour)

	input := TimeRangeInput{
		Start: start,
		End:   now,
	}

	result := parseTimeRange(input)

	if !result.Start.Equal(start) {
		t.Errorf("Start = %v, want %v", result.Start, start)
	}
	if !result.End.Equal(now) {
		t.Errorf("End = %v, want %v", result.End, now)
	}
}

func TestParseAggFunction(t *testing.T) {
	tests := []struct {
		input    AggFunction
		expected dataquery.AggFunction
	}{
		{AggFunctionAvg, dataquery.AggAvg},
		{AggFunctionMin, dataquery.AggMin},
		{AggFunctionMax, dataquery.AggMax},
		{AggFunctionSum, dataquery.AggSum},
		{AggFunctionCount, dataquery.AggCount},
		{AggFunction("UNKNOWN"), dataquery.AggAvg}, // default
	}

	for _, tt := range tests {
		result := parseAggFunction(tt.input)
		if result != tt.expected {
			t.Errorf("parseAggFunction(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"1", 1, false},
		{"123", 123, false},
		{"0", 0, false},
		{"-1", -1, false},
		{"abc", 0, true},
		{"", 0, true},
		{"1.5", 0, true},
	}

	for _, tt := range tests {
		result, err := parseID(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("parseID(%q) should return error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseID(%q) error = %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parseID(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestNewResolver(t *testing.T) {
	svc := newMockService()
	resolver := NewResolver(svc)

	if resolver == nil {
		t.Fatal("NewResolver returned nil")
	}
	if resolver.service == nil {
		t.Error("Resolver.service should not be nil")
	}
}

// Resolver method tests

func TestResolverEndpoints(t *testing.T) {
	svc := newMockService()
	resolver := NewResolver(svc)

	endpoints, err := resolver.service.GetEndpoints(context.Background())
	if err != nil {
		t.Fatalf("GetEndpoints error = %v", err)
	}

	if len(endpoints) != 2 {
		t.Errorf("Endpoints length = %d, want 2", len(endpoints))
	}
}

func TestResolverMetrics(t *testing.T) {
	svc := newMockService()
	resolver := NewResolver(svc)

	metrics, err := resolver.service.GetMetrics(context.Background(), "/api/metrics")
	if err != nil {
		t.Fatalf("GetMetrics error = %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Metrics length = %d, want 2", len(metrics))
	}
}

func TestResolverQuerySeries(t *testing.T) {
	svc := newMockService()
	resolver := NewResolver(svc)

	now := time.Now()
	req := &dataquery.SeriesQuery{
		TimeRange: dataquery.TimeRange{
			Start: now.Add(-1 * time.Hour),
			End:   now,
		},
	}

	series, err := resolver.service.QuerySeries(context.Background(), req)
	if err != nil {
		t.Fatalf("QuerySeries error = %v", err)
	}

	if len(series) == 0 {
		t.Error("QuerySeries returned no series")
	}
}

func TestResolverGetSeriesByID(t *testing.T) {
	svc := newMockService()
	resolver := NewResolver(svc)

	now := time.Now()
	tr := dataquery.TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	series, err := resolver.service.GetSeriesByID(context.Background(), 1, &tr)
	if err != nil {
		t.Fatalf("GetSeriesByID error = %v", err)
	}

	if series == nil {
		t.Error("GetSeriesByID returned nil for existing series")
	}
}

func TestResolverGetSeriesByIDNotFound(t *testing.T) {
	svc := newMockService()
	resolver := NewResolver(svc)

	now := time.Now()
	tr := dataquery.TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	series, err := resolver.service.GetSeriesByID(context.Background(), 999, &tr)
	if err != nil {
		t.Fatalf("GetSeriesByID error = %v", err)
	}

	if series != nil {
		t.Error("GetSeriesByID should return nil for non-existent series")
	}
}
