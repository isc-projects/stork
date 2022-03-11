#!/bin/bash
set -x
echo "Database type: ${DB_TYPE}"

echo "CREATE USER"
create_user_query="CREATE USER IF NOT EXISTS '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASSWORD}';"
if [ ${DB_TYPE} = 'mysql' ]; then
    mysql \
        --user=root \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        -e "$create_user_query"
else
    PGPASSWORD=${DB_PASSWORD} psql \
        -U ${DB_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} \
        -c "$create_user_query"
fi

echo "Checking if the database exists"
exist_query="select * from schema_version;"
if [ ${DB_TYPE} = 'mysql' ]; then
    mysql \
        --user=root \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        -e "$exist_query" \
        ${DB_NAME}
elif [ ${DB_TYPE} = 'pgsql']; then
    PGPASSWORD=${DB_PASSWORD} psql \
        -U ${DB_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} \
        -c "$exist_query"
else
    echo "Unsupported DB_TYPE, choose mysql or pgsql"
    exit 1
fi

has_db=$?
set -e

if [ $has_db -ne 0 ]
then
echo "Create the database"
    create_db_query="CREATE DATABASE IF NOT EXISTS ${DB_NAME};"
    if [ ${DB_TYPE} = 'mysql' ]; then
        mysql \
            --user=root \
            --password=${DB_ROOT_PASSWORD} \
            --host=${DB_HOST} \
            -e "$create_db_query"
    else
        PGPASSWORD=${DB_PASSWORD} psql \
            -U ${DB_USER} \
            -h ${DB_HOST} \
            -d ${DB_NAME} \
            -c "$create_db_query"
    fi
fi

grant_query="GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'%';"
if [ ${DB_TYPE} = 'mysql' ]; then
    mysql \
        --user=root \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        -e "$grant_query"
else
    PGPASSWORD=${DB_PASSWORD} psql \
        -U ${DB_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} \
        -c "$grant_query"
fi

if [ $has_db -eq 0 ]
then
    exit 0
fi

echo "Initializing the database"
kea-admin db-init ${DB_TYPE} \
    -u ${DB_USER} \
    -p ${DB_PASSWORD} \
    -n ${DB_NAME} \
    -h ${DB_HOST}

echo "Seed database"
seed_file="${BASH_SOURCE%/*}/init_query.sql"
if [ ${DB_TYPE} = 'mysql' ]; then
    mysql \
        --user=root \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        ${DB_NAME} < $seed_file
else
    PGPASSWORD=${DB_PASSWORD} psql \
        -U ${DB_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} < cat $seed_file
fi
