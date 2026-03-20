package dataquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/db-cockpit/pkg/domain/dataquery/labels"
)

// PGRepository implements Repository using PostgreSQL/TimescaleDB
type PGRepository struct {
	pool *pgxpool.Pool
}

// NewPGRepository creates a new PostgreSQL repository
func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// GetEndpoints retrieves all distinct endpoints
func (r *PGRepository) GetEndpoints(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT endpoint FROM series_meta ORDER BY endpoint
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []string
	for rows.Next() {
		var endpoint string
		if err := rows.Scan(&endpoint); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		endpoints = append(endpoints, endpoint)
	}

	if endpoints == nil {
		endpoints = []string{}
	}
	return endpoints, nil
}

// GetMetrics retrieves all distinct metrics for an endpoint
func (r *PGRepository) GetMetrics(ctx context.Context, endpoint string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT metric FROM series_meta
		WHERE endpoint = $1
		ORDER BY metric
	`, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []string
	for rows.Next() {
		var metric string
		if err := rows.Scan(&metric); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}
		metrics = append(metrics, metric)
	}

	if metrics == nil {
		metrics = []string{}
	}
	return metrics, nil
}

// QuerySeries queries series metadata based on filters
func (r *PGRepository) QuerySeries(ctx context.Context, req *SeriesQueryRequest) ([]SeriesMeta, error) {
	query := `
		SELECT id, endpoint, metric, labels, labels_hash, created_at
		FROM series_meta
		WHERE ($1::text IS NULL OR endpoint = $1)
		  AND ($2::text IS NULL OR metric = $2)
	`

	args := []interface{}{nullIfEmpty(req.Endpoint), nullIfEmpty(req.Metric)}

	// Parse and apply label filter
	if req.LabelFilter != "" {
		expr, err := labels.Parse(req.LabelFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to parse label filter: %w", err)
		}
		if err := labels.Validate(expr); err != nil {
			return nil, fmt.Errorf("invalid label filter: %w", err)
		}
		sqlFragment, err := labels.ToSQL(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert label filter to SQL: %w", err)
		}
		query += fmt.Sprintf(" AND (%s)", sqlFragment)
	}

	query += " ORDER BY id"

	if req.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", req.Limit)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query series: %w", err)
	}
	defer rows.Close()

	var series []SeriesMeta
	for rows.Next() {
		var s SeriesMeta
		var labelsJSON []byte
		if err := rows.Scan(&s.ID, &s.Endpoint, &s.Metric, &labelsJSON, &s.LabelsHash, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}
		if err := json.Unmarshal(labelsJSON, &s.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
		series = append(series, s)
	}

	if series == nil {
		series = []SeriesMeta{}
	}
	return series, nil
}

// GetSeriesByID retrieves series metadata by ID
func (r *PGRepository) GetSeriesByID(ctx context.Context, id int64) (*SeriesMeta, error) {
	var s SeriesMeta
	var labelsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, endpoint, metric, labels, labels_hash, created_at
		FROM series_meta WHERE id = $1
	`, id).Scan(&s.ID, &s.Endpoint, &s.Metric, &labelsJSON, &s.LabelsHash, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get series by ID: %w", err)
	}
	if err := json.Unmarshal(labelsJSON, &s.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}
	return &s, nil
}

// GetSeriesPoints retrieves data points for multiple series
func (r *PGRepository) GetSeriesPoints(ctx context.Context, req *PointsQueryRequest) (map[int64][]DataPoint, error) {
	if len(req.SeriesIDs) == 0 {
		return map[int64][]DataPoint{}, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT series_id, "time", value
		FROM series_points
		WHERE series_id = ANY($1)
		  AND "time" >= $2 AND "time" <= $3
		ORDER BY series_id, "time"
	`, req.SeriesIDs, req.TimeRange.Start, req.TimeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to query points: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]DataPoint)
	for rows.Next() {
		var seriesID int64
		var point DataPoint
		if err := rows.Scan(&seriesID, &point.Time, &point.Value); err != nil {
			return nil, fmt.Errorf("failed to scan point: %w", err)
		}
		result[seriesID] = append(result[seriesID], point)
	}

	return result, nil
}

// GetAggregatedPoints retrieves aggregated data points for multiple series
func (r *PGRepository) GetAggregatedPoints(ctx context.Context, req *AggregationRequest) (map[int64][]AggregatedPoint, error) {
	if len(req.SeriesIDs) == 0 {
		return map[int64][]AggregatedPoint{}, nil
	}

	aggFunc := strings.ToUpper(req.Function)
	switch aggFunc {
	case "AVG", "MIN", "MAX", "SUM", "COUNT":
		// valid
	default:
		aggFunc = "AVG"
	}

	query := fmt.Sprintf(`
		SELECT series_id,
			   time_bucket('%s', "time") AS bucket,
			   %s(value) AS value,
			   COUNT(*) AS count
		FROM series_points
		WHERE series_id = ANY($1)
		  AND "time" >= $2 AND "time" <= $3
		GROUP BY series_id, bucket
		ORDER BY bucket
	`, req.Interval, aggFunc)

	rows, err := r.pool.Query(ctx, query, req.SeriesIDs, req.TimeRange.Start, req.TimeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated points: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]AggregatedPoint)
	for rows.Next() {
		var seriesID int64
		var point AggregatedPoint
		if err := rows.Scan(&seriesID, &point.Time, &point.Value, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan aggregated point: %w", err)
		}
		result[seriesID] = append(result[seriesID], point)
	}

	return result, nil
}

// GetSeriesStatistics retrieves statistics for multiple series
func (r *PGRepository) GetSeriesStatistics(ctx context.Context, req *StatsRequest) (map[int64]*SeriesStatistics, error) {
	if len(req.SeriesIDs) == 0 {
		return map[int64]*SeriesStatistics{}, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT series_id,
			   MIN(value), MAX(value), AVG(value), SUM(value), COUNT(*)
		FROM series_points
		WHERE series_id = ANY($1)
		  AND "time" >= $2 AND "time" <= $3
		GROUP BY series_id
	`, req.SeriesIDs, req.TimeRange.Start, req.TimeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to query statistics: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]*SeriesStatistics)
	for rows.Next() {
		var seriesID int64
		var stats SeriesStatistics
		if err := rows.Scan(&seriesID, &stats.Min, &stats.Max, &stats.Avg, &stats.Sum, &stats.Count); err != nil {
			return nil, fmt.Errorf("failed to scan statistics: %w", err)
		}
		result[seriesID] = &stats
	}

	return result, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// stringsToUpper is not used, removing

