#!/bin/bash

set -eu

echo "Database type: ${DB_TYPE}"

until PGPASSWORD=${DB_ROOT_PASSWORD} psql -h "${DB_HOST}" -U "${DB_ROOT_USER}" -c "SELECT 1" > /dev/null 2>&1;
do
    echo "Waiting for database connection..."
    sleep 5
done

echo "CREATE USER"

set +e
create_user_query="CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';"
PGPASSWORD=${DB_ROOT_PASSWORD} \
psql \
    -U "${DB_ROOT_USER}" \
    -h "${DB_HOST}" \
    -c "${create_user_query}"

echo "Checking if the database exists"
set -e

exist_query="\c ${DB_NAME}"
set +e
PGPASSWORD=${DB_ROOT_PASSWORD} \
psql \
    -U "${DB_ROOT_USER}" \
    -h "${DB_HOST}" \
    -d "${DB_NAME}" \
    -c "${exist_query}"
has_db=$?
set -e

if [ $has_db -ne 0 ]
then
    echo "Create the database"
    create_db_query="CREATE DATABASE ${DB_NAME};"
    PGPASSWORD=${DB_ROOT_PASSWORD} \
    psql \
        -U "${DB_ROOT_USER}" \
        -h "${DB_HOST}" \
        -c "${create_db_query}"
fi

echo "Grant all privileges on the database"
grant_query="GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};"
PGPASSWORD=${DB_ROOT_PASSWORD} \
psql \
    -U "${DB_ROOT_USER}" \
    -h "${DB_HOST}" \
    -c "${grant_query}"

echo "Grant all privileges on the public schema"
PGPASSWORD=${DB_ROOT_PASSWORD} \
psql \
    -U "${DB_ROOT_USER}" \
    -h "${DB_HOST}" \
    -d "${DB_NAME}" \
    -c "GRANT ALL PRIVILEGES ON SCHEMA public TO ${DB_USER};"

if [ $has_db -eq 0 ]
then
    exit 0
fi

echo "Initializing the database"
kea-admin db-init "${DB_TYPE}" \
    -u "${DB_USER}" \
    -p "${DB_PASSWORD}" \
    -n "${DB_NAME}" \
    -h "${DB_HOST}"

echo "Seed database"
path=$(dirname "${BASH_SOURCE[0]}")
seed_file="${path}/init_pgsql_query.sql"

PGPASSWORD=${DB_ROOT_PASSWORD} \
psql \
    -U "${DB_ROOT_USER}" \
    -h "${DB_HOST}" \
    -d "${DB_NAME}" \
    < "${seed_file}"
