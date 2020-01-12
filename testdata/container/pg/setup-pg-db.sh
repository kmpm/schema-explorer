#!/bin/bash
set -e
echo ---- Starting $0

USER=${DB_USER:-postgres}
DB=${DB:-postgres}

DB_OPTS=${PG_OPTS:--d $PG_URL$DB}

SETUP_DB=${SETUP_DB:-ssetest}
SETUP_USER=${SETUP_USER:-ssetestusr}
SETUP_PASSWD=${SETUP_PASSWD:-ssetestusrpass}
SETUP_DB_SCRIPT=${SETUP_DB_SCRIPT:-test-db.sql}
SETUP_DB_OPTS=${SETUP_DB_OPTS:--d $PG_URL$SETUP_DB}

echo SETUP_DB=$SETUP_DB
echo SETUP_USER=$SETUP_USER
echo SETUP_PASSWD=$SETUP_PASSWD
echo SETUP_DB_SCRIPT=$SETUP_DB_SCRIPT

echo Create database and set owner
psql -v ON_ERROR_STOP=1 $DB_OPTS <<-EOSQL
    DROP DATABASE IF EXISTS $SETUP_DB;
    CREATE DATABASE $SETUP_DB;
    ALTER DATABASE $SETUP_DB OWNER TO $SETUP_USER;
EOSQL

echo Running $SETUP_DB_SCRIPT using $SETUP_DB_OPTS
psql -v ON_ERROR_STOP=1 $SETUP_DB_OPTS -q < $SETUP_DB_SCRIPT

echo Granting user privileges
psql -v ON_ERROR_STOP=1 $SETUP_DB_OPTS <<-EOSQL
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $SETUP_USER;
    GRANT ALL PRIVILEGES ON SCHEMA "identity" TO $SETUP_USER;
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA "identity" TO $SETUP_USER;
EOSQL

echo ---- Done with $0