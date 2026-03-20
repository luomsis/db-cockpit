package dataquery

import (
	"testing"
	"time"
)

func TestAggFunctionValues(t *testing.T) {
	tests := []struct {
		value    AggFunction
		expected string
	}{
		{AggAvg, "AVG"},
		{AggMin, "MIN"},
		{AggMax, "MAX"},
		{AggSum, "SUM"},
		{AggCount, "COUNT"},
	}

	for _, tt := range tests {
		if string(tt.value) != tt.expected {
			t.Errorf("AggFunction %v = %q, want %q", tt.value, string(tt.value), tt.expected)
		}
	}
}

func TestTimeRange(t *testing.T) {
	now := time.Now()
	start := now.Add(-1 * time.Hour)

	tr := TimeRange{
		Start: start,
		End:   now,
	}

	if tr.Start != start {
		t.Errorf("TimeRange.Start = %v, want %v", tr.Start, start)
	}
	if tr.End != now {
		t.Errorf("TimeRange.End = %v, want %v", tr.End, now)
	}
}

func TestSeriesMeta(t *testing.T) {
	now := time.Now()
	meta := SeriesMeta{
		ID:         1,
		Endpoint:   "/api/metrics",
		Metric:     "cpu_usage",
		Labels:     map[string]string{"host": "server1", "region": "us-east"},
		LabelsHash: "abc123",
		CreatedAt:  now,
	}

	if meta.ID != 1 {
		t.Errorf("SeriesMeta.ID = %d, want 1", meta.ID)
	}
	if meta.Endpoint != "/api/metrics" {
		t.Errorf("SeriesMeta.Endpoint = %q, want %q", meta.Endpoint, "/api/metrics")
	}
	if meta.Metric != "cpu_usage" {
		t.Errorf("SeriesMeta.Metric = %q, want %q", meta.Metric, "cpu_usage")
	}
	if len(meta.Labels) != 2 {
		t.Errorf("SeriesMeta.Labels length = %d, want 2", len(meta.Labels))
	}
}

func TestDataPoint(t *testing.T) {
	now := time.Now()
	point := DataPoint{
		Time:  now,
		Value: 75.5,
	}

	if !point.Time.Equal(now) {
		t.Errorf("DataPoint.Time = %v, want %v", point.Time, now)
	}
	if point.Value != 75.5 {
		t.Errorf("DataPoint.Value = %f, want 75.5", point.Value)
	}
}

func TestAggregatedPoint(t *testing.T) {
	now := time.Now()
	point := AggregatedPoint{
		Time:  now,
		Value: 75.0,
		Count: 10,
	}

	if point.Count != 10 {
		t.Errorf("AggregatedPoint.Count = %d, want 10", point.Count)
	}
	if point.Value != 75.0 {
		t.Errorf("AggregatedPoint.Value = %f, want 75.0", point.Value)
	}
}

func TestSeriesStatistics(t *testing.T) {
	stats := SeriesStatistics{
		Min:   10.0,
		Max:   100.0,
		Avg:   55.0,
		Sum:   550.0,
		Count: 10,
	}

	if stats.Min != 10.0 {
		t.Errorf("SeriesStatistics.Min = %f, want 10.0", stats.Min)
	}
	if stats.Max != 100.0 {
		t.Errorf("SeriesStatistics.Max = %f, want 100.0", stats.Max)
	}
	if stats.Avg != 55.0 {
		t.Errorf("SeriesStatistics.Avg = %f, want 55.0", stats.Avg)
	}
	if stats.Sum != 550.0 {
		t.Errorf("SeriesStatistics.Sum = %f, want 550.0", stats.Sum)
	}
	if stats.Count != 10 {
		t.Errorf("SeriesStatistics.Count = %d, want 10", stats.Count)
	}
}

