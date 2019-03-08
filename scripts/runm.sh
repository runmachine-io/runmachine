#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
VERBOSE=${VERBOSE:-0}
VERSION=$(git describe --tags --always --dirty)
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
SCRIPTS_DIR=$ROOT_DIR/scripts
LIB_DIR=$SCRIPTS_DIR/lib
BUILD_BIN_DIR=$ROOT_DIR/build/bin
RUNM_BIN_PATH=$BUILD_BIN_DIR/runm

source $LIB_DIR/common

if debug_enabled; then
    set -o xtrace
fi

runm_user=${RUNM_USER:-admin}
runm_project=${RUNM_PROJECT:-proj0}
runm_partition=${RUNM_PARTITION:-part0}

RUNM_USER="$runm_user" RUNM_PROJECT="$runm_project" RUNM_PARTITION="$runm_partition" $RUNM_BIN_PATH "$@"
