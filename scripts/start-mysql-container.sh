#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
LIB_DIR=$ROOT_DIR/scripts/lib

source $LIB_DIR/common

check_is_installed docker

source $LIB_DIR/container
source $LIB_DIR/mysql

if debug_enabled; then
    set -o xtrace
fi

CONTAINER_NAME=${1:-${CONTAINER_NAME:-"mysql"}}

echo -n "Starting mysql container named $CONTAINER_NAME ..."
if mysql_start_container "$CONTAINER_NAME"; then
    echo " ok."
else
    echo "FAIL"
fi

echo -n "Grabbing IP for $CONTAINER_NAME ... "
if container_get_ip "$CONTAINER_NAME" container_ip; then
    echo "ok."
    echo "mysql running in container at ${container_ip}:3306."
else
    echo "FAIL"
fi

