#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
VERBOSE=${VERBOSE:-0}
VERSION=$(git describe --tags --always --dirty)
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
SCRIPTS_DIR=$ROOT_DIR/scripts
LIB_DIR=$SCRIPTS_DIR/lib

source $LIB_DIR/common

check_is_installed docker

source $LIB_DIR/container
source $LIB_DIR/etcd
source $LIB_DIR/mysql

if debug_enabled; then
    set -o xtrace
fi

if ! container_image_exists "runmachine.io/runmachine/api:$VERSION"; then
    make build-api
fi

ETCD_CONTAINER_NAME=${ETCD_CONTAINER_NAME:-"runm-test-etcd"}
MYSQL_CONTAINER_NAME=${MYSQL_CONTAINER_NAME:-"runm-test-mysql"}
METADATA_CONTAINER_NAME=${METADATA_CONTAINER_NAME:-"runm-test-metadata"}
RESOURCE_CONTAINER_NAME=${RESOURCE_CONTAINER_NAME:-"runm-test-resource"}
API_CONTAINER_NAME=${API_CONTAINER_NAME:-"runm-test-api"}

if ! container_is_running "$ETCD_CONTAINER_NAME"; then
    $SCRIPTS_DIR/start-etcd-container.sh "$ETCD_CONTAINER_NAME"
fi

if ! container_get_ip "$ETCD_CONTAINER_NAME" etcd_container_ip; then
    echo "ERROR: could not get IP for etcd container"
    exit 1
fi

if ! container_is_running "$MYSQL_CONTAINER_NAME"; then
    $SCRIPTS_DIR/start-mysql-container.sh "$MYSQL_CONTAINER_NAME"
fi

if ! container_get_ip "$MYSQL_CONTAINER_NAME" mysql_container_ip; then
    echo "ERROR: could not get IP for mysql container"
    exit 1
fi

$SCRIPTS_DIR/start-runm-metadata-container.sh

inline_if_verbose "Grabbing IP for $METADATA_CONTAINER_NAME ... "
if container_get_ip "$METADATA_CONTAINER_NAME" metadata_container_ip; then
    print_if_verbose "ok."
    print_if_verbose "runm-metadata running in container at ${metadata_container_ip}:10000."
else
    echo "ERROR: could not get IP for runm-metadata container"
    exit 1
fi

$SCRIPTS_DIR/start-runm-resource-container.sh

inline_if_verbose "Grabbing IP for $RESOURCE_CONTAINER_NAME ... "
if container_get_ip "$RESOURCE_CONTAINER_NAME" resource_container_ip; then
    print_if_verbose "ok."
    print_if_verbose "runm-resource running in container at ${resource_container_ip}:10001."
else
    echo "ERROR: could not get IP for runm-resource container"
    exit 1
fi

if ! container_is_running "$API_CONTAINER_NAME"; then
    inline_if_verbose "Starting runm-api container named $API_CONTAINER_NAME... "
    docker run -d \
        --rm \
        -p 10000:10000 \
        --name $API_CONTAINER_NAME \
        -e GSR_LOG_LEVEL=3 \
        -e GSR_ETCD_ENDPOINTS="http://$etcd_container_ip:2379" \
        -e RUNM_LOG_LEVEL=3 \
        runmachine.io/runmachine/api:$VERSION >/dev/null 2>&1
    print_if_verbose "ok."
fi

inline_if_verbose "Grabbing IP for $API_CONTAINER_NAME ... "
if container_get_ip "$API_CONTAINER_NAME" api_container_ip; then
    print_if_verbose "ok."
    print_if_verbose "runm-api running in container at ${api_container_ip}:10002."
else
    echo "ERROR: could not get IP for runm-api container"
    exit 1
fi
