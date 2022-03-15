#!/bin/bash
__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "$DB_TYPE" == "pgsql" ] || [ "$DB_TYPE" == "mysql" ]; then
    source ${__dir}/init_${DB_TYPE}_db.sh
elif [ "$DB_TYPE" != "none" ]; then
    echo "Unknown DB_TYPE value: ${DB_TYPE}"
    exit 1
fi
