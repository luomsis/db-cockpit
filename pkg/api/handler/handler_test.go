package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/db-cockpit/pkg/domain"
	"github.com/db-cockpit/pkg/domain/llm"
	"github.com/db-cockpit/pkg/domain/performance"
	"github.com/db-cockpit/pkg/domain/sqlgovernance"
	"github.com/db-cockpit/pkg/domain/threshold"
)

// =====================
// Mock Implementations
// =====================

type mockSQLGovernanceService struct {
	reviewResult   *sqlgovernance.SQLReviewResult
	executeResult  *sqlgovernance.SQLExecuteResult
	auditEntries   []sqlgovernance.AuditEntry
	err            error
	lastReviewReq  *sqlgovernance.SQLReviewRequest
	lastExecuteReq *sqlgovernance.SQLExecuteRequest
}

func (m *mockSQLGovernanceService) Name() string { return "MockSQLGovernanceService" }
func (m *mockSQLGovernanceService) Initialize(ctx context.Context) error {
	return nil
}
func (m *mockSQLGovernanceService) Shutdown(ctx context.Context) error {
	return nil
}
func (m *mockSQLGovernanceService) Health(ctx context.Context) error {
	return nil
}
func (m *mockSQLGovernanceService) ReviewSQL(ctx *domain.DomainContext, req *sqlgovernance.SQLReviewRequest) (*sqlgovernance.SQLReviewResult, error) {
	m.lastReviewReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.reviewResult != nil {
		return m.reviewResult, nil
	}
	return &sqlgovernance.SQLReviewResult{
		ReviewID:  "review-123",
		Approved:  true,
		RiskLevel: sqlgovernance.RiskLevelLow,
	}, nil
}
func (m *mockSQLGovernanceService) ExecuteSQL(ctx *domain.DomainContext, req *sqlgovernance.SQLExecuteRequest) (*sqlgovernance.SQLExecuteResult, error) {
	m.lastExecuteReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.executeResult != nil {
		return m.executeResult, nil
	}
	return &sqlgovernance.SQLExecuteResult{
		ExecutionID:     "exec-123",
		Status:          "completed",
		RowsAffected:    10,
		ExecutionTimeMs: 100,
	}, nil
}
func (m *mockSQLGovernanceService) ExplainSQL(ctx *domain.DomainContext, databaseID, sql string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "EXPLAIN: " + sql, nil
}
func (m *mockSQLGovernanceService) GetAuditTrail(ctx *domain.DomainContext, startTime, endTime time.Time, limit int) ([]sqlgovernance.AuditEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.auditEntries != nil {
		return m.auditEntries, nil
	}
	return []sqlgovernance.AuditEntry{
		{AuditID: "audit-1", SQLText: "SELECT 1", Status: "completed"},
	}, nil
}
func (m *mockSQLGovernanceService) CreateRule(ctx *domain.DomainContext, rule *sqlgovernance.GovernanceRule) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "rule-123", nil
}
func (m *mockSQLGovernanceService) ListRules(ctx *domain.DomainContext, enabledOnly bool) ([]sqlgovernance.GovernanceRule, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []sqlgovernance.GovernanceRule{
		{RuleID: "rule-1", Name: "Test Rule", Enabled: true},
	}, nil
}
func (m *mockSQLGovernanceService) ValidateSQL(ctx *domain.DomainContext, databaseID, sql string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []string{}, nil
}

type mockPerformanceService struct {
	diagnosisResult *performance.DiagnosisResult
	metrics         []performance.MetricSeries
	slowQueries     []performance.SlowQuery
	err             error
}

