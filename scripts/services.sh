#!/bin/bash

# Database Intelligent Cockpit - Services Management Script
# Usage: ./scripts/services.sh [start|stop|restart|status|logs]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GATEWAY_PORT=8080
DATAQUERY_PORT=8084
FRONTEND_PORT=3000

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="/tmp/db-cockpit"
PID_DIR="${PROJECT_ROOT}/.pids"

# Create necessary directories
mkdir -p "${LOG_DIR}" "${PID_DIR}"

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if a port is in use
check_port() {
    local port=$1
    if lsof -i :${port} -t >/dev/null 2>&1; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Start Data Query Service
start_dataquery() {
    if check_port ${DATAQUERY_PORT}; then
        log_info "Data Query Service already running on port ${DATAQUERY_PORT}"
        return 0
    fi

    log_info "Starting Data Query Service..."
    cd "${PROJECT_ROOT}"
    nohup go run cmd/dataquery/main.go > "${LOG_DIR}/dataquery.log" 2>&1 &
    echo $! > "${PID_DIR}/dataquery.pid"

    sleep 2
    if check_port ${DATAQUERY_PORT}; then
        log_success "Data Query Service started on port ${DATAQUERY_PORT}"
    else
        log_error "Failed to start Data Query Service"
        return 1
    fi
}

# Stop Data Query Service
stop_dataquery() {
    if [ -f "${PID_DIR}/dataquery.pid" ]; then
        local pid=$(cat "${PID_DIR}/dataquery.pid")
        if kill -0 ${pid} 2>/dev/null; then
            log_info "Stopping Data Query Service (PID: ${pid})..."
            kill ${pid} 2>/dev/null || true
            rm -f "${PID_DIR}/dataquery.pid"
        fi
    fi

    # Also kill any process on the port
    local pids=$(lsof -t -i :${DATAQUERY_PORT} 2>/dev/null || true)
    if [ -n "${pids}" ]; then
        log_info "Killing processes on port ${DATAQUERY_PORT}..."
        echo "${pids}" | xargs kill 2>/dev/null || true
    fi

    log_success "Data Query Service stopped"
}

# Start Gateway Service
start_gateway() {
    if check_port ${GATEWAY_PORT}; then
        log_info "Gateway Service already running on port ${GATEWAY_PORT}"
        return 0
    fi

    log_info "Starting Gateway Service..."
    cd "${PROJECT_ROOT}"
    nohup go run cmd/gateway/main.go > "${LOG_DIR}/gateway.log" 2>&1 &
    echo $! > "${PID_DIR}/gateway.pid"

    sleep 2
    if check_port ${GATEWAY_PORT}; then
        log_success "Gateway Service started on port ${GATEWAY_PORT}"
    else
        log_error "Failed to start Gateway Service"
        return 1
    fi
}

# Stop Gateway Service
stop_gateway() {
    if [ -f "${PID_DIR}/gateway.pid" ]; then
        local pid=$(cat "${PID_DIR}/gateway.pid")
        if kill -0 ${pid} 2>/dev/null; then
            log_info "Stopping Gateway Service (PID: ${pid})..."
            kill ${pid} 2>/dev/null || true
            rm -f "${PID_DIR}/gateway.pid"
        fi
    fi

    # Also kill any process on the port
    local pids=$(lsof -t -i :${GATEWAY_PORT} 2>/dev/null || true)
    if [ -n "${pids}" ]; then
        log_info "Killing processes on port ${GATEWAY_PORT}..."
        echo "${pids}" | xargs kill 2>/dev/null || true
    fi

    log_success "Gateway Service stopped"
}

# Start Frontend
start_frontend() {
    if check_port ${FRONTEND_PORT}; then
        log_info "Frontend already running on port ${FRONTEND_PORT}"
        return 0
    fi

    log_info "Starting Frontend..."
    cd "${PROJECT_ROOT}/web/dashboard"
    nohup npm run dev > "${LOG_DIR}/frontend.log" 2>&1 &
    echo $! > "${PID_DIR}/frontend.pid"

    sleep 5
    if check_port ${FRONTEND_PORT}; then
        log_success "Frontend started on port ${FRONTEND_PORT}"
    else
        log_error "Failed to start Frontend"
        return 1
    fi
}

# Stop Frontend
stop_frontend() {
    if [ -f "${PID_DIR}/frontend.pid" ]; then
        local pid=$(cat "${PID_DIR}/frontend.pid")
        if kill -0 ${pid} 2>/dev/null; then
            log_info "Stopping Frontend (PID: ${pid})..."
            kill ${pid} 2>/dev/null || true
            rm -f "${PID_DIR}/frontend.pid"
        fi
    fi

    # Also kill any process on the port
    local pids=$(lsof -t -i :${FRONTEND_PORT} 2>/dev/null || true)
    if [ -n "${pids}" ]; then
        log_info "Killing processes on port ${FRONTEND_PORT}..."
        echo "${pids}" | xargs kill 2>/dev/null || true
    fi

    log_success "Frontend stopped"
}

# Show status
show_status() {
    echo ""
    echo "========================================"
    echo "  Database Intelligent Cockpit Status"
    echo "========================================"
    echo ""

    # Data Query
    printf "%-20s " "Data Query:"
    if check_port ${DATAQUERY_PORT}; then
        echo -e "${GREEN}Running${NC} (port ${DATAQUERY_PORT})"
    else
        echo -e "${RED}Stopped${NC}"
    fi

    # Gateway
    printf "%-20s " "Gateway:"
    if check_port ${GATEWAY_PORT}; then
        echo -e "${GREEN}Running${NC} (port ${GATEWAY_PORT})"
    else
        echo -e "${RED}Stopped${NC}"
    fi

    # Frontend
    printf "%-20s " "Frontend:"
    if check_port ${FRONTEND_PORT}; then
        echo -e "${GREEN}Running${NC} (port ${FRONTEND_PORT})"
    else
        echo -e "${RED}Stopped${NC}"
    fi

    echo ""
    echo "Endpoints:"
    echo "  Gateway GraphQL:   http://localhost:${GATEWAY_PORT}/graphql"
    echo "  Data Query:        http://localhost:${DATAQUERY_PORT}/graphql"
    echo "  Frontend:          http://localhost:${FRONTEND_PORT}"
    echo ""
    echo "Log files: ${LOG_DIR}/"
    echo ""
}

# Show logs
show_logs() {
    local service=$1
    case ${service} in
        gateway)
            tail -f "${LOG_DIR}/gateway.log"
            ;;
        dataquery)
            tail -f "${LOG_DIR}/dataquery.log"
            ;;
        frontend)
            tail -f "${LOG_DIR}/frontend.log"
            ;;
        *)
            echo "Usage: $0 logs [gateway|dataquery|frontend]"
            ;;
    esac
}