func TestSeriesData(t *testing.T) {
	now := time.Now()
	series := SeriesData{
		Meta: SeriesMeta{
			ID:       1,
			Endpoint: "/api/metrics",
			Metric:   "cpu_usage",
			Labels:   map[string]string{"host": "server1"},
		},
		Points: []DataPoint{
			{Time: now, Value: 75.0},
			{Time: now.Add(-5 * time.Minute), Value: 72.0},
		},
		Statistics: &SeriesStatistics{
			Min: 70.0, Max: 80.0, Avg: 75.0, Sum: 150.0, Count: 2,
		},
	}

	if series.Meta.ID != 1 {
		t.Errorf("SeriesData.Meta.ID = %d, want 1", series.Meta.ID)
	}
	if len(series.Points) != 2 {
		t.Errorf("SeriesData.Points length = %d, want 2", len(series.Points))
	}
	if series.Statistics == nil {
		t.Error("SeriesData.Statistics should not be nil")
	}
}

func TestAggregation(t *testing.T) {
	agg := Aggregation{
		Interval: "5m",
		Function: AggAvg,
	}

	if agg.Interval != "5m" {
		t.Errorf("Aggregation.Interval = %q, want %q", agg.Interval, "5m")
	}
	if agg.Function != AggAvg {
		t.Errorf("Aggregation.Function = %v, want %v", agg.Function, AggAvg)
	}
}

func TestSeriesQuery(t *testing.T) {
	now := time.Now()
	query := SeriesQuery{
		Endpoint:    "/api/metrics",
		Metric:      "cpu_usage",
		LabelFilter: `host="server1"`,
		TimeRange: TimeRange{
			Start: now.Add(-1 * time.Hour),
			End:   now,
		},
		Limit: 10,
	}

	if query.Endpoint != "/api/metrics" {
		t.Errorf("SeriesQuery.Endpoint = %q, want %q", query.Endpoint, "/api/metrics")
	}
	if query.Limit != 10 {
		t.Errorf("SeriesQuery.Limit = %d, want 10", query.Limit)
	}
}

func TestMultiSeriesQuery(t *testing.T) {
	now := time.Now()
	query := MultiSeriesQuery{
		Endpoints:   []string{"/api/metrics", "/api/health"},
		Metrics:     []string{"cpu_usage", "memory_usage"},
		LabelFilter: `env="prod"`,
		TimeRange: TimeRange{
			Start: now.Add(-1 * time.Hour),
			End:   now,
		},
		Aggregation: &Aggregation{
			Interval: "5m",
			Function: AggAvg,
		},
	}

	if len(query.Endpoints) != 2 {
		t.Errorf("MultiSeriesQuery.Endpoints length = %d, want 2", len(query.Endpoints))
	}
	if len(query.Metrics) != 2 {
		t.Errorf("MultiSeriesQuery.Metrics length = %d, want 2", len(query.Metrics))
	}
	if query.Aggregation == nil {
		t.Error("MultiSeriesQuery.Aggregation should not be nil")
	}
}

func TestRepositoryRequestTypes(t *testing.T) {
	now := time.Now()

	// SeriesQueryRequest
	seriesReq := SeriesQueryRequest{
		Endpoint:    "/api/metrics",
		Metric:      "cpu_usage",
		LabelFilter: `host="server1"`,
		TimeRange:   TimeRange{Start: now, End: now},
		Limit:       10,
	}
	if seriesReq.Endpoint != "/api/metrics" {
		t.Errorf("SeriesQueryRequest.Endpoint = %q", seriesReq.Endpoint)
	}

	// PointsQueryRequest
	pointsReq := PointsQueryRequest{
		SeriesIDs: []int64{1, 2, 3},
		TimeRange: TimeRange{Start: now, End: now},
	}
	if len(pointsReq.SeriesIDs) != 3 {
		t.Errorf("PointsQueryRequest.SeriesIDs length = %d", len(pointsReq.SeriesIDs))
	}

	// AggregationRequest
	aggReq := AggregationRequest{
		SeriesIDs: []int64{1, 2},
		TimeRange: TimeRange{Start: now, End: now},
		Interval:  "5m",
		Function:  "AVG",
	}
	if aggReq.Interval != "5m" {
		t.Errorf("AggregationRequest.Interval = %q", aggReq.Interval)
	}

	// StatsRequest
	statsReq := StatsRequest{
		SeriesIDs: []int64{1},
		TimeRange: TimeRange{Start: now, End: now},
	}
	if len(statsReq.SeriesIDs) != 1 {
		t.Errorf("StatsRequest.SeriesIDs length = %d", len(statsReq.SeriesIDs))
	}
}
