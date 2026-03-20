package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/db-cockpit/pkg/common/task"
	"github.com/db-cockpit/pkg/common/utils"
	"github.com/db-cockpit/pkg/data/pgmq"
)

// TaskSubmitterImpl implements TaskSubmitter interface
// Domain Layer uses this to submit tasks directly to PGMQ
type TaskSubmitterImpl struct {
	store task.TaskStore
	pgmq  *pgmq.PGMQClient
	queue string
}

// NewTaskSubmitter creates a new TaskSubmitter
func NewTaskSubmitter(store task.TaskStore, pgmqClient *pgmq.PGMQClient, queue string) *TaskSubmitterImpl {
	if queue == "" {
		queue = "tasks"
	}
	return &TaskSubmitterImpl{
		store: store,
		pgmq:  pgmqClient,
		queue: queue,
	}
}

// SubmitTask submits a new task
func (s *TaskSubmitterImpl) SubmitTask(ctx context.Context, req *task.SubmitTaskRequest) (*task.SubmitTaskResponse, error) {
	// Create task record
	t := &task.Task{
		TaskID:   utils.GenerateID(),
		TenantID: req.TenantID,
		TaskType: req.TaskType,
		Priority: req.Priority,
		Payload:  req.Payload,
		Options:  req.Options,
		Schedule: req.Schedule,
		Status:   task.TaskStatusPending,
		Timing: task.TaskTiming{
			SubmittedAt: time.Now(),
		},
	}

	// Set defaults
	if t.Priority == "" {
		t.Priority = task.PriorityMedium
	}
	if t.Options.MaxRetries <= 0 {
		t.Options.MaxRetries = 3
	}

	// 1. Save task to database first (ensures persistence)
	if err := s.store.CreateTask(ctx, t); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 2. Send message to PGMQ (only task_id for efficiency)
	msg := map[string]string{"task_id": t.TaskID}
	_, err := s.pgmq.SendJSON(ctx, s.queue, msg, pgmq.SendMessageOptions{
		Delay: time.Duration(req.Options.DelaySeconds) * time.Second,
	})
	if err != nil {
		// Rollback: mark task as failed
		s.store.UpdateTaskStatus(ctx, t.TaskID, task.TaskStatusFailed, nil, &task.TaskError{
			Code:    "QUEUE_ERROR",
			Message: fmt.Sprintf("Failed to enqueue: %v", err),
		})
		return nil, fmt.Errorf("failed to send task to queue: %w", err)
	}

	// 3. Update status to queued
	s.store.UpdateTaskStatus(ctx, t.TaskID, task.TaskStatusQueued, nil, nil)

	return &task.SubmitTaskResponse{
		Success:     true,
		TaskID:      t.TaskID,
		QueueName:   s.queue,
		SubmittedAt: t.Timing.SubmittedAt,
	}, nil
}

// SubmitSQLAnalysisTask submits a SQL analysis task
func (s *TaskSubmitterImpl) SubmitSQLAnalysisTask(ctx context.Context, tenantID, databaseID, sql string, priority string) (string, error) {
	if priority == "" {
		priority = string(task.PriorityMedium)
	}

	payload := task.TaskPayload{
		Handler: "sql_analysis",
		Data:    []byte(sql),
		Metadata: map[string]string{
			"database_id": databaseID,
		},
	}

	resp, err := s.SubmitTask(ctx, &task.SubmitTaskRequest{
		TenantID: tenantID,
		TaskType: task.TaskTypeSQLAnalysis,
		Priority: task.TaskPriority(priority),
		Payload:  payload,
	})
	if err != nil {
		return "", err
	}
	return resp.TaskID, nil
}

// SubmitReportGenerationTask submits a report generation task
func (s *TaskSubmitterImpl) SubmitReportGenerationTask(ctx context.Context, tenantID, databaseID, reportType string, startTime, endTime int64) (string, error) {
	payload := task.TaskPayload{
		Handler: "report_generation",
		Metadata: map[string]string{
			"database_id": databaseID,
			"report_type": reportType,
			"start_time":  fmt.Sprintf("%d", startTime),
			"end_time":    fmt.Sprintf("%d", endTime),
		},
	}

	resp, err := s.SubmitTask(ctx, &task.SubmitTaskRequest{
		TenantID: tenantID,
		TaskType: task.TaskTypeReportGeneration,
		Priority: task.PriorityMedium,
		Payload:  payload,
	})
	if err != nil {
		return "", err
	}
	return resp.TaskID, nil
}

// SubmitDiagnosisTask submits a diagnosis task
func (s *TaskSubmitterImpl) SubmitDiagnosisTask(ctx context.Context, tenantID, databaseID string, startTime, endTime int64, deepAnalysis bool) (string, error) {
	payload := task.TaskPayload{
		Handler: "diagnosis",
		Metadata: map[string]string{
			"database_id":   databaseID,
			"start_time":    fmt.Sprintf("%d", startTime),
			"end_time":      fmt.Sprintf("%d", endTime),
			"deep_analysis": fmt.Sprintf("%v", deepAnalysis),
		},
	}

	resp, err := s.SubmitTask(ctx, &task.SubmitTaskRequest{
		TenantID: tenantID,
		TaskType: task.TaskTypeDiagnosis,
		Priority: task.PriorityHigh,
		Payload:  payload,
	})
	if err != nil {
		return "", err
	}
	return resp.TaskID, nil
}

// SubmitThresholdCalculationTask submits a threshold calculation task
func (s *TaskSubmitterImpl) SubmitThresholdCalculationTask(ctx context.Context, tenantID, databaseID, metricName, method string, startTime, endTime int64) (string, error) {
	payload := task.TaskPayload{
		Handler: "threshold_calculation",
		Metadata: map[string]string{
			"database_id": databaseID,
			"metric_name": metricName,
			"method":      method,
			"start_time":  fmt.Sprintf("%d", startTime),
			"end_time":    fmt.Sprintf("%d", endTime),
		},
	}

	resp, err := s.SubmitTask(ctx, &task.SubmitTaskRequest{
		TenantID: tenantID,
		TaskType: task.TaskTypeThresholdCalc,
		Priority: task.PriorityMedium,
		Payload:  payload,
	})
	if err != nil {
		return "", err
	}
	return resp.TaskID, nil
}

// SubmitScheduledTask submits a scheduled task
func (s *TaskSubmitterImpl) SubmitScheduledTask(ctx context.Context, req *task.SubmitTaskRequest, scheduledAt time.Time) (string, error) {
	req.Schedule = task.TaskSchedule{
		Scheduled:   true,
		ScheduledAt: scheduledAt,
	}
	resp, err := s.SubmitTask(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.TaskID, nil
}

// SubmitCronTask submits a cron-based recurring task
func (s *TaskSubmitterImpl) SubmitCronTask(ctx context.Context, req *task.SubmitTaskRequest, cronExpr string, timezone string) (string, error) {
	req.Schedule = task.TaskSchedule{
		Scheduled:      true,
		CronExpression: cronExpr,
		Timezone:       timezone,
	}
	resp, err := s.SubmitTask(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.TaskID, nil
}

// Ensure TaskSubmitterImpl implements TaskSubmitter interface
var _ task.TaskSubmitter = (*TaskSubmitterImpl)(nil)

// TaskMessage represents the message structure sent to PGMQ
type TaskMessage struct {
	TaskID   string `json:"task_id"`
	Priority string `json:"priority,omitempty"`
}

// ParseTaskMessage parses a PGMQ message body
func ParseTaskMessage(body []byte) (*TaskMessage, error) {
	var msg TaskMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse task message: %w", err)
	}
	return &msg, nil
}
