package task

import (
	"context"
	"time"
)

// TaskStore defines the interface for task persistence
// This interface is implemented by both:
// - pkg/task/store_pg.go (Task Engine side)
// - Can be used directly by Domain Layer for queries
type TaskStore interface {
	// CreateTask creates a new task record
	CreateTask(ctx context.Context, task *Task) error

	// GetTask retrieves a task by ID
	GetTask(ctx context.Context, taskID string) (*Task, error)

	// UpdateTaskStatus updates task status and result
	UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, result *TaskResult, err *TaskError) error

	// UpdateTaskProgress updates task progress (for long-running tasks)
	UpdateTaskProgress(ctx context.Context, taskID string, progress int, message string) error

	// ListTasks lists tasks with filters
	ListTasks(ctx context.Context, filter *TaskFilter) (*TaskListResult, error)

	// SaveEvent saves a task event
	SaveEvent(ctx context.Context, event *TaskEvent) error

	// GetEvents gets events for a task
	GetEvents(ctx context.Context, taskID string, limit int) ([]TaskEvent, error)

	// CancelTask marks a task as cancelled (if not running)
	CancelTask(ctx context.Context, taskID string, reason string) error

	// RetryTask resets a failed task for retry
	RetryTask(ctx context.Context, taskID string) error
}

// TaskSubmitter defines the interface for submitting tasks
// Domain Layer uses this to submit tasks to PGMQ
type TaskSubmitter interface {
	// SubmitTask submits a new task
	SubmitTask(ctx context.Context, req *SubmitTaskRequest) (*SubmitTaskResponse, error)

	// SubmitSQLAnalysisTask submits a SQL analysis task
	SubmitSQLAnalysisTask(ctx context.Context, tenantID, databaseID, sql string, priority string) (string, error)

	// SubmitReportGenerationTask submits a report generation task
	SubmitReportGenerationTask(ctx context.Context, tenantID, databaseID, reportType string, startTime, endTime int64) (string, error)

	// SubmitDiagnosisTask submits a diagnosis task
	SubmitDiagnosisTask(ctx context.Context, tenantID, databaseID string, startTime, endTime int64, deepAnalysis bool) (string, error)

	// SubmitThresholdCalculationTask submits a threshold calculation task
	SubmitThresholdCalculationTask(ctx context.Context, tenantID, databaseID, metricName, method string, startTime, endTime int64) (string, error)
}

// TaskStatusQuerier defines the interface for querying task status
// Domain Layer uses this to check task status directly from DB
type TaskStatusQuerier interface {
	// GetTaskStatus gets task status
	GetTaskStatus(ctx context.Context, taskID string) (status TaskStatus, progress int, message string, err error)

	// GetTaskResult gets task result (if completed)
	GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error)

	// WaitTask waits for task completion with timeout
	WaitTask(ctx context.Context, taskID string, timeout time.Duration) (*Task, error)
}
