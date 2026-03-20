package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/db-cockpit/pkg/common/logger"
	"github.com/db-cockpit/pkg/common/task"
	"github.com/db-cockpit/pkg/data/pgmq"
	"go.uber.org/zap"
)

// TaskHandler defines the interface for task handlers
type TaskHandler interface {
	// Handle handles a task
	Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error)

	// TaskType returns the task type this handler handles
	TaskType() task.TaskType
}

// TaskEngineConfig represents task engine configuration
type TaskEngineConfig struct {
	QueueName         string
	WorkerCount       int
	PollInterval      time.Duration
	VisibilityTimeout time.Duration
	MaxRetries        int
}

// TaskEngine implements the task processing engine
// It consumes tasks from PGMQ and executes them via registered handlers
type TaskEngine struct {
	config   *TaskEngineConfig
	store    task.TaskStore
	queue    *pgmq.PGMQClient
	handlers map[task.TaskType]TaskHandler

	handlersMux sync.RWMutex
	running     bool
	runningMux  sync.Mutex
	wg          sync.WaitGroup
}

// NewTaskEngine creates a new task engine
func NewTaskEngine(config *TaskEngineConfig, store task.TaskStore, queue *pgmq.PGMQClient) *TaskEngine {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 10
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 1 * time.Second
	}
	if config.VisibilityTimeout <= 0 {
		config.VisibilityTimeout = 5 * time.Minute
	}
	if config.QueueName == "" {
		config.QueueName = "tasks"
	}

	return &TaskEngine{
		config:   config,
		store:    store,
		queue:    queue,
		handlers: make(map[task.TaskType]TaskHandler),
	}
}

// Start starts the task engine
func (e *TaskEngine) Start(ctx context.Context) error {
	e.runningMux.Lock()
	defer e.runningMux.Unlock()

	if e.running {
		return nil
	}

	// Create queue if not exists
	if err := e.queue.CreateQueueIfNotExists(ctx, e.config.QueueName); err != nil {
		return fmt.Errorf("failed to create queue: %w", err)
	}

	e.running = true

	// Start workers
	for i := 0; i < e.config.WorkerCount; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}

	logger.Info("Task Engine started",
		zap.String("queue", e.config.QueueName),
		zap.Int("workers", e.config.WorkerCount),
	)

	return nil
}

// Stop stops the task engine
func (e *TaskEngine) Stop(ctx context.Context) error {
	e.runningMux.Lock()
	defer e.runningMux.Unlock()

	if !e.running {
		return nil
	}

	e.running = false
	e.wg.Wait()

	logger.Info("Task Engine stopped")
	return nil
}

// RegisterHandler registers a task handler
func (e *TaskEngine) RegisterHandler(handler TaskHandler) error {
	e.handlersMux.Lock()
	defer e.handlersMux.Unlock()

	e.handlers[handler.TaskType()] = handler
	logger.Info("Registered task handler", zap.String("type", string(handler.TaskType())))
	return nil
}

// Health checks task engine health
func (e *TaskEngine) Health(ctx context.Context) error {
	if !e.running {
		return fmt.Errorf("task engine not running")
	}
	return e.queue.Ping(ctx)
}

// worker processes tasks from the queue
func (e *TaskEngine) worker(ctx context.Context, id int) {
	defer e.wg.Done()

	logger.Debug("Worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Worker stopped", zap.Int("worker_id", id))
			return
		default:
			if !e.running {
				return
			}

			// Read from queue
			msg, err := e.queue.Read(ctx, e.config.QueueName, e.config.VisibilityTimeout)
			if err != nil {
				logger.Error("Failed to read from queue", zap.Error(err))
				time.Sleep(e.config.PollInterval)
				continue
			}

			if msg == nil {
				time.Sleep(e.config.PollInterval)
				continue
			}

			// Parse message
			taskMsg, err := ParseTaskMessage(msg.Body)
			if err != nil {
				logger.Error("Failed to parse task message", zap.Error(err), zap.Int64("msg_id", msg.MsgID))
				_ = e.queue.Archive(ctx, e.config.QueueName, msg.MsgID)
				continue
			}

			// Process the task
			e.processTask(ctx, taskMsg.TaskID, msg.MsgID)
		}
	}
}

