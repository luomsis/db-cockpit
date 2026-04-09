package dataquery

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/db-cockpit/pkg/common/logger"
	"go.uber.org/zap"
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
	ID         string            `json:"id"`
	Endpoint   string            `json:"endpoint"`
	Metric     string            `json:"metric"`
	Labels     map[string]string `json:"labels"`
	LabelsHash string            `json:"labels_hash"`
	CreatedAt  time.Time         `json:"created_at"`
	Points     []DataPointDTO    `json:"points,omitempty"`
	Statistics *SeriesStatisticsDTO `json:"statistics,omitempty"`
}

// DataPointDTO is the JSON representation of DataPoint
type DataPointDTO struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
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
	Endpoints []string   `json:"endpoints"`
	Metrics   []string   `json:"metrics"`
	Labels    string     `json:"labels"`
	Start     *time.Time `json:"start"`
	End       *time.Time `json:"end"`
}

// InstanceMetaResponse is the response for instance metadata
type InstanceMetaResponse struct {
	Data *InstanceMeta `json:"data"`
}

// InstancesListResponse is the response for GetInstances
type InstancesListResponse struct {
	Data       []*InstanceMeta  `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
}

// AlertsListResponse is the response for GetAlerts with pagination
type AlertsListResponse struct {
	Data       []*Alert        `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
}

