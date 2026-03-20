package sqlgovernance

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/db-cockpit/pkg/common/utils"
	"github.com/db-cockpit/pkg/domain"
)

// Service implements the SQLGovernanceService interface
type Service struct {
	repo       Repository
	agent      ExecutionAgentClient
	rulesCache []GovernanceRule
}

// NewService creates a new SQL governance service
func NewService(repo Repository, agent ExecutionAgentClient) *Service {
	return &Service{
		repo:  repo,
		agent: agent,
	}
}

// Name returns the service name
func (s *Service) Name() string {
	return "sqlgovernance"
}

// Initialize initializes the service
func (s *Service) Initialize(ctx context.Context) error {
	// Load rules into cache
	return s.reloadRules(ctx)
}

// Shutdown shuts down the service
func (s *Service) Shutdown(ctx context.Context) error {
	return nil
}

// Health returns the health status
func (s *Service) Health(ctx context.Context) error {
	return nil
}

// reloadRules reloads governance rules
func (s *Service) reloadRules(ctx context.Context) error {
	rules, err := s.repo.GetRules(ctx, "", false)
	if err != nil {
		return err
	}
	s.rulesCache = rules
	return nil
}

// ReviewSQL reviews SQL before execution
func (s *Service) ReviewSQL(ctx *domain.DomainContext, req *SQLReviewRequest) (*SQLReviewResult, error) {
	result := &SQLReviewResult{
		ReviewID:    utils.GenerateID(),
		Approved:    true,
		RiskLevel:   RiskLevelLow,
		Warnings:    []string{},
		Violations:  []string{},
		Suggestions: []SQLSuggestion{},
	}

	// Analyze SQL
	analysis := s.analyzeSQL(req.SQLText)
	result.Analysis = analysis

	// Check against governance rules
	for _, rule := range s.rulesCache {
		if !rule.Enabled {
			continue
		}

		matched, err := regexp.MatchString(rule.Pattern, req.SQLText)
		if err != nil {
			continue
		}

		if matched {
			switch rule.RuleType {
			case RuleTypeBlock:
				result.Approved = false
				result.Violations = append(result.Violations, rule.Description)
				if rule.Severity == RiskLevelCritical || rule.Severity == RiskLevelHigh {
					result.RiskLevel = rule.Severity
				}
			case RuleTypeWarn:
				result.Warnings = append(result.Warnings, rule.Description)
				if result.RiskLevel == RiskLevelLow {
					result.RiskLevel = RiskLevelMedium
				}
			case RuleTypeApprove:
				// Requires approval workflow
				result.Warnings = append(result.Warnings, "Requires approval: "+rule.Description)
			}
		}
	}

	// Add optimization suggestions
	suggestions := s.generateSuggestions(req.SQLText, analysis)
	result.Suggestions = suggestions

	return result, nil
}

// ExecuteSQL executes SQL with governance controls
func (s *Service) ExecuteSQL(ctx *domain.DomainContext, req *SQLExecuteRequest) (*SQLExecuteResult, error) {
	// Set defaults
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 30
	}
	if req.MaxRows <= 0 {
		req.MaxRows = 1000
	}

	// Execute via agent
	result, err := s.agent.ExecuteSQL(ctx.Context(), req)
	if err != nil {
		return nil, err
	}

	// Create audit entry
	auditEntry := &AuditEntry{
		AuditID:         utils.GenerateID(),
		UserID:          ctx.UserID,
		DatabaseID:      req.DatabaseID,
		SQLText:         req.SQLText,
		Status:          result.Status,
		ExecutionTimeMs: result.ExecutionTimeMs,
		RowsAffected:    result.RowsAffected,
		Timestamp:       time.Now(),
	}

	// Save audit entry
	if err := s.repo.SaveAuditEntry(ctx.Context(), auditEntry); err != nil {
		// Log error but don't fail the execution
	}

	result.AuditID = auditEntry.AuditID
	return result, nil
}