// processTask processes a single task
func (e *TaskEngine) processTask(ctx context.Context, taskID string, msgID int64) {
	// Load task from store
	t, err := e.store.GetTask(ctx, taskID)
	if err != nil {
		logger.Error("Failed to get task", zap.String("task_id", taskID), zap.Error(err))
		_ = e.queue.Archive(ctx, e.config.QueueName, msgID)
		return
	}

	// Check if task was cancelled
	if t.Status == task.TaskStatusCancelled {
		logger.Info("Task was cancelled, skipping", zap.String("task_id", taskID))
		_ = e.queue.Archive(ctx, e.config.QueueName, msgID)
		return
	}

	// Update status to running
	if err := e.store.UpdateTaskStatus(ctx, taskID, task.TaskStatusRunning, nil, nil); err != nil {
		logger.Error("Failed to update task status", zap.String("task_id", taskID), zap.Error(err))
	}

	logger.Info("Processing task",
		zap.String("task_id", taskID),
		zap.String("type", string(t.TaskType)),
		zap.String("tenant_id", t.TenantID),
	)

	// Execute task
	result, execErr := e.executeTask(ctx, t)

	// Update task result
	var newStatus task.TaskStatus
	var taskErr *task.TaskError

	if execErr != nil {
		newStatus = task.TaskStatusFailed
		taskErr = &task.TaskError{
			Message: execErr.Error(),
		}

		// Check if should retry
		if t.Timing.RetryCount < t.Options.MaxRetries {
			newStatus = task.TaskStatusRetrying
			logger.Warn("Task failed, will retry",
				zap.String("task_id", taskID),
				zap.Int("retry_count", t.Timing.RetryCount),
				zap.Int("max_retries", t.Options.MaxRetries),
				zap.Error(execErr),
			)
		}
	} else {
		newStatus = task.TaskStatusCompleted
		logger.Info("Task completed", zap.String("task_id", taskID))
	}

	// Update store
	if err := e.store.UpdateTaskStatus(ctx, taskID, newStatus, result, taskErr); err != nil {
		logger.Error("Failed to update task result", zap.String("task_id", taskID), zap.Error(err))
	}

	// Archive or delete message from queue
	if newStatus == task.TaskStatusCompleted || newStatus == task.TaskStatusFailed {
		_ = e.queue.Archive(ctx, e.config.QueueName, msgID)
	} else if newStatus == task.TaskStatusRetrying {
		// For retry, keep message but release it back to queue after delay
		// Or delete and let retry mechanism handle it
		_ = e.queue.Delete(ctx, e.config.QueueName, msgID)

		// Re-queue for retry
		t.Timing.RetryCount++
		_ = e.store.RetryTask(ctx, taskID)

		// Send new message to queue
		_, _ = e.queue.SendJSON(ctx, e.config.QueueName, map[string]string{"task_id": taskID}, pgmq.SendMessageOptions{})
	}
}

// executeTask executes a task using registered handler
func (e *TaskEngine) executeTask(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	e.handlersMux.RLock()
	handler, ok := e.handlers[t.TaskType]
	e.handlersMux.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no handler registered for task type: %s", t.TaskType)
	}

	return handler.Handle(ctx, t)
}

// GetStats returns task engine statistics
func (e *TaskEngine) GetStats(ctx context.Context) (*EngineStats, error) {
	stats := &EngineStats{
		QueueName:   e.config.QueueName,
		WorkerCount: e.config.WorkerCount,
		Running:     e.running,
	}

	// Get queue stats
	queueStats, err := e.queue.GetQueueStats(ctx, e.config.QueueName)
	if err == nil {
		stats.QueueLength = queueStats.TotalMessages
	}

	return stats, nil
}

// EngineStats represents task engine statistics
type EngineStats struct {
	QueueName   string `json:"queue_name"`
	WorkerCount int    `json:"worker_count"`
	Running     bool   `json:"running"`
	QueueLength int64  `json:"queue_length"`
}
