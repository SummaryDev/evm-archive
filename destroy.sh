#!/usr/bin/env bash

env | grep 'namespace\|db_host\|image'

function fun() {
    evm_chain=$1
    evm_network=$2
    evm_shard=$3

    kubectl --namespace ${namespace} delete deployment evm-archive-${evm_chain}-${evm_network}${evm_shard}
}

fun moonbeam mainnet
