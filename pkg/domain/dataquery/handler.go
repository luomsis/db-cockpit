package dataquery

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// Handler provides REST handlers for the DataQuery service
type Handler struct {
	service DataQueryService
}

// NewHandler creates a new Handler
func NewHandler(service DataQueryService) *Handler {
	return &Handler{service: service}
}

// Response DTOs

// EndpointsResponse is the response for GetEndpoints
type EndpointsResponse struct {
	Data []string `json:"data"`
}

// MetricsResponse is the response for GetMetrics
type MetricsResponse struct {
	Data []string `json:"data"`
}

// SeriesResponse is the response for series queries
type SeriesResponse struct {
	Data []SeriesDataDTO `json:"data"`
}

// SeriesDataDTO is the JSON representation of SeriesData
type SeriesDataDTO struct {
	ID               int64                `json:"id"`
	Endpoint         string               `json:"endpoint"`
	Metric           string               `json:"metric"`
	Labels           map[string]string    `json:"labels"`
	LabelsHash       string               `json:"labels_hash"`
	CreatedAt        time.Time            `json:"created_at"`
	Points           []DataPointDTO       `json:"points,omitempty"`
	AggregatedPoints []AggregatedPointDTO `json:"aggregated_points,omitempty"`
	Statistics       *SeriesStatisticsDTO `json:"statistics,omitempty"`
}

// DataPointDTO is the JSON representation of DataPoint
type DataPointDTO struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// AggregatedPointDTO is the JSON representation of AggregatedPoint
type AggregatedPointDTO struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
	Count int       `json:"count"`
}

// SeriesStatisticsDTO is the JSON representation of SeriesStatistics
type SeriesStatisticsDTO struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Sum   float64 `json:"sum"`
	Count int     `json:"count"`
}

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Handlers

// GetEndpoints handles GET /endpoints requests
func (h *Handler) GetEndpoints(ctx context.Context, c *app.RequestContext) {
	endpoints, err := h.service.GetEndpoints(ctx)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(200, EndpointsResponse{Data: endpoints})
}

// GetMetrics handles GET /endpoints/:endpoint/metrics requests
func (h *Handler) GetMetrics(ctx context.Context, c *app.RequestContext) {
	endpoint := c.Param("endpoint")
	if endpoint == "" {
		c.JSON(400, ErrorResponse{Error: "endpoint parameter is required"})
		return
	}

	metrics, err := h.service.GetMetrics(ctx, endpoint)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(200, MetricsResponse{Data: metrics})
}

// GetSeries handles GET /series requests with query parameters
func (h *Handler) GetSeries(ctx context.Context, c *app.RequestContext) {
	query, err := h.parseSeriesQuery(c)
	if err != nil {
		c.JSON(400, ErrorResponse{Error: err.Error()})
		return
	}

	series, err := h.service.QuerySeries(ctx, query)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(200, SeriesResponse{Data: toSeriesDataDTOs(series)})
}

// GetSeriesByID handles GET /series/:id requests
func (h *Handler) GetSeriesByID(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(400, ErrorResponse{Error: "id parameter is required"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, ErrorResponse{Error: "invalid id parameter"})
		return
	}

	timeRange, err := h.parseTimeRange(c)
	if err != nil {
		c.JSON(400, ErrorResponse{Error: err.Error()})
		return
	}

	series, err := h.service.GetSeriesByID(ctx, id, &timeRange)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: err.Error()})
		return
	}

	if series == nil {
		c.JSON(404, ErrorResponse{Error: "series not found"})
		return
	}

	c.JSON(200, toSeriesDataDTO(series))
}

// QuerySeries handles POST /series/query requests for complex queries
func (h *Handler) QuerySeries(ctx context.Context, c *app.RequestContext) {
	var req SeriesQueryRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, ErrorResponse{Error: err.Error()})
		return
	}

	query := &SeriesQuery{
		Endpoint:    req.Endpoint,
		Metric:      req.Metric,
		LabelFilter: req.LabelFilter,
		TimeRange:   req.TimeRange,
		Limit:       req.Limit,
	}

	series, err := h.service.QuerySeries(ctx, query)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(200, SeriesResponse{Data: toSeriesDataDTOs(series)})
}