func (m *mockPerformanceService) Name() string { return "MockPerformanceService" }
func (m *mockPerformanceService) Initialize(ctx context.Context) error {
	return nil
}
func (m *mockPerformanceService) Shutdown(ctx context.Context) error {
	return nil
}
func (m *mockPerformanceService) Health(ctx context.Context) error {
	return nil
}
func (m *mockPerformanceService) Diagnose(ctx *domain.DomainContext, req *performance.DiagnosisRequest) (*performance.DiagnosisResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.diagnosisResult != nil {
		return m.diagnosisResult, nil
	}
	return &performance.DiagnosisResult{
		DiagnosisID: "diag-123",
		Status:      "completed",
		HealthScore: 85.5,
	}, nil
}
func (m *mockPerformanceService) GetMetrics(ctx *domain.DomainContext, databaseID string, metricNames []string, startTime, endTime time.Time) ([]performance.MetricSeries, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.metrics != nil {
		return m.metrics, nil
	}
	return []performance.MetricSeries{
		{Name: "cpu_usage", Points: []performance.MetricPoint{{Value: 75.5}}},
	}, nil
}
func (m *mockPerformanceService) GetSlowQueries(ctx *domain.DomainContext, databaseID string, startTime, endTime time.Time, minDurationMs float64, limit int) ([]performance.SlowQuery, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.slowQueries != nil {
		return m.slowQueries, nil
	}
	return []performance.SlowQuery{
		{QueryID: "q-1", SQLText: "SELECT * FROM large_table", DurationMs: 5000},
	}, nil
}
func (m *mockPerformanceService) AnalyzeQuery(ctx *domain.DomainContext, databaseID, sql string) (*performance.QueryAnalysis, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &performance.QueryAnalysis{AnalysisID: "analysis-1"}, nil
}
func (m *mockPerformanceService) GetResourceUtilization(ctx *domain.DomainContext, databaseID string, startTime, endTime time.Time) (*performance.ResourceUtilization, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &performance.ResourceUtilization{}, nil
}
func (m *mockPerformanceService) GenerateReport(ctx *domain.DomainContext, databaseID string, reportType string, startTime, endTime time.Time) (*performance.Report, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &performance.Report{ReportID: "report-1"}, nil
}

type mockThresholdService struct {
	thresholds []threshold.Threshold
	err        error
}

func (m *mockThresholdService) Name() string { return "MockThresholdService" }
func (m *mockThresholdService) Initialize(ctx context.Context) error {
	return nil
}
func (m *mockThresholdService) Shutdown(ctx context.Context) error {
	return nil
}
func (m *mockThresholdService) Health(ctx context.Context) error {
	return nil
}
func (m *mockThresholdService) GetThresholds(ctx *domain.DomainContext, databaseID string, metricNames []string) ([]threshold.Threshold, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.thresholds != nil {
		return m.thresholds, nil
	}
	return []threshold.Threshold{
		{ThresholdID: "thresh-1", MetricName: "cpu_usage", Value: 80.0},
	}, nil
}
func (m *mockThresholdService) UpdateThreshold(ctx *domain.DomainContext, thresholdID string, value float64, thresholdType threshold.ThresholdType) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
func (m *mockThresholdService) CalculateThresholds(ctx *domain.DomainContext, databaseID, metricName string, startTime, endTime time.Time, method threshold.CalculationMethod) ([]threshold.CalculatedThreshold, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []threshold.CalculatedThreshold{}, nil
}
func (m *mockThresholdService) CheckThreshold(ctx context.Context, tenantID, databaseID, metricName string, value float64) (bool, string, error) {
	if m.err != nil {
		return false, "", m.err
	}
	return false, "", nil
}
func (m *mockThresholdService) GetThresholdHistory(ctx *domain.DomainContext, thresholdID string, startTime, endTime time.Time, limit int) ([]threshold.ThresholdHistoryEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []threshold.ThresholdHistoryEntry{}, nil
}
func (m *mockThresholdService) CreateThresholdRule(ctx *domain.DomainContext, rule *threshold.ThresholdRule) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "rule-123", nil
}
func (m *mockThresholdService) SubscribeAlerts(ctx *domain.DomainContext, callback func(alert *threshold.ThresholdAlert) error) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

type mockLLMService struct {
	chatResult       *llm.ChatResponse
	sqlGenResult     *llm.SQLGenerationResult
	recommendations  []llm.Recommendation
	err              error
}

