#!/bin/bash

# Database Intelligent Cockpit - Services Management Script
# Usage: ./scripts/services.sh [start|stop|restart|status|logs] [service]
# Examples:
#   ./scripts/services.sh start              # Start all services
#   ./scripts/services.sh start gateway      # Start only gateway
#   ./scripts/services.sh stop               # Stop all services
#   ./scripts/services.sh restart gateway    # Restart gateway
#   ./scripts/services.sh status             # Show status of all services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Service port configuration
GATEWAY_PORT=8080
DATAQUERY_PORT=8084
COLLECTOR_PORT=8081
AGENT_PORT=8082
TASKENGINE_PORT=8083
FRONTEND_PORT=3000

# Service list (Go services)
GO_SERVICES="gateway dataquery collector agent taskengine"

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

# Get port for a service
get_port() {
    local service=$1
    case "$service" in
        gateway)    echo $GATEWAY_PORT ;;
        dataquery)  echo $DATAQUERY_PORT ;;
        collector)  echo $COLLECTOR_PORT ;;
        agent)      echo $AGENT_PORT ;;
        taskengine) echo $TASKENGINE_PORT ;;
        frontend)   echo $FRONTEND_PORT ;;
        *)          echo "" ;;
    esac
}

# Check if a port is in use
check_port() {
    local port=$1
    if lsof -i :${port} -t >/dev/null 2>&1; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Get PID for a service
get_pid() {
    local service=$1
    local pid_file="${PID_DIR}/${service}.pid"
    if [ -f "$pid_file" ]; then
        cat "$pid_file"
    else
        echo ""
    fi
}

# Save PID for a service
save_pid() {
    local service=$1
    local pid=$2
    echo "$pid" > "${PID_DIR}/${service}.pid"
}

# Remove PID file for a service
remove_pid() {
    local service=$1
    rm -f "${PID_DIR}/${service}.pid"
}

# Start a Go service
start_go_service() {
    local service=$1
    local port=$(get_port "$service")

    if [ -z "$port" ]; then
        log_error "Unknown service: $service"
        return 1
    fi

    if check_port ${port}; then
        log_info "${service} already running on port ${port}"
        return 0
    fi

    log_info "Starting ${service}..."
    cd "${PROJECT_ROOT}"

    # Build first if binary doesn't exist
    if [ ! -f "${PROJECT_ROOT}/bin/${service}" ]; then
        log_info "Building ${service}..."
        go build -o "${PROJECT_ROOT}/bin/${service}" "cmd/${service}/main.go"
    fi

    nohup "${PROJECT_ROOT}/bin/${service}" > "${LOG_DIR}/${service}.log" 2>&1 &
    local pid=$!
    save_pid "$service" "$pid"

    sleep 2
    if check_port ${port}; then
        log_success "${service} started on port ${port} (PID: ${pid})"
    else
        log_error "Failed to start ${service}"
        log_info "Check logs: ${LOG_DIR}/${service}.log"
        return 1
    fi
}

# Stop a Go service
stop_go_service() {
    local service=$1
    local port=$(get_port "$service")

    # Try to kill by PID file first
    local pid=$(get_pid "$service")
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        log_info "Stopping ${service} (PID: ${pid})..."
        kill "$pid" 2>/dev/null || true
        remove_pid "$service"
        sleep 1
    fi

    # Also kill any process on the port
    local pids=$(lsof -t -i :${port} 2>/dev/null || true)
    if [ -n "${pids}" ]; then
        log_info "Killing processes on port ${port}..."
        echo "${pids}" | xargs kill 2>/dev/null || true
    fi

    log_success "${service} stopped"
}

# Start frontend service
start_frontend() {
    local port=$FRONTEND_PORT

    if check_port ${port}; then
        log_info "Frontend already running on port ${port}"
        return 0
    fi

    log_info "Starting Frontend..."
    cd "${PROJECT_ROOT}/web/dashboard"

    if [ ! -d "node_modules" ]; then
        log_info "Installing frontend dependencies..."
        npm install
    fi

    nohup npm run dev > "${LOG_DIR}/frontend.log" 2>&1 &
    local pid=$!
    save_pid "frontend" "$pid"

    sleep 5
    if check_port ${port}; then
        log_success "Frontend started on port ${port} (PID: ${pid})"
    else
        log_error "Failed to start Frontend"
        log_info "Check logs: ${LOG_DIR}/frontend.log"
        return 1
    fi
}

# Stop frontend service
stop_frontend() {
    local port=$FRONTEND_PORT

    local pid=$(get_pid "frontend")
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        log_info "Stopping Frontend (PID: ${pid})..."
        kill "$pid" 2>/dev/null || true
        remove_pid "frontend"
    fi

    # Also kill any process on the port
    local pids=$(lsof -t -i :${port} 2>/dev/null || true)
    if [ -n "${pids}" ]; then
        log_info "Killing processes on port ${port}..."
        echo "${pids}" | xargs kill 2>/dev/null || true
    fi

    log_success "Frontend stopped"
}

# Show status of a single service
show_service_status() {
    local service=$1
    local port=$(get_port "$service")

    printf "  %-15s " "${service}:"
    if [ -n "$port" ] && check_port ${port}; then
        local pid=$(get_pid "$service")
        if [ -n "$pid" ]; then
            echo -e "${GREEN}Running${NC} (port ${port}, PID: ${pid})"
        else
            echo -e "${GREEN}Running${NC} (port ${port})"
        fi
    else
        echo -e "${RED}Stopped${NC}"
    fi
}

# Show status of all services
show_status() {
    local service=$1

    echo ""
    echo "========================================"
    echo "  Database Intelligent Cockpit Status"
    echo "========================================"
    echo ""

    if [ -n "$service" ]; then
        show_service_status "$service"
    else
        for svc in $GO_SERVICES frontend; do
            show_service_status "$svc"
        done
    fi

    echo ""
    echo "Endpoints:"
    echo "  Gateway:       http://localhost:${GATEWAY_PORT}"
    echo "  Data Query:    http://localhost:${DATAQUERY_PORT}/graphql"
    echo "  Frontend:      http://localhost:${FRONTEND_PORT}"
    echo ""
    echo "Log files: ${LOG_DIR}/"
    echo ""
}

# Show logs for a service
show_logs() {
    local service=$1

    if [ -z "$service" ]; then
        echo "Usage: $0 logs <service>"
        echo "Services: gateway, dataquery, collector, agent, taskengine, frontend"
        return 1
    fi

    local log_file="${LOG_DIR}/${service}.log"
    if [ -f "$log_file" ]; then
        tail -f "$log_file"
    else
        log_error "Log file not found: $log_file"
        return 1
    fi
}

# Start services
start_services() {
    local service=$1

    echo ""
    echo "========================================"
    echo "  Starting Database Intelligent Cockpit"
    echo "========================================"
    echo ""

    if [ -n "$service" ]; then
        if [ "$service" = "frontend" ]; then
            start_frontend
        else
            start_go_service "$service"
        fi
    else
        # Start all services in order
        start_go_service dataquery
        start_go_service gateway
        # Optional services (can be started individually)
        # start_go_service collector
        # start_go_service agent
        # start_go_service taskengine
        # start_frontend
    fi

    echo ""
    show_status
}

# Stop services
stop_services() {
    local service=$1

    echo ""
    echo "========================================"
    echo "  Stopping Database Intelligent Cockpit"
    echo "========================================"
    echo ""

    if [ -n "$service" ]; then
        if [ "$service" = "frontend" ]; then
            stop_frontend
        else
            stop_go_service "$service"
        fi
    else
        # Stop all services
        stop_frontend
        for svc in gateway dataquery collector agent taskengine; do
            stop_go_service "$svc"
        done
    fi

    echo ""
    log_success "Stop completed"
}

# Restart services
restart_services() {
    local service=$1

    echo ""
    echo "========================================"
    echo "  Restarting Database Intelligent Cockpit"
    echo "========================================"
    echo ""

    stop_services "$service"
    sleep 2
    start_services "$service"
}

# Check if service is valid
is_valid_service() {
    local service=$1
    for s in $GO_SERVICES frontend; do
        if [ "$s" = "$service" ]; then
            return 0
        fi
    done
    return 1
}

# Show help
show_help() {
    echo "Database Intelligent Cockpit - Services Management"
    echo ""
    echo "Usage: $0 <command> [service]"
    echo ""
    echo "Commands:"
    echo "  start [service]    Start all services or a specific service"
    echo "  stop [service]     Stop all services or a specific service"
    echo "  restart [service]  Restart all services or a specific service"
    echo "  status [service]   Show status of all services or a specific service"
    echo "  logs <service>     Tail logs for a specific service"
    echo ""
    echo "Services:"
    echo "  gateway      API Gateway (port ${GATEWAY_PORT})"
    echo "  dataquery    Data Query Service (port ${DATAQUERY_PORT})"
    echo "  collector    Collector Service (port ${COLLECTOR_PORT})"
    echo "  agent        Execution Agent (port ${AGENT_PORT})"
    echo "  taskengine   Task Engine (port ${TASKENGINE_PORT})"
    echo "  frontend     Web Frontend (port ${FRONTEND_PORT})"
    echo ""
    echo "Examples:"
    echo "  $0 start              # Start all services"
    echo "  $0 start gateway      # Start only gateway"
    echo "  $0 stop               # Stop all services"
    echo "  $0 restart gateway    # Restart gateway"
    echo "  $0 status             # Show status of all services"
    echo "  $0 logs gateway       # Tail gateway logs"
    echo ""
}

# Main command
case "$1" in
    start)
        start_services "$2"
        ;;
    stop)
        stop_services "$2"
        ;;
    restart)
        restart_services "$2"
        ;;
    status)
        show_status "$2"
        ;;
    logs)
        show_logs "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    "")
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac