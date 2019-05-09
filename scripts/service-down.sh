#!/usr/bin/env bash

# This script stops the following containers if they are running:
#  * runm-test-api
#  * runm-test-metadata
#  * runm-test-resource
#  * runm-test-etcd

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

ETCD_CONTAINER_NAME=${ETCD_CONTAINER_NAME:-"runm-test-etcd"}
MYSQL_CONTAINER_NAME=${MYSQL_CONTAINER_NAME:-"runm-test-mysql"}
METADATA_CONTAINER_NAME=${METADATA_CONTAINER_NAME:-"runm-test-metadata"}
RESOURCE_CONTAINER_NAME=${RESOURCE_CONTAINER_NAME:-"runm-test-resource"}
API_CONTAINER_NAME=${API_CONTAINER_NAME:-"runm-test-api"}

if container_exists "$ETCD_CONTAINER_NAME"; then
    inline_if_verbose "Destroying etcd container named $ETCD_CONTAINER_NAME... "
    container_destroy "$ETCD_CONTAINER_NAME"
    print_if_verbose "ok."
fi

if container_exists "$API_CONTAINER_NAME"; then
    inline_if_verbose "Destroying runm-api container named $API_CONTAINER_NAME... "
    container_destroy "$API_CONTAINER_NAME"
    print_if_verbose "ok."
fi

if container_exists "$METADATA_CONTAINER_NAME"; then
    inline_if_verbose "Destroying runm-metadata container named $METADATA_CONTAINER_NAME... "
    container_destroy "$METADATA_CONTAINER_NAME"
    print_if_verbose "ok."
fi

if container_exists "$RESOURCE_CONTAINER_NAME"; then
    inline_if_verbose "Destroying runm-resource container named $RESOURCE_CONTAINER_NAME... "
    container_destroy "$RESOURCE_CONTAINER_NAME"
    print_if_verbose "ok."
fi
