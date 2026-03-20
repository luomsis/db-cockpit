package adapter

import (
	"context"

	"github.com/db-cockpit/pkg/domain/sqlgovernance"
	agentrpc "github.com/db-cockpit/pkg/rpc/agent"
)

// AgentClientAdapter adapts the RPC AgentClient to the domain ExecutionAgentClient interface
type AgentClientAdapter struct {
	client *agentrpc.AgentClient
}

// NewAgentClientAdapter creates a new adapter
func NewAgentClientAdapter(client *agentrpc.AgentClient) *AgentClientAdapter {
	return &AgentClientAdapter{client: client}
}

// ExecuteSQL implements sqlgovernance.ExecutionAgentClient
func (a *AgentClientAdapter) ExecuteSQL(ctx context.Context, req *sqlgovernance.SQLExecuteRequest) (*sqlgovernance.SQLExecuteResult, error) {
	return a.client.ExecuteSQL(ctx, req)
}

// ExplainSQL implements sqlgovernance.ExecutionAgentClient
func (a *AgentClientAdapter) ExplainSQL(ctx context.Context, databaseID, sql string) (string, error) {
	return a.client.ExplainSQL(ctx, databaseID, sql)
}

// Ensure interface compliance
var _ sqlgovernance.ExecutionAgentClient = (*AgentClientAdapter)(nil)
