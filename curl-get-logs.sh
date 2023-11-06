#!/usr/bin/env bash

#rpc=https://moonbeam.api.onfinality.io/public
#rpc=http://localhost:8545
# rpc=https://docs-demo.quiknode.pro/
#address=0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
#fromBlock=0x`(printf '%x\n' 18266721)`
#toBlock=0x`(printf '%x\n' 18266721)`

rpc=https://sepolia.infura.io/v3/aaa2d467d67d42c9861f4afdbc599e3e
#address='["0xc45634B24447FA468baEE75A6695bD6547d7be98","0xEfe67EdcC7b8641204E9312Caeeb8CAf3C169846","0xbB55Eec677403BBa73D0B439676DB715B7C7FA45","0xcfd90fc58637a10b435ee80584e81d542341d0c7","0x6a13200debd8efe07bdfdadc66f4b7bb46f51206"]'
address='["0xbB55Eec677403BBa73D0B439676DB715B7C7FA45"]'
fromBlock=0x`(printf '%x\n' 4638833)`
toBlock=0x`(printf '%x\n' 4638833)`

req="{ \"jsonrpc\":\"2.0\", \"method\":\"eth_getLogs\",\"id\":1,\"params\": [{\"address\":$address, \"fromBlock\": \"$fromBlock\", \"toBlock\": \"$toBlock\"}] }"

echo $req

echo $req | jq

curl --silent --insecure $rpc -X POST -H "Content-Type: application/json" --data "$req" | jq
