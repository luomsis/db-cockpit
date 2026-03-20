package pgmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/db-cockpit/pkg/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGMQClient wraps the PGMQ client with pgxpool connection
type PGMQClient struct {
	config *config.PgMQConfig
	pool   *pgxpool.Pool
}

// Message represents a queue message
type Message struct {
	MsgID      int64
	ReadCount  int
	EnqueuedAt time.Time
	Visibility time.Time
	Body       []byte
}

// SendMessageOptions represents options for sending a message
type SendMessageOptions struct {
	Delay    time.Duration
	Priority int
	MaxReads int
}

// ReceiveMessageOptions represents options for receiving a message
type ReceiveMessageOptions struct {
	VisibilityTimeout time.Duration
	PollTimeout       time.Duration
	BatchSize         int
}

// QueueStats represents queue statistics
type QueueStats struct {
	QueueName     string
	TotalMessages int64
	PendingCount  int64
	InFlightCount int64
}

// NewPGMQClient creates a new PGMQ client with config
func NewPGMQClient(cfg *config.PgMQConfig) (*PGMQClient, error) {
	return &PGMQClient{
		config: cfg,
	}, nil
}

// NewPGMQClientWithPool creates a new PGMQ client with existing pool
func NewPGMQClientWithPool(pool *pgxpool.Pool) (*PGMQClient, error) {
	return &PGMQClient{
		pool: pool,
	}, nil
}

// Connect establishes connection to PGMQ
func (c *PGMQClient) Connect(ctx context.Context) error {
	if c.pool != nil {
		return nil
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.config.Host, c.config.Port, c.config.User, c.config.Password, c.config.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	c.pool = pool
	return nil
}

// Close closes the connection
func (c *PGMQClient) Close() error {
	if c.pool != nil {
		c.pool.Close()
	}
	return nil
}

// Pool returns the underlying connection pool
func (c *PGMQClient) Pool() *pgxpool.Pool {
	return c.pool
}

// CreateQueue creates a new queue
func (c *PGMQClient) CreateQueue(ctx context.Context, name string) error {
	_, err := c.pool.Exec(ctx, "SELECT pgmq.create($1)", name)
	if err != nil {
		// Check if it's "queue already exists" error
		// PGMQ returns specific error for duplicate queues
		return fmt.Errorf("failed to create queue %s: %w", name, err)
	}
	return nil
}

// CreateQueueIfNotExists creates a queue if it doesn't exist
func (c *PGMQClient) CreateQueueIfNotExists(ctx context.Context, name string) error {
	err := c.CreateQueue(ctx, name)
	if err != nil && !isQueueExistsError(err) {
		return err
	}
	return nil
}

// DropQueue drops a queue
func (c *PGMQClient) DropQueue(ctx context.Context, name string) error {
	_, err := c.pool.Exec(ctx, "SELECT pgmq.drop_queue($1)", name)
	return err
}

// ListQueues lists all queues
func (c *PGMQClient) ListQueues(ctx context.Context) ([]string, error) {
	rows, err := c.pool.Query(ctx, "SELECT queue_name FROM pgmq.meta")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queues []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		queues = append(queues, name)
	}
	return queues, nil
}

// Send sends a message to a queue
func (c *PGMQClient) Send(ctx context.Context, queue string, body []byte, opts SendMessageOptions) (int64, error) {
	var msgID int64
	var err error

	if opts.Delay > 0 {
		err = c.pool.QueryRow(ctx,
			"SELECT * FROM pgmq.send($1, $2, $3)",
			queue, body, int(opts.Delay.Seconds()),
		).Scan(&msgID)
	} else {
		err = c.pool.QueryRow(ctx,
			"SELECT * FROM pgmq.send($1, $2)",
			queue, body,
		).Scan(&msgID)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to send message to queue %s: %w", queue, err)
	}
	return msgID, nil
}

// SendBatch sends multiple messages to a queue
func (c *PGMQClient) SendBatch(ctx context.Context, queue string, bodies [][]byte, opts SendMessageOptions) ([]int64, error) {
	rows, err := c.pool.Query(ctx,
		"SELECT * FROM pgmq.send_batch($1, $2)",
		queue, bodies,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		msgIDs = append(msgIDs, id)
	}
	return msgIDs, nil
}

