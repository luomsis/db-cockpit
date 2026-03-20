package taskclient

import (
	"context"
	"time"

	"github.com/db-cockpit/pkg/common/task"
)

// TaskClientInterface defines the interface for domain services to interact with Task System
// This interface combines:
// - TaskSubmitter: for submitting tasks (via PGMQ)
// - TaskStatusQuerier: for querying task status (via DB)
type TaskClientInterface interface {
	// Task submission methods (via PGMQ)
	task.TaskSubmitter

	// Task status methods (via DB direct query)
	task.TaskStatusQuerier

	// CancelTask cancels a task
	CancelTask(ctx context.Context, taskID, reason string) error

	// ListTasks lists tasks with filters
	ListTasks(ctx context.Context, tenantID string, taskType task.TaskType, status task.TaskStatus, limit int) ([]task.Task, error)

	// GetTaskEvents gets events for a task
	GetTaskEvents(ctx context.Context, taskID string, limit int) ([]task.TaskEvent, error)
}

// TaskClient combines submitter and store for Domain Layer
type TaskClient struct {
	submitter task.TaskSubmitter
	store     task.TaskStore
}

// NewTaskClient creates a new TaskClient
func NewTaskClient(submitter task.TaskSubmitter, store task.TaskStore) *TaskClient {
	return &TaskClient{
		submitter: submitter,
		store:     store,
	}
}

// SubmitTask submits a new task
func (c *TaskClient) SubmitTask(ctx context.Context, req *task.SubmitTaskRequest) (*task.SubmitTaskResponse, error) {
	return c.submitter.SubmitTask(ctx, req)
}

// SubmitSQLAnalysisTask submits a SQL analysis task
func (c *TaskClient) SubmitSQLAnalysisTask(ctx context.Context, tenantID, databaseID, sql string, priority string) (string, error) {
	return c.submitter.SubmitSQLAnalysisTask(ctx, tenantID, databaseID, sql, priority)
}

// SubmitReportGenerationTask submits a report generation task
func (c *TaskClient) SubmitReportGenerationTask(ctx context.Context, tenantID, databaseID, reportType string, startTime, endTime int64) (string, error) {
	return c.submitter.SubmitReportGenerationTask(ctx, tenantID, databaseID, reportType, startTime, endTime)
}

// SubmitDiagnosisTask submits a diagnosis task
func (c *TaskClient) SubmitDiagnosisTask(ctx context.Context, tenantID, databaseID string, startTime, endTime int64, deepAnalysis bool) (string, error) {
	return c.submitter.SubmitDiagnosisTask(ctx, tenantID, databaseID, startTime, endTime, deepAnalysis)
}

// SubmitThresholdCalculationTask submits a threshold calculation task
func (c *TaskClient) SubmitThresholdCalculationTask(ctx context.Context, tenantID, databaseID, metricName, method string, startTime, endTime int64) (string, error) {
	return c.submitter.SubmitThresholdCalculationTask(ctx, tenantID, databaseID, metricName, method, startTime, endTime)
}

// GetTaskStatus gets task status
func (c *TaskClient) GetTaskStatus(ctx context.Context, taskID string) (task.TaskStatus, int, string, error) {
	t, err := c.store.GetTask(ctx, taskID)
	if err != nil {
		return "", 0, "", err
	}

	// Get latest event for message
	events, _ := c.store.GetEvents(ctx, taskID, 1)
	message := ""
	if len(events) > 0 {
		message = events[0].Message
	}

	return t.Status, events[0].Progress, message, nil
}

// GetTaskResult gets task result (if completed)
func (c *TaskClient) GetTaskResult(ctx context.Context, taskID string) (*task.TaskResult, error) {
	t, err := c.store.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if t.Status != task.TaskStatusCompleted {
		return nil, nil
	}

	return t.Result, nil
}

// WaitTask waits for task completion with timeout
func (c *TaskClient) WaitTask(ctx context.Context, taskID string, timeout time.Duration) (*task.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			t, err := c.store.GetTask(ctx, taskID)
			if err != nil {
				return nil, err
			}

			if t.Status == task.TaskStatusCompleted ||
				t.Status == task.TaskStatusFailed ||
				t.Status == task.TaskStatusCancelled {
				return t, nil
			}
		}
	}
}

// CancelTask cancels a task
func (c *TaskClient) CancelTask(ctx context.Context, taskID, reason string) error {
	return c.store.CancelTask(ctx, taskID, reason)
}

// ListTasks lists tasks with filters
func (c *TaskClient) ListTasks(ctx context.Context, tenantID string, taskType task.TaskType, status task.TaskStatus, limit int) ([]task.Task, error) {
	result, err := c.store.ListTasks(ctx, &task.TaskFilter{
		TenantID: tenantID,
		TaskType: taskType,
		Status:   status,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// GetTaskEvents gets events for a task
func (c *TaskClient) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]task.TaskEvent, error) {
	return c.store.GetEvents(ctx, taskID, limit)
}

// GetTask retrieves full task details
func (c *TaskClient) GetTask(ctx context.Context, taskID string) (*task.Task, error) {
	return c.store.GetTask(ctx, taskID)
}

// RetryTask retries a failed task
func (c *TaskClient) RetryTask(ctx context.Context, taskID string) error {
	return c.store.RetryTask(ctx, taskID)
}

// Ensure TaskClient implements TaskClientInterface
var _ TaskClientInterface = (*TaskClient)(nil)
