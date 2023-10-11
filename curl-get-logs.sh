#!/usr/bin/env bash

#rpc=https://moonbeam.api.onfinality.io/public
rpc=http://localhost:8545
# rpc=https://docs-demo.quiknode.pro/
address=0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
fromBlock=0x`(printf '%x\n' 18266721)`
toBlock=0x`(printf '%x\n' 18266721)`

req="{ \"jsonrpc\":\"2.0\", \"method\":\"eth_getLogs\",\"id\":1,\"params\": [{\"address\":\"$address\", \"fromBlock\": \"$fromBlock\", \"toBlock\": \"$toBlock\"}] }"

echo $req | jq

curl $rpc -X POST -H "Content-Type: application/json" --data "$req" | jq
