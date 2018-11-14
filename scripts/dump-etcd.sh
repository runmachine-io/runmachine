#!/usr/bin/env bash

# Dumps the contents of an etcd data store's keys. Accepts a single optional
# CLI argument of the key prefix to filter results by.

DEBUG=${DEBUG:-0}
VERBOSE=${VERBOSE:-0}
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

EXEC_COMMAND="$@"
ETCD_CONTAINER_NAME=${ETCD_CONTAINER_NAME:-"runm-test-etcd"}

if ! container_is_running "$ETCD_CONTAINER_NAME"; then
    $SCRIPTS_DIR/start-etcd-container.sh "$ETCD_CONTAINER_NAME"
fi

if ! container_get_ip "$ETCD_CONTAINER_NAME" etcd_container_ip; then
    echo "ERROR: could not get IP for etcd container"
    exit 1
fi

key_prefix=${1:-"/"}

etcd_results=$(
    KEY=`echo "$key_prefix" | base64`; curl -LsS -X POST -d "{\"key\": \"$KEY\", \"range_end\": \"AA==\"}" \
       http://$etcd_container_ip:2379/v3alpha/kv/range | \
       python -mjson.tool | \
       grep "key" | \
       cut -d':' -f2 \
       | tr -d ' ",'
)
for payload in $etcd_results; do
        echo $payload | base64 --decode; echo "";
done
