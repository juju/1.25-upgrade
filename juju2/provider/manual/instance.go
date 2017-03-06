// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package manual

import (
	"github.com/juju/1.25-upgrade/juju2/environs/manual"
	"github.com/juju/1.25-upgrade/juju2/instance"
	"github.com/juju/1.25-upgrade/juju2/network"
	"github.com/juju/1.25-upgrade/juju2/status"
)

type manualBootstrapInstance struct {
	host string
}

func (manualBootstrapInstance) Id() instance.Id {
	return BootstrapInstanceId
}

func (manualBootstrapInstance) Status() instance.InstanceStatus {
	// We asume that if we are deploying in manual provider the
	// underlying machine is clearly running.
	return instance.InstanceStatus{
		Status: status.Running,
	}
}

func (manualBootstrapInstance) Refresh() error {
	return nil
}

func (inst manualBootstrapInstance) Addresses() (addresses []network.Address, err error) {
	addr, err := manual.HostAddress(inst.host)
	if err != nil {
		return nil, err
	}
	return []network.Address{addr}, nil
}

func (manualBootstrapInstance) OpenPorts(machineId string, ports []network.PortRange) error {
	return nil
}

func (manualBootstrapInstance) ClosePorts(machineId string, ports []network.PortRange) error {
	return nil
}

func (manualBootstrapInstance) Ports(machineId string) ([]network.PortRange, error) {
	return nil, nil
}
