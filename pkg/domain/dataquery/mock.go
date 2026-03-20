package dataquery

import (
	"context"
	"time"
)

// MockService implements DataQueryService with mock data
type MockService struct {
	endpoints []string
	series    map[int64]*SeriesData
}

// NewMockService creates a new mock service
func NewMockService() *MockService {
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	// Create mock series data
	series := make(map[int64]*SeriesData)

	// Create some mock series
	for i := int64(1); i <= 3; i++ {
		meta := SeriesMeta{
			ID:        i,
			Endpoint:  "/api/metrics",
			Metric:    "cpu_usage",
			Labels:    map[string]string{"host": "server" + string(rune('0'+i)), "region": "us-east"},
			CreatedAt: start,
		}

		// Generate mock data points
		var points []DataPoint
		for t := start; t.Before(now); t = t.Add(5 * time.Minute) {
			points = append(points, DataPoint{
				Time:  t,
				Value: 50 + float64(i*10) + float64(t.Unix()%20),
			})
		}

		series[i] = &SeriesData{
			Meta:   meta,
			Points: points,
		}
	}

	return &MockService{
		endpoints: []string{"/api/metrics", "/api/health", "/api/query"},
		series:    series,
	}
}

// Name returns the service name
func (s *MockService) Name() string {
	return "MockDataQueryService"
}

// Initialize initializes the service
func (s *MockService) Initialize(ctx context.Context) error {
	return nil
}

// Shutdown shuts down the service
func (s *MockService) Shutdown(ctx context.Context) error {
	return nil
}

// Health returns the health status
func (s *MockService) Health(ctx context.Context) error {
	return nil
}

// GetEndpoints returns all mock endpoints
func (s *MockService) GetEndpoints(ctx context.Context) ([]string, error) {
	return s.endpoints, nil
}

// GetMetrics returns mock metrics for an endpoint
func (s *MockService) GetMetrics(ctx context.Context, endpoint string) ([]string, error) {
	switch endpoint {
	case "/api/metrics":
		return []string{"cpu_usage", "memory_usage", "disk_io", "network_bytes"}, nil
	case "/api/health":
		return []string{"response_time", "status_code"}, nil
	case "/api/query":
		return []string{"query_count", "query_latency"}, nil
	default:
		return []string{}, nil
	}
}

// QuerySeries queries series data based on filters
func (s *MockService) QuerySeries(ctx context.Context, req *SeriesQuery) ([]*SeriesData, error) {
	var result []*SeriesData
	for _, sd := range s.series {
		// Filter by endpoint
		if req.Endpoint != "" && sd.Meta.Endpoint != req.Endpoint {
			continue
		}
		// Filter by metric
		if req.Metric != "" && sd.Meta.Metric != req.Metric {
			continue
		}
		// Filter points by time range
		var filteredPoints []DataPoint
		for _, p := range sd.Points {
			if (p.Time.Equal(req.TimeRange.Start) || p.Time.After(req.TimeRange.Start)) &&
				(p.Time.Equal(req.TimeRange.End) || p.Time.Before(req.TimeRange.End)) {
				filteredPoints = append(filteredPoints, p)
			}
		}
		result = append(result, &SeriesData{
			Meta:   sd.Meta,
			Points: filteredPoints,
		})
	}
	return result, nil
}

// QuerySeriesMulti queries multiple series at once
func (s *MockService) QuerySeriesMulti(ctx context.Context, req *MultiSeriesQuery) ([]*SeriesData, error) {
	var result []*SeriesData
	for _, sd := range s.series {
		// Filter by endpoints
		if len(req.Endpoints) > 0 {
			found := false
			for _, ep := range req.Endpoints {
				if sd.Meta.Endpoint == ep {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Filter by metrics
		if len(req.Metrics) > 0 {
			found := false
			for _, m := range req.Metrics {
				if sd.Meta.Metric == m {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Filter points by time range
		var filteredPoints []DataPoint
		for _, p := range sd.Points {
			if (p.Time.Equal(req.TimeRange.Start) || p.Time.After(req.TimeRange.Start)) &&
				(p.Time.Equal(req.TimeRange.End) || p.Time.Before(req.TimeRange.End)) {
				filteredPoints = append(filteredPoints, p)
			}
		}
		result = append(result, &SeriesData{
			Meta:   sd.Meta,
			Points: filteredPoints,
		})
	}
	return result, nil
}

// GetSeriesByID retrieves a single series by ID
func (s *MockService) GetSeriesByID(ctx context.Context, id int64, timeRange *TimeRange) (*SeriesData, error) {
	sd, ok := s.series[id]
	if !ok {
		return nil, nil
	}

	// Filter points by time range
	var filteredPoints []DataPoint
	for _, p := range sd.Points {
		if (p.Time.Equal(timeRange.Start) || p.Time.After(timeRange.Start)) &&
			(p.Time.Equal(timeRange.End) || p.Time.Before(timeRange.End)) {
			filteredPoints = append(filteredPoints, p)
		}
	}

	return &SeriesData{
		Meta:   sd.Meta,
		Points: filteredPoints,
	}, nil
}
