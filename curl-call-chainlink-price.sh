#!/usr/bin/env bash

rpc=http://localhost:8545
# rpc=https://docs-demo.quiknode.pro/
data=0x50d25bcd # latestAnswer
# see oracle addresses https://data.chain.link/ethereum/mainnet/crypto-usd
address=0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419 # ETH
# address=0xf4030086522a5beea4988f8ca5b36dbc97bee88c # BTC
blockNumber=0x`(printf '%x\n' 18327729)`
# blockNumber=latest

req="{ \"jsonrpc\":\"2.0\", \"method\":\"eth_call\",\"id\":1,\"params\": [{\"to\":\"$address\", \"data\": \"$data\"}, \"$blockNumber\"] }"

echo $req | jq

curl --silent --insecure $rpc -X POST -H "Content-Type: application/json" --data "$req" | jq

# curl --silent --insecure $rpc -X POST -H "Content-Type: application/json" --data "$req" | jq -r .result | xargs printf '%d\n'
