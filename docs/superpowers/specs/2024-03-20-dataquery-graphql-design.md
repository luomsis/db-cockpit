# DataQuery GraphQL Query Interface Design

**Date:** 2024-03-20
**Status:** Draft

## Overview

Enhance the DataQuery service to support comprehensive time-series queries against TimescaleDB tables (`series_meta` + `series_points`). The implementation includes a PromQL-style label filter parser, basic time-based aggregations, and a GraphQL API.

## Goals

1. Query all distinct endpoints
2. Query metrics for a specific endpoint
3. Query time-series data with label filtering and time range
4. Support basic aggregations (avg, min, max, sum, count) over time buckets
5. Support PromQL-style label expression syntax
6. Provide test data insertion and validation scripts

## Non-Goals

- Data insertion via GraphQL (write operations)
- Advanced statistical functions (percentiles, standard deviation, rate calculations)
- Multi-tenancy at the query layer (assumed handled by middleware)

## Architecture

```
GraphQL Request
      │
      ▼
┌─────────────────┐
│  GraphQL Layer  │  schema.graphqls, resolvers
│  (gqlgen)       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Service Layer  │  DataQueryService interface
│                 │  Query orchestration
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Labels Parser   │────▶│  SQL Fragment   │
│                 │     │  Generator      │
└─────────────────┘     └─────────────────┘
         │
         ▼
┌─────────────────┐
│  Repository     │  PostgreSQL queries
│  (pgx/v5)       │  TimescaleDB functions
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   TimescaleDB   │  series_meta, series_points
└─────────────────┘
```

## Component Details

### 1. Label Expression Parser

**Supported Syntax:**
```
Operators:
  =    exact match          host="server1"
  !=   not equal            host!="server1"
  =~   regex match          region=~"us-.*"
  !~   regex not match      region!~"eu-.*"

Logical:
  AND, OR (case insensitive)
  ()     grouping

Examples:
  host="server1" AND region="us-east"
  host!="localhost" OR region=~"us-.*"
  (host="server1" OR host="server2") AND region="us-east"
```

**AST Structure:**
```go
type Expr interface {
    exprNode()
}

type BinaryExpr struct {
    Op       BinaryOp // AND, OR
    Left     Expr
    Right    Expr
}

type Comparison struct {
    Key      string
    Op       ComparisonOp // =, !=, =~, !~
    Value    string
}
```

**SQL Translation:**
| Expression | SQL Fragment |
|------------|--------------|
| `key="value"` | `labels->>'key' = 'value'` |
| `key!="value"` | `labels->>'key' != 'value'` |
| `key=~"pattern"` | `labels->>'key' ~ 'pattern'` |
| `key!~"pattern"` | `labels->>'key' !~ 'pattern'` |
| `a AND b` | `(a_sql) AND (b_sql)` |
| `a OR b` | `(a_sql) OR (b_sql)` |

### 2. GraphQL Schema

```graphql
input LabelFilter {
  expression: String!
}

input TimeRange {
  start: Time!
  end: Time!
}

input Aggregation {
  interval: String!    # "1m", "5m", "1h", "1d"
  function: AggFunction!
}

enum AggFunction {
  AVG
  MIN
  MAX
  SUM
  COUNT
}

type SeriesMeta {
  id: ID!
  endpoint: String!
  metric: String!
  labels: Labels!
  labelsHash: String!
  createdAt: Time!
}

type Labels {
  keys: [String!]!
  value(key: String!): String
  entries: [LabelEntry!]!
}

type LabelEntry {
  key: String!
  value: String!
}

type AggregatedPoint {
  time: Time!
  value: Float!
  count: Int!
}

type Series {
  meta: SeriesMeta!
  points: [DataPoint!]!
  aggregatedPoints(interval: String!, function: AggFunction!): [AggregatedPoint!]!
  statistics: SeriesStatistics
}

type DataPoint {
  time: Time!
  value: Float!
}

type SeriesStatistics {
  min: Float!
  max: Float!
  avg: Float!
  sum: Float!
  count: Int!
}

type Query {
  endpoints: [String!]!
  metrics(endpoint: String!): [String!]!
  series(
    endpoint: String
    metric: String
    labels: LabelFilter
    timeRange: TimeRange!
    limit: Int
  ): [Series!]!
  seriesById(id: ID!, timeRange: TimeRange!): Series
  seriesMulti(
    endpoints: [String!]
    metrics: [String!]
    labels: LabelFilter
    timeRange: TimeRange!
    aggregation: Aggregation
  ): [Series!]!
}

scalar Time
```

### 3. Repository Interface

