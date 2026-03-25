#!/bin/bash

# Database Intelligent Cockpit - Build Script
# Usage: ./scripts/build.sh [all|gateway|collector|agent|taskengine|dataquery]
# Examples:
#   ./scripts/build.sh          # Build all services
#   ./scripts/build.sh all      # Build all services
#   ./scripts/build.sh gateway  # Build only gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${PROJECT_ROOT}"

# Output directory
BIN_DIR="${PROJECT_ROOT}/bin"

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Service list
SERVICES="gateway collector agent taskengine dataquery"

# Get source path for a service
get_source() {
    local service=$1
    echo "cmd/${service}/main.go"
}

# Build a single service
build_service() {
    local service=$1
    local source=$(get_source "$service")

    if [ ! -f "$source" ]; then
        log_error "Source file not found: $source"
        return 1
    fi

    log_info "Building ${service}..."
    go build -o "${BIN_DIR}/${service}" "${source}"

    if [ $? -eq 0 ]; then
        log_success "${service} built successfully -> ${BIN_DIR}/${service}"
    else
        log_error "Failed to build ${service}"
        return 1
    fi
}

# Build all services
build_all() {
    echo ""
    echo "========================================"
    echo "  Building All Services"
    echo "========================================"
    echo ""

    # Create bin directory
    mkdir -p "${BIN_DIR}"

    local failed=0
    for service in $SERVICES; do
        if ! build_service "$service"; then
            failed=1
        fi
    done

    echo ""
    if [ $failed -eq 0 ]; then
        log_success "All services built successfully!"
        echo ""
        echo "Binaries available in ${BIN_DIR}/:"
        ls -la "${BIN_DIR}/" 2>/dev/null || echo "  (empty)"
    else
        log_error "Some services failed to build"
        return 1
    fi
}

# Clean build artifacts
clean() {
    log_info "Cleaning build artifacts..."
    rm -rf "${BIN_DIR}"
    log_success "Build artifacts cleaned"
}

# Show help
show_help() {
    echo "Database Intelligent Cockpit - Build Script"
    echo ""
    echo "Usage: $0 [command|service]"
    echo ""
    echo "Commands:"
    echo "  all           Build all services (default)"
    echo "  clean         Remove all build artifacts"
    echo "  help          Show this help message"
    echo ""
    echo "Services:"
    for service in $SERVICES; do
        printf "  %-12s Build only %s service\n" "$service" "$service"
    done
    echo ""
    echo "Examples:"
    echo "  $0                  # Build all services"
    echo "  $0 all              # Build all services"
    echo "  $0 gateway          # Build only gateway"
    echo "  $0 clean            # Clean build artifacts"
    echo ""
}

# Check if service is valid
is_valid_service() {
    local service=$1
    for s in $SERVICES; do
        if [ "$s" = "$service" ]; then
            return 0
        fi
    done
    return 1
}

# Main logic
mkdir -p "${BIN_DIR}"

case "$1" in
    "")
        build_all
        ;;
    all)
        build_all
        ;;
    clean)
        clean
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        if is_valid_service "$1"; then
            build_service "$1"
        else
            log_error "Unknown command or service: $1"
            echo ""
            show_help
            exit 1
        fi
        ;;
esac