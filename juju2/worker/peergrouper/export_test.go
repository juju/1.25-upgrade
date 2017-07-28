// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package peergrouper

import (
	"github.com/juju/1.25-upgrade/juju2/instance"
	"github.com/juju/1.25-upgrade/juju2/network"
)

var (
	NewWorker    = newWorker
	NewPublisher = newPublisher
)

func (pub *publisher) PublishAPIServers(apiServers [][]network.HostPort, instanceIds []instance.Id) error {
	return pub.publishAPIServers(apiServers, instanceIds)
}
