# EVM Archive

This is an ETL utility to archive logs emitted by smart contracts in Ethereum or another EVM
compatible blockchain into a Postgres database.

- Extracts logs from a full blockchain node by calling
  [eth_getLogs](https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_getlogs)
  RPC method for a range of blocks and smart contract addresses.
- Transforms at a minimum by splitting topics array into 4 individual attributes and converting
  indexes from hex to decimals.
- Loads into `logs` table in the Postgres database.

From this response from a blockchain node (try it with [curl-get-logs.sh](./curl-get-logs.sh)):

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

## Quick start

To try it out quickly start in docker containers by `docker-compose up`:
- event log [archiver](./docker-compose.yaml#L16)
- postgres [database](./docker-compose.yaml#L4) with SQL [functions](https://github.com/SummaryDev/ethereum-sql) and views to decode event payloads 
- GraphQL [engine](./docker-compose.yaml#L43) by [Graphile](https://www.graphile.org/postgraphile/introduction/)

When starting with an empty database for the first time, [init.sql](./init.sql) script will create the schema, 
functions to decode payload. Add view definitions for your contracts' events at the end of the file. 
You can generate them from contracts' ABIs with [ethereum-sql](https://github.com/SummaryDev/ethereum-sql).

The postgres container keeps its data on your host in a local folder `postgres-data` to persist it between restarts.

That was just a quick start to try things out: all the data are kpt in one `public` schema and there a few event definitions we're interested in. 
The instructions below assume a more thorough setup with a separate db instance, deployments to Kubernetes.

## Create the database

We name our database by the name of the blockchain (ethereum or moonbeam), its network (mainnet or
goerli), and a namespace to separate dev and prod deployments, like `ethereum_goerli_dev` or
`moonbeam_mainnet_prod`.

We create a separate database and not a schema because we use schemas to group views that decode raw
logs into readable events: `event.Transfer_address_from_address_to_uint256_value_d` is grouped into
schema `event` together with other logs we can decode.

Other views that decode contract events are grouped into schemas representing labels like project
names or standards (aave, beamswap, erc20) like `beamswap.GlintTokenV1_evt_Transfer`.

You don't have to follow this pattern and can create `logs` table in any schema in any database
you'd like. Use the table's definition from [schema.sql](./schema.sql).

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
- `EVM_ARCHIVE_FROM_BLOCK` will start getting logs from this block number unless the block saved in the db is higher. Use this as a starting point when the logs table is empty. Note that inserts won't fail (we ignore with `do nothing` violations of primary key in the insert statement).
- `EVM_ARCHIVE_TO_BLOCK` block number to end with, if not specified will keep querying to infinity with some sleep in between.
- `EVM_ARCHIVE_BLOCK_STEP` how many blocks to extract the logs from, if omitted will default to 100.
  Adjust this parameter to the frequency of the logs you're extracting. Too big a value can lead to
  the blockchain node refusing to return too many records, too small can slow down the extraction
  process.
- `EVM_ARCHIVE_SLEEP_SECONDS` sleep between queries. Makes sense to adjust to about block creation time or longer.

Set these parameters from an [env](./example.env) file and run.

```shell
. .env
./evm-archive
```

## Build and deploy to Kubernetes

Build docker image and push it to your repository on docker.io.

```shell
image_evm_archive="mydockeriousername/evm-archive:latest" ./build.sh
```

Deploy to your k8s cluster with *kubectl*. Edit [deploy.yaml](./deploy.sh) to set env variables to
control the logs filter like contract addresses. Script [deploy.sh](./deploy.sh) has a convenient
function with which you can deploy multiple  
instances each archiving a different chain or its shard, like for
`ethereum-mainnet-prod`, `ethereum-mainnet-dev`, `ethereum-goerli2023-prod`,
`moonbeam-mainnet-dev` etc. Will first try to create the database, schema and tables
with [db-create.sh](./db-create.sh). Docker image if not specified will default to
`olegabu/evm-archive:latest`.

```shell
db_host=db-postgresql.local db_password_evm_archive=evm_archive123 namespace=prod ./deploy.sh
```

## Why we store raw logs

The raw event log data is hard to analyze. It needs to be decoded so its numeric values can be used
in calculations and its hex values be seen as text and addresses.

The usual approach to archive onchain data is to *index* smart contracts with *subgraphs*: extract
data of specific contracts, decode it on the fly by code specially written for them, and store
decoded events for querying.

We postpone the decoding stage to the time of the actual query: the data is stored encoded as
received. This gives us flexibility in interpreting data: we don't have to know the contract's ABI
at the time of event capture. This approach does not require efforts to index specific contracts or
write subgraph code to interpret them. We can extract and load into our database raw logs of as 
many contracts as we may be interested in, or of all the contracts for a range of dates. Then 
we decode on the fly when we run a query.

For decoding we employ user defined functions within SQL select statements. Once we obtain the
contract's ABI we know what functions (to_uint256, to_address) to apply to raw data and what
names to give to the resulting columns.

This turns the usual ETL approach to ELT: extract, load raw, transform when reading. We extract logs
of all contracts and load them raw into the database to be queried by functions and views that know
how to decode them.

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

Which you compose knowing the event's ABI:

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
[ethereum-sql](https://github.com/SummaryDev/ethereum-sql) for SQL functions to decode event values
and scripts to turn ABI files into database views.
