#!/usr/bin/env bash

curl https://moonbeam.api.onfinality.io/public --header 'Content-Type: application/json' --data-raw \
'{ "jsonrpc":"2.0", "method":"eth_getLogs","id":1,"params": {"filter":{"address":["0x985BcA32293A7A496300a48081947321177a86FD"], "fromBlock": 4009513, "toBlock": 4009513}} }' \
| jq