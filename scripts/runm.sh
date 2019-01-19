#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
VERBOSE=${VERBOSE:-0}
VERSION=$(git describe --tags --always --dirty)
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
SCRIPTS_DIR=$ROOT_DIR/scripts
LIB_DIR=$SCRIPTS_DIR/lib

source $LIB_DIR/common

if debug_enabled; then
    set -o xtrace
fi

source $SCRIPTS_DIR/service-up.sh

runm_user=${RUNM_USER:-admin}
runm_project=${RUNM_PROJECT:-proj0}
runm_partition=${RUNM_PARTITION:-part0}

docker run --rm --network host -v $ROOT_DIR/tests/data/:/tests/data \
    -e RUNM_USER="$runm_user" \
    -e RUNM_PROJECT="$runm_project" \
    -e RUNM_PARTITION="$runm_partition" \
    runm/runm:$VERSION "$@"
