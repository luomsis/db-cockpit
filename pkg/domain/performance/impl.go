package performance

import (
	"context"
	"time"

	"github.com/db-cockpit/pkg/common/utils"
	"github.com/db-cockpit/pkg/domain"
)

// Service implements the PerformanceService interface
type Service struct {
	repo      Repository
	threshold ThresholdClient
}

// NewService creates a new performance service
func NewService(repo Repository, threshold ThresholdClient) *Service {
	return &Service{
		repo:      repo,
		threshold: threshold,
	}
}

// Name returns the service name
func (s *Service) Name() string {
	return "performance"
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

// Diagnose runs performance diagnosis
func (s *Service) Diagnose(ctx *domain.DomainContext, req *DiagnosisRequest) (*DiagnosisResult, error) {
	result := &DiagnosisResult{
		DiagnosisID:     utils.GenerateID(),
		Status:          "running",
		HealthScore:     100,
		Issues:          []DiagnosisIssue{},
		Recommendations: []DiagnosisRecommendation{},
		GeneratedAt:     time.Now(),
	}

	// Set default time range
	if req.StartTime.IsZero() {
		req.StartTime = time.Now().Add(-1 * time.Hour)
	}
	if req.EndTime.IsZero() {
		req.EndTime = time.Now()
	}

	// Collect metrics
	metricNames := []string{
		"cpu_usage",
		"memory_usage",
		"disk_io",
		"connection_count",
		"query_latency",
	}

	metrics, err := s.repo.QueryMetrics(ctx.Context(), req.DatabaseID, metricNames, req.StartTime, req.EndTime, 60)
	if err != nil {
		return nil, err
	}

	// Analyze metrics and detect issues
	for _, series := range metrics {
		issues := s.analyzeMetrics(series, ctx.TenantID, req.DatabaseID)
		result.Issues = append(result.Issues, issues...)

		// Update health score based on issues
		for _, issue := range issues {
			switch issue.Severity {
			case SeverityCritical:
				result.HealthScore -= 20
			case SeverityWarning:
				result.HealthScore -= 5
			}
		}
	}

	// Get slow queries if scope includes queries
	if req.Scope == DiagnosisScopeFull || req.Scope == DiagnosisScopeQueries {
		slowQueries, err := s.repo.QuerySlowQueries(ctx.Context(), req.DatabaseID, req.StartTime, req.EndTime, 1000, 20)
		if err == nil {
			for _, sq := range slowQueries {
				issue := DiagnosisIssue{
					IssueID:     utils.GenerateID(),
					Severity:    SeverityWarning,
					Category:    "query",
					Title:       "Slow Query Detected",
					Description: sq.SQLText,
					Impact:      "Performance degradation",
					Metrics: map[string]float64{
						"duration_ms": sq.DurationMs,
					},
				}
				result.Issues = append(result.Issues, issue)

				// Add recommendation
				if len(sq.Suggestions) > 0 {
					rec := DiagnosisRecommendation{
						RecommendationID: utils.GenerateID(),
						Priority:         "medium",
						Title:            "Optimize Slow Query",
						Description:      sq.Suggestions[0],
						Action:           "Review and optimize query",
						ExpectedImpact:   "Reduced query latency",
					}
					result.Recommendations = append(result.Recommendations, rec)
				}
			}
		}
	}

	// Generate summary
	result.Summary = s.generateSummary(result.Issues, metrics)

	// Ensure health score is within bounds
	if result.HealthScore < 0 {
		result.HealthScore = 0
	}
	if result.HealthScore > 100 {
		result.HealthScore = 100
	}

	result.Status = "completed"

	// Save diagnosis result
	go func() {
		_ = s.repo.SaveDiagnosisResult(context.Background(), result)
	}()

	return result, nil
}

// GetMetrics retrieves performance metrics
func (s *Service) GetMetrics(ctx *domain.DomainContext, databaseID string, metricNames []string, startTime, endTime time.Time) ([]MetricSeries, error) {
	return s.repo.QueryMetrics(ctx.Context(), databaseID, metricNames, startTime, endTime, 60)
}

// GetSlowQueries retrieves slow queries
func (s *Service) GetSlowQueries(ctx *domain.DomainContext, databaseID string, startTime, endTime time.Time, minDurationMs float64, limit int) ([]SlowQuery, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.QuerySlowQueries(ctx.Context(), databaseID, startTime, endTime, minDurationMs, limit)
}

// AnalyzeQuery analyzes query performance
func (s *Service) AnalyzeQuery(ctx *domain.DomainContext, databaseID, sql string) (*QueryAnalysis, error) {
	// TODO: Implement query analysis
	// This would involve:
	// 1. Getting execution plan
	// 2. Analyzing query structure
	// 3. Generating recommendations

	return &QueryAnalysis{
		AnalysisID:    utils.GenerateID(),
		Optimizations: []string{},
	}, nil
}

// GetResourceUtilization retrieves resource utilization
func (s *Service) GetResourceUtilization(ctx *domain.DomainContext, databaseID string, startTime, endTime time.Time) (*ResourceUtilization, error) {
	// Query resource metrics
	metricNames := []string{"cpu_usage", "memory_usage", "disk_usage", "connection_count"}
	series, err := s.repo.QueryMetrics(ctx.Context(), databaseID, metricNames, startTime, endTime, 60)
	if err != nil {
		return nil, err
	}

	utilization := &ResourceUtilization{}

	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}

		switch s.Name {
		case "cpu_usage":
			utilization.CPU = CPUUtilization{
				CurrentPct: s.Points[len(s.Points)-1].Value,
				AvgPct:     s.Statistics.Avg,
				MaxPct:     s.Statistics.Max,
			}
		case "memory_usage":
			utilization.Memory = MemoryUtilization{
				CurrentPct: s.Points[len(s.Points)-1].Value,
			}
		}
	}

	return utilization, nil
}