```go
type Repository interface {
    GetEndpoints(ctx context.Context) ([]string, error)
    GetMetrics(ctx context.Context, endpoint string) ([]string, error)
    QuerySeries(ctx context.Context, req *SeriesQueryRequest) ([]SeriesMeta, error)
    GetSeriesPoints(ctx context.Context, req *PointsQueryRequest) (map[int64][]DataPoint, error)
    GetAggregatedPoints(ctx context.Context, req *AggregationRequest) (map[int64][]AggregatedPoint, error)
    GetSeriesStatistics(ctx context.Context, req *StatsRequest) (map[int64]*SeriesStatistics, error)
}

type SeriesQueryRequest struct {
    Endpoint    string
    Metric      string
    LabelFilter string  // raw expression
    TimeRange   TimeRange
    Limit       int
}

type PointsQueryRequest struct {
    SeriesIDs []int64
    TimeRange TimeRange
}

type AggregationRequest struct {
    SeriesIDs []int64
    TimeRange TimeRange
    Interval  string
    Function  string
}
```

### 4. Service Interface

```go
type DataQueryService interface {
    Name() string
    Initialize(ctx context.Context) error
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error

    GetEndpoints(ctx context.Context) ([]string, error)
    GetMetrics(ctx context.Context, endpoint string) ([]string, error)
    QuerySeries(ctx context.Context, req *SeriesQuery) ([]*SeriesData, error)
    QuerySeriesMulti(ctx context.Context, req *MultiSeriesQuery) ([]*SeriesData, error)
    GetSeriesByID(ctx context.Context, id int64, timeRange *TimeRange) (*SeriesData, error)
}

type SeriesData struct {
    Meta             SeriesMeta
    Points           []DataPoint
    AggregatedPoints []AggregatedPoint
    Statistics       *SeriesStatistics
}
```

### 5. Key SQL Queries

**GetEndpoints:**
```sql
SELECT DISTINCT endpoint FROM series_meta ORDER BY endpoint
```

**GetMetrics:**
```sql
SELECT DISTINCT metric FROM series_meta
WHERE endpoint = $1 ORDER BY metric
```

**QuerySeries:**
```sql
SELECT id, endpoint, metric, labels, labels_hash, created_at
FROM series_meta
WHERE ($1::text IS NULL OR endpoint = $1)
  AND ($2::text IS NULL OR metric = $2)
  AND ({label_filter_sql})
ORDER BY id
LIMIT $3
```

**GetAggregatedPoints:**
```sql
SELECT series_id,
       time_bucket($1, "time") AS bucket,
       {agg_func}(value) AS value,
       COUNT(*) AS count
FROM series_points
WHERE series_id = ANY($2)
  AND "time" >= $3 AND "time" <= $4
GROUP BY series_id, bucket
ORDER BY bucket
```

**GetSeriesStatistics:**
```sql
SELECT series_id,
       MIN(value), MAX(value), AVG(value), SUM(value), COUNT(*)
FROM series_points
WHERE series_id = ANY($1)
  AND "time" >= $2 AND "time" <= $3
GROUP BY series_id
```

## File Structure

```
pkg/domain/dataquery/
├── service.go              # Service interface (updated)
├── impl.go                 # Service implementation (new)
├── repository.go           # Repository interface (updated)
├── pg_repository.go        # PostgreSQL implementation (new)
├── models.go               # Domain models (new)
├── mock.go                 # Mock service (updated)
├── generate.go             # Code generation marker
├── gqlgen.yml              # gqlgen config (updated)
├── labels/
│   ├── parser.go           # Label expression parser
│   ├── ast.go              # AST types
│   └── sql.go              # SQL translation
└── graph/
    ├── schema.graphqls     # GraphQL schema (rewritten)
    ├── resolver.go         # Root resolver
    ├── schema.resolvers.go # Resolver implementations
    ├── generated.go        # gqlgen generated
    └── models_gen.go       # gqlgen generated

scripts/
├── insert_test_data.go     # Test data insertion
├── test_dataquery.sh       # GraphQL test script
└── graphql_helper.sh       # Helper functions
```

## Test Data

**Endpoints:**
- `/api/metrics`
- `/api/health`
- `/api/query`

**Metrics:**
- `/api/metrics`: cpu_usage, memory_usage, disk_io, network_bytes
- `/api/health`: response_time, status_code
- `/api/query`: query_count, query_latency

**Labels:**
```json
{"host": "server1", "region": "us-east", "env": "prod"}
{"host": "server2", "region": "us-east", "env": "prod"}
{"host": "server3", "region": "eu-west", "env": "prod"}
{"host": "server1", "region": "us-east", "env": "staging"}
```

**Data Points:**
- Time range: last 24 hours
- Interval: every 5 minutes
- ~288 points per series
- ~10 series total

## Test Cases

1. Query all endpoints
2. Query metrics for specific endpoint
3. Query series with time range only
4. Query series with exact label match (`host="server1"`)
5. Query series with regex label match (`region=~"us-.*"`)
6. Query series with AND/OR combinations
7. Query series with aggregation
8. Query series by ID
9. Query multiple metrics at once
10. Query with statistics calculation

## Dependencies

- `github.com/99designs/gqlgen` - GraphQL code generation
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- TimescaleDB extension (for `time_bucket` function)

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Label parser complexity | Start with core operators, extend incrementally |
| SQL injection via label expressions | Use parameterized queries, validate parsed AST |
| Performance on large datasets | Leverage TimescaleDB indexes, add query limits |
| Regex DoS | Validate regex patterns, add timeout |