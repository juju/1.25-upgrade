// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.
package machiner

import (
	"github.com/juju/errors"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/api/machiner"
	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/network"
	"github.com/juju/1.25-upgrade/juju2/status"
	"github.com/juju/1.25-upgrade/juju2/watcher"
)

type MachineAccessor interface {
	Machine(names.MachineTag) (Machine, error)
}

type Machine interface {
	Refresh() error
	Life() params.Life
	EnsureDead() error
	SetMachineAddresses(addresses []network.Address) error
	SetStatus(machineStatus status.Status, info string, data map[string]interface{}) error
	Watch() (watcher.NotifyWatcher, error)
	SetObservedNetworkConfig(netConfig []params.NetworkConfig) error
}

type APIMachineAccessor struct {
	State *machiner.State
}

func (a APIMachineAccessor) Machine(tag names.MachineTag) (Machine, error) {
	m, err := a.State.Machine(tag)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return m, nil
}
