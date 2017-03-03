#!/bin/bash

# basic stress test

set -e

while true; do
	go get -u -v github.com/juju/1.25-upgrade/juju1/utils
	export GOMAXPROCS=$[ 1 + $[ RANDOM % 128 ]]
        go test github.com/juju/1.25-upgrade/juju1/... 2>&1
done