// Read reads a single message from a queue
func (c *PGMQClient) Read(ctx context.Context, queue string, visibilityTimeout time.Duration) (*Message, error) {
	var msg Message
	var body []byte

	err := c.pool.QueryRow(ctx,
		"SELECT msg_id, read_count, enqueued_at, vt, message FROM pgmq.read($1, $2, 1)",
		queue, int(visibilityTimeout.Seconds()),
	).Scan(&msg.MsgID, &msg.ReadCount, &msg.EnqueuedAt, &msg.Visibility, &body)

	if err != nil {
		// No message available is not an error
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	msg.Body = body
	return &msg, nil
}

// ReadBatch reads multiple messages from a queue
func (c *PGMQClient) ReadBatch(ctx context.Context, queue string, opts ReceiveMessageOptions) ([]Message, error) {
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	vt := int(opts.VisibilityTimeout.Seconds())
	if vt <= 0 {
		vt = 30
	}

	rows, err := c.pool.Query(ctx,
		"SELECT msg_id, read_count, enqueued_at, vt, message FROM pgmq.read($1, $2, $3)",
		queue, vt, batchSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var body []byte
		if err := rows.Scan(&msg.MsgID, &msg.ReadCount, &msg.EnqueuedAt, &msg.Visibility, &body); err != nil {
			return nil, err
		}
		msg.Body = body
		messages = append(messages, msg)
	}
	return messages, nil
}

// Pop reads and deletes a message from a queue
func (c *PGMQClient) Pop(ctx context.Context, queue string) (*Message, error) {
	var msg Message
	var body []byte

	err := c.pool.QueryRow(ctx,
		"SELECT msg_id, read_count, enqueued_at, vt, message FROM pgmq.pop($1)",
		queue,
	).Scan(&msg.MsgID, &msg.ReadCount, &msg.EnqueuedAt, &msg.Visibility, &body)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	msg.Body = body
	return &msg, nil
}

// Archive archives a message (removes from queue but keeps in archive)
func (c *PGMQClient) Archive(ctx context.Context, queue string, msgID int64) error {
	var archived bool
	err := c.pool.QueryRow(ctx,
		"SELECT pgmq.archive($1, $2)",
		queue, msgID,
	).Scan(&archived)
	if err != nil {
		return fmt.Errorf("failed to archive message %d: %w", msgID, err)
	}
	return nil
}

// Delete deletes a message from the queue
func (c *PGMQClient) Delete(ctx context.Context, queue string, msgID int64) error {
	var deleted bool
	err := c.pool.QueryRow(ctx,
		"SELECT pgmq.delete($1, $2)",
		queue, msgID,
	).Scan(&deleted)
	if err != nil {
		return fmt.Errorf("failed to delete message %d: %w", msgID, err)
	}
	return nil
}

// DeleteBatch deletes multiple messages
func (c *PGMQClient) DeleteBatch(ctx context.Context, queue string, msgIDs []int64) error {
	_, err := c.pool.Exec(ctx,
		"SELECT pgmq.delete($1, $2::bigint[])",
		queue, msgIDs,
	)
	return err
}

// Purge purges all messages from a queue
func (c *PGMQClient) Purge(ctx context.Context, queue string) (int64, error) {
	var count int64
	err := c.pool.QueryRow(ctx,
		"SELECT pgmq.purge_queue($1)",
		queue,
	).Scan(&count)
	return count, err
}

// SetVisibilityTimeout sets the visibility timeout for a message
func (c *PGMQClient) SetVisibilityTimeout(ctx context.Context, queue string, msgID int64, visibilityTimeout time.Duration) error {
	_, err := c.pool.Exec(ctx,
		"SELECT pgmq.set_vt($1, $2, $3)",
		queue, msgID, int(visibilityTimeout.Seconds()),
	)
	return err
}

// GetQueueStats gets statistics for a queue
func (c *PGMQClient) GetQueueStats(ctx context.Context, queue string) (*QueueStats, error) {
	stats := &QueueStats{QueueName: queue}

	err := c.pool.QueryRow(ctx,
		"SELECT queue_length FROM pgmq.queue_metrics($1)",
		queue,
	).Scan(&stats.TotalMessages)

	if err != nil {
		return nil, err
	}

	stats.PendingCount = stats.TotalMessages
	return stats, nil
}

// Ping checks the connection
func (c *PGMQClient) Ping(ctx context.Context) error {
	if c.pool == nil {
		return fmt.Errorf("connection pool is nil")
	}
	return c.pool.Ping(ctx)
}

// SendJSON sends a JSON message to a queue
func (c *PGMQClient) SendJSON(ctx context.Context, queue string, data interface{}, opts SendMessageOptions) (int64, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Send(ctx, queue, body, opts)
}

// ReadJSON reads a message and unmarshals as JSON
func (c *PGMQClient) ReadJSON(ctx context.Context, queue string, visibilityTimeout time.Duration, out interface{}) (*Message, error) {
	msg, err := c.Read(ctx, queue, visibilityTimeout)
	if err != nil || msg == nil {
		return msg, err
	}

	if err := json.Unmarshal(msg.Body, out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return msg, nil
}

// isQueueExistsError checks if the error indicates queue already exists
func isQueueExistsError(err error) bool {
	// PGMQ returns "relation ... already exists" error
	return err != nil && (contains(err.Error(), "already exists") || contains(err.Error(), "duplicate"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
