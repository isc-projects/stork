#!/bin/bash

set -eu

echo "Database type: ${DB_TYPE}"

until mysqladmin ping -h"${DB_HOST}" --silent;
do
    echo "Waiting for database connection..."
    sleep 5
done

echo "CREATE USER"

create_user_query="CREATE USER IF NOT EXISTS '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASSWORD}';"
mysql \
    --user="${DB_ROOT_USER}" \
    --password="${DB_ROOT_PASSWORD}" \
    --host="${DB_HOST}" \
    -e "${create_user_query}"

echo "Checking if the database exists"

exist_query="USE ${DB_NAME}"
set +e
mysql \
    --user="${DB_ROOT_USER}" \
    --password="${DB_ROOT_PASSWORD}" \
    --host="${DB_HOST}" \
    -e "${exist_query}" \
    "${DB_NAME}"
has_db=$?
set -e

if [ $has_db -ne 0 ]
then
    echo "Create the database"
    create_db_query="CREATE DATABASE ${DB_NAME};"
    mysql \
        --user="${DB_ROOT_USER}" \
        --password="${DB_ROOT_PASSWORD}" \
        --host="${DB_HOST}" \
        -e "$create_db_query"
fi

grant_query="GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'%';"
mysql \
    --user="${DB_ROOT_USER}" \
    --password="${DB_ROOT_PASSWORD}" \
    --host="${DB_HOST}" \
    -e "${grant_query}"

if [ $has_db -eq 0 ]
then
    exit 0
fi

echo "Initializing the database"
set +e
kea-admin db-init "${DB_TYPE}" \
    -u "${DB_USER}" \
    -p "${DB_PASSWORD}" \
    -n "${DB_NAME}" \
    -h "${DB_HOST}"
init_status=$?
set -e

# Status 0 - initialization successful, status 2 - database already initialized.
if [ $init_status -ne 0 ] && [ $init_status -ne 2 ]
then
    echo "Database initialization failed with status ${init_status}"
    exit $init_status
fi

path=$(dirname "${BASH_SOURCE[0]}")
seed_file="${path}/init_mysql_query.sql"

if [ -f "${seed_file}" ]; then
    echo "Seed database"
    mysql \
        --user="${DB_ROOT_USER}" \
        --password="${DB_ROOT_PASSWORD}" \
        --host="${DB_HOST}" \
        "${DB_NAME}" \
        < "${seed_file}"
fi