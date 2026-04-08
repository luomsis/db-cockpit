package performance

// Performance Diagnosis Domain Service
// Status: Interface defined, basic implementation exists
// Note: Repository is nil when used in Gateway, making this a stub.
// Full implementation requires database repository and Threshold client.

import (
	"context"
	"time"

	"github.com/db-cockpit/pkg/domain"
)

// DiagnosisScope represents the scope of diagnosis
type DiagnosisScope string

const (
	DiagnosisScopeFull        DiagnosisScope = "full"
	DiagnosisScopeQueries     DiagnosisScope = "queries"
	DiagnosisScopeResources   DiagnosisScope = "resources"
	DiagnosisScopeConnections DiagnosisScope = "connections"
	DiagnosisScopeStorage     DiagnosisScope = "storage"
)

// Severity represents issue severity
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// DiagnosisRequest represents a diagnosis request
type DiagnosisRequest struct {
	DatabaseID   string
	Scope        DiagnosisScope
	StartTime    time.Time
	EndTime      time.Time
	DeepAnalysis bool
}

// DiagnosisResult represents a diagnosis result
type DiagnosisResult struct {
	DiagnosisID     string
	Status          string
	HealthScore     float64 // 0-100
	Issues          []DiagnosisIssue
	Recommendations []DiagnosisRecommendation
	Summary         *DiagnosisSummary
	GeneratedAt     time.Time
}

// DiagnosisIssue represents a diagnosed issue
type DiagnosisIssue struct {
	IssueID            string
	Severity           Severity
	Category           string // "query", "resource", "connection", "storage"
	Title              string
	Description        string
	Impact             string
	AffectedComponents []string
	Metrics            map[string]float64
}

// DiagnosisRecommendation represents a recommendation
type DiagnosisRecommendation struct {
	RecommendationID string
	Priority         string // "low", "medium", "high"
	Title            string
	Description      string
	Action           string
	ExpectedImpact   string
	RelatedIssues    []string
}

// DiagnosisSummary represents a diagnosis summary
type DiagnosisSummary struct {
	TotalIssues            int
	CriticalIssues         int
	WarningIssues          int
	InfoIssues             int
	AvgQueryTimeMs         float64
	P99QueryTimeMs         float64
	ResourceUtilizationPct float64
}

// MetricSeries represents a series of metric data
type MetricSeries struct {
	Name       string
	Unit       string
	Points     []MetricPoint
	Statistics MetricStatistics
}

// MetricPoint represents a single metric point
type MetricPoint struct {
	Timestamp time.Time
	Value     float64
	Tags      map[string]string
}

// MetricStatistics represents metric statistics
type MetricStatistics struct {
	Min   float64
	Max   float64
	Avg   float64
	Sum   float64
	Count int64
	P50   float64
	P95   float64
	P99   float64
}

// SlowQuery represents a slow query
type SlowQuery struct {
	QueryID       string
	SQLText       string
	DurationMs    float64
	RowsExamined  int64
	RowsSent      int64
	User          string
	Database      string
	Timestamp     time.Time
	ExecutionPlan string
	Suggestions   []string
}

// ResourceUtilization represents resource utilization
type ResourceUtilization struct {
	CPU            CPUUtilization
	Memory         MemoryUtilization
	Storage        StorageUtilization
	ConnectionPool ConnectionPool
	IO             IOStatistics
}

// CPUUtilization represents CPU utilization
type CPUUtilization struct {
	CurrentPct float64
	AvgPct     float64
	MaxPct     float64
}

// MemoryUtilization represents memory utilization
type MemoryUtilization struct {
	CurrentPct        float64
	UsedBytes         int64
	TotalBytes        int64
	BufferPoolHitRate float64
}

// StorageUtilization represents storage utilization
type StorageUtilization struct {
	CurrentPct float64
	UsedBytes  int64
	TotalBytes int64
	FreeBytes  int64
}

// ConnectionPool represents connection pool status
type ConnectionPool struct {
	ActiveConnections int
	IdleConnections   int
	MaxConnections    int
	UtilizationPct    float64
}

// IOStatistics represents I/O statistics
type IOStatistics struct {
	ReadIOPS            float64
	WriteIOPS           float64
	ReadThroughputMBps  float64
	WriteThroughputMBps float64
	AvgLatencyMs        float64
}

