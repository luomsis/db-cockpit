# Database Schema

This directory contains DDL files for database tables used in the project.

## Tables

| File | Table | Description |
|------|-------|-------------|
| [instance_meta.sql](instance_meta.sql) | `instance_meta` | Database instance metadata |
| [series_meta.sql](series_meta.sql) | `series_meta` | Time series metadata |
| [series_points.sql](series_points.sql) | `series_points` | Time series data points |

## Table Relationships

```
instance_meta (standalone metadata table)

series_meta ←── series_points
    1              ∞
```

## Usage

Apply individual schema:

```bash
psql -h localhost -U postgres -d your_database -f instance_meta.sql
```

Apply all schemas:

```bash
psql -h localhost -U postgres -d your_database -f scripts/db/schema/*.sql
```

Or use the data management script:

```bash
./scripts/db/db-data.sh ensure-tables
```

## Requirements

- PostgreSQL 12+
- TimescaleDB extension (optional, for hypertable support on `series_points`)