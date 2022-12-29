#!/bin/bash

set -eu

# Script directory
__dir="$(cd "$(dirname "${0}")" && pwd)"

# Select the specific script. It must be done in a shell script,
# not in the Dockerfile, An environment variable substitution in
# Dockerfile is done during the build phase, but in the shell script
# in runtime.
if [ "$DB_TYPE" = "pgsql" ] || [ "$DB_TYPE" = "mysql" ]; then
    # shellcheck source=./docker/init/init_mysql_db.sh
    # shellcheck source=./docker/init/init_pgsql_db.sh
    . "${__dir}/init_${DB_TYPE}_db.sh"
elif [ "$DB_TYPE" != "none" ]; then
    echo "Unknown DB_TYPE value: ${DB_TYPE}"
    exit 1
fi
