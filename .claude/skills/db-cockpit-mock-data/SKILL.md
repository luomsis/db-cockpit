---
name: db-cockpit-mock-data
description: Use when setting up test data for db-cockpit, regenerating mock time-series/alert/slow-query data, or resetting database to known state for testing. Supports configurable time ranges (7/15/30 days).
---

# Mock Data Generation for db-cockpit

## Overview

Generate realistic mock data for testing: time-series points, alerts, and slow queries based on actual instance metadata. Supports configurable time ranges.

## When to Use

- Setting up fresh test environment
- Regenerating data after schema changes
- Resetting database to known state
- Testing data query API endpoints

## Prerequisites

- TimescaleDB container running (`docker ps | grep db-cockpit-postgres`)
- Database connection: `localhost:5432`, user `postgres`, db `postgres`
- Python 3 with psycopg2: `pip3 install psycopg2-binary`

## Quick Reference

| Time Range | Command | Expected Data |
|------------|---------|---------------|
| 1 week | `python3 scripts/data/generate_mock_data.py --days 7` | ~1.6M points |
| 15 days | `python3 scripts/data/generate_mock_data.py --days 15` | ~3.4M points |
| 1 month | `python3 scripts/data/generate_mock_data.py --days 30` | ~6.8M points |
| Custom | `python3 scripts/data/generate_mock_data.py --start-date 2026-03-01 --end-date 2026-03-31` | varies |

## Implementation

### Step 1: Generate SQL Files

```bash
# Default: 7 days
python3 scripts/data/generate_mock_data.py

# 15 days (half month)
python3 scripts/data/generate_mock_data.py --days 15

# 30 days (one month)
python3 scripts/data/generate_mock_data.py --days 30

# Custom date range
python3 scripts/data/generate_mock_data.py --start-date 2026-03-01 --end-date 2026-03-31
```

Output files:
- `cleanup_data.sql` - Clears existing data
- `series_points_insert.sql` - Time-series points
- `alert_insert.sql` - Alert records
- `slow_query_insert.sql` - Slow query records

### Step 2: Execute Cleanup

```bash
PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f scripts/data/cleanup_data.sql
```

### Step 3: Insert Data (in order)

```bash
# Series points (largest, most time)
PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f scripts/data/series_points_insert.sql

# Alerts
PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f scripts/data/alert_insert.sql

# Slow queries
PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f scripts/data/slow_query_insert.sql
```

### Step 4: Verify Data

```bash
# Check counts
PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -c "
SELECT 'series_points' as table_name, COUNT(*) FROM series_points
UNION ALL SELECT 'alert', COUNT(*) FROM alert
UNION ALL SELECT 'slow_query', COUNT(*) FROM slow_query;
"

# Check time range
PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -c "
SELECT MIN(time), MAX(time) FROM series_points;
"
```

## Data Characteristics

| Metric | MySQL | PostgreSQL | Oracle | Redis |
|--------|-------|------------|--------|-------|
| cpu_usage_percent | 15-75% | 10-65% | 20-80% | 5-40% |
| memory_usage_percent | 60-85% | 50-80% | 70-90% | 80-95% |
| queries_per_second | 100-800 | 50-400 | 30-200 | 5000-50000 |

- **Time pattern**: Business hours higher, night lower, weekends reduced
- **Environment pattern**: prod more alerts/slow-queries, dev fewer
- **Sampling interval**: 1 minute

## Common Issues

| Issue | Fix |
|-------|-----|
| psycopg2 not found | `pip3 install psycopg2-binary` |
| Connection refused | Check Docker container running |
| Slow insert | Expected for large datasets, ~2 min per million points |
| SQL syntax error | Re-run Python script (regenerates clean SQL) |
| Memory error (30 days) | Script generates in batches; if fails, try smaller range |

## Files

- Generator script: `scripts/data/generate_mock_data.py`
- Schema: `sql/schema/*.sql`
- Models: `pkg/domain/dataquery/models.go`