// Helper functions

// parseSeriesQuery parses query parameters into a SeriesQuery
func (h *Handler) parseSeriesQuery(c *app.RequestContext) (*SeriesQuery, error) {
	query := &SeriesQuery{}

	// Parse endpoint
	query.Endpoint = c.Query("endpoint")

	// Parse metric
	query.Metric = c.Query("metric")

	// Parse label filter
	query.LabelFilter = c.Query("label_filter")

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, err
		}
		query.Limit = limit
	}

	// Parse time range
	timeRange, err := h.parseTimeRange(c)
	if err != nil {
		return nil, err
	}
	query.TimeRange = timeRange

	return query, nil
}

// parseTimeRange parses time range from query parameters
func (h *Handler) parseTimeRange(c *app.RequestContext) (TimeRange, error) {
	timeRange := TimeRange{
		Start: time.Now().Add(-1 * time.Hour), // Default: last 1 hour
		End:   time.Now(),
	}

	if startStr := c.Query("start"); startStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Try parsing as Unix timestamp
			unixStart, err := strconv.ParseInt(startStr, 10, 64)
			if err != nil {
				return TimeRange{}, err
			}
			start = time.Unix(unixStart, 0)
		}
		timeRange.Start = start
	}

	if endStr := c.Query("end"); endStr != "" {
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			// Try parsing as Unix timestamp
			unixEnd, err := strconv.ParseInt(endStr, 10, 64)
			if err != nil {
				return TimeRange{}, err
			}
			end = time.Unix(unixEnd, 0)
		}
		timeRange.End = end
	}

	return timeRange, nil
}

// toSeriesDataDTO converts SeriesData to SeriesDataDTO
func toSeriesDataDTO(s *SeriesData) SeriesDataDTO {
	return SeriesDataDTO{
		ID:               s.Meta.ID,
		Endpoint:         s.Meta.Endpoint,
		Metric:           s.Meta.Metric,
		Labels:           s.Meta.Labels,
		LabelsHash:       s.Meta.LabelsHash,
		CreatedAt:        s.Meta.CreatedAt,
		Points:           toDataPointDTOs(s.Points),
		AggregatedPoints: toAggregatedPointDTOs(s.AggregatedPoints),
		Statistics:       toSeriesStatisticsDTO(s.Statistics),
	}
}

// toSeriesDataDTOs converts a slice of SeriesData to SeriesDataDTO
func toSeriesDataDTOs(series []*SeriesData) []SeriesDataDTO {
	if len(series) == 0 {
		return []SeriesDataDTO{}
	}
	result := make([]SeriesDataDTO, len(series))
	for i, s := range series {
		result[i] = toSeriesDataDTO(s)
	}
	return result
}

// toDataPointDTOs converts a slice of DataPoint to DataPointDTO
func toDataPointDTOs(points []DataPoint) []DataPointDTO {
	if len(points) == 0 {
		return nil
	}
	result := make([]DataPointDTO, len(points))
	for i, p := range points {
		result[i] = DataPointDTO{
			Time:  p.Time,
			Value: p.Value,
		}
	}
	return result
}

// toAggregatedPointDTOs converts a slice of AggregatedPoint to AggregatedPointDTO
func toAggregatedPointDTOs(points []AggregatedPoint) []AggregatedPointDTO {
	if len(points) == 0 {
		return nil
	}
	result := make([]AggregatedPointDTO, len(points))
	for i, p := range points {
		result[i] = AggregatedPointDTO{
			Time:  p.Time,
			Value: p.Value,
			Count: p.Count,
		}
	}
	return result
}

// toSeriesStatisticsDTO converts SeriesStatistics to SeriesStatisticsDTO
func toSeriesStatisticsDTO(stats *SeriesStatistics) *SeriesStatisticsDTO {
	if stats == nil {
		return nil
	}
	return &SeriesStatisticsDTO{
		Min:   stats.Min,
		Max:   stats.Max,
		Avg:   stats.Avg,
		Sum:   stats.Sum,
		Count: stats.Count,
	}
}