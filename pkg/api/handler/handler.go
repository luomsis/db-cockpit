package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/db-cockpit/pkg/domain"
	"github.com/db-cockpit/pkg/domain/llm"
	"github.com/db-cockpit/pkg/domain/performance"
	"github.com/db-cockpit/pkg/domain/sqlgovernance"
	"github.com/db-cockpit/pkg/domain/threshold"
)

// GatewayHandler handles gateway requests for domain services
// Note: Data Query operations are proxied to Data Query Service REST API
type GatewayHandler struct {
	sqlGovern   sqlgovernance.SQLGovernanceService
	performance performance.PerformanceService
	threshold   threshold.ThresholdService
	llm         llm.LLMOrchestratorService
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(
	sqlGovern sqlgovernance.SQLGovernanceService,
	perf performance.PerformanceService,
	thresh threshold.ThresholdService,
	llm llm.LLMOrchestratorService,
) *GatewayHandler {
	return &GatewayHandler{
		sqlGovern:   sqlGovern,
		performance: perf,
		threshold:   thresh,
		llm:         llm,
	}
}

// Health handles health check requests
func (h *GatewayHandler) Health(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "db-cockpit-gateway",
	})
}

// =====================
// SQL Governance Handlers
// =====================

// ReviewSQL handles SQL review requests
func (h *GatewayHandler) ReviewSQL(ctx context.Context, c *app.RequestContext) {
	var req SQLReviewRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	reviewReq := &sqlgovernance.SQLReviewRequest{
		DatabaseID: req.DatabaseID,
		SQLText:    req.SQL,
		Context:    req.Context,
	}

	result, err := h.sqlGovern.ReviewSQL(domainCtx, reviewReq)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// ExecuteSQL handles SQL execution requests
func (h *GatewayHandler) ExecuteSQL(ctx context.Context, c *app.RequestContext) {
	var req ExecuteSQLRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	execReq := &sqlgovernance.SQLExecuteRequest{
		DatabaseID:      req.DatabaseID,
		SQLText:         req.SQL,
		TimeoutSeconds:  req.TimeoutSeconds,
		MaxRows:         req.MaxRows,
		DryRun:          req.DryRun,
		RequireApproval: req.RequireApproval,
	}

	result, err := h.sqlGovern.ExecuteSQL(domainCtx, execReq)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// GetSQLAudit handles SQL audit retrieval
func (h *GatewayHandler) GetSQLAudit(ctx context.Context, c *app.RequestContext) {
	domainCtx := h.getDomainContext(c, "")

	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	entries, err := h.sqlGovern.GetAuditTrail(domainCtx, startTime, endTime, 100)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"entries": entries,
		"total":   len(entries),
	})
}

// =====================
// Performance Handlers
// =====================

// Diagnose handles performance diagnosis requests
func (h *GatewayHandler) Diagnose(ctx context.Context, c *app.RequestContext) {
	var req DiagnosisRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	diagReq := &performance.DiagnosisRequest{
		DatabaseID:   req.DatabaseID,
		Scope:        performance.DiagnosisScope(req.Scope),
		StartTime:    time.Unix(req.StartTime, 0),
		EndTime:      time.Unix(req.EndTime, 0),
		DeepAnalysis: req.DeepAnalysis,
	}

	result, err := h.performance.Diagnose(domainCtx, diagReq)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// GetMetrics handles metrics retrieval
func (h *GatewayHandler) GetMetrics(ctx context.Context, c *app.RequestContext) {
	var req GetMetricsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	metrics, err := h.performance.GetMetrics(
		domainCtx,
		req.DatabaseID,
		req.MetricNames,
		time.Unix(req.StartTime, 0),
		time.Unix(req.EndTime, 0),
	)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"metrics": metrics,
	})
}

// GetSlowQueries handles slow query retrieval
func (h *GatewayHandler) GetSlowQueries(ctx context.Context, c *app.RequestContext) {
	var req GetSlowQueriesRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	queries, err := h.performance.GetSlowQueries(
		domainCtx,
		req.DatabaseID,
		time.Unix(req.StartTime, 0),
		time.Unix(req.EndTime, 0),
		req.MinDurationMs,
		req.Limit,
	)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"queries": queries,
	})
}

// =====================
// Threshold Handlers
// =====================

