package llm

// LLM Orchestrator Domain Service
// Status: Interface defined, basic implementation exists
// Note: Repository and Provider are nil when used in Gateway, making this a stub.
// Full implementation requires LLM provider client and vector database.

import (
	"context"
	"time"

	"github.com/db-cockpit/pkg/domain"
)

// ChatRole represents the role in a chat conversation
type ChatRole string

const (
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
	ChatRoleSystem    ChatRole = "system"
)

// AnalysisDepth represents the depth of analysis
type AnalysisDepth string

const (
	AnalysisDepthQuick    AnalysisDepth = "quick"
	AnalysisDepthStandard AnalysisDepth = "standard"
	AnalysisDepthDeep     AnalysisDepth = "deep"
)

// OptimizationGoal represents the goal of SQL optimization
type OptimizationGoal string

const (
	GoalPerformance   OptimizationGoal = "performance"
	GoalReadability   OptimizationGoal = "readability"
	GoalResourceUsage OptimizationGoal = "resource_usage"
)

// RecommendationCategory represents recommendation category
type RecommendationCategory string

const (
	CategoryPerformance RecommendationCategory = "performance"
	CategorySecurity    RecommendationCategory = "security"
	CategoryCost        RecommendationCategory = "cost"
	CategoryReliability RecommendationCategory = "reliability"
)

// ChatRequest represents a chat request
type ChatRequest struct {
	SessionID string
	Message   string
	Context   *ChatContext
	Options   *ChatOptions
}

// ChatContext represents chat context
type ChatContext struct {
	History    []ChatMessage
	DatabaseID string
	SchemaInfo []string
	Metadata   map[string]string
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role      ChatRole
	Content   string
	Timestamp time.Time
}

// ChatOptions represents chat options
type ChatOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
	Stream      bool
}

// ChatResponse represents a chat response
type ChatResponse struct {
	ResponseID string
	SessionID  string
	Message    string
	Sources    []string
	Actions    []SuggestedAction
	Timestamp  time.Time
	TokenUsage TokenUsage
}

// SuggestedAction represents a suggested action
type SuggestedAction struct {
	ActionID    string
	Type        string // "execute_sql", "view_metrics", "apply_recommendation"
	Label       string
	Description string
	Parameters  map[string]string
}

// TokenUsage represents token usage
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// IssueContext represents issue context for analysis
type IssueContext struct {
	IssueType   string // "slow_query", "high_cpu", "deadlock", "error"
	Description string
	Metrics     []MetricSnapshot
	Logs        []LogEntry
	Context     map[string]string
}

// MetricSnapshot represents a metric snapshot
type MetricSnapshot struct {
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Fields    map[string]string
}

// IssueAnalysis represents issue analysis result
type IssueAnalysis struct {
	AnalysisID      string
	Summary         *IssueSummary
	RootCauses      []RootCause
	Solutions       []Solution
	RelatedIssues   []string
	ConfidenceScore float64
}

// IssueSummary represents issue summary
type IssueSummary struct {
	Title              string
	Description        string
	Severity           string
	Category           string
	AffectedComponents []string
}

// RootCause represents a root cause
type RootCause struct {
	CauseID             string
	Description         string
	Probability         float64
	Evidence            []string
	ContributingFactors []string
}

// Solution represents a solution
type Solution struct {
	SolutionID      string
	Title           string
	Description     string
	Priority        string
	Steps           []string
	ExpectedOutcome string
	EstimatedEffort float64
	Prerequisites   []string
}

// SQLGenerationRequest represents SQL generation request
type SQLGenerationRequest struct {
	NaturalLanguage string
	SchemaContext   string
	Options         *SQLGenerationOptions
}

// SQLGenerationOptions represents SQL generation options
type SQLGenerationOptions struct {
	Dialect            string // "postgres", "mysql", "clickhouse"
	IncludeExplanation bool
	ValidateSyntax     bool
	Optimize           bool
}

// SQLGenerationResult represents SQL generation result
type SQLGenerationResult struct {
	GenerationID string
	SQL          string
	Explanation  string
	Warnings     []string
	TablesUsed   []string
	Confidence   float64
}

