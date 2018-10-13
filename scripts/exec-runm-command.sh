#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
VERSION=$(git describe --tags --always --dirty)
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
LIB_DIR=$ROOT_DIR/scripts/lib

source $LIB_DIR/common

check_is_installed docker

source $LIB_DIR/container
source $LIB_DIR/etcd

if debug_enabled; then
    set -o xtrace
fi

docker image inspect runm-metadata:$VERSION >/dev/null 2>&1
if [ $? -ne 0 ]; then
    make build
fi

EXEC_COMMAND=$1
METADATA_CONTAINER_NAME=${METADATA_CONTAINER_NAME:-"runm-metadata"}

if ! container_is_running "$METADATA_CONTAINER_NAME"; then
    echo -n "Starting runm-metadata container named $METADATA_CONTAINER_NAME... "
    docker run -d \
        --rm \
        -p 10000:10000 \
        --name $METADATA_CONTAINER_NAME \
        -e RUNM_LOG_LEVEL=3 \
        runm-metadata:$VERSION # >/dev/null 2>&1
    echo "ok."
fi

echo -n "Grabbing IP for $METADATA_CONTAINER_NAME ... "
if container_get_ip "$METADATA_CONTAINER_NAME" metadata_container_ip; then
    echo "ok."
    echo "runm-metadata running in container at ${metadata_container_ip}:10000."
else
    echo "FAIL"
fi

docker run --rm -e RUNM_USER=$USER -e RUNM_HOST="http://$metadata_container_ip" runm:$VERSION runm $EXEC_COMMAND
