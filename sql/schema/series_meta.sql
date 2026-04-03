-- Series metadata table
-- Stores metadata for time series including endpoint, metric, and labels

CREATE TABLE IF NOT EXISTS series_meta (
    id BIGSERIAL PRIMARY KEY,
    endpoint TEXT NOT NULL,
    metric TEXT NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    labels_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT series_meta_endpoint_metric_labels_hash_key UNIQUE (endpoint, metric, labels_hash)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_series_meta_labels_hash ON series_meta(labels_hash);
CREATE INDEX IF NOT EXISTS idx_series_meta_metric ON series_meta(metric);
CREATE INDEX IF NOT EXISTS series_meta_endpoint_idx ON series_meta(endpoint);
CREATE INDEX IF NOT EXISTS series_meta_endpoint_metric_idx ON series_meta(endpoint, metric);

-- Comments for documentation
COMMENT ON TABLE series_meta IS 'Time series metadata including endpoint, metric, and labels';
COMMENT ON COLUMN series_meta.id IS 'Primary key';
COMMENT ON COLUMN series_meta.endpoint IS 'API endpoint or service name';
COMMENT ON COLUMN series_meta.metric IS 'Metric name (e.g., cpu_usage, memory_usage)';
COMMENT ON COLUMN series_meta.labels IS 'Key-value labels for dimensionality (JSONB)';
COMMENT ON COLUMN series_meta.labels_hash IS 'MD5 hash of labels for uniqueness constraint';
COMMENT ON COLUMN series_meta.created_at IS 'Record creation timestamp';