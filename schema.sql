-- db-default-privileges
alter default privileges for role evm_archive in schema ${evm_chain}_${evm_network}${evm_shard} grant select on tables to hasura;
alter default privileges for role evm_archive in schema ${evm_chain}_${evm_network}${evm_shard} grant select on tables to superset;
alter default privileges for role evm_archive in schema ${evm_chain}_${evm_network}${evm_shard} grant select on tables to metabase;
alter default privileges for role evm_archive in schema ${evm_chain}_${evm_network}${evm_shard} grant select on tables to redash_evm;
alter default privileges for role evm_archive in schema ${evm_chain}_${evm_network}${evm_shard} grant select on tables to graphile;

set search_path to ${evm_chain}_${evm_network}${evm_shard};

-- drop table if exists Log cascade;

create table if not exists Log
(
  address             text,
  topic0              text,
  topic1              text,
  topic2              text,
  topic3              text,
  data                text,
  blockHash           text,
  blockNumber         text,
  transactionHash     text,
  transactionIndex    text,
  logIndex            text,
  transactionLogIndex text,
  removed             boolean,
  primary key (blockHash, transactionHash, logIndex)
);

create index on Log (address);
create index on Log (topic0);
create index on Log (topic1);
create index on Log (topic2);
create index on Log (topic3);
create index on Log (address, topic0);
create index on Log (transactionHash);
create index on Log (blockHash);
create index on Log (blockNumber);

comment on table Log
is 'evmEvent';

comment on column Log.address
is 'Address of the smart contract emitting the event';
comment on column Log.topic0
is 'Hash of the event name';


