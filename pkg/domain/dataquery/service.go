package dataquery

import (
	"context"
)

// DataQueryService defines the service interface for time series operations
type DataQueryService interface {
	// Domain service methods
	Name() string
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Health(ctx context.Context) error

	// GetEndpoints retrieves all distinct endpoints
	GetEndpoints(ctx context.Context) ([]string, error)

	// GetMetrics retrieves all distinct metrics for an endpoint
	GetMetrics(ctx context.Context, endpoint string) ([]string, error)

	// QuerySeries queries series data based on filters
	QuerySeries(ctx context.Context, req *SeriesQuery) ([]*SeriesData, error)

	// QuerySeriesMulti queries multiple series at once with optional aggregation
	QuerySeriesMulti(ctx context.Context, req *MultiSeriesQuery) ([]*SeriesData, error)

	// GetSeriesByID retrieves a single series by ID with data points
	GetSeriesByID(ctx context.Context, id int64, timeRange *TimeRange) (*SeriesData, error)

	// GetInstanceByEndpoint retrieves instance metadata by endpoint
	GetInstanceByEndpoint(ctx context.Context, endpoint string) (*InstanceMeta, error)

	// GetAllInstances retrieves instance metadata with pagination
	GetAllInstances(ctx context.Context, req *InstancesQueryRequest) (*InstancesListResponse, error)

	// GetAlertsByEndpoint retrieves all alerts for a specific endpoint
	GetAlertsByEndpoint(ctx context.Context, endpoint string) ([]*Alert, error)
}

// Service implements DataQueryService
type Service struct {
	repo Repository
}

// NewService creates a new data query service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Name returns the service name
func (s *Service) Name() string {
	return "DataQueryService"
}

// Initialize initializes the service
func (s *Service) Initialize(ctx context.Context) error {
	return nil
}

// Shutdown shuts down the service
func (s *Service) Shutdown(ctx context.Context) error {
	return nil
}

// Health returns the health status
func (s *Service) Health(ctx context.Context) error {
	return nil
}

// GetEndpoints retrieves all distinct endpoints
func (s *Service) GetEndpoints(ctx context.Context) ([]string, error) {
	return s.repo.GetEndpoints(ctx)
}

// GetMetrics retrieves all distinct metrics for an endpoint
func (s *Service) GetMetrics(ctx context.Context, endpoint string) ([]string, error) {
	return s.repo.GetMetrics(ctx, endpoint)
}

// QuerySeries queries series data based on filters
func (s *Service) QuerySeries(ctx context.Context, req *SeriesQuery) ([]*SeriesData, error) {
	// Query series metadata
	seriesMeta, err := s.repo.QuerySeries(ctx, &SeriesQueryRequest{
		Endpoint:    req.Endpoint,
		Metric:      req.Metric,
		LabelFilter: req.LabelFilter,
		TimeRange:   req.TimeRange,
		Limit:       req.Limit,
	})
	if err != nil {
		return nil, err
	}

	if len(seriesMeta) == 0 {
		return []*SeriesData{}, nil
	}

	// Extract series IDs
	seriesIDs := make([]int64, len(seriesMeta))
	for i, meta := range seriesMeta {
		seriesIDs[i] = meta.ID
	}

	// Get data points with interval
	pointsMap, err := s.repo.GetSeriesPoints(ctx, &PointsQueryRequest{
		SeriesIDs: seriesIDs,
		TimeRange: req.TimeRange,
		Interval:  req.Interval,
	})
	if err != nil {
		return nil, err
	}

	// Build result
	result := make([]*SeriesData, len(seriesMeta))
	for i, meta := range seriesMeta {
		result[i] = &SeriesData{
			Meta:   meta,
			Points: pointsMap[meta.ID],
		}
		if result[i].Points == nil {
			result[i].Points = []DataPoint{}
		}
	}

	return result, nil
}

