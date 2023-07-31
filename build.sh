#!/usr/bin/env bash

export image_evm_archive=${image_evm_archive-"olegabu/evm-archive:latest"}

echo "build and push $image_evm_archive"

docker build -t $image_evm_archive . && docker push $image_evm_archive