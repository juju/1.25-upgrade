#!/bin/bash
# Copyright 2012 Canonical Ltd.
# Licensed under the AGPLv3, see LICENCE file for details.

# basic stress test

set -e

while true; do
	go get -u -v github.com/juju/1.25-upgrade/juju2/utils
	export GOMAXPROCS=$[ 1 + $[ RANDOM % 128 ]]
        go test github.com/juju/1.25-upgrade/juju2/... 2>&1
done
