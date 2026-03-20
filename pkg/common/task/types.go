package task

import (
	"time"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusRetrying  TaskStatus = "retrying"
)

// TaskPriority represents task priority
type TaskPriority string

const (
	PriorityLow      TaskPriority = "low"
	PriorityMedium   TaskPriority = "medium"
	PriorityHigh     TaskPriority = "high"
	PriorityCritical TaskPriority = "critical"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeSQLAnalysis      TaskType = "sql_analysis"
	TaskTypeReportGeneration TaskType = "report_generation"
	TaskTypeThresholdCalc    TaskType = "threshold_calculation"
	TaskTypeDiagnosis        TaskType = "diagnosis"
	TaskTypeDataSync         TaskType = "data_sync"
	TaskTypeCleanup          TaskType = "cleanup"
)

// Task represents a task
type Task struct {
	TaskID   string
	TenantID string
	TaskType TaskType
	Priority TaskPriority
	Payload  TaskPayload
	Options  TaskOptions
	Schedule TaskSchedule
	Status   TaskStatus
	Result   *TaskResult
	Error    *TaskError
	Timing   TaskTiming
	Metadata map[string]string
}

// TaskPayload represents a task payload
type TaskPayload struct {
	Handler  string
	Data     []byte
	Metadata map[string]string
}

// TaskOptions represents task options
type TaskOptions struct {
	MaxRetries     int
	TimeoutSeconds int
	DelaySeconds   int
	Labels         map[string]string
	Queue          string
}

// TaskSchedule represents task schedule
type TaskSchedule struct {
	Scheduled      bool
	ScheduledAt    time.Time
	CronExpression string
	Timezone       string
}

// TaskResult represents a task result
type TaskResult struct {
	Data     []byte
	Metadata map[string]string
}

// TaskError represents a task error
type TaskError struct {
	Code       string
	Message    string
	Details    string
	StackTrace []string
}

// Error implements the error interface
func (e *TaskError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// TaskTiming represents task timing information
type TaskTiming struct {
	SubmittedAt time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
	RetryCount  int
}

// TaskEvent represents a task state change event
type TaskEvent struct {
	EventID   string
	TaskID    string
	OldStatus TaskStatus
	NewStatus TaskStatus
	Message   string
	Progress  int
	Timestamp time.Time
}

// SubmitTaskRequest represents a submit task request
type SubmitTaskRequest struct {
	TenantID string
	TaskType TaskType
	Priority TaskPriority
	Payload  TaskPayload
	Options  TaskOptions
	Schedule TaskSchedule
}

// SubmitTaskResponse represents a submit task response
type SubmitTaskResponse struct {
	Success     bool
	TaskID      string
	QueueName   string
	SubmittedAt time.Time
	Error       string
}

// TaskFilter represents filters for listing tasks
type TaskFilter struct {
	TenantID  string
	TaskType  TaskType
	Status    TaskStatus
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}

// TaskListResult represents the result of listing tasks
type TaskListResult struct {
	Tasks []Task
	Total int64
}
