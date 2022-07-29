#! /bin/sh
# Script for running the Stork demo with minimal dependencies

# Exit on error
set -e

# Check if the docker-compose exists
if ! [ -x $(command -v docker-compose) ]
then
    echo "The docker-compose command could not be found"
    exit 127
fi

# The directory with this script
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

# Prompt for CloudSmith access token
usage()
{
    echo "Usage: stork-demo.sh [ -f | --no-prompt ] [ -s | --stop ] [ CLOUD_SMITH_ACCESS_TOKEN ]"
    echo "You can also set the access token using environment variable CS_REPO_ACCESS_TOKEN."
    exit 2
}

logo()
{
    echo ""
    echo "-----------------------------------------------"
    echo " ____  _             _    "
    echo "/ ___|| |_ ___  _ __| | __"
    echo "\___ \| __/ _ \| '__| |/ /"
    echo " ___) | || (_) | |  |   < "
    echo "|____/ \__\___/|_|  |_|\_\\"
    echo ""
}

# Parse arguments
set +e
PARSED_ARGUMENTS=$(getopt -o fsh --long no-prompt,stop,help -- "$@")
parse_status=$?
set -e

if [ ${parse_status} -ne 0 ]
then
    usage
    exit ${parse_status}
fi

eval set -- "$PARSED_ARGUMENTS"
NO_PROMPT=0
STOP=0
ACCESS_TOKEN=${CS_REPO_ACCESS_TOKEN}
while :
do
  case "$1" in
    -f | --no-prompt)   NO_PROMPT=1      ; shift   ;;
    -s | --stop)        STOP=1           ; shift   ;;
    -h | --help)        usage            ; exit 2  ;;
    # -- means the end of the arguments; drop this, and break out of the while loop
    --) shift; break ;;
    # If invalid options were passed, then getopt should have reported an error,
    # which we checked as VALID_ARGUMENTS when getopt was called...
    *) echo "Unexpected option: $1 - this should not happen."
       usage ;;
  esac
done

# Stop the demo
if [ ${STOP} -eq 1 ]
then
    docker-compose \
        --project-directory "${SCRIPT_DIR}" \
        -f "${SCRIPT_DIR}/docker/docker-compose.yaml" \
        -f "${SCRIPT_DIR}/docker/docker-compose-premium.yaml" \
        down --volumes

    if [ ${NO_PROMPT} -eq 0 ]
    then
        logo
        echo "The demo was stopped."
    fi
    exit 0
fi

# Set access token from CMD
if [ $# -gt 0 ]
then
    ACCESS_TOKEN=$1
fi

# Prompt necessary?
if [ -z "${ACCESS_TOKEN}" ]
then
    # Prompt allowed?
    if [ ${NO_PROMPT} -eq 0 ]
    then
        echo "To run the Demo with a Kea instance that includes the premium features, you need to provide your CloudSmith access token."
        echo "Leave this value empty to use only open-source features."
        echo "Enter CloudSmith access token (or leave empty):"
        # No echo the secret
        stty -echo
        read ACCESS_TOKEN
        stty echo
    fi
fi

PREMIUM_COMPOSE=
if ! [ -z "${ACCESS_TOKEN}" ]
then
    PREMIUM_COMPOSE="-f${SCRIPT_DIR}/docker/docker-compose-premium.yaml"
else
    # This variable cannot be empty because it causes the docker-compose fails.
    # The below line sets any default value explicitly.
    PREMIUM_COMPOSE="--env-file="
fi

# Run the demo
# Build Docker containers
DOCKER_BUILDKIT=1 \
COMPOSE_DOCKER_CLI_BUILD=1 \
CS_REPO_ACCESS_TOKEN=${ACCESS_TOKEN} \
docker-compose \
    --project-directory "${SCRIPT_DIR}" \
    -f "${SCRIPT_DIR}/docker/docker-compose.yaml" \
    "${PREMIUM_COMPOSE}" \
    build
# Start Docker containers
docker-compose \
    --project-directory "${SCRIPT_DIR}" \
    -f "${SCRIPT_DIR}/docker/docker-compose.yaml" \
    "${PREMIUM_COMPOSE}" \
    up -d

if [ ${NO_PROMPT} -eq 0 ]
then
    logo
    echo "Open the demo in the browser:"
    echo "Stork Server:      http://127.0.0.1:8080"
    echo "Grafana:           http://127.0.0.1:3000"
    echo "Prometheus:        http://127.0.0.1:9090"
    echo "Traffic simulator: http://127.0.0.1:5000"
    echo "Default username: admin password: admin"
    echo ""
    echo "Use './stork-demo.sh --stop' to shutdown the demo"
fi