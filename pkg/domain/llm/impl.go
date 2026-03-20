package llm

import (
	"context"
	"strings"
	"time"

	"github.com/db-cockpit/pkg/common/utils"
	"github.com/db-cockpit/pkg/domain"
)

// Service implements the LLMOrchestratorService interface
type Service struct {
	repo     Repository
	provider LLMProvider
}

// NewService creates a new LLM orchestrator service
func NewService(repo Repository, provider LLMProvider) *Service {
	return &Service{
		repo:     repo,
		provider: provider,
	}
}

// Name returns the service name
func (s *Service) Name() string {
	return "llm"
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

// Chat handles chat interactions
func (s *Service) Chat(ctx *domain.DomainContext, req *ChatRequest) (*ChatResponse, error) {
	// Generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = utils.GenerateID()
	}

	// Get chat history
	history := []ChatMessage{}
	if req.Context != nil {
		history = req.Context.History
	}

	// Build prompt with context
	prompt := s.buildPrompt(req.Message, history, req.Context)

	// Generate response
	options := req.Options
	if options == nil {
		options = &ChatOptions{
			Model:       "default",
			Temperature: 0.7,
			MaxTokens:   2048,
		}
	}

	response, err := s.provider.Generate(ctx.Context(), prompt, options)
	if err != nil {
		return nil, err
	}

	// Extract suggested actions
	actions := s.extractActions(response)

	// Create response
	result := &ChatResponse{
		ResponseID: utils.GenerateID(),
		SessionID:  req.SessionID,
		Message:    response,
		Sources:    []string{},
		Actions:    actions,
		Timestamp:  time.Now(),
		TokenUsage: TokenUsage{},
	}

	// Save chat session
	userMessage := ChatMessage{
		Role:      ChatRoleUser,
		Content:   req.Message,
		Timestamp: time.Now(),
	}
	assistantMessage := ChatMessage{
		Role:      ChatRoleAssistant,
		Content:   response,
		Timestamp: time.Now(),
	}
	history = append(history, userMessage, assistantMessage)
	_ = s.repo.SaveChatSession(ctx.Context(), req.SessionID, ctx.TenantID, ctx.UserID, history)

	return result, nil
}

// StreamChat handles streaming chat
func (s *Service) StreamChat(ctx *domain.DomainContext, req *ChatRequest, callback func(chunk string) error) error {
	prompt := s.buildPrompt(req.Message, nil, req.Context)

	options := req.Options
	if options == nil {
		options = &ChatOptions{
			Model:       "default",
			Temperature: 0.7,
			MaxTokens:   2048,
		}
	}

	return s.provider.GenerateStream(ctx.Context(), prompt, options, callback)
}

// AnalyzeIssue analyzes a database issue
func (s *Service) AnalyzeIssue(ctx *domain.DomainContext, databaseID string, issue *IssueContext, depth AnalysisDepth) (*IssueAnalysis, error) {
	// Build analysis prompt
	prompt := s.buildAnalysisPrompt(issue, depth)

	options := &ChatOptions{
		Model:       "default",
		Temperature: 0.3,
		MaxTokens:   4096,
	}

	response, err := s.provider.Generate(ctx.Context(), prompt, options)
	if err != nil {
		return nil, err
	}

	// Parse response into structured analysis
	analysis := s.parseAnalysisResponse(response)
	analysis.AnalysisID = utils.GenerateID()

	return analysis, nil
}

// GenerateSQL generates SQL from natural language
func (s *Service) GenerateSQL(ctx *domain.DomainContext, databaseID string, req *SQLGenerationRequest) (*SQLGenerationResult, error) {
	// Build SQL generation prompt
	prompt := s.buildSQLGenerationPrompt(req)

	options := &ChatOptions{
		Model:       "default",
		Temperature: 0.2,
		MaxTokens:   2048,
	}

	response, err := s.provider.Generate(ctx.Context(), prompt, options)
	if err != nil {
		return nil, err
	}

	// Extract SQL from response
	sql, explanation := s.extractSQLAndExplanation(response)

	result := &SQLGenerationResult{
		GenerationID: utils.GenerateID(),
		SQL:          sql,
		Explanation:  explanation,
		Confidence:   0.85,
	}

	return result, nil
}

// ExplainSQL explains a SQL query
func (s *Service) ExplainSQL(ctx *domain.DomainContext, databaseID, sql string) (string, error) {
	prompt := "Explain the following SQL query in simple terms:\n\n" + sql

	options := &ChatOptions{
		Model:       "default",
		Temperature: 0.3,
		MaxTokens:   2048,
	}

	return s.provider.Generate(ctx.Context(), prompt, options)
}

// OptimizeSQL optimizes a SQL query
func (s *Service) OptimizeSQL(ctx *domain.DomainContext, databaseID, sql string, goal OptimizationGoal) (*SQLOptimizationResult, error) {
	prompt := s.buildOptimizationPrompt(sql, goal)

	options := &ChatOptions{
		Model:       "default",
		Temperature: 0.2,
		MaxTokens:   2048,
	}

	response, err := s.provider.Generate(ctx.Context(), prompt, options)
	if err != nil {
		return nil, err
	}

	result := &SQLOptimizationResult{
		OptimizationID:   utils.GenerateID(),
		OptimizedSQL:     s.extractSQL(response),
		Changes:          []OptimizationChange{},
		IndexSuggestions: []IndexSuggestion{},
	}

	return result, nil
}

