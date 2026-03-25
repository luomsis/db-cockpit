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

// SeriesSingleResponse is the response for a single series
type SeriesSingleResponse struct {
	Data *SeriesDataDTO `json:"data"`
}

// SeriesDataDTO is the JSON representation of SeriesData
type SeriesDataDTO struct {
	ID               string               `json:"id"`
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
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error code and message
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SeriesQueryRequestBody is the request body for POST /series/query
type SeriesQueryRequestBody struct {
	Endpoints   []string           `json:"endpoints"`
	Metrics     []string           `json:"metrics"`
	Labels      string             `json:"labels"`
	Start       *time.Time         `json:"start"`
	End         *time.Time         `json:"end"`
	Aggregation *AggregationInput  `json:"aggregation"`
}

// AggregationInput is the input for aggregation
type AggregationInput struct {
	Interval string     `json:"interval"`
	Function AggFunction `json:"function"`
}

// Handlers

// GetEndpoints handles GET /endpoints requests
// @Summary Get all endpoints
// @Description Get all distinct endpoints from the time series database
// @Tags endpoints
// @Produce json
// @Success 200 {object} EndpointsResponse
// @Failure 500 {object} ErrorResponse
// @Router /endpoints [get]
func (h *Handler) GetEndpoints(ctx context.Context, c *app.RequestContext) {
	endpoints, err := h.service.GetEndpoints(ctx)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	c.JSON(200, EndpointsResponse{Data: endpoints})
}

// GetMetrics handles GET /metrics requests with endpoint as query parameter
// @Summary Get metrics for an endpoint
// @Description Get all distinct metrics for a specific endpoint
// @Tags metrics
// @Param endpoint query string true "Endpoint name"
// @Produce json
// @Success 200 {object} MetricsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /metrics [get]
func (h *Handler) GetMetrics(ctx context.Context, c *app.RequestContext) {
	endpoint := c.Query("endpoint")
	if endpoint == "" {
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "endpoint query parameter is required"}})
		return
	}

	metrics, err := h.service.GetMetrics(ctx, endpoint)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	c.JSON(200, MetricsResponse{Data: metrics})
}

// GetSeries handles GET /series requests with query parameters
// @Summary Query series data
// @Description Query time series data with optional filters
// @Tags series
// @Param endpoint query string false "Endpoint filter"
// @Param metric query string false "Metric filter"
// @Param label_filter query string false "Label filter expression"
// @Param start query string true "Start time (RFC3339 or Unix timestamp)"
// @Param end query string true "End time (RFC3339 or Unix timestamp)"
// @Param limit query int false "Maximum number of results"
// @Produce json
// @Success 200 {object} SeriesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /series [get]
func (h *Handler) GetSeries(ctx context.Context, c *app.RequestContext) {
	query, err := h.parseSeriesQuery(c)
	if err != nil {
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: err.Error()}})
		return
	}

	series, err := h.service.QuerySeries(ctx, query)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	c.JSON(200, SeriesResponse{Data: toSeriesDataDTOs(series)})
}

// GetSeriesByID handles GET /series/:id requests
// @Summary Get series by ID
// @Description Get a single time series by its ID with optional time range
// @Tags series
// @Param id path string true "Series ID"
// @Param start query string false "Start time (RFC3339 or Unix timestamp, default: 1 hour ago)"
// @Param end query string false "End time (RFC3339 or Unix timestamp, default: now)"
// @Produce json
// @Success 200 {object} SeriesSingleResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /series/{id} [get]
func (h *Handler) GetSeriesByID(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "id parameter is required"}})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "invalid id parameter"}})
		return
	}

	timeRange := h.parseOptionalTimeRange(c)

	series, err := h.service.GetSeriesByID(ctx, id, &timeRange)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	if series == nil {
		c.JSON(404, ErrorResponse{Error: ErrorDetail{Code: "NOT_FOUND", Message: "series not found"}})
		return
	}

	dto := toSeriesDataDTO(series)
	c.JSON(200, SeriesSingleResponse{Data: &dto})
}

