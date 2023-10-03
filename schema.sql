-- this schema data is for raw undecoded records received from an rpc node like logs, transactions, blocks, traces
-- schema metadata is for contracts and abis
-- schema event is for decoded events like event.Transfer
-- other schemas will be created as namespaces to hold views per contract like erc20, aave, beamswap

create schema if not exists data;

-- no need to explicitly grant when the schema is created by this archive user
-- grant usage, create on schema data to evm_archive;
-- grant select, insert, update, delete on all tables in schema data to evm_archive;
-- grant select, update, usage on all sequences in schema data to evm_archive;

-- users who are readers of the data like BI tools
-- grant usage on schema data to redash;
-- grant usage on schema data to hasura;
-- grant usage on schema data to metabase;
-- grant usage on schema data to superset;
-- grant usage on schema data to graphile;

-- grant select on all tables in schema data to redash;
-- grant select on all tables in schema data to hasura;
-- grant select on all tables in schema data to metabase;
-- grant select on all tables in schema data to superset;
-- grant select on all tables in schema data to graphile;

-- alter default privileges for role evm_archive in schema data grant select on tables to hasura;
-- alter default privileges for role evm_archive in schema data grant select on tables to superset;
-- alter default privileges for role evm_archive in schema data grant select on tables to metabase;
-- alter default privileges for role evm_archive in schema data grant select on tables to redash;
-- alter default privileges for role evm_archive in schema data grant select on tables to graphile;

set search_path to data;

-- drop table if exists logs cascade;

create table if not exists logs
(
  address               text,
  topic0                text,
  topic1                text,
  topic2                text,
  topic3                text,
  data                  text,
  block_hash            text,
  block_number          numeric,
  transaction_hash      text,
  transaction_index     numeric,
  log_index             numeric,
  removed               boolean,
  block_timestamp       timestamp,
  primary key (block_hash, transaction_hash, log_index)
);

create index on logs (address);
create index on logs (topic0);
create index on logs (topic1);
create index on logs (topic2);
create index on logs (topic3);
create index on logs (address, topic0);
create index on logs (transaction_hash);
create index on logs (block_hash);
create index on logs (block_number);

comment on table logs is 'Event emitted by a smart contract';
comment on column logs.address is 'Address of the smart contract emitting the event';
comment on column logs.topic0 is 'Hash of the event name';
