package dataquery

import (
	"context"
)

// Repository defines the data access interface for time series data
type Repository interface {
	// GetEndpoints retrieves all distinct endpoints
	GetEndpoints(ctx context.Context) ([]string, error)

	// GetMetrics retrieves all distinct metrics for an endpoint
	GetMetrics(ctx context.Context, endpoint string) ([]string, error)

	// QuerySeries queries series metadata based on filters
	QuerySeries(ctx context.Context, req *SeriesQueryRequest) ([]SeriesMeta, error)

	// GetSeriesByID retrieves series metadata by ID
	GetSeriesByID(ctx context.Context, id int64) (*SeriesMeta, error)

	// GetSeriesPoints retrieves data points for multiple series
	GetSeriesPoints(ctx context.Context, req *PointsQueryRequest) (map[int64][]DataPoint, error)

	// GetSeriesStatistics retrieves statistics for multiple series
	GetSeriesStatistics(ctx context.Context, req *StatsRequest) (map[int64]*SeriesStatistics, error)

	// GetInstanceByEndpoint retrieves instance metadata by endpoint
	GetInstanceByEndpoint(ctx context.Context, endpoint string) (*InstanceMeta, error)

	// GetAllInstances retrieves instance metadata with pagination
	// Returns instances slice and total count
	GetAllInstances(ctx context.Context, req *InstancesQueryRequest) ([]*InstanceMeta, int64, error)

	// GetAlertsByEndpoint retrieves all alerts for a specific endpoint
	GetAlertsByEndpoint(ctx context.Context, endpoint string) ([]*Alert, error)
}