// SQLOptimizationResult represents SQL optimization result
type SQLOptimizationResult struct {
	OptimizationID      string
	OptimizedSQL        string
	Changes             []OptimizationChange
	PerformanceEstimate PerformanceEstimate
	IndexSuggestions    []IndexSuggestion
}

// OptimizationChange represents an optimization change
type OptimizationChange struct {
	Type        string
	Description string
	Before      string
	After       string
}

// PerformanceEstimate represents performance estimate
type PerformanceEstimate struct {
	EstimatedImprovementPct       float64
	EstimatedTimeSavedMs          float64
	EstimatedResourceReductionPct float64
}

// IndexSuggestion represents an index suggestion
type IndexSuggestion struct {
	TableName           string
	Columns             []string
	Reason              string
	EstimatedBenefitPct float64
}

// Recommendation represents a recommendation
type Recommendation struct {
	RecommendationID string
	Title            string
	Description      string
	Category         RecommendationCategory
	Priority         string
	ImpactScore      float64
	EffortScore      float64
	Actions          []string
	DocumentationURL string
}

// LLMOrchestratorService defines the interface for LLM orchestrator domain
type LLMOrchestratorService interface {
	domain.DomainService

	// Chat handles chat interactions
	Chat(ctx *domain.DomainContext, req *ChatRequest) (*ChatResponse, error)

	// StreamChat handles streaming chat
	StreamChat(ctx *domain.DomainContext, req *ChatRequest, callback func(chunk string) error) error

	// AnalyzeIssue analyzes a database issue
	AnalyzeIssue(ctx *domain.DomainContext, databaseID string, issue *IssueContext, depth AnalysisDepth) (*IssueAnalysis, error)

	// GenerateSQL generates SQL from natural language
	GenerateSQL(ctx *domain.DomainContext, databaseID string, req *SQLGenerationRequest) (*SQLGenerationResult, error)

	// ExplainSQL explains a SQL query
	ExplainSQL(ctx *domain.DomainContext, databaseID, sql string) (string, error)

	// OptimizeSQL optimizes a SQL query
	OptimizeSQL(ctx *domain.DomainContext, databaseID, sql string, goal OptimizationGoal) (*SQLOptimizationResult, error)

	// GetRecommendations gets intelligent recommendations
	GetRecommendations(ctx *domain.DomainContext, databaseID string, category RecommendationCategory, limit int) ([]Recommendation, error)

	// CreateEmbedding creates an embedding for text
	CreateEmbedding(ctx *domain.DomainContext, text string) ([]float32, error)

	// SemanticSearch performs semantic search
	SemanticSearch(ctx *domain.DomainContext, query string, collections []string, topK int) ([]SemanticSearchResult, error)
}

// SemanticSearchResult represents a semantic search result
type SemanticSearchResult struct {
	ID         string
	Content    string
	Similarity float32
	Metadata   map[string]string
	Source     string
}

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	// Generate generates text from prompt
	Generate(ctx context.Context, prompt string, options *ChatOptions) (string, error)

	// GenerateStream generates text stream
	GenerateStream(ctx context.Context, prompt string, options *ChatOptions, callback func(chunk string) error) error

	// CreateEmbedding creates an embedding
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// Repository defines the data access interface for LLM domain
type Repository interface {
	// SaveChatSession saves a chat session
	SaveChatSession(ctx context.Context, sessionID, tenantID, userID string, messages []ChatMessage) error

	// GetChatSession retrieves a chat session
	GetChatSession(ctx context.Context, sessionID string) ([]ChatMessage, error)

	// SaveEmbedding saves an embedding
	SaveEmbedding(ctx context.Context, collection, id string, embedding []float32, metadata map[string]string) error

	// SearchEmbeddings searches similar embeddings
	SearchEmbeddings(ctx context.Context, collection string, embedding []float32, topK int) ([]SemanticSearchResult, error)

	// SaveRecommendation saves a recommendation
	SaveRecommendation(ctx context.Context, tenantID, databaseID string, rec *Recommendation) error

	// GetRecommendations retrieves recommendations
	GetRecommendations(ctx context.Context, tenantID, databaseID string, category RecommendationCategory, limit int) ([]Recommendation, error)
}
