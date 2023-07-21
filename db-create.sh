#!/usr/bin/env bash

# run with
# evm_chain=moonriver evm_network=mainnet ./db-create.sh

export evm_network=${evm_network-mainnet}

source ../infra/env.sh

env | grep '^db' | sort

# as postgres user: create database user evm_archive, grants, schema

export PGHOST=${db_host} && \
export PGPASSWORD=${db_password} && \
export PGUSER=postgres && \
export PGDATABASE=${namespace}

env | grep '^PG' | sort

envsubst < db-create.sql | psql --file -

# as evm_archive user: create tables and default privileges for reader users

export PGUSER=evm_archive && \
export PGPASSWORD=${db_password_evm_archive} && \
export PGDATABASE=${namespace}

env | grep '^PG' | sort

envsubst < schema.sql | psql --file -