func (m *mockLLMService) Name() string { return "MockLLMService" }
func (m *mockLLMService) Initialize(ctx context.Context) error {
	return nil
}
func (m *mockLLMService) Shutdown(ctx context.Context) error {
	return nil
}
func (m *mockLLMService) Health(ctx context.Context) error {
	return nil
}
func (m *mockLLMService) Chat(ctx *domain.DomainContext, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.chatResult != nil {
		return m.chatResult, nil
	}
	return &llm.ChatResponse{
		ResponseID: "resp-123",
		SessionID:  req.SessionID,
		Message:    "This is a test response",
	}, nil
}
func (m *mockLLMService) StreamChat(ctx *domain.DomainContext, req *llm.ChatRequest, callback func(chunk string) error) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
func (m *mockLLMService) AnalyzeIssue(ctx *domain.DomainContext, databaseID string, issue *llm.IssueContext, depth llm.AnalysisDepth) (*llm.IssueAnalysis, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.IssueAnalysis{AnalysisID: "analysis-1"}, nil
}
func (m *mockLLMService) GenerateSQL(ctx *domain.DomainContext, databaseID string, req *llm.SQLGenerationRequest) (*llm.SQLGenerationResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.sqlGenResult != nil {
		return m.sqlGenResult, nil
	}
	return &llm.SQLGenerationResult{
		GenerationID: "gen-123",
		SQL:          "SELECT * FROM users WHERE id = 1",
	}, nil
}
func (m *mockLLMService) ExplainSQL(ctx *domain.DomainContext, databaseID, sql string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "Explanation: " + sql, nil
}
func (m *mockLLMService) OptimizeSQL(ctx *domain.DomainContext, databaseID, sql string, goal llm.OptimizationGoal) (*llm.SQLOptimizationResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.SQLOptimizationResult{OptimizationID: "opt-1"}, nil
}
func (m *mockLLMService) GetRecommendations(ctx *domain.DomainContext, databaseID string, category llm.RecommendationCategory, limit int) ([]llm.Recommendation, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.recommendations != nil {
		return m.recommendations, nil
	}
	return []llm.Recommendation{
		{RecommendationID: "rec-1", Title: "Add index on users.email"},
	}, nil
}
func (m *mockLLMService) CreateEmbedding(ctx *domain.DomainContext, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []float32{0.1, 0.2, 0.3}, nil
}
func (m *mockLLMService) SemanticSearch(ctx *domain.DomainContext, query string, collections []string, topK int) ([]llm.SemanticSearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []llm.SemanticSearchResult{}, nil
}

// =====================
// Test Helper Functions
// =====================

func newTestHandler() (*GatewayHandler, *mockSQLGovernanceService, *mockPerformanceService, *mockThresholdService, *mockLLMService) {
	sqlMock := &mockSQLGovernanceService{}
	perfMock := &mockPerformanceService{}
	threshMock := &mockThresholdService{}
	llmMock := &mockLLMService{}

	handler := NewGatewayHandler(sqlMock, perfMock, threshMock, llmMock)
	return handler, sqlMock, perfMock, threshMock, llmMock
}

func createTestContext(body string) (context.Context, *app.RequestContext) {
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.SetBody([]byte(body))
	return ctx, reqCtx
}

// =====================
// Health Tests
// =====================

func TestHealth(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()
	ctx, reqCtx := createTestContext("")

	handler.Health(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Health() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Health() status = %v, want ok", resp["status"])
	}
	if resp["service"] != "db-cockpit-gateway" {
		t.Errorf("Health() service = %v, want db-cockpit-gateway", resp["service"])
	}
}

// =====================
// SQL Governance Tests
// =====================

