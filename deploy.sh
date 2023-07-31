#!/usr/bin/env bash

export image_evm_archive=${image_evm_archive-"olegabu/evm-archive:latest"}

env | grep 'namespace\|evm\|db_host\|image'

function fun {
    evm_chain=$1
    evm_network=$2
    evm_shard=$3

    namespace=$namespace evm_chain=$evm_chain evm_network=$evm_network ./db-create.sh

    cat deploy.yaml | \
    sed 's/${namespace}/namespace/g' | sed "s/namespace/$namespace/g" | \
    sed 's/${evm_chain}/evm_chain/g' | sed "s/evm_chain/$evm_chain/g" | \
    sed 's/${evm_network}/evm_network/g' | sed "s/evm_network/$evm_network/g" | \
    sed 's/${evm_shard}/evm_shard/g' | sed "s/evm_shard/$evm_shard/g" | \
    sed 's/${image_evm_archive}/image_evm_archive/g' | sed "s@image_evm_archive@${image_evm_archive}@g" |\
    sed 's/${db_password_evm_archive}/db_password_evm_archive/g' | sed "s/db_password_evm_archive/$db_password_evm_archive/g" | \
    sed 's/${db_host}/db_host/g' | sed "s/db_host/$db_host/g" | \
    kubectl --namespace ${namespace} -f - apply

#    --dry-run=client
}

fun moonbeam mainnet
