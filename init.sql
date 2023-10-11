/*
using public schema for simplicity and to have all in one place as GraphileQL supports one schema only
*/

set search_path to public;

/*
this table holds raw events
*/

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

/*
creates functions present on redshift https://docs.aws.amazon.com/redshift/latest/dg/r_STRTOL.html
but missing on the latest postgres https://www.postgresql.org/docs/current/functions-string.html
to keep functions.sql uniform across postgres and redshift

see https://github.com/SummaryDev/ethereum-sql
 */

set search_path to public;

create or replace function to_int64 (pos int, data text) returns bigint immutable
as $$
select concat('x', substring(lpad($2, 64, '0'), $1+49, 16))::bit(64)::bigint
$$ language sql;

create or replace function to_uint64 (pos int, data text) returns dec immutable
as $$
select concat('x', substring(lpad($2, 64, '0'), $1+49, 8))::bit(32)::bigint::dec*4294967296 + concat('x', substring(lpad($2, 64, '0'), $1+57, 8))::bit(32)::bigint::dec
$$ language sql;

create or replace function to_uint32 (pos int, data text) returns bigint immutable
as $$
select concat('x', substring(lpad($2, 64, '0'), $1+57, 8))::bit(32)::bigint
$$ language sql;

--todo test it
create or replace function to_int32 (pos int, data text) returns bigint immutable
as $$
select to_int32($1, $2)
$$ language sql;

create or replace function to_uint128 (pos int, data text) returns dec immutable
as $$
select concat('x', substring(lpad($2, 64, '0'), $1+33, 8))::bit(32)::bigint::dec*4294967296*4294967296*4294967296 + concat('x', substring(lpad($2, 64, '0'), $1+41, 8))::bit(32)::bigint::dec*4294967296*4294967296 + concat('x', substring(lpad($2, 64, '0'), $1+49, 8))::bit(32)::bigint::dec*4294967296 + concat('x', substring(lpad($2, 64, '0'), $1+57, 8))::bit(32)::bigint::dec
$$ language sql;

--todo don't downshift to_uint256 to to_uint128
create or replace function to_uint256 (pos int, data text) returns dec immutable
as $$
select to_uint128 ($1, $2)
$$ language sql;

create or replace function strtol (data text, bits int) returns bigint immutable
as $$
select concat('x', substr(lpad(data, 64, '0'), 49, 64))::bit(64)::bigint
$$ language sql;

create or replace function from_hex (data text)
  returns bytea
immutable
as $$
select decode(data, 'hex')
$$ language sql;

create or replace function from_varbyte (data bytea, encoding text)
  returns text
immutable
as $$
select convert_from(data, encoding)
$$ language sql;

/*
library of functions to decode abi encoded data https://docs.soliditylang.org/en/develop/abi-spec.html
uses lower level functions created for redshift or postgres

see https://github.com/SummaryDev/ethereum-sql
 */

set search_path to public;

create or replace function to_location (pos int, data text) returns int immutable
as $$
select to_uint32($1, $2)::int
$$ language sql;

create or replace function to_size (pos int, data text) returns int immutable
as $$
select to_uint32(to_location($1, $2)*2, $2)::int
$$ language sql;

create or replace function to_raw_bytes (pos int, data text)
  returns text
immutable
as $$
select substring($2, 1 + to_location($1, $2)*2 + 64, to_size($1, $2)*2)
$$ language sql;

create or replace function to_bytes (pos int, data text)
  returns text
immutable
as $$
select '0x' || to_raw_bytes($1, $2)
$$ language sql;

create or replace function to_fixed_bytes (pos int, data text, size int)
  returns text
immutable
as $$
select '0x' || rtrim(substring($2, $1+1, $3*2), '0')
$$ language sql;

create or replace function to_string (pos int, data text)
  returns text
immutable
as $$
select from_varbyte(from_hex(to_raw_bytes($1, $2)), 'utf8')
-- select convert_from(decode(to_raw_bytes($1, $2), 'hex'), 'utf8')
$$ language sql;

create or replace function to_address (pos int, data text)
  returns text
immutable
as $$
select '0x' || substring($2, $1+25, 40)
$$ language sql;

create or replace function to_bool (pos int, data text)
  returns bool
immutable
as $$
select to_uint32($1, $2)::int::bool
$$ language sql;

create or replace function to_element (pos int, data text, type text)
  returns text
immutable
as $$
select case
       when $3 = 'string' then quote_ident(to_string($1, $2))
       when $3 = 'bytes' then quote_ident(to_bytes($1, $2))
       when $3 = 'address' then quote_ident(to_address($1, $2))
       when $3 = 'int32' then to_int32($1, $2)::text
       when $3 = 'uint32' then to_int32($1, $2)::text
       when $3 = 'int64' then to_int64($1, $2)::text
       when $3 = 'uint64' then to_uint64($1, $2)::text
       when $3 = 'uint128' then to_uint128($1, $2)::text
