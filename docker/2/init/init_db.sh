#!/bin/bash
set -x
echo "Database type: ${DB_TYPE}"
if [ ${DB_TYPE} = 'mysql' ]; then
    until mysqladmin ping -h"${DB_HOST}" --silent;
    do
    echo "Waiting for database connection..."
    # wait for 5 seconds before check again
    sleep 5
    done
elif [ ${DB_TYPE} = 'pgsql' ]; then
    db_port=5432
else
    echo "Unsupported DB_TYPE, choose mysql or pgsql"
    exit 1
fi


echo "CREATE USER"
if [ ${DB_TYPE} = 'mysql' ]; then
    create_user_query="CREATE USER IF NOT EXISTS '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASSWORD}';"
    mysql \
        --user=${DB_ROOT_USER} \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        -e "$create_user_query"
else
    create_user_query="CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';"
    PGPASSWORD=${DB_ROOT_PASSWORD} psql \
        -U ${DB_ROOT_USER} \
        -h ${DB_HOST} \
        -c "$create_user_query"
fi

echo "Checking if the database exists"
exist_query="select * from schema_version;"
if [ ${DB_TYPE} = 'mysql' ]; then
    mysql \
        --user=${DB_ROOT_USER} \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        -e "$exist_query" \
        ${DB_NAME}
else
    PGPASSWORD=${DB_ROOT_PASSWORD} psql \
        -U ${DB_ROOT_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} \
        -c "$exist_query"
fi

has_db=$?
set -e

if [ $has_db -ne 0 ]
then
    echo "Create the database"
    create_db_query="CREATE DATABASE ${DB_NAME};"
    if [ ${DB_TYPE} = 'mysql' ]; then
        mysql \
            --user=${DB_ROOT_USER} \
            --password=${DB_ROOT_PASSWORD} \
            --host=${DB_HOST} \
            -e "$create_db_query"
    else
        PGPASSWORD=${DB_ROOT_PASSWORD} psql \
            -U ${DB_ROOT_USER} \
            -h ${DB_HOST} \
            -c "$create_db_query"
    fi
fi

if [ ${DB_TYPE} = 'mysql' ]; then
    grant_query="GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'%';"
    mysql \
        --user=${DB_ROOT_USER} \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        -e "$grant_query"
else
    grant_query="GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};"
    PGPASSWORD=${DB_ROOT_PASSWORD} psql \
        -U ${DB_ROOT_USER} \
        -h ${DB_HOST} \
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
        --user=${DB_ROOT_USER} \
        --password=${DB_ROOT_PASSWORD} \
        --host=${DB_HOST} \
        ${DB_NAME} < $seed_file
else
    PGPASSWORD=${DB_ROOT_PASSWORD} psql \
        -U ${DB_ROOT_USER} \
        -h ${DB_HOST} \
        -d ${DB_NAME} < $seed_file
fi