# Start all services
start_all() {
    echo ""
    echo "========================================"
    echo "  Starting Database Intelligent Cockpit"
    echo "========================================"
    echo ""

    start_dataquery
    start_gateway
    start_frontend

    echo ""
    log_success "All services started!"
    show_status
}

# Stop all services
stop_all() {
    echo ""
    echo "========================================"
    echo "  Stopping Database Intelligent Cockpit"
    echo "========================================"
    echo ""

    stop_frontend
    stop_gateway
    stop_dataquery

    echo ""
    log_success "All services stopped!"
}

# Main command
case "$1" in
    start)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        stop_all
        sleep 2
        start_all
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs "$2"
        ;;
    start-dataquery)
        start_dataquery
        ;;
    stop-dataquery)
        stop_dataquery
        ;;
    start-gateway)
        start_gateway
        ;;
    stop-gateway)
        stop_gateway
        ;;
    start-frontend)
        start_frontend
        ;;
    stop-frontend)
        stop_frontend
        ;;
    *)
        echo "Database Intelligent Cockpit - Services Management"
        echo ""
        echo "Usage: $0 {start|stop|restart|status|logs [service]}"
        echo ""
        echo "Commands:"
        echo "  start              Start all services"
        echo "  stop               Stop all services"
        echo "  restart            Restart all services"
        echo "  status             Show service status"
        echo "  logs <service>     Tail logs for a service (gateway|dataquery|frontend)"
        echo ""
        echo "Individual services:"
        echo "  start-dataquery    Start Data Query Service"
        echo "  stop-dataquery     Stop Data Query Service"
        echo "  start-gateway      Start Gateway Service"
        echo "  stop-gateway       Stop Gateway Service"
        echo "  start-frontend     Start Frontend"
        echo "  stop-frontend      Stop Frontend"
        echo ""
        ;;
esac