func TestReviewSQL(t *testing.T) {
	handler, sqlMock, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","sql":"SELECT * FROM users","context":{}}`
	ctx, reqCtx := createTestContext(body)

	handler.ReviewSQL(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("ReviewSQL() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	if sqlMock.lastReviewReq == nil {
		t.Fatal("ReviewSQL() did not call service")
	}
	if sqlMock.lastReviewReq.DatabaseID != "db-1" {
		t.Errorf("DatabaseID = %s, want db-1", sqlMock.lastReviewReq.DatabaseID)
	}
	if sqlMock.lastReviewReq.SQLText != "SELECT * FROM users" {
		t.Errorf("SQLText = %s, want SELECT * FROM users", sqlMock.lastReviewReq.SQLText)
	}
}

func TestReviewSQL_InvalidJSON(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	ctx, reqCtx := createTestContext("invalid json")

	handler.ReviewSQL(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("ReviewSQL() status = %d, want 400", reqCtx.Response.StatusCode())
	}
}

func TestExecuteSQL(t *testing.T) {
	handler, sqlMock, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","sql":"UPDATE users SET name='test'","timeout_seconds":30,"max_rows":1000}`
	ctx, reqCtx := createTestContext(body)

	handler.ExecuteSQL(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("ExecuteSQL() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	if sqlMock.lastExecuteReq == nil {
		t.Fatal("ExecuteSQL() did not call service")
	}
	if sqlMock.lastExecuteReq.DatabaseID != "db-1" {
		t.Errorf("DatabaseID = %s, want db-1", sqlMock.lastExecuteReq.DatabaseID)
	}
	if sqlMock.lastExecuteReq.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d, want 30", sqlMock.lastExecuteReq.TimeoutSeconds)
	}
	if sqlMock.lastExecuteReq.MaxRows != 1000 {
		t.Errorf("MaxRows = %d, want 1000", sqlMock.lastExecuteReq.MaxRows)
	}
}

func TestGetSQLAudit(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	ctx, reqCtx := createTestContext("")

	handler.GetSQLAudit(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetSQLAudit() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["total"] == nil {
		t.Error("GetSQLAudit() missing total field")
	}
}

func TestSQLGovernance_ServiceError(t *testing.T) {
	handler, sqlMock, _, _, _ := newTestHandler()
	sqlMock.err = context.Canceled

	body := `{"database_id":"db-1","sql":"SELECT 1"}`
	ctx, reqCtx := createTestContext(body)

	handler.ReviewSQL(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("ReviewSQL() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}

// =====================
// Performance Tests
// =====================

func TestDiagnose(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Unix()
	endTime := now.Unix()
	body := `{"database_id":"db-1","scope":"full","start_time":` + fmt.Sprintf("%d", startTime) + `,"end_time":` + fmt.Sprintf("%d", endTime) + `,"deep_analysis":false}`
	ctx, reqCtx := createTestContext(body)

	handler.Diagnose(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Diagnose() status = %d, want 200", reqCtx.Response.StatusCode())
	}
}

func TestGetMetrics(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","metric_names":["cpu_usage","memory"],"start_time":1700000000,"end_time":1700086400}`
	ctx, reqCtx := createTestContext(body)

	handler.GetMetrics(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetMetrics() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["metrics"] == nil {
		t.Error("GetMetrics() missing metrics field")
	}
}

func TestGetSlowQueries(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","start_time":1700000000,"end_time":1700086400,"min_duration_ms":1000,"limit":10}`
	ctx, reqCtx := createTestContext(body)

	handler.GetSlowQueries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetSlowQueries() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["queries"] == nil {
		t.Error("GetSlowQueries() missing queries field")
	}
}

func TestPerformance_ServiceError(t *testing.T) {
	handler, _, perfMock, _, _ := newTestHandler()
	perfMock.err = context.Canceled

	body := `{"database_id":"db-1","start_time":1700000000,"end_time":1700086400}`
	ctx, reqCtx := createTestContext(body)

	handler.GetMetrics(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetMetrics() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}

// =====================
// Threshold Tests
// =====================

func TestGetThresholds(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","metric_names":["cpu_usage","memory"]}`
	ctx, reqCtx := createTestContext(body)

	handler.GetThresholds(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetThresholds() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["thresholds"] == nil {
		t.Error("GetThresholds() missing thresholds field")
	}
}

func TestUpdateThreshold(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	body := `{"threshold_id":"thresh-1","value":90.0,"type":"static"}`
	ctx, reqCtx := createTestContext(body)

	handler.UpdateThreshold(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("UpdateThreshold() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != true {
		t.Error("UpdateThreshold() success should be true")
	}
}

func TestThreshold_ServiceError(t *testing.T) {
	handler, _, _, threshMock, _ := newTestHandler()
	threshMock.err = context.Canceled

	body := `{"database_id":"db-1","metric_names":["cpu"]}`
	ctx, reqCtx := createTestContext(body)

	handler.GetThresholds(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("GetThresholds() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}

// =====================
// LLM Tests
// =====================

func TestChat(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	body := `{"session_id":"session-1","message":"How do I optimize this query?"}`
	ctx, reqCtx := createTestContext(body)

	handler.Chat(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Chat() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp llm.ChatResponse
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.SessionID != "session-1" {
		t.Errorf("SessionID = %s, want session-1", resp.SessionID)
	}
}

func TestGenerateSQL(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","natural_language":"Get all users created last month","schema_context":"users table with id, name, created_at"}`
	ctx, reqCtx := createTestContext(body)

	handler.GenerateSQL(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GenerateSQL() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp llm.SQLGenerationResult
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.SQL == "" {
		t.Error("GenerateSQL() should return non-empty SQL")
	}
}

func TestGetRecommendations(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","category":"performance","limit":5}`
	ctx, reqCtx := createTestContext(body)

	handler.GetRecommendations(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetRecommendations() status = %d, want 200", reqCtx.Response.StatusCode())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["recommendations"] == nil {
		t.Error("GetRecommendations() missing recommendations field")
	}
}

func TestLLM_ServiceError(t *testing.T) {
	handler, _, _, _, llmMock := newTestHandler()
	llmMock.err = context.Canceled

	body := `{"session_id":"s1","message":"test"}`
	ctx, reqCtx := createTestContext(body)

	handler.Chat(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("Chat() status = %d, want 500", reqCtx.Response.StatusCode())
	}
}

// =====================
// Domain Context Tests
// =====================

func TestGetDomainContext(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	ctx, reqCtx := createTestContext(`{"database_id":"db-1","sql":"SELECT 1"}`)

	// Simulate middleware setting tenant_id
	reqCtx.Set("tenant_id", "tenant-123")

	handler.ReviewSQL(ctx, reqCtx)

	// The handler should extract tenant_id from context
	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("ReviewSQL() status = %d, want 200", reqCtx.Response.StatusCode())
	}
}

// =====================
// Request DTO Tests
// =====================

func TestSQLReviewRequest_Binding(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid request", `{"database_id":"db-1","sql":"SELECT 1"}`, false},
		{"with context", `{"database_id":"db-1","sql":"SELECT 1","context":{"user":"admin"}}`, false},
		{"empty body", `{}`, false}, // Empty body is valid JSON, just empty fields
		{"invalid JSON", `not json`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req SQLReviewRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecuteSQLRequest_Binding(t *testing.T) {
	body := `{"database_id":"db-1","sql":"UPDATE users SET x=1","timeout_seconds":30,"max_rows":100,"dry_run":true,"require_approval":true}`

	var req ExecuteSQLRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if req.DatabaseID != "db-1" {
		t.Errorf("DatabaseID = %s, want db-1", req.DatabaseID)
	}
	if req.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d, want 30", req.TimeoutSeconds)
	}
	if req.MaxRows != 100 {
		t.Errorf("MaxRows = %d, want 100", req.MaxRows)
	}
	if !req.DryRun {
		t.Error("DryRun should be true")
	}
	if !req.RequireApproval {
		t.Error("RequireApproval should be true")
	}
}

func TestDiagnosisRequest_Binding(t *testing.T) {
	body := `{"database_id":"db-1","scope":"full","start_time":1700000000,"end_time":1700086400,"deep_analysis":true}`

	var req DiagnosisRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if req.DatabaseID != "db-1" {
		t.Errorf("DatabaseID = %s, want db-1", req.DatabaseID)
	}
	if req.Scope != "full" {
		t.Errorf("Scope = %s, want full", req.Scope)
	}
	if !req.DeepAnalysis {
		t.Error("DeepAnalysis should be true")
	}
}

func TestChatRequest_Binding(t *testing.T) {
	body := `{"session_id":"s-123","message":"Hello"}`

	var req ChatRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if req.SessionID != "s-123" {
		t.Errorf("SessionID = %s, want s-123", req.SessionID)
	}
	if req.Message != "Hello" {
		t.Errorf("Message = %s, want Hello", req.Message)
	}
}

func TestGenerateSQLRequest_Binding(t *testing.T) {
	body := `{"database_id":"db-1","natural_language":"Get all users","schema_context":"users(id, name)"}`

	var req GenerateSQLRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if req.DatabaseID != "db-1" {
		t.Errorf("DatabaseID = %s, want db-1", req.DatabaseID)
	}
	if req.NaturalLanguage != "Get all users" {
		t.Errorf("NaturalLanguage = %s, want 'Get all users'", req.NaturalLanguage)
	}
}

func TestGetRecommendationsRequest_Binding(t *testing.T) {
	body := `{"database_id":"db-1","category":"performance","limit":10}`

	var req GetRecommendationsRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if req.DatabaseID != "db-1" {
		t.Errorf("DatabaseID = %s, want db-1", req.DatabaseID)
	}
	if req.Category != "performance" {
		t.Errorf("Category = %s, want performance", req.Category)
	}
	if req.Limit != 10 {
		t.Errorf("Limit = %d, want 10", req.Limit)
	}
}

// =====================
// Edge Cases Tests
// =====================

func TestReviewSQL_EmptySQL(t *testing.T) {
	handler, sqlMock, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","sql":""}`
	ctx, reqCtx := createTestContext(body)

	handler.ReviewSQL(ctx, reqCtx)

	// Service should still be called even with empty SQL
	if sqlMock.lastReviewReq == nil {
		t.Error("ReviewSQL() should still call service with empty SQL")
	}
}

func TestExecuteSQL_DryRun(t *testing.T) {
	handler, sqlMock, _, _, _ := newTestHandler()

	body := `{"database_id":"db-1","sql":"DELETE FROM users","dry_run":true}`
	ctx, reqCtx := createTestContext(body)

	handler.ExecuteSQL(ctx, reqCtx)

	if sqlMock.lastExecuteReq == nil {
		t.Fatal("ExecuteSQL() did not call service")
	}
	if !sqlMock.lastExecuteReq.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestGetSlowQueries_DefaultLimit(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	// Without limit specified
	body := `{"database_id":"db-1","start_time":1700000000,"end_time":1700086400}`
	ctx, reqCtx := createTestContext(body)

	handler.GetSlowQueries(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("GetSlowQueries() status = %d, want 200", reqCtx.Response.StatusCode())
	}
}

// =====================
// JSON Response Tests
// =====================

func TestReviewSQL_ResponseFormat(t *testing.T) {
	handler, sqlMock, _, _, _ := newTestHandler()
	sqlMock.reviewResult = &sqlgovernance.SQLReviewResult{
		ReviewID:    "review-abc",
		Approved:    true,
		RiskLevel:   sqlgovernance.RiskLevelLow,
		Warnings:    []string{"Consider adding LIMIT"},
		Violations:  []string{},
		Suggestions: []sqlgovernance.SQLSuggestion{{Type: "optimization", Message: "Add index"}},
	}

	body := `{"database_id":"db-1","sql":"SELECT * FROM users"}`
	ctx, reqCtx := createTestContext(body)

	handler.ReviewSQL(ctx, reqCtx)

	var resp sqlgovernance.SQLReviewResult
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.ReviewID != "review-abc" {
		t.Errorf("ReviewID = %s, want review-abc", resp.ReviewID)
	}
	if !resp.Approved {
		t.Error("Approved should be true")
	}
	if len(resp.Warnings) != 1 {
		t.Errorf("Warnings count = %d, want 1", len(resp.Warnings))
	}
}

func TestExecuteSQL_ResponseFormat(t *testing.T) {
	handler, sqlMock, _, _, _ := newTestHandler()
	sqlMock.executeResult = &sqlgovernance.SQLExecuteResult{
		ExecutionID:     "exec-xyz",
		Status:          "completed",
		RowsAffected:    42,
		ExecutionTimeMs: 150,
		Columns:         []sqlgovernance.Column{{Name: "id", Type: "int"}},
	}

	body := `{"database_id":"db-1","sql":"SELECT * FROM users"}`
	ctx, reqCtx := createTestContext(body)

	handler.ExecuteSQL(ctx, reqCtx)

	var resp sqlgovernance.SQLExecuteResult
	if err := json.Unmarshal(reqCtx.Response.Body(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.ExecutionID != "exec-xyz" {
		t.Errorf("ExecutionID = %s, want exec-xyz", resp.ExecutionID)
	}
	if resp.RowsAffected != 42 {
		t.Errorf("RowsAffected = %d, want 42", resp.RowsAffected)
	}
}

// =====================
// Benchmark Tests
// =====================

func BenchmarkReviewSQL(b *testing.B) {
	handler, _, _, _, _ := newTestHandler()
	body := `{"database_id":"db-1","sql":"SELECT * FROM users WHERE id = 1"}`
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reqCtx := &app.RequestContext{}
		reqCtx.Request.SetBody([]byte(body))
		handler.ReviewSQL(ctx, reqCtx)
	}
}

func BenchmarkExecuteSQL(b *testing.B) {
	handler, _, _, _, _ := newTestHandler()
	body := `{"database_id":"db-1","sql":"UPDATE users SET name='test'","timeout_seconds":30}`
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reqCtx := &app.RequestContext{}
		reqCtx.Request.SetBody([]byte(body))
		handler.ExecuteSQL(ctx, reqCtx)
	}
}

// =====================
// HTTP Handler Integration Tests
// =====================

func TestHandler_NewGatewayHandler(t *testing.T) {
	sqlMock := &mockSQLGovernanceService{}
	perfMock := &mockPerformanceService{}
	threshMock := &mockThresholdService{}
	llmMock := &mockLLMService{}

	handler := NewGatewayHandler(sqlMock, perfMock, threshMock, llmMock)

	if handler == nil {
		t.Fatal("NewGatewayHandler() returned nil")
	}
	if handler.sqlGovern != sqlMock {
		t.Error("SQL Governance service not set correctly")
	}
	if handler.performance != perfMock {
		t.Error("Performance service not set correctly")
	}
	if handler.threshold != threshMock {
		t.Error("Threshold service not set correctly")
	}
	if handler.llm != llmMock {
		t.Error("LLM service not set correctly")
	}
}

// =====================
// GraphQL Proxy Tests
// =====================

func TestGraphQLProxy_Path(t *testing.T) {
	tests := []struct {
		path       string
		shouldProxy bool
	}{
		{"/graphql", true},
		{"/graphql/playground", true},
		{"/graphql?query={metrics}", true},
		{"/api/v1/sql/review", false},
		{"/health", false},
		{"/unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// Check if path should be proxied
			isGraphQLPath := strings.HasPrefix(tt.path, "/graphql")
			if isGraphQLPath != tt.shouldProxy {
				t.Errorf("Path %s: isGraphQLPath = %v, want %v", tt.path, isGraphQLPath, tt.shouldProxy)
			}
		})
	}
}

func TestGraphQLProxy_MethodHandling(t *testing.T) {
	// Test that GraphQL endpoint should handle both GET and POST
	methods := []string{"GET", "POST"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Create a simple request context
			reqCtx := &app.RequestContext{}
			reqCtx.Request.SetMethod(method)
			reqCtx.Request.SetRequestURI("/graphql")

			// Verify method is set
			if string(reqCtx.Method()) != method {
				t.Errorf("Method = %s, want %s", string(reqCtx.Method()), method)
			}
		})
	}
}

// Test using httptest for full HTTP request simulation
func TestHandler_WithHTTPTestRequest(t *testing.T) {
	handler, _, _, _, _ := newTestHandler()

	// Create a test HTTP request
	req := httptest.NewRequest("POST", "/api/v1/sql/review", strings.NewReader(`{"database_id":"db-1","sql":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")

	// Create Hertz context
	reqCtx := &app.RequestContext{}
	reqCtx.Request.SetMethod("POST")
	reqCtx.Request.SetRequestURI("/api/v1/sql/review")
	reqCtx.Request.SetBody([]byte(`{"database_id":"db-1","sql":"SELECT 1"}`))
	reqCtx.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	ctx := context.Background()
	handler.ReviewSQL(ctx, reqCtx)

	// Verify response
	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Status = %d, want 200", reqCtx.Response.StatusCode())
	}
}