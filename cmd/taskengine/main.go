package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/db-cockpit/pkg/common/config"
	"github.com/db-cockpit/pkg/common/logger"
	"github.com/db-cockpit/pkg/common/task"
	"github.com/db-cockpit/pkg/data/pgmq"
	taskengine "github.com/db-cockpit/pkg/task"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v, using defaults\n", err)
		cfg = config.DefaultConfig()
	}

	// Initialize logger
	if err := logger.Init(&logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting Task Engine Service")

	// Initialize PostgreSQL connection pool
	ctx := context.Background()
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.TimescaleDB.Host,
		cfg.Database.TimescaleDB.Port,
		cfg.Database.TimescaleDB.User,
		cfg.Database.TimescaleDB.Password,
		cfg.Database.TimescaleDB.Database,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		logger.Fatal("Failed to create connection pool", zap.Error(err))
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	logger.Info("Connected to PostgreSQL")

	// Initialize schema
	if err := initSchema(ctx, pool); err != nil {
		logger.Fatal("Failed to initialize schema", zap.Error(err))
	}

	// Create task store
	store := taskengine.NewPGTaskStore(pool)

	// Create PGMQ client (shares the same pool)
	pgmqClient, err := pgmq.NewPGMQClientWithPool(pool)
	if err != nil {
		logger.Fatal("Failed to create PGMQ client", zap.Error(err))
	}

	// Create task engine (consumer)
	engine := taskengine.NewTaskEngine(&taskengine.TaskEngineConfig{
		QueueName:         "tasks",
		WorkerCount:       10,
		PollInterval:      1 * time.Second,
		VisibilityTimeout: 5 * time.Minute,
		MaxRetries:        3,
	}, store, pgmqClient)

	// Register task handlers
	registerTaskHandlers(engine)

	// Start task engine
	if err := engine.Start(ctx); err != nil {
		logger.Fatal("Failed to start task engine", zap.Error(err))
	}

	logger.Info("Task Engine started",
		zap.String("host", cfg.Server.TaskEngine.Host),
		zap.Int("port", cfg.Server.TaskEngine.Port),
	)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Task Engine...")

	// Stop task engine
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := engine.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping task engine", zap.Error(err))
	}

	logger.Info("Task Engine stopped")
}

// registerTaskHandlers registers task handlers
func registerTaskHandlers(engine *taskengine.TaskEngine) {
	// Register SQL Analysis handler
	engine.RegisterHandler(&sqlAnalysisHandler{})

	// Register Report Generation handler
	engine.RegisterHandler(&reportGenerationHandler{})

	// Register Threshold Calculation handler
	engine.RegisterHandler(&thresholdCalcHandler{})

	// Register Diagnosis handler
	engine.RegisterHandler(&diagnosisHandler{})

	// Register Data Sync handler
	engine.RegisterHandler(&dataSyncHandler{})

	// Register Cleanup handler
	engine.RegisterHandler(&cleanupHandler{})
}

// initSchema initializes database schema
func initSchema(ctx context.Context, pool *pgxpool.Pool) error {
	// Create tasks table
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tasks (
			task_id VARCHAR(36) PRIMARY KEY,
			tenant_id VARCHAR(36) NOT NULL,
			task_type VARCHAR(50) NOT NULL,
			priority VARCHAR(20) NOT NULL DEFAULT 'medium',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			payload JSONB NOT NULL,
			options JSONB,
			schedule JSONB,
			result JSONB,
			error JSONB,
			retry_count INT DEFAULT 0,
			submitted_at TIMESTAMPTZ NOT NULL,
			started_at TIMESTAMPTZ,
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_tasks_tenant ON tasks(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(task_type);
		CREATE INDEX IF NOT EXISTS idx_tasks_submitted ON tasks(submitted_at DESC);
	`)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Create task_events table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS task_events (
			event_id VARCHAR(36) PRIMARY KEY,
			task_id VARCHAR(36) NOT NULL,
			old_status VARCHAR(20),
			new_status VARCHAR(20) NOT NULL,
			message TEXT,
			progress INT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			CONSTRAINT fk_task FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_events_task ON task_events(task_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create task_events table: %w", err)
	}

	// Enable PGMQ extension
	_, err = pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgmq CASCADE`)
	if err != nil {
		logger.Warn("Failed to create pgmq extension (may already exist)", zap.Error(err))
	}

	return nil
}

// Task handler implementations

type sqlAnalysisHandler struct{}

func (h *sqlAnalysisHandler) Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	// TODO: Implement SQL analysis task
	// This would typically:
	// 1. Parse the SQL from payload
	// 2. Analyze SQL for performance issues
	// 3. Generate recommendations

	logger.Info("Processing SQL analysis task", zap.String("task_id", t.TaskID))

	return &task.TaskResult{
		Data: []byte("SQL analysis completed"),
		Metadata: map[string]string{
			"analysis_type": "performance",
		},
	}, nil
}

func (h *sqlAnalysisHandler) TaskType() task.TaskType {
	return task.TaskTypeSQLAnalysis
}

type reportGenerationHandler struct{}

func (h *reportGenerationHandler) Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	// TODO: Implement report generation task
	logger.Info("Processing report generation task", zap.String("task_id", t.TaskID))

	return &task.TaskResult{
		Data: []byte("Report generated"),
	}, nil
}

func (h *reportGenerationHandler) TaskType() task.TaskType {
	return task.TaskTypeReportGeneration
}

type thresholdCalcHandler struct{}

func (h *thresholdCalcHandler) Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	// TODO: Implement threshold calculation task
	logger.Info("Processing threshold calculation task", zap.String("task_id", t.TaskID))

	return &task.TaskResult{
		Data: []byte("Thresholds calculated"),
	}, nil
}

func (h *thresholdCalcHandler) TaskType() task.TaskType {
	return task.TaskTypeThresholdCalc
}

type diagnosisHandler struct{}

func (h *diagnosisHandler) Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	// TODO: Implement diagnosis task
	logger.Info("Processing diagnosis task", zap.String("task_id", t.TaskID))

	return &task.TaskResult{
		Data: []byte("Diagnosis completed"),
	}, nil
}

func (h *diagnosisHandler) TaskType() task.TaskType {
	return task.TaskTypeDiagnosis
}

type dataSyncHandler struct{}

func (h *dataSyncHandler) Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	// TODO: Implement data sync task
	logger.Info("Processing data sync task", zap.String("task_id", t.TaskID))

	return &task.TaskResult{
		Data: []byte("Data sync completed"),
	}, nil
}

func (h *dataSyncHandler) TaskType() task.TaskType {
	return task.TaskTypeDataSync
}

type cleanupHandler struct{}

func (h *cleanupHandler) Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	// TODO: Implement cleanup task
	logger.Info("Processing cleanup task", zap.String("task_id", t.TaskID))

	return &task.TaskResult{
		Data: []byte("Cleanup completed"),
	}, nil
}

func (h *cleanupHandler) TaskType() task.TaskType {
	return task.TaskTypeCleanup
}