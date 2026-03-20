package collector

import (
	"context"
	"sync"
	"time"

	"github.com/db-cockpit/pkg/common/utils"
	"github.com/db-cockpit/pkg/data"
	"github.com/db-cockpit/pkg/data/timescaledb"
)

// SourceType represents the type of data source
type SourceType string

const (
	SourceTypeDatabase    SourceType = "database"
	SourceTypeServer      SourceType = "server"
	SourceTypeApplication SourceType = "application"
)

// MetricData represents a metric data point
type MetricData struct {
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
	Tags      map[string]string
	Fields    map[string]float64
}

// LogData represents a log entry
type LogData struct {
	Timestamp time.Time
	Level     string
	Message   string
	Logger    string
	Fields    map[string]string
	TraceID   string
	SpanID    string
}

// EventData represents an event
type EventData struct {
	EventID    string
	EventType  string
	Timestamp  time.Time
	Severity   string
	Source     string
	Message    string
	Attributes map[string]string
}

// CollectorConfig represents collector configuration
type CollectorConfig struct {
	CollectorID   string
	Name          string
	Type          SourceType
	Version       string
	BatchSize     int
	FlushInterval time.Duration
}

// SourceConfig represents a data source configuration
type SourceConfig struct {
	SourceID   string
	SourceType SourceType
	Endpoint   string
	Interval   time.Duration
	Enabled    bool
	Config     map[string]string
}

// MetricBatch represents a batch of metrics
type MetricBatch struct {
	SourceID string
	Metrics  []MetricData
}

// LogBatch represents a batch of logs
type LogBatch struct {
	SourceID string
	Logs     []LogData
}

// EventBatch represents a batch of events
type EventBatch struct {
	SourceID string
	Events   []EventData
}

// CollectorService defines the collector service interface
type CollectorService interface {
	// Start starts the collector
	Start(ctx context.Context) error

	// Stop stops the collector
	Stop(ctx context.Context) error

	// RegisterSource registers a data source
	RegisterSource(ctx context.Context, source *SourceConfig) error

	// UnregisterSource unregisters a data source
	UnregisterSource(ctx context.Context, sourceID string) error

	// CollectMetrics collects metrics from sources
	CollectMetrics(ctx context.Context, batch *MetricBatch) error

	// CollectLogs collects logs from sources
	CollectLogs(ctx context.Context, batch *LogBatch) error

	// CollectEvents collects events from sources
	CollectEvents(ctx context.Context, batch *EventBatch) error

	// GetStatus returns collector status
	GetStatus(ctx context.Context) (*CollectorStatus, error)
}

// CollectorStatus represents collector status
type CollectorStatus struct {
	CollectorID      string
	Status           string
	LastHeartbeat    time.Time
	MetricsCollected int64
	LogsCollected    int64
	EventsCollected  int64
	Errors           []string
}

// DataWriter defines the interface for writing data
type DataWriter interface {
	// WriteMetrics writes metrics to TSDB
	WriteMetrics(ctx context.Context, tenantID string, metrics []MetricData) error

	// WriteLogs writes logs to storage
	WriteLogs(ctx context.Context, tenantID string, logs []LogData) error

	// WriteEvents writes events to storage
	WriteEvents(ctx context.Context, tenantID string, events []EventData) error
}

// Collector implements the CollectorService interface
type Collector struct {
	config     *CollectorConfig
	dataWriter DataWriter
	sources    map[string]*SourceConfig
	sourcesMux sync.RWMutex
	metricsCh  chan *MetricBatch
	logsCh     chan *LogBatch
	eventsCh   chan *EventBatch
	stopCh     chan struct{}
	running    bool
	runningMux sync.Mutex

	// Statistics
	metricsCount int64
	logsCount    int64
	eventsCount  int64
	statsMux     sync.Mutex
}

