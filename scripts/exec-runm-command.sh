#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
VERSION=$(git describe --tags --always --dirty)
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
SCRIPTS_DIR=$ROOT_DIR/scripts
LIB_DIR=$SCRIPTS_DIR/lib

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
ETCD_CONTAINER_NAME=${ETCD_CONTAINER_NAME:-"runm-test-etcd"}
METADATA_CONTAINER_NAME=${METADATA_CONTAINER_NAME:-"runm-test-metadata"}

if ! container_is_running "$ETCD_CONTAINER_NAME"; then
    $SCRIPTS_DIR/start-etcd-container.sh "$ETCD_CONTAINER_NAME"
fi

if ! container_get_ip "$ETCD_CONTAINER_NAME" etcd_container_ip; then
    echo "ERROR: failed to get etcd container IP"
    exit 1
fi

if ! container_is_running "$METADATA_CONTAINER_NAME"; then
    echo -n "Starting runm-metadata container named $METADATA_CONTAINER_NAME... "
    docker run -d \
        --rm \
        -p 10000:10000 \
        --name $METADATA_CONTAINER_NAME \
        -e GSR_LOG_LEVEL=3 \
        -e GSR_ETCD_ENDPOINTS="http://$etcd_container_ip:2379" \
        -e RUNM_LOG_LEVEL=3 \
        -e RUNM_METADATA_STORAGE_ETCD_ENDPOINTS="http://$etcd_container_ip:2379" \
        -e RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX="$METADATA_CONTAINER_NAME" \
        runm-metadata:$VERSION # >/dev/null 2>&1
    echo "ok."
fi

echo -n "Grabbing IP for $METADATA_CONTAINER_NAME ... "
if container_get_ip "$METADATA_CONTAINER_NAME" metadata_container_ip; then
    echo "ok."
    print_if_verbose "runm-metadata running in container at ${metadata_container_ip}:10000."
else
    echo "FAIL"
    exit 1
fi

docker run --rm --network host runm:$VERSION runm $EXEC_COMMAND