// ExplainSQL explains SQL execution plan
func (s *Service) ExplainSQL(ctx *domain.DomainContext, databaseID, sql string) (string, error) {
	return s.agent.ExplainSQL(ctx.Context(), databaseID, sql)
}

// GetAuditTrail retrieves audit trail
func (s *Service) GetAuditTrail(ctx *domain.DomainContext, startTime, endTime time.Time, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.GetAuditEntries(ctx.Context(), ctx.TenantID, startTime, endTime, limit)
}

// CreateRule creates a governance rule
func (s *Service) CreateRule(ctx *domain.DomainContext, rule *GovernanceRule) (string, error) {
	rule.RuleID = utils.GenerateID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	if err := s.repo.SaveRule(ctx.Context(), rule); err != nil {
		return "", err
	}

	// Reload rules cache
	go func() {
		_ = s.reloadRules(context.Background())
	}()

	return rule.RuleID, nil
}

// ListRules lists governance rules
func (s *Service) ListRules(ctx *domain.DomainContext, enabledOnly bool) ([]GovernanceRule, error) {
	return s.repo.GetRules(ctx.Context(), ctx.TenantID, enabledOnly)
}

// ValidateSQL validates SQL against rules
func (s *Service) ValidateSQL(ctx *domain.DomainContext, databaseID, sql string) ([]string, error) {
	violations := []string{}

	for _, rule := range s.rulesCache {
		if !rule.Enabled || rule.RuleType != RuleTypeBlock {
			continue
		}

		matched, err := regexp.MatchString(rule.Pattern, sql)
		if err != nil {
			continue
		}

		if matched {
			violations = append(violations, rule.Description)
		}
	}

	return violations, nil
}

// analyzeSQL analyzes SQL and returns analysis
func (s *Service) analyzeSQL(sql string) *SQLAnalysis {
	analysis := &SQLAnalysis{
		AffectedTables: []string{},
	}

	// Detect operation type
	upperSQL := strings.ToUpper(strings.TrimSpace(sql))
	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		analysis.OperationType = "SELECT"
	case strings.HasPrefix(upperSQL, "INSERT"):
		analysis.OperationType = "INSERT"
	case strings.HasPrefix(upperSQL, "UPDATE"):
		analysis.OperationType = "UPDATE"
	case strings.HasPrefix(upperSQL, "DELETE"):
		analysis.OperationType = "DELETE"
	default:
		analysis.OperationType = "DDL"
	}

	// Extract table names (simplified)
	// TODO: Implement proper SQL parsing
	tablePatterns := []string{
		`(?i)FROM\s+(\w+)`,
		`(?i)JOIN\s+(\w+)`,
		`(?i)INTO\s+(\w+)`,
		`(?i)UPDATE\s+(\w+)`,
	}

	for _, pattern := range tablePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(sql, -1)
		for _, match := range matches {
			if len(match) > 1 {
				analysis.AffectedTables = append(analysis.AffectedTables, match[1])
			}
		}
	}

	return analysis
}

// generateSuggestions generates optimization suggestions
func (s *Service) generateSuggestions(sql string, analysis *SQLAnalysis) []SQLSuggestion {
	suggestions := []SQLSuggestion{}

	// Check for SELECT *
	if strings.Contains(strings.ToUpper(sql), "SELECT *") {
		suggestions = append(suggestions, SQLSuggestion{
			Type:         "best_practice",
			Message:      "Consider specifying columns instead of SELECT *",
			SuggestedFix: "List only required columns",
		})
	}

	// Check for missing WHERE clause in UPDATE/DELETE
	upperSQL := strings.ToUpper(sql)
	if (strings.HasPrefix(upperSQL, "UPDATE") || strings.HasPrefix(upperSQL, "DELETE")) &&
		!strings.Contains(upperSQL, "WHERE") {
		suggestions = append(suggestions, SQLSuggestion{
			Type:         "security",
			Message:      "UPDATE/DELETE without WHERE clause affects all rows",
			SuggestedFix: "Add WHERE clause to limit affected rows",
		})
	}

	return suggestions
}
