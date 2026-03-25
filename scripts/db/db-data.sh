#!/bin/bash

# Database Intelligent Cockpit - Data Management Script
# Usage: ./scripts/db/db-data.sh [clear|seed|reset|status]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-postgres}"
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if PostgreSQL is running
check_postgres() {
    if docker ps --format '{{.Names}}' | grep -q "db-cockpit-postgres"; then
        return 0
    else
        # Try to connect via psql
        if pg_isready -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} >/dev/null 2>&1; then
            return 0
        fi
        return 1
    fi
}

# Execute SQL command
exec_sql() {
    local sql="$1"
    docker exec db-cockpit-postgres psql -U ${DB_USER} -d ${DB_NAME} -c "$sql" 2>/dev/null || \
    psql "${DATABASE_URL}" -c "$sql" 2>/dev/null
}

# Execute SQL file
exec_sql_file() {
    local file="$1"
    docker exec -i db-cockpit-postgres psql -U ${DB_USER} -d ${DB_NAME} < "$file" 2>/dev/null || \
    psql "${DATABASE_URL}" -f "$file" 2>/dev/null
}

# Clear all data
clear_data() {
    log_info "Clearing database data..."

    exec_sql "TRUNCATE TABLE series_points CASCADE;"
    exec_sql "TRUNCATE TABLE series_meta CASCADE;"

    log_success "All data cleared!"
}

# Show data status
show_status() {
    log_info "Checking database status..."

    echo ""
    echo "========================================"
    echo "  Database Data Status"
    echo "========================================"
    echo ""

    # Table counts
    echo "Table counts:"
    exec_sql "
        SELECT 'series_meta' as table_name, COUNT(*) as count FROM series_meta
        UNION ALL
        SELECT 'series_points', COUNT(*) FROM series_points;
    " 2>/dev/null || echo "  Tables not found or empty"

    echo ""

    # Sample series_meta
    echo "Sample series_meta (first 5):"
    exec_sql "SELECT id, endpoint, metric, labels FROM series_meta LIMIT 5;" 2>/dev/null || echo "  No data"

    echo ""

    # Endpoints
    echo "Available endpoints:"
    exec_sql "SELECT DISTINCT endpoint FROM series_meta ORDER BY endpoint;" 2>/dev/null || echo "  No endpoints"

    echo ""

    # Metrics per endpoint
    echo "Metrics per endpoint:"
    exec_sql "
        SELECT endpoint, string_agg(metric, ', ' ORDER BY metric) as metrics
        FROM (SELECT DISTINCT endpoint, metric FROM series_meta) t
        GROUP BY endpoint
        ORDER BY endpoint;
    " 2>/dev/null || echo "  No metrics"

    echo ""
}

# Seed test data
seed_data() {
    log_info "Seeding test data..."

    cd "${PROJECT_ROOT}"

    # Run the Go seed script
    DATABASE_URL="${DATABASE_URL}" go run scripts/db/insert_test_data.go

    log_success "Test data seeded!"
}

# Reset data (clear + seed)
reset_data() {
    log_info "Resetting database data..."
    clear_data
    seed_data
    show_status
}

# Check tables exist
check_tables() {
    log_info "Checking tables..."

    local result=$(exec_sql "
        SELECT EXISTS (
            SELECT FROM information_schema.tables
            WHERE table_name = 'series_meta'
        ) AND EXISTS (
            SELECT FROM information_schema.tables
            WHERE table_name = 'series_points'
        );
    " 2>/dev/null | grep -c 't' || echo "0")

    if [ "${result}" -eq 1 ]; then
        return 0
    else
        return 1
    fi
}

# Ensure tables exist
ensure_tables() {
    if check_tables; then
        log_info "Tables already exist"
        return 0
    fi

    log_info "Creating tables..."

    # Use Go script to ensure tables (it has EnsureTables function)
    cd "${PROJECT_ROOT}"
    DATABASE_URL="${DATABASE_URL}" go run -exec 'echo' scripts/db/insert_test_data.go 2>/dev/null || true

    # Create tables directly via SQL
    exec_sql "
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
    "

    exec_sql "
        CREATE TABLE IF NOT EXISTS series_points (
            \"time\" TIMESTAMPTZ NOT NULL,
            series_id BIGINT NOT NULL REFERENCES series_meta(id),
            value DOUBLE PRECISION NOT NULL
        );
        CREATE INDEX IF NOT EXISTS series_points_series_time_idx ON series_points(series_id, \"time\" DESC);
        CREATE INDEX IF NOT EXISTS series_points_time_idx ON series_points(\"time\" DESC);
    "

    # Try to create hypertable (will fail if TimescaleDB not installed, which is OK)
    exec_sql "SELECT create_hypertable('series_points', 'time', if_not_exists => TRUE);" 2>/dev/null || true

    log_success "Tables created!"
}

# Quick test
quick_test() {
    log_info "Running quick REST API tests..."

    echo ""
    echo "Testing endpoints query..."
    curl -s http://localhost:8084/api/v1/endpoints | python3 -m json.tool 2>/dev/null || echo "Failed to connect to DataQuery service"

    echo ""
    echo "Testing metrics query..."
    curl -s "http://localhost:8084/api/v1/metrics?endpoint=/api/metrics" | python3 -m json.tool 2>/dev/null || echo "Failed"

    echo ""
}

# Main command
case "$1" in
    clear)
        clear_data
        ;;
    seed)
        seed_data
        ;;
    reset)
        reset_data
        ;;
    status)
        show_status
        ;;
    ensure-tables)
        ensure_tables
        ;;
    test)
        quick_test
        ;;
    *)
        echo "Database Intelligent Cockpit - Data Management"
        echo ""
        echo "Usage: $0 {clear|seed|reset|status|ensure-tables|test}"
        echo ""
        echo "Commands:"
        echo "  clear          Clear all data from tables"
        echo "  seed           Insert test data"
        echo "  reset          Clear and re-seed data"
        echo "  status         Show database data status"
        echo "  ensure-tables  Create tables if they don't exist"
        echo "  test           Run quick REST API tests"
        echo ""
        echo "Environment variables:"
        echo "  DB_HOST        Database host (default: localhost)"
        echo "  DB_PORT        Database port (default: 5432)"
        echo "  DB_USER        Database user (default: postgres)"
        echo "  DB_PASSWORD    Database password (default: postgres)"
        echo "  DB_NAME        Database name (default: postgres)"
        echo ""
        ;;
esac