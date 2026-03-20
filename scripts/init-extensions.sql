-- Initialize all required extensions

-- TimescaleDB for time-series data
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- pgvector for vector similarity search
CREATE EXTENSION IF NOT EXISTS vector CASCADE;

-- PGMQ for message queue
CREATE EXTENSION IF NOT EXISTS pgmq CASCADE;

-- Create default queue for tasks
SELECT pgmq.create('tasks');

-- Verify all extensions are installed
SELECT extname, extversion FROM pg_extension
WHERE extname IN ('timescaledb', 'vector', 'pgmq');