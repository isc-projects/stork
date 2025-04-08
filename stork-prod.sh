#! /bin/sh
# Production deployment script for Stork with enhanced security and monitoring

# Exit on error
set -e
# Exit on undefined variable
set -u
# Exit if any command in a pipe fails
set -o pipefail

# The directory with this script
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

# Default values for configuration
DEFAULT_POSTGRES_PASSWORD=$(openssl rand -base64 32)
DEFAULT_STORK_PASSWORD=$(openssl rand -base64 32)
DEFAULT_ADMIN_PASSWORD=$(openssl rand -base64 16)

# Configuration file
CONFIG_FILE="${SCRIPT_DIR}/.stork-prod.env"

# Function to validate environment
validate_environment() {
    # Check for required commands
    for cmd in docker openssl curl; do
        if ! command -v "$cmd" > /dev/null; then
            echo "Error: Required command '$cmd' not found"
            exit 1
        fi
    done

    # Check if the docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        echo "Error: Docker daemon is not running"
        exit 1
    }

    # Check for docker compose
    if docker compose > /dev/null 2>&1; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose > /dev/null; then
        DOCKER_COMPOSE=docker-compose
    else
        echo "Error: Neither \"docker compose\" nor \"docker-compose\" found"
        exit 1
    fi
}

# Function to create/load configuration
setup_configuration() {
    if [ ! -f "${CONFIG_FILE}" ]; then
        echo "Creating new configuration file..."
        cat > "${CONFIG_FILE}" << EOF
# Stork Production Configuration
POSTGRES_PASSWORD=${DEFAULT_POSTGRES_PASSWORD}
STORK_DATABASE_PASSWORD=${DEFAULT_STORK_PASSWORD}
ADMIN_PASSWORD=${DEFAULT_ADMIN_PASSWORD}
# Monitoring Configuration
ENABLE_MONITORING=true
GRAFANA_ADMIN_PASSWORD=$(openssl rand -base64 16)
# Backup Configuration
BACKUP_ENABLED=true
BACKUP_RETENTION_DAYS=7
# Resource Limits
POSTGRES_MAX_CONNECTIONS=200
SERVER_MAX_CONNECTIONS=1000
# Network Configuration
STORK_SERVER_PORT=8080
GRAFANA_PORT=3000
PROMETHEUS_PORT=9090
EOF
        chmod 600 "${CONFIG_FILE}"
    fi

    # Load configuration
    . "${CONFIG_FILE}"
}

# Function to check system resources
check_resources() {
    # Check available disk space (need at least 10GB)
    available_space=$(df -BG "${SCRIPT_DIR}" | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "${available_space}" -lt 10 ]; then
        echo "Error: Insufficient disk space. Need at least 10GB, have ${available_space}GB"
        exit 1
    fi

    # Check available memory (need at least 4GB)
    total_mem=$(free -g | awk '/^Mem:/{print $2}')
    if [ "${total_mem}" -lt 4 ]; then
        echo "Error: Insufficient memory. Need at least 4GB RAM"
        exit 1
    fi
}

# Function to setup backup
setup_backup() {
    if [ "${BACKUP_ENABLED}" = "true" ]; then
        mkdir -p "${SCRIPT_DIR}/backups"
        chmod 700 "${SCRIPT_DIR}/backups"

        # Create backup script
        cat > "${SCRIPT_DIR}/backup.sh" << 'EOF'
#!/bin/sh
BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7

# Backup PostgreSQL
docker exec stork-postgres pg_dump -U stork stork > "${BACKUP_DIR}/stork_${TIMESTAMP}.sql"

# Backup configuration
cp .stork-prod.env "${BACKUP_DIR}/config_${TIMESTAMP}.env"

# Cleanup old backups
find "${BACKUP_DIR}" -type f -mtime +${RETENTION_DAYS} -delete
EOF
        chmod 700 "${SCRIPT_DIR}/backup.sh"

        # Setup daily cron job for backup
        (crontab -l 2>/dev/null; echo "0 2 * * * ${SCRIPT_DIR}/backup.sh") | crontab -
    fi
}

# Function to display usage
usage() {
    echo "Usage: stork-prod.sh [OPTIONS]"
    echo "Options:"
    echo "  -s, --start        Start the production stack"
    echo "  -d, --stop         Stop the production stack"
    echo "  -b, --backup       Perform immediate backup"
    echo "  -r, --restore FILE Restore from backup file"
    echo "  -h, --help         Show this help message"
}

# Function to start the stack
start_stack() {
    echo "Starting Stork production stack..."

    # Pull latest images
    $DOCKER_COMPOSE \
        --project-directory "${SCRIPT_DIR}" \
        -f "${SCRIPT_DIR}/docker/docker-compose-prod.yaml" \
        pull

    # Start services
    $DOCKER_COMPOSE \
        --project-directory "${SCRIPT_DIR}" \
        -f "${SCRIPT_DIR}/docker/docker-compose-prod.yaml" \
        up -d

    # Wait for services to be healthy
    echo "Waiting for services to be healthy..."
    timeout 300 sh -c 'until curl -s http://localhost:${STORK_SERVER_PORT}/api/version; do sleep 5; done'

    echo "Stork production stack is running"
    echo "Access URLs:"
    echo "Stork Server:   http://localhost:${STORK_SERVER_PORT}"
    echo "Grafana:        http://localhost:${GRAFANA_PORT}"
    echo "Prometheus:     http://localhost:${PROMETHEUS_PORT}"
    echo "Default admin credentials are in ${CONFIG_FILE}"
}

# Function to stop the stack
stop_stack() {
    echo "Stopping Stork production stack..."

    # Perform backup before stopping if enabled
    if [ "${BACKUP_ENABLED}" = "true" ]; then
        "${SCRIPT_DIR}/backup.sh"
    fi

    $DOCKER_COMPOSE \
        --project-directory "${SCRIPT_DIR}" \
        -f "${SCRIPT_DIR}/docker/docker-compose-prod.yaml" \
        down --volumes

    echo "Stork production stack stopped"
}

# Main script execution
main() {
    # Validate environment first
    validate_environment

    # Process command line arguments
    case "${1:-}" in
        -s|--start)
            check_resources
            setup_configuration
            setup_backup
            start_stack
            ;;
        -d|--stop)
            setup_configuration
            stop_stack
            ;;
        -b|--backup)
            setup_configuration
            "${SCRIPT_DIR}/backup.sh"
            ;;
        -r|--restore)
            if [ -z "${2:-}" ]; then
                echo "Error: Restore file not specified"
                usage
                exit 1
            fi
            setup_configuration
            # TODO: Implement restore functionality
            ;;
        -h|--help)
            usage
            ;;
        *)
            usage
            exit 1
            ;;
    esac
}

# Execute main function with all arguments
main "$@"
