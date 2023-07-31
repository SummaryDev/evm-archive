# EVM Archive

This is an ETL utility to archive logs emitted by smart contracts in Ethereum or another EVM compatible
blockchain into a Postgres database.

- Extracts logs from a full blockchain node by calling
  [eth_getLogs](https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_getlogs)
  RPC method for a range of blocks and smart contract addresses.
- Transforms at a minimum by splitting topics array into 4 individual attributes and converting
  indexes from hex to decimals.
- Loads into `logs` table in the Postgres database.

From this response from a blockchain node (try it with [curl_get_logs.sh](./curl_get_logs.sh)):

```json
{
  "address": "0xcd3b51d98478d53f4515a306be565c6eebef1d58",
  "topics": [
    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
    "0x0000000000000000000000000000000000000000000000000000000000000000",
    "0x000000000000000000000000f78031c993afb43e79f017938326ff34418ec36e"
  ],
  "data": "0x000000000000000000000000000000000000000000000000aad50c474db4eb50",
  "blockHash": "0x09f1e5619fcbfaa873fcf4e924b724dac6b84e0f9c02341f75c11393d586792b",
  "blockNumber": "0x364df",
  "transactionHash": "0xf9a7cefb1ab525781aac1b0ca29bf76b90cd2f16e22ee9e91cf7d2dcae78aa08",
  "transactionIndex": "0x6",
  "logIndex": "0x12",
  "transactionLogIndex": "0x1",
  "removed": false
}
```

We store this row in Postgres:

| address                                    | topic0                                                             | topic1                                                             | topic2                                                             | topic3 | data                                                               | block_hash                                                         | block_number | transaction_hash                                                   | transaction_index | log_index | transaction_log_index | removed | block_timestamp |
|--------------------------------------------|--------------------------------------------------------------------|--------------------------------------------------------------------|--------------------------------------------------------------------|--------|--------------------------------------------------------------------|--------------------------------------------------------------------|--------------|--------------------------------------------------------------------|-------------------|-----------|-----------------------|---------|-----------------|
| 0xcd3b51d98478d53f4515a306be565c6eebef1d58 | 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef | 0x0000000000000000000000000000000000000000000000000000000000000000 | 0x000000000000000000000000f78031c993afb43e79f017938326ff34418ec36e |        | 0x000000000000000000000000000000000000000000000000aad50c474db4eb50 | 0x09f1e5619fcbfaa873fcf4e924b724dac6b84e0f9c02341f75c11393d586792b | 222431       | 0xf9a7cefb1ab525781aac1b0ca29bf76b90cd2f16e22ee9e91cf7d2dcae78aa08 | 6                 | 18        | 1                     | false   |                 |

## Create the database

We name our database by the name of the blockchain (ethereum or moonbeam), its network (mainnet
or goerli), and a namespace to separate dev and prod deployments, like `ethereum_goerli_dev` or
`moonbeam_mainnet_prod`.

We create a separate database and not a schema because we use schemas to group views that decode raw
logs into readable events: `event.Transfer_address_from_address_to_uint256_value_d` is grouped into
schema `event` together with other logs we can decode.

Other views that decode contract events are grouped into schemas representing labels like project
names or standards (aave, beamswap, erc20) like `beamswap.GlintTokenV1_evt_Transfer`.

You don't have to follow this pattern and can create `logs` table in any schema in any database
you'd like.
Use the table's definition from [schema.sql](./schema.sql).

This script [db-create.sh](./db-create.sh) will create the database using admin user `postgres`
authenticated by `db_password`, and the user this app will connect as: `evm_archive` authenticated
by `db_password_evm_archive`. Make sure Postgres client `psql` is installed on your local machine.

```shell
db_host=db-postgresql.local db_password=postgres123 db_password_evm_archive=evm_archive123 \
namespace=prod evm_chain=moonbeam evm_network=mainnet \
./db-create.sh
```

## Build and run

This app was tested with Go 1.19 and should build with other versions.

```shell
go build
```

You supply database and blockchain node connection parameters via env variables. We list them in
[example.env](./example.env): the usual Postgres `PGHOST`, `PGPASSWORD` etc.
and `EVM_ARCHIVE_ENDPOINT`.

Query filter is controlled by env variables:

- `EVM_ARCHIVE_CONTRACTS` comma delimited contract addresses, if omitted will query for logs emitted
  by all contracts
- `EVM_ARCHIVE_FROM_BLOCK` block number to start with, if omitted will query repeatedly for the
  latest block. TODO:
  Need to first get the latest block number inserted from the database then start the query from it.
  Otherwise we will query from the same block defined by this env variable on process restart. Note
  that inserts won't
  fail (we ignore with `do nothing` violations of primary key in the insert statement).
