-- Series data points table
-- Stores time-series data points linked to series_meta
-- Designed for TimescaleDB hypertable

CREATE TABLE IF NOT EXISTS series_points (
    "time" TIMESTAMPTZ NOT NULL,
    series_id BIGINT NOT NULL REFERENCES series_meta(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS series_points_series_time_idx ON series_points(series_id, "time" DESC);
CREATE INDEX IF NOT EXISTS series_points_time_idx ON series_points("time" DESC);

-- Create TimescaleDB hypertable (requires TimescaleDB extension)
-- This will fail silently if TimescaleDB is not installed
SELECT create_hypertable('series_points', 'time', if_not_exists => TRUE);

-- Comments for documentation
COMMENT ON TABLE series_points IS 'Time series data points with timestamp and value';
COMMENT ON COLUMN series_points.time IS 'Timestamp of the data point';
COMMENT ON COLUMN series_points.series_id IS 'Foreign key to series_meta';
COMMENT ON COLUMN series_points.value IS 'Numeric value of the data point';