// NewCollector creates a new collector instance
func NewCollector(config *CollectorConfig, dataWriter DataWriter) *Collector {
	if config.BatchSize <= 0 {
		config.BatchSize = 1000
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	return &Collector{
		config:     config,
		dataWriter: dataWriter,
		sources:    make(map[string]*SourceConfig),
		metricsCh:  make(chan *MetricBatch, 10000),
		logsCh:     make(chan *LogBatch, 10000),
		eventsCh:   make(chan *EventBatch, 10000),
		stopCh:     make(chan struct{}),
	}
}

// Start starts the collector
func (c *Collector) Start(ctx context.Context) error {
	c.runningMux.Lock()
	defer c.runningMux.Unlock()

	if c.running {
		return nil
	}

	c.running = true

	// Start workers
	go c.metricsWorker(ctx)
	go c.logsWorker(ctx)
	go c.eventsWorker(ctx)
	go c.heartbeatWorker(ctx)

	return nil
}

// Stop stops the collector
func (c *Collector) Stop(ctx context.Context) error {
	c.runningMux.Lock()
	defer c.runningMux.Unlock()

	if !c.running {
		return nil
	}

	close(c.stopCh)
	c.running = false

	return nil
}

// RegisterSource registers a data source
func (c *Collector) RegisterSource(ctx context.Context, source *SourceConfig) error {
	c.sourcesMux.Lock()
	defer c.sourcesMux.Unlock()

	source.SourceID = utils.GenerateID()
	c.sources[source.SourceID] = source

	return nil
}

// UnregisterSource unregisters a data source
func (c *Collector) UnregisterSource(ctx context.Context, sourceID string) error {
	c.sourcesMux.Lock()
	defer c.sourcesMux.Unlock()

	delete(c.sources, sourceID)

	return nil
}

// CollectMetrics collects metrics from sources
func (c *Collector) CollectMetrics(ctx context.Context, batch *MetricBatch) error {
	select {
	case c.metricsCh <- batch:
		c.statsMux.Lock()
		c.metricsCount += int64(len(batch.Metrics))
		c.statsMux.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CollectLogs collects logs from sources
func (c *Collector) CollectLogs(ctx context.Context, batch *LogBatch) error {
	select {
	case c.logsCh <- batch:
		c.statsMux.Lock()
		c.logsCount += int64(len(batch.Logs))
		c.statsMux.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CollectEvents collects events from sources
func (c *Collector) CollectEvents(ctx context.Context, batch *EventBatch) error {
	select {
	case c.eventsCh <- batch:
		c.statsMux.Lock()
		c.eventsCount += int64(len(batch.Events))
		c.statsMux.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetStatus returns collector status
func (c *Collector) GetStatus(ctx context.Context) (*CollectorStatus, error) {
	c.statsMux.Lock()
	defer c.statsMux.Unlock()

	status := "running"
	if !c.running {
		status = "stopped"
	}

	return &CollectorStatus{
		CollectorID:      c.config.CollectorID,
		Status:           status,
		LastHeartbeat:    time.Now(),
		MetricsCollected: c.metricsCount,
		LogsCollected:    c.logsCount,
		EventsCollected:  c.eventsCount,
	}, nil
}

// metricsWorker processes metrics batches
func (c *Collector) metricsWorker(ctx context.Context) {
	batch := make([]MetricData, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			// Flush remaining metrics
			if len(batch) > 0 {
				_ = c.dataWriter.WriteMetrics(ctx, "default", batch)
			}
			return
		case <-ticker.C:
			// Flush batch
			if len(batch) > 0 {
				_ = c.dataWriter.WriteMetrics(ctx, "default", batch)
				batch = batch[:0]
			}
		case mb := <-c.metricsCh:
			batch = append(batch, mb.Metrics...)
			if len(batch) >= c.config.BatchSize {
				_ = c.dataWriter.WriteMetrics(ctx, "default", batch)
				batch = batch[:0]
			}
		}
	}
}

// logsWorker processes logs batches
func (c *Collector) logsWorker(ctx context.Context) {
	batch := make([]LogData, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			if len(batch) > 0 {
				_ = c.dataWriter.WriteLogs(ctx, "default", batch)
			}
			return
		case <-ticker.C:
			if len(batch) > 0 {
				_ = c.dataWriter.WriteLogs(ctx, "default", batch)
				batch = batch[:0]
			}
		case lb := <-c.logsCh:
			batch = append(batch, lb.Logs...)
			if len(batch) >= c.config.BatchSize {
				_ = c.dataWriter.WriteLogs(ctx, "default", batch)
				batch = batch[:0]
			}
		}
	}
}

// eventsWorker processes events batches
func (c *Collector) eventsWorker(ctx context.Context) {
	batch := make([]EventData, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			if len(batch) > 0 {
				_ = c.dataWriter.WriteEvents(ctx, "default", batch)
			}
			return
		case <-ticker.C:
			if len(batch) > 0 {
				_ = c.dataWriter.WriteEvents(ctx, "default", batch)
				batch = batch[:0]
			}
		case eb := <-c.eventsCh:
			batch = append(batch, eb.Events...)
			if len(batch) >= c.config.BatchSize {
				_ = c.dataWriter.WriteEvents(ctx, "default", batch)
				batch = batch[:0]
			}
		}
	}
}

// heartbeatWorker sends periodic heartbeats
func (c *Collector) heartbeatWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			// Send heartbeat
		}
	}
}

// DataLayerWriter implements DataWriter using data layer
type DataLayerWriter struct {
	dataLayer *data.DataLayer
}

// NewDataLayerWriter creates a new data layer writer
func NewDataLayerWriter(dataLayer *data.DataLayer) *DataLayerWriter {
	return &DataLayerWriter{
		dataLayer: dataLayer,
	}
}

// WriteMetrics writes metrics to TSDB
func (w *DataLayerWriter) WriteMetrics(ctx context.Context, tenantID string, metrics []MetricData) error {
	// Convert to TimescaleDB format
	tsMetrics := make([]timescaledb.MetricPoint, len(metrics))
	for i, m := range metrics {
		tsMetrics[i] = timescaledb.MetricPoint{
			Name:      m.Name,
			Value:     m.Value,
			Timestamp: m.Timestamp,
			Tags:      m.Tags,
			Fields:    m.Fields,
		}
	}

	return w.dataLayer.TimescaleDB.InsertMetrics(ctx, tsMetrics)
}

// WriteLogs writes logs to storage
func (w *DataLayerWriter) WriteLogs(ctx context.Context, tenantID string, logs []LogData) error {
	// TODO: Implement log storage
	// Logs can be stored in TimescaleDB or a dedicated log store
	return nil
}

// WriteEvents writes events to storage
func (w *DataLayerWriter) WriteEvents(ctx context.Context, tenantID string, events []EventData) error {
	// TODO: Implement event storage
	return nil
}
