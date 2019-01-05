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

docker image inspect runm/metadata:$VERSION >/dev/null 2>&1
if [ $? -ne 0 ]; then
    make build
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

inline_if_verbose "Creating resource database ... "
mysql -uroot -P3306 -h$mysql_container_ip -e "CREATE DATABASE IF NOT EXISTS runm_resource;"
print_if_verbose "ok."

if ! container_is_running "$METADATA_CONTAINER_NAME"; then
    inline_if_verbose "Starting runm-metadata container named $METADATA_CONTAINER_NAME... "
    docker run -d \
        --rm \
        -p 10000:10000 \
        --name $METADATA_CONTAINER_NAME \
        -e GSR_LOG_LEVEL=3 \
        -e GSR_ETCD_ENDPOINTS="http://$etcd_container_ip:2379" \
        -e RUNM_LOG_LEVEL=3 \
        -e RUNM_METADATA_STORAGE_ETCD_ENDPOINTS="http://$etcd_container_ip:2379" \
        -e RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX="$METADATA_CONTAINER_NAME" \
        -e RUNM_METADATA_BOOTSTRAP_TOKEN="bootstrapme" \
        runm/metadata:$VERSION >/dev/null 2>&1
    print_if_verbose "ok."
fi

inline_if_verbose "Grabbing IP for $METADATA_CONTAINER_NAME ... "
if container_get_ip "$METADATA_CONTAINER_NAME" metadata_container_ip; then
    print_if_verbose "ok."
    print_if_verbose "runm-metadata running in container at ${metadata_container_ip}:10000."
else
    echo "ERROR: could not get IP for runm-metadata container"
    exit 1
fi

if ! container_is_running "$RESOURCE_CONTAINER_NAME"; then
    inline_if_verbose "Starting runm-resource container named $RESOURCE_CONTAINER_NAME... "
    docker run -d \
        --rm \
        -p 10001:10001 \
        --name $RESOURCE_CONTAINER_NAME \
        -e GSR_LOG_LEVEL=3 \
        -e GSR_ETCD_ENDPOINTS="http://$etcd_container_ip:2379" \
        -e RUNM_LOG_LEVEL=3 \
        -e RUNM_RESOURCE_STORAGE_DSN="root:@tcp($mysql_container_ip:3306)/runm_resource" \
        runm/resource:$VERSION >/dev/null 2>&1
    print_if_verbose "ok."
fi

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
        -p 10002:10002 \
        --name $API_CONTAINER_NAME \
        -e GSR_LOG_LEVEL=3 \
        -e GSR_ETCD_ENDPOINTS="http://$etcd_container_ip:2379" \
        -e RUNM_LOG_LEVEL=3 \
        runm/api:$VERSION >/dev/null 2>&1
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

print_if_verbose ""
print_if_verbose "*********************************************************************"
print_if_verbose "Running \`runm $*\` in single-use docker container..."
print_if_verbose "*********************************************************************"
print_if_verbose ""

runm_user=${RUNM_USER:-admin}
runm_project=${RUNM_PROJECT:-proj0}
runm_partition=${RUNM_PARTITION:-part0}

docker run --rm --network host -v $ROOT_DIR/tests/data/:/tests/data \
    -e RUNM_USER="$runm_user" \
    -e RUNM_PROJECT="$runm_project" \
    -e RUNM_PARTITION="$runm_partition" \
    runm/runm:$VERSION "$@"