// InsertSeriesMeta inserts a new series metadata record
func (r *PGRepository) InsertSeriesMeta(ctx context.Context, endpoint, metric string, lbls map[string]string) (*SeriesMeta, error) {
	labelsJSON, err := json.Marshal(lbls)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}

	// Convert to string for PostgreSQL
	labelsStr := string(labelsJSON)

	var s SeriesMeta
	var returnedLabels []byte
	err = r.pool.QueryRow(ctx, `
		INSERT INTO series_meta (endpoint, metric, labels, labels_hash, created_at)
		VALUES ($1, $2, $3::jsonb, md5($3::text), NOW())
		RETURNING id, endpoint, metric, labels, labels_hash, created_at
	`, endpoint, metric, labelsStr).Scan(&s.ID, &s.Endpoint, &s.Metric, &returnedLabels, &s.LabelsHash, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert series meta: %w", err)
	}

	if err := json.Unmarshal(returnedLabels, &s.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	return &s, nil
}

// InsertPoints inserts data points for a series
func (r *PGRepository) InsertPoints(ctx context.Context, seriesID int64, points []DataPoint) error {
	if len(points) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(points))
	for i, p := range points {
		// Column order: time, series_id, value (matching the DDL)
		rows[i] = []interface{}{p.Time, seriesID, p.Value}
	}

	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"series_points"},
		[]string{"time", "series_id", "value"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("failed to insert points: %w", err)
	}

	return nil
}

// EnsureTables creates the necessary tables if they don't exist
func (r *PGRepository) EnsureTables(ctx context.Context) error {
	// Create series_meta table (matching the provided DDL)
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS series_meta (
			id BIGSERIAL PRIMARY KEY,
			endpoint TEXT NOT NULL,
			metric TEXT NOT NULL,
			labels JSONB NOT NULL DEFAULT '{}'::jsonb,
			labels_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT series_meta_endpoint_metric_labels_hash_key UNIQUE (endpoint, metric, labels_hash)
		);
		CREATE INDEX IF NOT EXISTS idx_series_meta_labels_hash ON series_meta(labels_hash);
		CREATE INDEX IF NOT EXISTS idx_series_meta_metric ON series_meta(metric);
		CREATE INDEX IF NOT EXISTS series_meta_endpoint_idx ON series_meta(endpoint);
		CREATE INDEX IF NOT EXISTS series_meta_endpoint_metric_idx ON series_meta(endpoint, metric);
	`)
	if err != nil {
		return fmt.Errorf("failed to create series_meta table: %w", err)
	}

	// Create series_points table (matching the provided DDL)
	_, err = r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS series_points (
			"time" TIMESTAMPTZ NOT NULL,
			series_id BIGINT NOT NULL REFERENCES series_meta(id),
			value DOUBLE PRECISION NOT NULL
		);
		CREATE INDEX IF NOT EXISTS series_points_series_time_idx ON series_points(series_id, "time" DESC);
		CREATE INDEX IF NOT EXISTS series_points_time_idx ON series_points("time" DESC);
	`)
	if err != nil {
		return fmt.Errorf("failed to create series_points table: %w", err)
	}

	// Try to create hypertable (will fail if TimescaleDB is not installed, which is OK)
	_, _ = r.pool.Exec(ctx, `
		SELECT create_hypertable('series_points', 'time', if_not_exists => TRUE);
	`)

	return nil
}
