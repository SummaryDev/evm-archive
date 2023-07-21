-- db-create-users
create user evm_archive with password '${db_password_evm_archive}';

-- db-grant-on-database
grant connect on database ${namespace} to evm_archive;

-- db-grant-on-schema
create schema if not exists ${evm_chain}_${evm_network}${evm_shard};

grant usage, create on schema {evm_chain}_${evm_network}${evm_shard} to evm_archive;
grant select, insert, update, delete on all tables in schema {evm_chain}_${evm_network}${evm_shard} to evm_archive;
grant select, update, usage on all sequences in schema {evm_chain}_${evm_network}${evm_shard} to evm_archive;

grant usage on schema {evm_chain}_${evm_network}${evm_shard} to redash_evm, hasura, metabase, superset, graphile;

grant select on all tables in schema {evm_chain}_${evm_network}${evm_shard} to redash_evm, hasura, metabase, superset, graphile;