--        when $3 = 'decimal' then to_decimal($1, $2)::text todo where do we use decimal type?
       when $3 = 'bool' then case when to_bool($1, $2) then 'true' else 'false' end
       else quote_ident(substring($2, $1+1, 64))
       end
$$ language sql;

create or replace function to_array (pos int, data text, type text)
  returns text
immutable
as $$
select case
       when to_size($1, $2) = 0 then '[]'
       when to_size($1, $2) = 1 then '[' || to_element($1+128, $2, $3) || ']'
       when to_size($1, $2) = 2 then '[' || to_element($1+128, $2, $3) || ',' || to_element($1+192, $2, $3) || ']'
                                else '[' || to_element($1+128, $2, $3) || ',' || to_element($1+192, $2, $3) || ',' || to_element($1+256, $2, $3) || ']'
       end
$$ language sql;

create or replace function to_fixed_array (pos int, data text, type text, size int)
  returns text
immutable
as $$
select case
       when $4 = 0 then '[]'
       when $4 = 1 then '[' || to_element($1, $2, $3) || ']'
       when $4 = 2 then '[' || to_element($1, $2, $3) || ',' || to_element($1+64, $2, $3) || ']'
                   else '[' || to_element($1, $2, $3) || ',' || to_element($1+64, $2, $3) || ',' || to_element($1+128, $2, $3) || ']'
       end
$$ language sql;

/*
views generated by https://github.com/SummaryDev/ethereum-sql from contracts ABIs
*/

create or replace view "Approval_address_owner_address_spender_uint256_amount_d" as select to_address(2,topic1::text) "owner",to_address(2,topic2::text) "spender",to_uint256(2,data::text) "amount", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925';
create or replace view "AuthorityUpdated_address_user_address_newAuthority" as select to_address(2,topic1::text) "user",to_address(2,topic2::text) "newAuthority", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0xa3396fd7f6e0a21b50e5089d2da70d5ac0a3bbbd1f617a93f134b76389980198';
create or replace view "Deposit_address_caller_address_owner_uint256_assets_d_uint256_shares_d" as select to_address(2,topic1::text) "caller",to_address(2,topic2::text) "owner",to_uint256(2,data::text) "assets",to_uint256(66,data::text) "shares", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0xdcbc1c05240f31ff3ad067ef1ee35ce4997762752e3a095284754544f4c709d7';
create or replace view "FeePercentUpdated_address_user_uint256_newFeePercent_d" as select to_address(2,topic1::text) "user",to_uint256(2,data::text) "newFeePercent", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0xec370615cc81fb334e5566fbc80664d9082377bf59288d64a79f3fbecf4323a9';
create or replace view "OwnershipTransferred_address_user_address_newOwner" as select to_address(2,topic1::text) "user",to_address(2,topic2::text) "newOwner", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0';
create or replace view "StrategyDeposit_address_user_uint256_underlyingAmount_d" as select to_address(2,topic1::text) "user",to_uint256(2,data::text) "underlyingAmount", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0xc6f6f91a48277d76f232cc08a9a30f6b05b3fd9b92c3180c25936e17a22a1025';
create or replace view "StrategyWithdrawal_address_user_uint256_underlyingAmount_d" as select to_address(2,topic1::text) "user",to_uint256(2,data::text) "underlyingAmount", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0xd5ad0f046bd35f48b421a3e575435de38cea1980177b1c6da935d2f26049f3fa';
create or replace view "TargetFloatPercentUpdated_address_user_uint256_newTargetFloatPercent_d" as select to_address(2,topic1::text) "user",to_uint256(2,data::text) "newTargetFloatPercent", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0x95bc4480b51f4860106d42850bcae222cf3303fb2b7d433e896205e0ebefe369';
create or replace view "Transfer_address_from_address_to_uint256_amount_d" as select to_address(2,topic1::text) "from",to_address(2,topic2::text) "to",to_uint256(2,data::text) "amount", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef';
create or replace view "Withdraw_address_caller_address_receiver_address_owner_uint256_assets_d_uint256_shares_d" as select to_address(2,topic1::text) "caller",to_address(2,topic2::text) "receiver",to_address(2,topic3::text) "owner",to_uint256(2,data::text) "assets",to_uint256(66,data::text) "shares", address contract_address, transaction_hash evt_tx_hash, log_index evt_index, block_timestamp evt_block_time, block_number evt_block_number from logs where topic0 = '0xfbde797d201c681b91056529119e0b02407c7bb96a4a2c75c01fc9667232c8db';
