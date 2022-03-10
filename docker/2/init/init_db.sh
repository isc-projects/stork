#!/bin/bash

echo "Database type: ${DB_TYPE}"
echo "Checking if the database exists"
exist_query = "select * from schema_version;"
if [ ${DB_TYPE} = 'mysql' ]; then
    mysql \
        --user=${DB_USER} \
        --password=${DB_PASSWORD} \
        --host=${DB_HOST} \
        ${DB_NAME} \
        -e ${exist_query}
elif [ ${DB_TYPE} = 'pgsql']; then
    PGPASSWORD=${DB_PASSWORD} psql \
        -U ${DB_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} \
        -c ${exist_query}
else
    echo "Unsupported DB_TYPE, choose mysql or pgsql"
    exit 1
fi

if [ $? -eq 0 ]
then
    echo "Database apparently exists"
    exit 0
fi

set -e

echo "Initializing the database"
kea-admin db-init ${DB_TYPE} \
    -u ${DB_USER} \
    -p ${DB_PASSWORD} \
    -n ${DB_NAME} \
    -h ${DB_HOST}

echo "Seed database"
seed_file = "${BASH_SOURCE%/*}/init_query.sql"
if [ ${DB_TYPE} = 'mysql' ]; then
    mysql \
        --user=${DB_USER} \
        --password=${DB_PASSWORD} \
        --host=${DB_HOST} \
        ${DB_NAME} < cat ${seed_file}
else
    PGPASSWORD=${DB_PASSWORD} psql \
        -U ${DB_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} < cat ${seed_file}
fi
