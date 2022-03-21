#!/bin/bash
# Script directory
__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Select the specific script. It must be done in a shell script,
# not in the Dockerfile, An environment variable substitution in
# Dockerfile is done during the build phase, but in the shell script
# in runtime.
if [ "$DB_TYPE" == "pgsql" ] || [ "$DB_TYPE" == "mysql" ]; then
    source ${__dir}/init_${DB_TYPE}_db.sh
elif [ "$DB_TYPE" != "none" ]; then
    echo "Unknown DB_TYPE value: ${DB_TYPE}"
    exit 1
fi
