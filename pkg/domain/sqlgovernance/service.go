package sqlgovernance

// SQL Governance Domain Service
// Status: Interface defined, basic implementation exists
// Note: Repository is nil when used in Gateway, making this a stub.
// Full implementation requires database repository and Agent RPC client.

import (
	"context"
	"time"

	"github.com/db-cockpit/pkg/domain"
)

// RiskLevel represents the risk level of SQL
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// RuleType represents the type of governance rule
type RuleType string

const (
	RuleTypeBlock   RuleType = "block"
	RuleTypeWarn    RuleType = "warn"
	RuleTypeApprove RuleType = "approve"
)

// SQLReviewRequest represents a SQL review request
type SQLReviewRequest struct {
	DatabaseID string
	SQLText    string
	Context    map[string]string
}

// SQLReviewResult represents a SQL review result
type SQLReviewResult struct {
	ReviewID    string
	Approved    bool
	RiskLevel   RiskLevel
	Warnings    []string
	Violations  []string
	Suggestions []SQLSuggestion
	Analysis    *SQLAnalysis
}

// SQLSuggestion represents a SQL optimization suggestion
type SQLSuggestion struct {
	Type         string // "optimization", "security", "best_practice"
	Message      string
	SuggestedFix string
}

// SQLAnalysis represents SQL analysis results
type SQLAnalysis struct {
	OperationType        string
	AffectedTables       []string
	EstimatedRows        int64
	EstimatedCost        int64
	UsesIndex            bool
	IndexRecommendations []string
}

// SQLExecuteRequest represents a SQL execution request
type SQLExecuteRequest struct {
	DatabaseID      string
	SQLText         string
	TimeoutSeconds  int
	MaxRows         int
	DryRun          bool
	RequireApproval bool
	ApprovalID      string
}

// SQLExecuteResult represents a SQL execution result
type SQLExecuteResult struct {
	ExecutionID     string
	Status          string // "pending", "running", "completed", "failed", "cancelled"
	Columns         []Column
	Rows            []Row
	RowsAffected    int64
	ExecutionTimeMs int64
	ErrorMessage    string
	AuditID         string
}

// Column represents a result column
type Column struct {
	Name string
	Type string
}

// Row represents a result row
type Row struct {
	Values []string
}

// GovernanceRule represents a governance rule
type GovernanceRule struct {
	RuleID      string
	Name        string
	Description string
	RuleType    RuleType
	Pattern     string
	Severity    RiskLevel
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	AuditID         string
	UserID          string
	DatabaseID      string
	SQLText         string
	Status          string
	ExecutionTimeMs int64
	RowsAffected    int64
	Timestamp       time.Time
	RiskLevel       RiskLevel
	Violations      []string
}

// SQLGovernanceService defines the interface for SQL governance domain
type SQLGovernanceService interface {
	domain.DomainService

	// ReviewSQL reviews SQL before execution
	ReviewSQL(ctx *domain.DomainContext, req *SQLReviewRequest) (*SQLReviewResult, error)

	// ExecuteSQL executes SQL with governance controls
	ExecuteSQL(ctx *domain.DomainContext, req *SQLExecuteRequest) (*SQLExecuteResult, error)

	// ExplainSQL explains SQL execution plan
	ExplainSQL(ctx *domain.DomainContext, databaseID, sql string) (string, error)

	// GetAuditTrail retrieves audit trail
	GetAuditTrail(ctx *domain.DomainContext, startTime, endTime time.Time, limit int) ([]AuditEntry, error)

	// CreateRule creates a governance rule
	CreateRule(ctx *domain.DomainContext, rule *GovernanceRule) (string, error)

	// ListRules lists governance rules
	ListRules(ctx *domain.DomainContext, enabledOnly bool) ([]GovernanceRule, error)

	// ValidateSQL validates SQL against rules
	ValidateSQL(ctx *domain.DomainContext, databaseID, sql string) ([]string, error)
}

// ExecutionAgentClient defines the interface for calling execution agent
type ExecutionAgentClient interface {
	// ExecuteSQL executes SQL on a database
	ExecuteSQL(ctx context.Context, req *SQLExecuteRequest) (*SQLExecuteResult, error)

	// ExplainSQL explains SQL execution plan
	ExplainSQL(ctx context.Context, databaseID, sql string) (string, error)
}

// Repository defines the data access interface for SQL governance
type Repository interface {
	// SaveAuditEntry saves an audit entry
	SaveAuditEntry(ctx context.Context, entry *AuditEntry) error

	// GetAuditEntries retrieves audit entries
	GetAuditEntries(ctx context.Context, tenantID string, startTime, endTime time.Time, limit int) ([]AuditEntry, error)

	// SaveRule saves a governance rule
	SaveRule(ctx context.Context, rule *GovernanceRule) error

	// GetRules retrieves governance rules
	GetRules(ctx context.Context, tenantID string, enabledOnly bool) ([]GovernanceRule, error)

	// GetRuleByID retrieves a rule by ID
	GetRuleByID(ctx context.Context, ruleID string) (*GovernanceRule, error)
}