// SlowQueryListResponse is the response for GetSlowQueries
type SlowQueryListResponse struct {
	Data       []*SlowQuery    `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
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
	logger.Debug("GetEndpoints called")

	endpoints, err := h.service.GetEndpoints(ctx)
	if err != nil {
		logger.Error("GetEndpoints failed", zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("GetEndpoints success", zap.Int("count", len(endpoints)))
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
		logger.Warn("GetMetrics missing endpoint parameter")
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "endpoint query parameter is required"}})
		return
	}

	logger.Debug("GetMetrics called", zap.String("endpoint", endpoint))

	metrics, err := h.service.GetMetrics(ctx, endpoint)
	if err != nil {
		logger.Error("GetMetrics failed", zap.String("endpoint", endpoint), zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("GetMetrics success", zap.String("endpoint", endpoint), zap.Int("count", len(metrics)))
	c.JSON(200, MetricsResponse{Data: metrics})
}

// GetSeries handles GET /series requests with query parameters
// @Summary Query series data
// @Description Query time series data with optional filters and sampling
// @Tags series
// @Param endpoint query string false "Endpoint filter"
// @Param metric query string false "Metric filter"
// @Param label_filter query string false "Label filter expression"
// @Param start query string true "Start time (RFC3339 or Unix timestamp)"
// @Param end query string true "End time (RFC3339 or Unix timestamp)"
// @Param step query string false "Sampling interval (e.g., 5m, 1h). Returns averaged values within each time bucket. Valid range: 1m to 1h."
// @Param limit query int false "Maximum number of results"
// @Produce json
// @Success 200 {object} SeriesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /series [get]
func (h *Handler) GetSeries(ctx context.Context, c *app.RequestContext) {
	query, err := h.parseSeriesQuery(c)
	if err != nil {
		logger.Warn("GetSeries invalid query", zap.Error(err))
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: err.Error()}})
		return
	}

	logger.Debug("GetSeries called",
		zap.String("endpoint", query.Endpoint),
		zap.String("metric", query.Metric),
		zap.Time("start", query.TimeRange.Start),
		zap.Time("end", query.TimeRange.End),
	)

	series, err := h.service.QuerySeries(ctx, query)
	if err != nil {
		logger.Error("GetSeries failed", zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("GetSeries success", zap.Int("count", len(series)))
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
		logger.Warn("GetSeriesByID missing id parameter")
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "id parameter is required"}})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Warn("GetSeriesByID invalid id", zap.String("id", idStr), zap.Error(err))
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "invalid id parameter"}})
		return
	}

	timeRange := h.parseOptionalTimeRange(c)

	logger.Debug("GetSeriesByID called", zap.Int64("id", id), zap.Time("start", timeRange.Start), zap.Time("end", timeRange.End))

	series, err := h.service.GetSeriesByID(ctx, id, &timeRange)
	if err != nil {
		logger.Error("GetSeriesByID failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	if series == nil {
		logger.Debug("GetSeriesByID not found", zap.Int64("id", id))
		c.JSON(404, ErrorResponse{Error: ErrorDetail{Code: "NOT_FOUND", Message: "series not found"}})
		return
	}

	logger.Debug("GetSeriesByID success", zap.Int64("id", id), zap.Int("points", len(series.Points)))
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
		logger.Warn("QuerySeries invalid JSON body", zap.Error(err))
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: err.Error()}})
		return
	}

	// Validate time range
	if req.Start == nil || req.End == nil {
		logger.Warn("QuerySeries missing time range")
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "start and end time are required"}})
		return
	}

	logger.Debug("QuerySeries called",
		zap.Int("endpoints", len(req.Endpoints)),
		zap.Int("metrics", len(req.Metrics)),
		zap.Time("start", *req.Start),
		zap.Time("end", *req.End),
	)

	query := &MultiSeriesQuery{
		Endpoints:   req.Endpoints,
		Metrics:     req.Metrics,
		LabelFilter: req.Labels,
		TimeRange: TimeRange{
			Start: *req.Start,
			End:   *req.End,
		},
	}

	series, err := h.service.QuerySeriesMulti(ctx, query)
	if err != nil {
		logger.Error("QuerySeries failed", zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("QuerySeries success", zap.Int("count", len(series)))
	c.JSON(200, SeriesResponse{Data: toSeriesDataDTOs(series)})
}

// GetInstance handles GET /instances/:endpoint requests
// @Summary Get instance metadata by endpoint
// @Description Get database instance metadata (db_type, entity_name, instance_vip, instance_port, etc.) by endpoint
// @Tags instances
// @Param endpoint path string true "Instance endpoint"
// @Produce json
// @Success 200 {object} InstanceMetaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /instances/{endpoint} [get]
func (h *Handler) GetInstance(ctx context.Context, c *app.RequestContext) {
	endpoint := c.Param("endpoint")
	if endpoint == "" {
		logger.Warn("GetInstance missing endpoint parameter")
		c.JSON(400, ErrorResponse{Error: ErrorDetail{Code: "INVALID_PARAMETER", Message: "endpoint parameter is required"}})
		return
	}

	logger.Debug("GetInstance called", zap.String("endpoint", endpoint))

	instance, err := h.service.GetInstanceByEndpoint(ctx, endpoint)
	if err != nil {
		logger.Error("GetInstance failed", zap.String("endpoint", endpoint), zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	if instance == nil {
		logger.Debug("GetInstance not found", zap.String("endpoint", endpoint))
		c.JSON(404, ErrorResponse{Error: ErrorDetail{Code: "NOT_FOUND", Message: "instance not found"}})
		return
	}

	logger.Debug("GetInstance success", zap.String("endpoint", endpoint), zap.String("db_type", instance.DbType))
	c.JSON(200, InstanceMetaResponse{Data: instance})
}

// GetInstances handles GET /instances requests
// @Summary Get all instances with pagination
// @Description Get all database instances with pagination support
// @Tags instances
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Produce json
// @Success 200 {object} InstancesListResponse
// @Failure 500 {object} ErrorResponse
// @Router /instances [get]
func (h *Handler) GetInstances(ctx context.Context, c *app.RequestContext) {
	logger.Debug("GetInstances called")

	// Parse pagination params
	page := parsePageParam(c, 1)       // default 1
	pageSize := parsePageSizeParam(c, 20) // default 20

	// Validate
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100 // max limit
	}

	req := &InstancesQueryRequest{
		Pagination: PaginationRequest{Page: page, PageSize: pageSize},
	}

	resp, err := h.service.GetAllInstances(ctx, req)
	if err != nil {
		logger.Error("GetInstances failed", zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("GetInstances success",
		zap.Int("count", len(resp.Data)),
		zap.Int64("total", resp.Pagination.TotalCount),
		zap.Int("page", resp.Pagination.CurrentPage))
	c.JSON(200, resp)
}

// parsePageParam parses the page query parameter with a default value
func parsePageParam(c *app.RequestContext, defaultVal int) int {
	if val := c.Query("page"); val != "" {
		parsed, err := strconv.Atoi(val)
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultVal
}

// parsePageSizeParam parses the page_size query parameter with a default value
func parsePageSizeParam(c *app.RequestContext, defaultVal int) int {
	if val := c.Query("page_size"); val != "" {
		parsed, err := strconv.Atoi(val)
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultVal
}

// GetAlerts handles GET /alerts requests
// @Summary Query alerts with optional filters
// @Description Query alert events with optional filters (all parameters are optional). Time filter uses overlap logic: alert matches if its time range overlaps with query time range.
// @Tags alerts
// @Param endpoint query string false "Instance endpoint filter"
// @Param alert_text query string false "Alert text keyword filter (case-insensitive LIKE match)"
// @Param start query string false "Start time filter - alerts active after this time (RFC3339 or Unix timestamp)"
// @Param end query string false "End time filter - alerts started before this time (RFC3339 or Unix timestamp)"
// @Param metric query string false "Metric name filter (exact match)"
// @Param status query string false "Status filter (exact match)"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Produce json
// @Success 200 {object} AlertsListResponse
// @Failure 500 {object} ErrorResponse
// @Router /alerts [get]
func (h *Handler) GetAlerts(ctx context.Context, c *app.RequestContext) {
	// Parse endpoint (optional query parameter)
	endpoint := c.Query("endpoint")

	// Parse alert_text (optional)
	alertText := c.Query("alert_text")

	// Parse metric (optional)
	metric := c.Query("metric")

	// Parse status (optional)
	status := c.Query("status")

	// Parse optional time range filters
	var startTime, endTime *time.Time

	if startStr := c.Query("start"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Try parsing as Unix timestamp
			unixStart, err := strconv.ParseInt(startStr, 10, 64)
			if err == nil {
				parsed := time.Unix(unixStart, 0)
				startTime = &parsed
			}
		} else {
			startTime = &t
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			// Try parsing as Unix timestamp
			unixEnd, err := strconv.ParseInt(endStr, 10, 64)
			if err == nil {
				parsed := time.Unix(unixEnd, 0)
				endTime = &parsed
			}
		} else {
			endTime = &t
		}
	}

	// Parse pagination params
	page := parsePageParam(c, 1)       // default 1
	pageSize := parsePageSizeParam(c, 20) // default 20

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100 // max limit
	}

	req := &AlertsQueryRequest{
		Endpoint:   endpoint,
		AlertText:  alertText,
		StartTime:  startTime,
		EndTime:    endTime,
		Metric:     metric,
		Status:     status,
		Pagination: PaginationRequest{Page: page, PageSize: pageSize},
	}

	logger.Debug("GetAlerts called",
		zap.String("endpoint", endpoint),
		zap.String("alert_text", alertText),
		zap.Bool("has_start_time", startTime != nil),
		zap.Bool("has_end_time", endTime != nil),
		zap.String("metric", metric),
		zap.String("status", status),
		zap.Int("page", page),
		zap.Int("page_size", pageSize))

	resp, err := h.service.GetAlertsByEndpoint(ctx, req)
	if err != nil {
		logger.Error("GetAlerts failed", zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("GetAlerts success",
		zap.Int("count", len(resp.Data)),
		zap.Int64("total", resp.Pagination.TotalCount))
	c.JSON(200, resp)
}

// GetSlowQueries handles GET /slow-queries requests
// @Summary Query slow queries with optional filters
// @Description Query slow SQL query records with optional filters (all parameters are optional). Supports filtering by endpoint, SQL keyword, and time range.
// @Tags slow-queries
// @Param endpoint query string false "Instance endpoint filter"
// @Param sql_keyword query string false "SQL text keyword filter (case-insensitive LIKE match)"
// @Param start query string false "Start time (RFC3339 or Unix timestamp)"
// @Param end query string false "End time (RFC3339 or Unix timestamp)"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Produce json
// @Success 200 {object} SlowQueryListResponse
// @Failure 500 {object} ErrorResponse
// @Router /slow-queries [get]
func (h *Handler) GetSlowQueries(ctx context.Context, c *app.RequestContext) {
	// Parse optional endpoint parameter
	endpoint := c.Query("endpoint")

	// Parse optional SQL keyword
	sqlKeyword := c.Query("sql_keyword")

	// Parse optional time range - only filter if parameters provided
	var timeRange *TimeRange
	if c.Query("start") != "" || c.Query("end") != "" {
		tr := h.parseProvidedTimeRange(c)
		timeRange = &tr
	}

	// Parse pagination params
	page := parsePageParam(c, 1)
	pageSize := parsePageSizeParam(c, 20)

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100 // max limit
	}

	req := &SlowQueryRequest{
		Endpoint:   endpoint,
		SqlKeyword: sqlKeyword,
		TimeRange:  timeRange,
		Pagination: PaginationRequest{Page: page, PageSize: pageSize},
	}

	logger.Debug("GetSlowQueries called",
		zap.String("endpoint", endpoint),
		zap.Bool("hasTimeRange", timeRange != nil),
		zap.String("sqlKeyword", sqlKeyword))

	resp, err := h.service.GetSlowQueries(ctx, req)
	if err != nil {
		logger.Error("GetSlowQueries failed", zap.String("endpoint", endpoint), zap.Error(err))
		c.JSON(500, ErrorResponse{Error: ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}

	logger.Debug("GetSlowQueries success",
		zap.String("endpoint", endpoint),
		zap.Int("count", len(resp.Data)),
		zap.Int64("total", resp.Pagination.TotalCount))
	c.JSON(200, resp)
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

	// Parse step (sampling interval)
	if stepStr := c.Query("step"); stepStr != "" {
		interval, err := parseStep(stepStr)
		if err != nil {
			return nil, err
		}
		query.Interval = interval
	}

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

// parseProvidedTimeRange parses start/end parameters only when provided
// Does not set default values - returns zero TimeRange if nothing provided
func (h *Handler) parseProvidedTimeRange(c *app.RequestContext) TimeRange {
	var timeRange TimeRange

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

// InvalidStepError indicates that the step parameter is invalid
type InvalidStepError struct {
	Value  string
	Reason string
}

func (e *InvalidStepError) Error() string {
	return "invalid step '" + e.Value + "': " + e.Reason
}

// Step parsing constants
const (
	MinStep = time.Minute // 1 minute minimum
	MaxStep = time.Hour   // 1 hour maximum
)

// parseStep parses and validates the step parameter
func parseStep(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil // No step = raw data
	}

	interval, err := time.ParseDuration(s)
	if err != nil {
		return 0, &InvalidStepError{Value: s, Reason: "invalid format"}
	}

	if interval < MinStep {
		return 0, &InvalidStepError{Value: s, Reason: "minimum step is 1m"}
	}

	if interval > MaxStep {
		return 0, &InvalidStepError{Value: s, Reason: "maximum step is 1h"}
	}

	return interval, nil
}

// toSeriesDataDTO converts SeriesData to SeriesDataDTO
func toSeriesDataDTO(s *SeriesData) SeriesDataDTO {
	return SeriesDataDTO{
		ID:         strconv.FormatInt(s.Meta.ID, 10),
		Endpoint:   s.Meta.Endpoint,
		Metric:     s.Meta.Metric,
		Labels:     s.Meta.Labels,
		LabelsHash: s.Meta.LabelsHash,
		CreatedAt:  s.Meta.CreatedAt,
		Points:     toDataPointDTOs(s.Points),
		Statistics: toSeriesStatisticsDTO(s.Statistics),
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