// PerformanceService defines the interface for performance diagnosis domain
type PerformanceService interface {
	domain.DomainService

	// Diagnose runs performance diagnosis
	Diagnose(ctx *domain.DomainContext, req *DiagnosisRequest) (*DiagnosisResult, error)

	// GetMetrics retrieves performance metrics
	GetMetrics(ctx *domain.DomainContext, databaseID string, metricNames []string, startTime, endTime time.Time) ([]MetricSeries, error)

	// GetSlowQueries retrieves slow queries
	GetSlowQueries(ctx *domain.DomainContext, databaseID string, startTime, endTime time.Time, minDurationMs float64, limit int) ([]SlowQuery, error)

	// AnalyzeQuery analyzes query performance
	AnalyzeQuery(ctx *domain.DomainContext, databaseID, sql string) (*QueryAnalysis, error)

	// GetResourceUtilization retrieves resource utilization
	GetResourceUtilization(ctx *domain.DomainContext, databaseID string, startTime, endTime time.Time) (*ResourceUtilization, error)

	// GenerateReport generates a performance report
	GenerateReport(ctx *domain.DomainContext, databaseID string, reportType string, startTime, endTime time.Time) (*Report, error)
}

// QueryAnalysis represents query analysis result
type QueryAnalysis struct {
	AnalysisID           string
	Complexity           QueryComplexity
	CostEstimate         QueryCostEstimate
	Optimizations        []string
	IndexRecommendations []IndexRecommendation
	ExecutionPlan        string
}

// QueryComplexity represents query complexity
type QueryComplexity struct {
	TableCount     int
	JoinCount      int
	SubqueryCount  int
	HasAggregation bool
	HasDistinct    bool
	HasOrderBy     bool
	HasGroupBy     bool
}

// QueryCostEstimate represents query cost estimate
type QueryCostEstimate struct {
	EstimatedRows    float64
	EstimatedTimeMs  float64
	MemoryUsageBytes float64
	ComplexityScore  string
}

// IndexRecommendation represents an index recommendation
type IndexRecommendation struct {
	TableName               string
	Columns                 []string
	Reason                  string
	EstimatedImprovementPct float64
}

// Report represents a performance report
type Report struct {
	ReportID    string
	ReportType  string
	GeneratedAt time.Time
	PeriodStart time.Time
	PeriodEnd   time.Time
	Summary     *ReportSummary
	Sections    []ReportSection
}

// ReportSummary represents report summary
type ReportSummary struct {
	HealthScore       float64
	TotalIssues       int
	ResolvedIssues    int
	AvgResponseTimeMs float64
	UptimePct         float64
}

// ReportSection represents a report section
type ReportSection struct {
	Title   string
	Content string
	Charts  []ChartData
}

// ChartData represents chart data
type ChartData struct {
	Title  string
	Type   string // "line", "bar", "pie"
	Labels []string
	Values []float64
}

// ThresholdClient defines the interface for threshold service
type ThresholdClient interface {
	// CheckThreshold checks if a value breaches threshold
	CheckThreshold(ctx context.Context, tenantID, databaseID, metricName string, value float64) (bool, string, error)

	// GetThresholds retrieves thresholds
	GetThresholds(ctx context.Context, tenantID, databaseID string, metricNames []string) (map[string]float64, error)
}

// Repository defines the data access interface for performance domain
type Repository interface {
	// QueryMetrics queries metrics from TSDB
	QueryMetrics(ctx context.Context, databaseID string, metricNames []string, startTime, endTime time.Time, resolution int) ([]MetricSeries, error)

	// QuerySlowQueries queries slow queries
	QuerySlowQueries(ctx context.Context, databaseID string, startTime, endTime time.Time, minDurationMs float64, limit int) ([]SlowQuery, error)

	// SaveDiagnosisResult saves diagnosis result
	SaveDiagnosisResult(ctx context.Context, result *DiagnosisResult) error

	// GetDiagnosisHistory retrieves diagnosis history
	GetDiagnosisHistory(ctx context.Context, databaseID string, limit int) ([]DiagnosisResult, error)
}
