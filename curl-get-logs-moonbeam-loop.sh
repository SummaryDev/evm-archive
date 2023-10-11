#!/usr/bin/env bash

from_block=222400
to_block=222430
block_step=10

filename="$from_block-$to_block.ndjson"

echo "starting with block $from_block, writing to to $filename"

for (( block_number=from_block; block_number<=to_block; block_number+=block_step ))
do
  filter_to_block=$((block_number+block_step-1))
  echo "$block_number $filter_to_block"
  curl https://moonbeam.api.onfinality.io/public --header 'Content-Type: application/json' --data-raw \
  "{ \"jsonrpc\":\"2.0\", \"method\":\"eth_getLogs\",\"id\":1,\"params\": {\"filter\":{\"fromBlock\": $block_number, \"toBlock\": $filter_to_block}} }" \
  | jq .result[] >> $filename
done