// GetThresholds handles threshold retrieval
func (h *GatewayHandler) GetThresholds(ctx context.Context, c *app.RequestContext) {
	var req GetThresholdsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	thresholds, err := h.threshold.GetThresholds(domainCtx, req.DatabaseID, req.MetricNames)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"thresholds": thresholds,
	})
}

// UpdateThreshold handles threshold updates
func (h *GatewayHandler) UpdateThreshold(ctx context.Context, c *app.RequestContext) {
	var req UpdateThresholdRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	err := h.threshold.UpdateThreshold(
		domainCtx,
		req.ThresholdID,
		req.Value,
		threshold.ThresholdType(req.Type),
	)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"success": true,
	})
}

// =====================
// LLM Handlers
// =====================

// Chat handles chat requests
func (h *GatewayHandler) Chat(ctx context.Context, c *app.RequestContext) {
	var req ChatRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	chatReq := &llm.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		Context:   &llm.ChatContext{},
		Options:   &llm.ChatOptions{},
	}

	result, err := h.llm.Chat(domainCtx, chatReq)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// GenerateSQL handles SQL generation requests
func (h *GatewayHandler) GenerateSQL(ctx context.Context, c *app.RequestContext) {
	var req GenerateSQLRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	result, err := h.llm.GenerateSQL(domainCtx, req.DatabaseID, &llm.SQLGenerationRequest{
		NaturalLanguage: req.NaturalLanguage,
		SchemaContext:   req.SchemaContext,
	})
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// GetRecommendations handles recommendation requests
func (h *GatewayHandler) GetRecommendations(ctx context.Context, c *app.RequestContext) {
	var req GetRecommendationsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	domainCtx := h.getDomainContext(c, "")

	recs, err := h.llm.GetRecommendations(
		domainCtx,
		req.DatabaseID,
		llm.RecommendationCategory(req.Category),
		req.Limit,
	)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"recommendations": recs,
	})
}

// Helper method to get domain context
func (h *GatewayHandler) getDomainContext(c *app.RequestContext, tenantID string) *domain.DomainContext {
	ctx := context.Background()

	if tid, exists := c.Get("tenant_id"); exists && tenantID == "" {
		if s, ok := tid.(string); ok {
			tenantID = s
		}
	}

	return domain.NewDomainContext(ctx, tenantID, "")
}

// Request/Response DTOs
type SQLReviewRequest struct {
	DatabaseID string            `json:"database_id"`
	SQL        string            `json:"sql"`
	Context    map[string]string `json:"context"`
}

type ExecuteSQLRequest struct {
	DatabaseID      string `json:"database_id"`
	SQL             string `json:"sql"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	MaxRows         int    `json:"max_rows"`
	DryRun          bool   `json:"dry_run"`
	RequireApproval bool   `json:"require_approval"`
}

type DiagnosisRequest struct {
	DatabaseID   string `json:"database_id"`
	Scope        string `json:"scope"`
	StartTime    int64  `json:"start_time"`
	EndTime      int64  `json:"end_time"`
	DeepAnalysis bool   `json:"deep_analysis"`
}

type GetMetricsRequest struct {
	DatabaseID  string   `json:"database_id"`
	MetricNames []string `json:"metric_names"`
	StartTime   int64    `json:"start_time"`
	EndTime     int64    `json:"end_time"`
}

type GetSlowQueriesRequest struct {
	DatabaseID    string  `json:"database_id"`
	StartTime     int64   `json:"start_time"`
	EndTime       int64   `json:"end_time"`
	MinDurationMs float64 `json:"min_duration_ms"`
	Limit         int     `json:"limit"`
}

type GetThresholdsRequest struct {
	DatabaseID  string   `json:"database_id"`
	MetricNames []string `json:"metric_names"`
}

type UpdateThresholdRequest struct {
	ThresholdID string  `json:"threshold_id"`
	Value       float64 `json:"value"`
	Type        string  `json:"type"`
}

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type GenerateSQLRequest struct {
	DatabaseID      string `json:"database_id"`
	NaturalLanguage string `json:"natural_language"`
	SchemaContext   string `json:"schema_context"`
}

type GetRecommendationsRequest struct {
	DatabaseID string `json:"database_id"`
	Category   string `json:"category"`
	Limit      int    `json:"limit"`
}
