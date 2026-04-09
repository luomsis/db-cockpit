-- Slow query tracking table
-- Stores slow SQL query execution records for performance analysis

CREATE TABLE IF NOT EXISTS public.slow_query (
    id bigserial NOT NULL,
    endpoint text NULL,
    hostname text NULL,
    port int8 NULL,
    database_name text NULL,
    username text NULL,
    sql_text text NULL,
    execute_time numeric NULL,
    execute_date timestamptz NULL,
    CONSTRAINT slow_query_pkey PRIMARY KEY(id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_slow_query_host_port ON public.slow_query(hostname, port);
CREATE INDEX IF NOT EXISTS idx_slow_query_execute_date ON public.slow_query(execute_date DESC);
CREATE INDEX IF NOT EXISTS idx_slow_query_endpoint ON public.slow_query(endpoint);

-- Comments for documentation
COMMENT ON TABLE public.slow_query IS 'Slow SQL query execution records for performance analysis';
COMMENT ON COLUMN public.slow_query.id IS 'Primary key';
COMMENT ON COLUMN public.slow_query.endpoint IS 'Instance endpoint from instance_meta (hostname+port lookup)';
COMMENT ON COLUMN public.slow_query.hostname IS 'Host name where query was executed';
COMMENT ON COLUMN public.slow_query.port IS 'Database instance port';
COMMENT ON COLUMN public.slow_query.database_name IS 'Database name';
COMMENT ON COLUMN public.slow_query.username IS 'User who executed the query';
COMMENT ON COLUMN public.slow_query.sql_text IS 'SQL query text';
COMMENT ON COLUMN public.slow_query.execute_time IS 'Query execution time in seconds';
COMMENT ON COLUMN public.slow_query.execute_date IS 'Query execution timestamp';