// GenerateReport generates a performance report
func (s *Service) GenerateReport(ctx *domain.DomainContext, databaseID string, reportType string, startTime, endTime time.Time) (*Report, error) {
	// Run diagnosis for the report
	diagReq := &DiagnosisRequest{
		DatabaseID:   databaseID,
		Scope:        DiagnosisScopeFull,
		StartTime:    startTime,
		EndTime:      endTime,
		DeepAnalysis: true,
	}

	diagResult, err := s.Diagnose(ctx, diagReq)
	if err != nil {
		return nil, err
	}

	report := &Report{
		ReportID:    utils.GenerateID(),
		ReportType:  reportType,
		GeneratedAt: time.Now(),
		PeriodStart: startTime,
		PeriodEnd:   endTime,
		Summary: &ReportSummary{
			HealthScore:       diagResult.HealthScore,
			TotalIssues:       len(diagResult.Issues),
			AvgResponseTimeMs: diagResult.Summary.AvgQueryTimeMs,
		},
	}

	// Generate sections
	report.Sections = []ReportSection{
		{
			Title:   "Overview",
			Content: "Performance report overview",
		},
		{
			Title:   "Issues",
			Content: "Detected issues and recommendations",
		},
	}

	return report, nil
}

// analyzeMetrics analyzes metrics and returns issues
func (s *Service) analyzeMetrics(series MetricSeries, tenantID, databaseID string) []DiagnosisIssue {
	issues := []DiagnosisIssue{}

	for _, point := range series.Points {
		breached, severity, err := s.threshold.CheckThreshold(context.Background(), tenantID, databaseID, series.Name, point.Value)
		if err != nil || !breached {
			continue
		}

		issue := DiagnosisIssue{
			IssueID:     utils.GenerateID(),
			Severity:    Severity(severity),
			Category:    "resource",
			Title:       series.Name + " threshold breach",
			Description: series.Name + " exceeded threshold",
			Impact:      "Potential performance degradation",
			Metrics: map[string]float64{
				series.Name: point.Value,
			},
		}
		issues = append(issues, issue)
	}

	return issues
}

// generateSummary generates diagnosis summary
func (s *Service) generateSummary(issues []DiagnosisIssue, metrics []MetricSeries) *DiagnosisSummary {
	summary := &DiagnosisSummary{
		TotalIssues: len(issues),
	}

	for _, issue := range issues {
		switch issue.Severity {
		case SeverityCritical:
			summary.CriticalIssues++
		case SeverityWarning:
			summary.WarningIssues++
		case SeverityInfo:
			summary.InfoIssues++
		}
	}

	// Calculate averages from metrics
	for _, series := range metrics {
		if series.Name == "query_latency" {
			summary.AvgQueryTimeMs = series.Statistics.Avg
			summary.P99QueryTimeMs = series.Statistics.P99
		}
		if series.Name == "cpu_usage" || series.Name == "memory_usage" {
			summary.ResourceUtilizationPct = series.Statistics.Avg
		}
	}

	return summary
}
