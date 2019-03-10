#!/usr/bin/env bash

# Used for "resetting" a test environment back to a clean start state.
#
# This script stops the following containers if they are running:
#  * runm-test-api
#  * runm-test-metadata
#  * runm-test-resource
#  * runm-test-etcd
#
# And then proceeds to clear out the SQL database. We do not attempt to
# stop/start the runm-test-mysql container because this container takes a
# stupid long time to start up due to the init scripts used by the MySQL Docker
# container. Instead, we just DROP and re-CREATE the database.
#
# When the runm-test-etcd container is re(started), the etcd service will be
# re-created from scratch. When the runm-test-metadata service (re)starts, it
# will create the necessary objects in the runm-test-etcd data store. When the
# runm-test-resource service (re)starts, it will create all the necessary DB
# tables in the database housed in the runm-test-mysql container.

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

if ! container_is_running "$MYSQL_CONTAINER_NAME"; then
    $SCRIPTS_DIR/start-mysql-container.sh "$MYSQL_CONTAINER_NAME"
fi

if ! container_get_ip "$MYSQL_CONTAINER_NAME" mysql_container_ip; then
    echo "ERROR: could not get IP for mysql container"
    exit 1
fi

inline_if_verbose "Dropping resource database ... "
mysql -uroot -P3306 -h$mysql_container_ip -e "DROP DATABASE IF EXISTS runm_resource;"
print_if_verbose "ok."

$SCRIPTS_DIR/start-etcd-container.sh "$ETCD_CONTAINER_NAME"
$SCRIPTS_DIR/start-runm-metadata-container.sh
$SCRIPTS_DIR/start-runm-resource-container.sh
$SCRIPTS_DIR/start-runm-api-container.sh
$SCRIPTS_DIR/populate.sh
