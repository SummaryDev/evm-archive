#!/usr/bin/env bash

# run with
# namespace=prod evm_chain=ethereum evm_network=mainnet ./db-create.sh
# namespace=prod evm_chain=ethereum evm_network=goerli ./db-create.sh
# namespace=dev evm_chain=ethereum evm_network=goerli ./db-create.sh
# namespace=prod evm_chain=moonbeam evm_network=mainnet ./db-create.sh

# defaults
export evm_chain=${evm_chain-moonbeam}
export evm_network=${evm_network-mainnet}
export namespace=${namespace-dev}

# you may already have db host and passwords in an env file:
# source .env
# source ../infra/env.sh
# otherwise export them to env variables:
# if the db is within k8s
# export db_host=db-postgresql.default.svc.cluster.local
# if the db is RDS
# export db_host=mydb.abcdelniptgt.eu-central-1.rds.amazonaws.com
# password of admin user postgres who will crete the db
# export db_password=postgres123
# password of user evm_archive this app will connect as to create the schema and insert records
# export db_password_evm_archive=evm_archive123

env | grep '^db' | sort

# as admin user postgres: create the database, user evm_archive, his grants

export PGHOST=${db_host-"db-postgresql.default.svc.cluster.local"}
export PGPASSWORD=${db_password}
export PGUSER=postgres

env | grep '^PG' | sort

envsubst < db-create.sql | psql --file -

# as evm_archive user: create schema, tables and default privileges for users who will be readers

export PGUSER=evm_archive
export PGPASSWORD=${db_password_evm_archive}
export PGDATABASE=${evm_chain}_${evm_network}${evm_shard}_${namespace}

env | grep '^PG' | sort

envsubst < schema.sql | psql --file -