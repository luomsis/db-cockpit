package timescaledb

import (
	"context"
	"sync"
	"time"
)

// MockTimescaleDB is a mock implementation of TimescaleDB
type MockTimescaleDB struct {
	mu      sync.RWMutex
	metrics map[string][]MetricPoint // key: metric_name
}

// NewMockTimescaleDB creates a new mock TimescaleDB
func NewMockTimescaleDB() *MockTimescaleDB {
	mock := &MockTimescaleDB{
		metrics: make(map[string][]MetricPoint),
	}
	mock.seedData()
	return mock
}

// seedData seeds the mock with sample data
func (m *MockTimescaleDB) seedData() {
	now := time.Now()

	// CPU metrics
	cpuMetrics := []MetricPoint{}
	for i := 0; i < 60; i++ {
		cpuMetrics = append(cpuMetrics, MetricPoint{
			Name:      "cpu_usage",
			Value:     30.0 + float64(i%20) + float64(i)*0.1,
			Timestamp: now.Add(-time.Duration(60-i) * time.Minute),
			Tags:      map[string]string{"host": "db-server-1", "database": "production"},
		})
	}
	m.metrics["cpu_usage"] = cpuMetrics

	// Memory metrics
	memMetrics := []MetricPoint{}
	for i := 0; i < 60; i++ {
		memMetrics = append(memMetrics, MetricPoint{
			Name:      "memory_usage",
			Value:     60.0 + float64(i%15) + float64(i)*0.05,
			Timestamp: now.Add(-time.Duration(60-i) * time.Minute),
			Tags:      map[string]string{"host": "db-server-1", "database": "production"},
		})
	}
	m.metrics["memory_usage"] = memMetrics

	// Query latency metrics
	latencyMetrics := []MetricPoint{}
	for i := 0; i < 60; i++ {
		latencyMetrics = append(latencyMetrics, MetricPoint{
			Name:      "query_latency_ms",
			Value:     10.0 + float64(i%30) + float64(i)*0.2,
			Timestamp: now.Add(-time.Duration(60-i) * time.Minute),
			Tags:      map[string]string{"database": "production", "query_type": "select"},
		})
	}
	m.metrics["query_latency_ms"] = latencyMetrics

	// Connection count
	connMetrics := []MetricPoint{}
	for i := 0; i < 60; i++ {
		connMetrics = append(connMetrics, MetricPoint{
			Name:      "connection_count",
			Value:     float64(50 + i%20),
			Timestamp: now.Add(-time.Duration(60-i) * time.Minute),
			Tags:      map[string]string{"database": "production"},
		})
	}
	m.metrics["connection_count"] = connMetrics

	// Slow queries
	slowQueryMetrics := []MetricPoint{}
	for i := 0; i < 10; i++ {
		slowQueryMetrics = append(slowQueryMetrics, MetricPoint{
			Name:      "slow_query_count",
			Value:     float64(i + 1),
			Timestamp: now.Add(-time.Duration(10-i) * time.Minute),
			Tags:      map[string]string{"database": "production", "threshold": "1000ms"},
		})
	}
	m.metrics["slow_query_count"] = slowQueryMetrics

	// Disk I/O
	diskMetrics := []MetricPoint{}
	for i := 0; i < 60; i++ {
		diskMetrics = append(diskMetrics, MetricPoint{
			Name:      "disk_io_read_mbps",
			Value:     50.0 + float64(i%30),
			Timestamp: now.Add(-time.Duration(60-i) * time.Minute),
			Tags:      map[string]string{"database": "production", "device": "sda"},
		})
	}
	m.metrics["disk_io_read_mbps"] = diskMetrics
}

// Connect mock implementation
func (m *MockTimescaleDB) Connect(ctx context.Context) error {
	return nil
}

// Close mock implementation
func (m *MockTimescaleDB) Close() error {
	return nil
}

// InsertMetrics mock implementation
func (m *MockTimescaleDB) InsertMetrics(ctx context.Context, metrics []MetricPoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		m.metrics[metric.Name] = append(m.metrics[metric.Name], metric)
	}
	return nil
}

// QueryMetrics mock implementation
func (m *MockTimescaleDB) QueryMetrics(ctx context.Context, query MetricQuery) ([]MetricSeries, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	series := []MetricSeries{}

	points, exists := m.metrics[query.Name]
	if !exists {
		return series, nil
	}

	// Filter by time range
	filteredPoints := []MetricPoint{}
	for _, p := range points {
		if (query.StartTime.IsZero() || p.Timestamp.After(query.StartTime) || p.Timestamp.Equal(query.StartTime)) &&
			(query.EndTime.IsZero() || p.Timestamp.Before(query.EndTime) || p.Timestamp.Equal(query.EndTime)) {
			// Filter by tags if specified
			if query.Tags != nil {
				match := true
				for k, v := range query.Tags {
					if p.Tags[k] != v {
						match = false
						break
					}
				}
				if !match {
					continue
				}
			}
			filteredPoints = append(filteredPoints, p)
		}
	}

	// Apply limit
	if query.Limit > 0 && len(filteredPoints) > query.Limit {
		filteredPoints = filteredPoints[:query.Limit]
	}

	// Calculate statistics
	stats := MetricStats{}
	if len(filteredPoints) > 0 {
		stats.Min = filteredPoints[0].Value
		stats.Max = filteredPoints[0].Value
		stats.Sum = 0
		for _, p := range filteredPoints {
			stats.Sum += p.Value
			if p.Value < stats.Min {
				stats.Min = p.Value
			}
			if p.Value > stats.Max {
				stats.Max = p.Value
			}
		}
		stats.Avg = stats.Sum / float64(len(filteredPoints))
		stats.Count = int64(len(filteredPoints))

		// Simple percentile calculations
		stats.P50 = stats.Avg
		stats.P95 = stats.Max * 0.95
		stats.P99 = stats.Max * 0.99
	}

	series = append(series, MetricSeries{
		Name:   query.Name,
		Points: filteredPoints,
		Stats:  stats,
	})

	return series, nil
}

// QueryMetricsAggregated mock implementation
func (m *MockTimescaleDB) QueryMetricsAggregated(ctx context.Context, query MetricQuery, interval time.Duration) ([]MetricSeries, error) {
	// For simplicity, return same as QueryMetrics
	return m.QueryMetrics(ctx, query)
}

// QueryMetricsStats mock implementation
func (m *MockTimescaleDB) QueryMetricsStats(ctx context.Context, query MetricQuery) (MetricStats, error) {
	series, err := m.QueryMetrics(ctx, query)
	if err != nil {
		return MetricStats{}, err
	}

	if len(series) == 0 {
		return MetricStats{}, nil
	}

	return series[0].Stats, nil
}

// CreateHypertable mock implementation
func (m *MockTimescaleDB) CreateHypertable(ctx context.Context, table string, timeColumn string) error {
	return nil
}

// CreateContinuousAggregate mock implementation
func (m *MockTimescaleDB) CreateContinuousAggregate(ctx context.Context, name string, query string, refreshInterval time.Duration) error {
	return nil
}

// DeleteMetrics mock implementation
func (m *MockTimescaleDB) DeleteMetrics(ctx context.Context, before time.Time) error {
	return nil
}

// Ping mock implementation
func (m *MockTimescaleDB) Ping(ctx context.Context) error {
	return nil
}

// GetMetricNames returns available metric names
func (m *MockTimescaleDB) GetMetricNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := []string{}
	for name := range m.metrics {
		names = append(names, name)
	}
	return names
}
