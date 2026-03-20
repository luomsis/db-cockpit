package rpc

import (
	"context"

	"github.com/db-cockpit/api/proto/agent"
	"github.com/db-cockpit/pkg/domain/sqlgovernance"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentClient wraps the gRPC client for ExecutionAgent
type AgentClient struct {
	conn   *grpc.ClientConn
	client agent.ExecutionAgentServiceClient
}

// NewAgentClient creates a new Agent RPC client
func NewAgentClient(addr string) (*AgentClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &AgentClient{
		conn:   conn,
		client: agent.NewExecutionAgentServiceClient(conn),
	}, nil
}

// Close closes the connection
func (c *AgentClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ExecuteSQL implements sqlgovernance.ExecutionAgentClient
func (c *AgentClient) ExecuteSQL(ctx context.Context, req *sqlgovernance.SQLExecuteRequest) (*sqlgovernance.SQLExecuteResult, error) {
	// Convert to proto request
	protoReq := &agent.ExecuteSQLRequest{
		DatabaseId: req.DatabaseID,
		Sql:        req.SQLText,
		Options: &agent.ExecutionOptions{
			TimeoutSeconds: int32(req.TimeoutSeconds),
			MaxRows:        int32(req.MaxRows),
		},
	}

	resp, err := c.client.ExecuteSQL(ctx, protoReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	result := &sqlgovernance.SQLExecuteResult{
		ExecutionID:     resp.ExecutionId,
		Status:          executionStatusToString(resp.Status),
		ExecutionTimeMs: resp.ExecutionTimeMs,
		ErrorMessage:    resp.ErrorMessage,
		AuditID:         resp.AuditId,
		RowsAffected:    resp.RowsAffected,
	}

	// Convert columns
	for _, col := range resp.Columns {
		result.Columns = append(result.Columns, sqlgovernance.Column{
			Name: col.Name,
			Type: col.Type,
		})
	}

	// Convert rows
	for _, row := range resp.Rows {
		result.Rows = append(result.Rows, sqlgovernance.Row{
			Values: row.Values,
		})
	}

	return result, nil
}

// ExplainSQL implements sqlgovernance.ExecutionAgentClient
func (c *AgentClient) ExplainSQL(ctx context.Context, databaseID, sql string) (string, error) {
	// For now, return empty string as the proto doesn't have ExplainSQL
	// In a real implementation, this would call the ExplainSQL RPC
	return "", nil
}

// ExecuteSQLDirect calls the underlying gRPC client directly
func (c *AgentClient) ExecuteSQLDirect(ctx context.Context, req *agent.ExecuteSQLRequest) (*agent.ExecuteSQLResponse, error) {
	return c.client.ExecuteSQL(ctx, req)
}

// ExecuteTransaction calls the transaction RPC
func (c *AgentClient) ExecuteTransaction(ctx context.Context, req *agent.ExecuteTransactionRequest) (*agent.ExecuteTransactionResponse, error) {
	return c.client.ExecuteTransaction(ctx, req)
}

// ExecuteAPICall calls the API execution RPC
func (c *AgentClient) ExecuteAPICall(ctx context.Context, req *agent.ExecuteAPICallRequest) (*agent.ExecuteAPICallResponse, error) {
	return c.client.ExecuteAPICall(ctx, req)
}

// GetExecutionStatus gets execution status
func (c *AgentClient) GetExecutionStatus(ctx context.Context, executionID string) (*agent.GetExecutionStatusResponse, error) {
	return c.client.GetExecutionStatus(ctx, &agent.GetExecutionStatusRequest{
		ExecutionId: executionID,
	})
}

// CancelExecution cancels an execution
func (c *AgentClient) CancelExecution(ctx context.Context, executionID, reason string) (*agent.CancelExecutionResponse, error) {
	return c.client.CancelExecution(ctx, &agent.CancelExecutionRequest{
		ExecutionId: executionID,
		Reason:      reason,
	})
}

// GetAuditLog gets audit log
func (c *AgentClient) GetAuditLog(ctx context.Context, req *agent.GetAuditLogRequest) (*agent.GetAuditLogResponse, error) {
	return c.client.GetAuditLog(ctx, req)
}

// Health checks agent health
func (c *AgentClient) Health(ctx context.Context) (*agent.HealthResponse, error) {
	return c.client.Health(ctx, &agent.HealthRequest{})
}

// executionStatusToString converts proto status to string
func executionStatusToString(status agent.ExecutionStatus) string {
	switch status {
	case agent.ExecutionStatus_EXECUTION_STATUS_PENDING:
		return "pending"
	case agent.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return "running"
	case agent.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case agent.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return "failed"
	case agent.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return "cancelled"
	case agent.ExecutionStatus_EXECUTION_STATUS_TIMEOUT:
		return "timeout"
	default:
		return "unknown"
	}
}