- `EVM_ARCHIVE_TO_BLOCK` block number to end with, if omitted will query indefinitely for subsequent
  blocks.
- `EVM_ARCHIVE_BLOCK_STEP` how many blocks to extract the logs from, if omitted will default to 100.
  Adjust this
  parameter to the frequency of the logs you're extracting. Too big a value can lead to the
  blockchain node
  refusing to return too many records, too small can slow down the extraction process. TODO: This
  logic can be
  improved by adjusting this value dynamically: start with a large number and decrease it when node
  complains,
  increase it again when start getting too few records per call.
- `EVM_ARCHIVE_SLEEP_SECONDS` will pause in between queries for the latest block
  when `EVM_ARCHIVE_FROM_BLOCK` is
  not set. Adjust this so the total time between calls (time to extract and insert plus sleep time)
  is less than
  the time to produce a new block. TODO: This logic of querying for the latest block is flawed as a
  query and insert
  which take longer than block time can make the next call for the latest skip a block.

Set these parameters from an [env](./example.env) file and run.

```shell
. .env
./evm-archive
```

## Why we store raw logs

The raw event log data is hard to analyze. It needs to be decoded so its numeric values can be used
in calculations and its hex values be seen as text and addresses.

The usual approach to archive onchain data is to *index* smart contracts with *subgraphs*: extract
data of specific contracts, decode it on the fly by code specially written for its events, and
store decoded for querying.

We postpone the decoding stage to the time of the actual query: the data is stored encoded as
received and gets decoded later. This gives us flexibility in interpreting data: we don't have 
to know the contract's ABI at the time of event capture. This approach does not require
efforts to index specific contracts or write subgraph code to interpret them.

For decoding we employ user defined functions within SQL select statements. Once we obtain 
the contract's ABI we know what functions (to_uint256, to_address) to apply to raw columns and what 
names to give to resulting columns.

This turns the usual ETL approach to ELT: extract, load raw, transform when reading. We extract
logs of all contracts and load them raw into the database to be queried by functions and views
that know how to decode them.

From this row with a raw log:

| address                                    | topic0                                                             | topic1                                                             | topic2                                                             | topic3 | data                                                               | block_hash                                                         | block_number | transaction_hash                                                   | transaction_index | log_index | transaction_log_index | removed | block_timestamp |
|--------------------------------------------|--------------------------------------------------------------------|--------------------------------------------------------------------|--------------------------------------------------------------------|--------|--------------------------------------------------------------------|--------------------------------------------------------------------|--------------|--------------------------------------------------------------------|-------------------|-----------|-----------------------|---------|-----------------|
| 0xcd3b51d98478d53f4515a306be565c6eebef1d58 | 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef | 0x0000000000000000000000000000000000000000000000000000000000000000 | 0x000000000000000000000000f78031c993afb43e79f017938326ff34418ec36e |        | 0x000000000000000000000000000000000000000000000000aad50c474db4eb50 | 0x09f1e5619fcbfaa873fcf4e924b724dac6b84e0f9c02341f75c11393d586792b | 222431       | 0xf9a7cefb1ab525781aac1b0ca29bf76b90cd2f16e22ee9e91cf7d2dcae78aa08 | 6                 | 18        | 1                     | false   |                 |

You can get to this row of a decoded `Transfer` event:

| from                                       | to                                         | value                | contract_address                           |
|--------------------------------------------|--------------------------------------------|----------------------|--------------------------------------------|
| 0x0000000000000000000000000000000000000000 | 0xf78031c993afb43e79f017938326ff34418ec36e | 12309758656873032448 | 0xcd3b51d98478d53f4515a306be565c6eebef1d58 |

With a SQL select like:

```sql
select to_address(topic1) "from",
       to_address(topic2) "to",
       to_uint256(data)   "value",
       address            contract_address
from data.logs
where topic0 = '0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef';
```

When you know the event's ABI:

```json
{
  "anonymous": false,
  "inputs": [
    {
      "indexed": true,
      "name": "from",
      "type": "address"
    },
    {
      "indexed": true,
      "name": "to",
      "type": "address"
    },
    {
      "indexed": false,
      "name": "value",
      "type": "uint256"
    }
  ],
  "name": "Transfer",
  "type": "event"
}
```

Please see our other repo
[ethereum-sql](https://github.com/SummaryDev/ethereum-sql) for SQL functions to decode event values and 
scripts to turn ABI files into database views.
