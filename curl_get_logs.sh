#!/usr/bin/env bash

curl https://moonbeam.api.onfinality.io/public --header 'Content-Type: application/json' --data-raw \
'{ "jsonrpc":"2.0", "method":"eth_getLogs","id":1,"params": {"filter":{"address":["0xcd3b51d98478d53f4515a306be565c6eebef1d58"], "fromBlock": 222431, "toBlock": 222431}} }' \
| jq