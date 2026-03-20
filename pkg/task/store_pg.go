package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/db-cockpit/pkg/common/task"
	"github.com/db-cockpit/pkg/common/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGTaskStore implements TaskStore using PostgreSQL
type PGTaskStore struct {
	pool *pgxpool.Pool
}

// NewPGTaskStore creates a new PostgreSQL task store
func NewPGTaskStore(pool *pgxpool.Pool) *PGTaskStore {
	return &PGTaskStore{pool: pool}
}

// CreateTask creates a new task record
func (s *PGTaskStore) CreateTask(ctx context.Context, t *task.Task) error {
	if t.TaskID == "" {
		t.TaskID = utils.GenerateID()
	}
	if t.Status == "" {
		t.Status = task.TaskStatusPending
	}
	if t.Timing.SubmittedAt.IsZero() {
		t.Timing.SubmittedAt = time.Now()
	}

	payloadJSON, err := json.Marshal(t.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	optionsJSON, _ := json.Marshal(t.Options)
	scheduleJSON, _ := json.Marshal(t.Schedule)
	metadataJSON, _ := json.Marshal(t.Metadata)

	_, err = s.pool.Exec(ctx, `
		INSERT INTO tasks (
			task_id, tenant_id, task_type, priority, status,
			payload, options, schedule, metadata,
			submitted_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`, t.TaskID, t.TenantID, t.TaskType, t.Priority, t.Status,
		payloadJSON, optionsJSON, scheduleJSON, metadataJSON,
		t.Timing.SubmittedAt)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Save initial event
	event := &task.TaskEvent{
		EventID:   utils.GenerateID(),
		TaskID:    t.TaskID,
		NewStatus: t.Status,
		Message:   "Task created",
		Timestamp: time.Now(),
	}
	return s.SaveEvent(ctx, event)
}

// GetTask retrieves a task by ID
func (s *PGTaskStore) GetTask(ctx context.Context, taskID string) (*task.Task, error) {
	var t task.Task
	var payloadJSON, optionsJSON, scheduleJSON, metadataJSON []byte
	var resultJSON, errorJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT task_id, tenant_id, task_type, priority, status,
			   payload, options, schedule, result, error,
			   retry_count, submitted_at, started_at, completed_at, metadata
		FROM tasks WHERE task_id = $1
	`, taskID).Scan(
		&t.TaskID, &t.TenantID, &t.TaskType, &t.Priority, &t.Status,
		&payloadJSON, &optionsJSON, &scheduleJSON, &resultJSON, &errorJSON,
		&t.Timing.RetryCount, &t.Timing.SubmittedAt, &t.Timing.StartedAt, &t.Timing.CompletedAt, &metadataJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(payloadJSON, &t.Payload)
	json.Unmarshal(optionsJSON, &t.Options)
	json.Unmarshal(scheduleJSON, &t.Schedule)
	json.Unmarshal(metadataJSON, &t.Metadata)

	if resultJSON != nil {
		t.Result = &task.TaskResult{}
		json.Unmarshal(resultJSON, t.Result)
	}
	if errorJSON != nil {
		t.Error = &task.TaskError{}
		json.Unmarshal(errorJSON, t.Error)
	}

	return &t, nil
}

// UpdateTaskStatus updates task status and result
func (s *PGTaskStore) UpdateTaskStatus(ctx context.Context, taskID string, status task.TaskStatus, result *task.TaskResult, taskErr *task.TaskError) error {
	var resultJSON, errorJSON []byte
	var err error

	if result != nil {
		resultJSON, err = json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
	}

	if taskErr != nil {
		errorJSON, err = json.Marshal(taskErr)
		if err != nil {
			return fmt.Errorf("failed to marshal error: %w", err)
		}
	}

	now := time.Now()

	// Update task record
	_, err = s.pool.Exec(ctx, `
		UPDATE tasks SET
			status = $2,
			result = $3,
			error = $4,
			updated_at = $5,
			started_at = CASE WHEN $2 = 'running' THEN $5 ELSE started_at END,
			completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN $5 ELSE completed_at END
		WHERE task_id = $1
	`, taskID, status, resultJSON, errorJSON, now)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Get old status for event
	var oldStatus task.TaskStatus
	s.pool.QueryRow(ctx, "SELECT status FROM tasks WHERE task_id = $1", taskID).Scan(&oldStatus)

	// Save event
	event := &task.TaskEvent{
		EventID:   utils.GenerateID(),
		TaskID:    taskID,
		OldStatus: oldStatus,
		NewStatus: status,
		Timestamp: now,
	}
	return s.SaveEvent(ctx, event)
}

// UpdateTaskProgress updates task progress
func (s *PGTaskStore) UpdateTaskProgress(ctx context.Context, taskID string, progress int, message string) error {
	event := &task.TaskEvent{
		EventID:   utils.GenerateID(),
		TaskID:    taskID,
		NewStatus: task.TaskStatusRunning,
		Progress:  progress,
		Message:   message,
		Timestamp: time.Now(),
	}
	return s.SaveEvent(ctx, event)
}

// ListTasks lists tasks with filters
func (s *PGTaskStore) ListTasks(ctx context.Context, filter *task.TaskFilter) (*task.TaskListResult, error) {
	// Build query
	query := `SELECT task_id, tenant_id, task_type, priority, status,
			  submitted_at, started_at, completed_at, retry_count
			  FROM tasks WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM tasks WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, filter.TenantID)
		argIdx++
	}
	if filter.TaskType != "" {
		query += fmt.Sprintf(" AND task_type = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND task_type = $%d", argIdx)
		args = append(args, filter.TaskType)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND submitted_at >= $%d", argIdx)
		countQuery += fmt.Sprintf(" AND submitted_at >= $%d", argIdx)
		args = append(args, filter.StartTime)
		argIdx++
	}
	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND submitted_at <= $%d", argIdx)
		countQuery += fmt.Sprintf(" AND submitted_at <= $%d", argIdx)
		args = append(args, filter.EndTime)
		argIdx++
	}

	// Get total count
	var total int64
	s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)

	// Add pagination
	query += " ORDER BY submitted_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		if err := rows.Scan(&t.TaskID, &t.TenantID, &t.TaskType, &t.Priority, &t.Status,
			&t.Timing.SubmittedAt, &t.Timing.StartedAt, &t.Timing.CompletedAt, &t.Timing.RetryCount); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	return &task.TaskListResult{
		Tasks: tasks,
		Total: total,
	}, nil
}