// QuerySeriesMulti queries multiple series at once
func (s *Service) QuerySeriesMulti(ctx context.Context, req *MultiSeriesQuery) ([]*SeriesData, error) {
	// Build series query request
	queryReq := &SeriesQueryRequest{
		TimeRange:   req.TimeRange,
		LabelFilter: req.LabelFilter,
	}

	// Use a map to deduplicate series by ID
	seriesMap := make(map[int64]SeriesMeta)

	if len(req.Endpoints) > 0 && len(req.Metrics) > 0 {
		// Query for each endpoint/metric combination
		for _, endpoint := range req.Endpoints {
			for _, metric := range req.Metrics {
				queryReq.Endpoint = endpoint
				queryReq.Metric = metric

				series, err := s.repo.QuerySeries(ctx, queryReq)
				if err != nil {
					return nil, err
				}
				for _, s := range series {
					seriesMap[s.ID] = s
				}
			}
		}
	} else if len(req.Endpoints) > 0 {
		// Query by endpoints only
		for _, endpoint := range req.Endpoints {
			queryReq.Endpoint = endpoint
			queryReq.Metric = ""

			series, err := s.repo.QuerySeries(ctx, queryReq)
			if err != nil {
				return nil, err
			}
			for _, s := range series {
				seriesMap[s.ID] = s
			}
		}
	} else if len(req.Metrics) > 0 {
		// Query by metrics only
		queryReq.Endpoint = ""
		for _, metric := range req.Metrics {
			queryReq.Metric = metric

			series, err := s.repo.QuerySeries(ctx, queryReq)
			if err != nil {
				return nil, err
			}
			for _, s := range series {
				seriesMap[s.ID] = s
			}
		}
	} else {
		// No endpoint/metric specified, query all matching label filter
		series, err := s.repo.QuerySeries(ctx, queryReq)
		if err != nil {
			return nil, err
		}
		for _, s := range series {
			seriesMap[s.ID] = s
		}
	}

	if len(seriesMap) == 0 {
		return []*SeriesData{}, nil
	}

	// Extract series IDs and convert map to slice
	seriesIDs := make([]int64, 0, len(seriesMap))
	allSeries := make([]SeriesMeta, 0, len(seriesMap))
	for id, meta := range seriesMap {
		seriesIDs = append(seriesIDs, id)
		allSeries = append(allSeries, meta)
	}

	// Get data points
	pointsMap, err := s.repo.GetSeriesPoints(ctx, &PointsQueryRequest{
		SeriesIDs: seriesIDs,
		TimeRange: req.TimeRange,
	})
	if err != nil {
		return nil, err
	}

	// Build result
	result := make([]*SeriesData, len(allSeries))
	for i, meta := range allSeries {
		result[i] = &SeriesData{
			Meta:   meta,
			Points: pointsMap[meta.ID],
		}
		if result[i].Points == nil {
			result[i].Points = []DataPoint{}
		}
	}

	return result, nil
}

// GetSeriesByID retrieves a single series by ID with data points
func (s *Service) GetSeriesByID(ctx context.Context, id int64, timeRange *TimeRange) (*SeriesData, error) {
	meta, err := s.repo.GetSeriesByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, nil
	}

	points, err := s.repo.GetSeriesPoints(ctx, &PointsQueryRequest{
		SeriesIDs: []int64{id},
		TimeRange: *timeRange,
	})
	if err != nil {
		return nil, err
	}

	stats, err := s.repo.GetSeriesStatistics(ctx, &StatsRequest{
		SeriesIDs: []int64{id},
		TimeRange: *timeRange,
	})
	if err != nil {
		return nil, err
	}

	result := &SeriesData{
		Meta:       *meta,
		Points:     points[id],
		Statistics: stats[id],
	}

	if result.Points == nil {
		result.Points = []DataPoint{}
	}

	return result, nil
}

// GetInstanceByEndpoint retrieves instance metadata by endpoint
func (s *Service) GetInstanceByEndpoint(ctx context.Context, endpoint string) (*InstanceMeta, error) {
	return s.repo.GetInstanceByEndpoint(ctx, endpoint)
}

// GetAllInstances retrieves instance metadata with pagination
func (s *Service) GetAllInstances(ctx context.Context, req *InstancesQueryRequest) (*InstancesListResponse, error) {
	instances, totalCount, err := s.repo.GetAllInstances(ctx, req)
	if err != nil {
		return nil, err
	}

	totalPages := int(totalCount) / req.Pagination.PageSize
	if int(totalCount) % req.Pagination.PageSize > 0 {
		totalPages++
	}

	return &InstancesListResponse{
		Data: instances,
		Pagination: &PaginationMeta{
			TotalCount:  totalCount,
			TotalPages:  totalPages,
			CurrentPage: req.Pagination.Page,
			PageSize:    req.Pagination.PageSize,
		},
	}, nil
}

// GetAlertsByEndpoint retrieves all alerts for a specific endpoint
func (s *Service) GetAlertsByEndpoint(ctx context.Context, endpoint string) ([]*Alert, error) {
	return s.repo.GetAlertsByEndpoint(ctx, endpoint)
}
