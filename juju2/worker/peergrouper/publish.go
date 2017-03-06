// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package peergrouper

import (
	"sync"

	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/instance"
	"github.com/juju/1.25-upgrade/juju2/network"
)

type apiHostPortsSetter interface {
	SetAPIHostPorts([][]network.HostPort) error
}

type publisher struct {
	st apiHostPortsSetter

	mu             sync.Mutex
	lastAPIServers [][]network.HostPort
}

func newPublisher(st apiHostPortsSetter) *publisher {
	return &publisher{st: st}
}

func (pub *publisher) publishAPIServers(apiServers [][]network.HostPort, instanceIds []instance.Id) error {
	if len(apiServers) == 0 {
		return errors.Errorf("no api servers specified")
	}
	pub.mu.Lock()
	defer pub.mu.Unlock()

	sortedAPIServers := make([][]network.HostPort, len(apiServers))
	for i, hostPorts := range apiServers {
		sortedAPIServers[i] = append([]network.HostPort{}, hostPorts...)
		network.SortHostPorts(sortedAPIServers[i])
	}
	if apiServersEqual(sortedAPIServers, pub.lastAPIServers) {
		logger.Debugf("API host ports have not changed")
		return nil
	}

	// TODO(rog) publish instanceIds in environment storage.
	err := pub.st.SetAPIHostPorts(sortedAPIServers)
	if err != nil {
		return err
	}
	pub.lastAPIServers = sortedAPIServers
	return nil
}

func apiServersEqual(a, b [][]network.HostPort) bool {
	if len(a) != len(b) {
		return false
	}
	for i, hostPortsA := range a {
		hostPortsB := b[i]
		if len(hostPortsA) != len(hostPortsB) {
			return false
		}
		for j := range hostPortsA {
			if hostPortsA[j] != hostPortsB[j] {
				return false
			}
		}
	}
	return true
}
