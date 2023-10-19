#!/usr/bin/env bash

from_block=222400
to_block=222430
block_step=10
filename="$from_block-$to_block.ndjson"
rpc=https://moonbeam.api.onfinality.io/public

echo "calling $rpc blocks $from_block to $to_block, writing to $filename"

for (( block_number=from_block; block_number<=to_block; block_number+=block_step ))
do
  filter_to_block=$((block_number+block_step-1))
  req="{ \"jsonrpc\":\"2.0\", \"method\":\"eth_getLogs\",\"id\":1,\"params\": {\"filter\":{\"fromBlock\": $block_number, \"toBlock\": $filter_to_block}} }"
  echo $req
  curl --silent --insecure $rpc --header 'Content-Type: application/json' --data-raw "$req" | jq .result[] >> $filename
done