// SaveEvent saves a task event
func (s *PGTaskStore) SaveEvent(ctx context.Context, event *task.TaskEvent) error {
	if event.EventID == "" {
		event.EventID = utils.GenerateID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO task_events (event_id, task_id, old_status, new_status, message, progress, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.EventID, event.TaskID, event.OldStatus, event.NewStatus, event.Message, event.Progress, event.Timestamp)

	return err
}

// GetEvents gets events for a task
func (s *PGTaskStore) GetEvents(ctx context.Context, taskID string, limit int) ([]task.TaskEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.pool.Query(ctx, `
		SELECT event_id, task_id, old_status, new_status, message, progress, created_at
		FROM task_events WHERE task_id = $1 ORDER BY created_at DESC LIMIT $2
	`, taskID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []task.TaskEvent
	for rows.Next() {
		var e task.TaskEvent
		if err := rows.Scan(&e.EventID, &e.TaskID, &e.OldStatus, &e.NewStatus, &e.Message, &e.Progress, &e.Timestamp); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// CancelTask marks a task as cancelled
func (s *PGTaskStore) CancelTask(ctx context.Context, taskID string, reason string) error {
	// Check current status
	var status task.TaskStatus
	err := s.pool.QueryRow(ctx, "SELECT status FROM tasks WHERE task_id = $1", taskID).Scan(&status)
	if err != nil {
		return fmt.Errorf("task not found")
	}

	if status == task.TaskStatusRunning || status == task.TaskStatusCompleted {
		return fmt.Errorf("cannot cancel task in %s status", status)
	}

	return s.UpdateTaskStatus(ctx, taskID, task.TaskStatusCancelled, nil, &task.TaskError{
		Code:    "CANCELLED",
		Message: reason,
	})
}

// RetryTask resets a failed task for retry
func (s *PGTaskStore) RetryTask(ctx context.Context, taskID string) error {
	// Get current task
	t, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if t.Status != task.TaskStatusFailed && t.Status != task.TaskStatusCancelled {
		return fmt.Errorf("can only retry failed or cancelled tasks")
	}

	// Reset task
	_, err = s.pool.Exec(ctx, `
		UPDATE tasks SET
			status = 'pending',
			result = NULL,
			error = NULL,
			retry_count = retry_count + 1,
			updated_at = NOW()
		WHERE task_id = $1
	`, taskID)

	if err != nil {
		return err
	}

	return s.SaveEvent(ctx, &task.TaskEvent{
		EventID:   utils.GenerateID(),
		TaskID:    taskID,
		OldStatus: t.Status,
		NewStatus: task.TaskStatusPending,
		Message:   "Task queued for retry",
		Timestamp: time.Now(),
	})
}

// Ensure PGTaskStore implements TaskStore interface
var _ task.TaskStore = (*PGTaskStore)(nil)