// GetRecommendations gets intelligent recommendations
func (s *Service) GetRecommendations(ctx *domain.DomainContext, databaseID string, category RecommendationCategory, limit int) ([]Recommendation, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get from repository first
	recs, err := s.repo.GetRecommendations(ctx.Context(), ctx.TenantID, databaseID, category, limit)
	if err == nil && len(recs) > 0 {
		return recs, nil
	}

	// Generate new recommendations
	prompt := s.buildRecommendationPrompt(databaseID, category)

	options := &ChatOptions{
		Model:       "default",
		Temperature: 0.5,
		MaxTokens:   2048,
	}

	response, err := s.provider.Generate(ctx.Context(), prompt, options)
	if err != nil {
		return nil, err
	}

	// Parse recommendations
	recommendations := s.parseRecommendations(response, category)

	// Save recommendations
	for i := range recommendations {
		recommendations[i].RecommendationID = utils.GenerateID()
		_ = s.repo.SaveRecommendation(ctx.Context(), ctx.TenantID, databaseID, &recommendations[i])
	}

	return recommendations, nil
}

// CreateEmbedding creates an embedding for text
func (s *Service) CreateEmbedding(ctx *domain.DomainContext, text string) ([]float32, error) {
	return s.provider.CreateEmbedding(ctx.Context(), text)
}

// SemanticSearch performs semantic search
func (s *Service) SemanticSearch(ctx *domain.DomainContext, query string, collections []string, topK int) ([]SemanticSearchResult, error) {
	if topK <= 0 {
		topK = 10
	}

	// Create embedding for query
	embedding, err := s.provider.CreateEmbedding(ctx.Context(), query)
	if err != nil {
		return nil, err
	}

	// Search in each collection
	results := []SemanticSearchResult{}
	for _, collection := range collections {
		collectionResults, err := s.repo.SearchEmbeddings(ctx.Context(), collection, embedding, topK)
		if err != nil {
			continue
		}
		results = append(results, collectionResults...)
	}

	// Sort by similarity and return top K
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Helper methods

func (s *Service) buildPrompt(message string, history []ChatMessage, ctx *ChatContext) string {
	var sb strings.Builder

	sb.WriteString("You are a database assistant helping users manage and optimize their databases.\n\n")

	if ctx != nil && len(ctx.SchemaInfo) > 0 {
		sb.WriteString("Database Schema:\n")
		for _, schema := range ctx.SchemaInfo {
			sb.WriteString(schema + "\n")
		}
		sb.WriteString("\n")
	}

	if len(history) > 0 {
		sb.WriteString("Conversation History:\n")
		for _, msg := range history {
			sb.WriteString(string(msg.Role) + ": " + msg.Content + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("User: " + message)

	return sb.String()
}

func (s *Service) buildAnalysisPrompt(issue *IssueContext, depth AnalysisDepth) string {
	var sb strings.Builder

	sb.WriteString("Analyze the following database issue and provide root causes and solutions.\n\n")
	sb.WriteString("Issue Type: " + issue.IssueType + "\n")
	sb.WriteString("Description: " + issue.Description + "\n\n")

	if len(issue.Metrics) > 0 {
		sb.WriteString("Related Metrics:\n")
		for _, m := range issue.Metrics {
			sb.WriteString("- " + m.Name + ": " + string(rune(int(m.Value))) + " " + m.Unit + "\n")
		}
	}

	if len(issue.Logs) > 0 {
		sb.WriteString("\nRelevant Logs:\n")
		for _, log := range issue.Logs {
			sb.WriteString("[" + log.Level + "] " + log.Message + "\n")
		}
	}

	return sb.String()
}

func (s *Service) buildSQLGenerationPrompt(req *SQLGenerationRequest) string {
	var sb strings.Builder

	sb.WriteString("Generate a SQL query for the following request:\n\n")
	sb.WriteString(req.NaturalLanguage + "\n\n")

	if req.SchemaContext != "" {
		sb.WriteString("Schema Context:\n" + req.SchemaContext + "\n\n")
	}

	if req.Options != nil {
		sb.WriteString("Dialect: " + req.Options.Dialect + "\n")
	}

	return sb.String()
}

func (s *Service) buildOptimizationPrompt(sql string, goal OptimizationGoal) string {
	return "Optimize the following SQL query for " + string(goal) + ":\n\n" + sql
}

func (s *Service) buildRecommendationPrompt(databaseID string, category RecommendationCategory) string {
	return "Provide " + string(category) + " recommendations for database: " + databaseID
}

func (s *Service) extractActions(response string) []SuggestedAction {
	// TODO: Implement action extraction from response
	return []SuggestedAction{}
}

func (s *Service) extractSQLAndExplanation(response string) (string, string) {
	// TODO: Implement SQL and explanation extraction
	return response, ""
}

func (s *Service) extractSQL(response string) string {
	// TODO: Implement SQL extraction
	return response
}

func (s *Service) parseAnalysisResponse(response string) *IssueAnalysis {
	// TODO: Implement response parsing
	return &IssueAnalysis{
		Summary: &IssueSummary{
			Title:       "Issue Analysis",
			Description: response,
		},
		RootCauses: []RootCause{},
		Solutions:  []Solution{},
	}
}

func (s *Service) parseRecommendations(response string, category RecommendationCategory) []Recommendation {
	// TODO: Implement recommendation parsing
	return []Recommendation{
		{
			Title:       "Recommendation",
			Description: response,
			Category:    category,
			Priority:    "medium",
		},
	}
}