// QuerySeries handles POST /series/query requests for complex queries
// @Summary Query multiple series
// @Description Query multiple time series with complex filters and optional aggregation
// @Tags series
// @Accept json
// @Produce json
// @Param request body SeriesQueryRequestBody true "Query parameters"
// @Success 200 {object} SeriesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /series/query [post]
func (h *Handler) QuerySeries(ctx context.Context, c *app.RequestContext) {
	var req SeriesQueryRequestBody
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: err.Error()}})
		return
	}

	// Validate time range
	if req.Start == nil || req.End == nil {
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "start and end time are required"}})
		return
	}

	query := &MultiSeriesQuery{
		Endpoints:   req.Endpoints,
		Metrics:     req.Metrics,
		LabelFilter: req.Labels,
		TimeRange: TimeRange{
			Start: *req.Start,
			End:   *req.End,
		},
	}

	if req.Aggregation != nil {
		query.Aggregation = &Aggregation{
			Interval: req.Aggregation.Interval,
			Function: req.Aggregation.Function,
		}
	}

	series, err := h.service.QuerySeriesMulti(ctx, query)
	if err != nil {
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
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

	// Parse time range (required)
	timeRange, err := h.parseTimeRange(c)
	if err != nil {
		return nil, err
	}
	query.TimeRange = timeRange

	return query, nil
}

// parseTimeRange parses time range from query parameters (required)
func (h *Handler) parseTimeRange(c *app.RequestContext) (TimeRange, error) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	// Time range is required for GetSeries
	if startStr == "" || endStr == "" {
		return TimeRange{}, &TimeRangeError{Message: "start and end time parameters are required"}
	}

	var timeRange TimeRange

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		// Try parsing as Unix timestamp
		unixStart, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil {
			return TimeRange{}, &TimeParseError{Field: "start", Value: startStr}
		}
		start = time.Unix(unixStart, 0)
	}
	timeRange.Start = start

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		// Try parsing as Unix timestamp
		unixEnd, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil {
			return TimeRange{}, &TimeParseError{Field: "end", Value: endStr}
		}
		end = time.Unix(unixEnd, 0)
	}
	timeRange.End = end

	return timeRange, nil
}

// parseOptionalTimeRange parses time range from query parameters (optional, defaults to last hour)
func (h *Handler) parseOptionalTimeRange(c *app.RequestContext) TimeRange {
	timeRange := TimeRange{
		Start: time.Now().Add(-1 * time.Hour), // Default: last 1 hour
		End:   time.Now(),
	}

	if startStr := c.Query("start"); startStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Try parsing as Unix timestamp
			unixStart, err := strconv.ParseInt(startStr, 10, 64)
			if err == nil {
				timeRange.Start = time.Unix(unixStart, 0)
			}
		} else {
			timeRange.Start = start
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			// Try parsing as Unix timestamp
			unixEnd, err := strconv.ParseInt(endStr, 10, 64)
			if err == nil {
				timeRange.End = time.Unix(unixEnd, 0)
			}
		} else {
			timeRange.End = end
		}
	}

	return timeRange
}

// TimeRangeError indicates that time range parameters are missing
type TimeRangeError struct {
	Message string
}

func (e *TimeRangeError) Error() string {
	return e.Message
}

// TimeParseError indicates that a time parameter could not be parsed
type TimeParseError struct {
	Field string
	Value string
}

func (e *TimeParseError) Error() string {
	return "invalid " + e.Field + " time format: " + e.Value
}

// toSeriesDataDTO converts SeriesData to SeriesDataDTO
func toSeriesDataDTO(s *SeriesData) SeriesDataDTO {
	return SeriesDataDTO{
		ID:               strconv.FormatInt(s.Meta.ID, 10),
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