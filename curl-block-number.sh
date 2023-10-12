#!/usr/bin/env bash

#rpc=https://moonbeam.api.onfinality.io/public
rpc=http://localhost:8545
# rpc=https://docs-demo.quiknode.pro/

req="{ \"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\",\"id\":1,\"params\": [] }"

echo $req | jq

# curl $rpc -X POST -H "Content-Type: application/json" --data "$req" | jq

curl $rpc -X POST -H "Content-Type: application/json" --data "$req" | jq -r .result | xargs printf '%d\n'
