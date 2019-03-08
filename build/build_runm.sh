#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
VERBOSE=${VERBOSE:-0}
VERSION=$(git describe --tags --always --dirty)
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
CMD_RUNM_DIR=$ROOT_DIR/cmd/runm
BUILD_BIN_DIR=$ROOT_DIR/build/bin

if [[ ! -d $BUILD_BIN_DIR ]]; then
    mkdir $BUILD_BIN_DIR
fi

# Clean up any previously-built binary
if [[ -f $BUILD_BIN_DIR/runm ]]; then
    echo -n "Removing old runm binary ... "
    rm $BUILD_BIN_DIR/runm
    echo "ok."
fi

cd $CMD_RUNM_DIR

echo -n "Building latest runm binary ... "
CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o $BUILD_BIN_DIR/runm .
echo "ok."
