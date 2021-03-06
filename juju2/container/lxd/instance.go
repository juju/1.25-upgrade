// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxd

import (
	"fmt"

	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/instance"
	"github.com/juju/1.25-upgrade/juju2/network"
	"github.com/juju/1.25-upgrade/juju2/status"
	"github.com/juju/1.25-upgrade/juju2/tools/lxdclient"
)

type lxdInstance struct {
	id     string
	client *lxdclient.Client
}

var _ instance.Instance = (*lxdInstance)(nil)

// Id implements instance.Instance.Id.
func (lxd *lxdInstance) Id() instance.Id {
	return instance.Id(lxd.id)
}

func (*lxdInstance) Refresh() error {
	return nil
}

func (lxd *lxdInstance) Addresses() ([]network.Address, error) {
	return nil, errors.NotImplementedf("lxdInstance.Addresses")
}

// Status implements instance.Instance.Status.
func (lxd *lxdInstance) Status() instance.InstanceStatus {
	jujuStatus := status.Pending
	instStatus, err := lxd.client.Status(lxd.id)
	if err != nil {
		return instance.InstanceStatus{
			Status:  status.Empty,
			Message: fmt.Sprintf("could not get status: %v", err),
		}
	}
	switch instStatus {
	case lxdclient.StatusStarting, lxdclient.StatusStarted:
		jujuStatus = status.Allocating
	case lxdclient.StatusRunning:
		jujuStatus = status.Running
	case lxdclient.StatusFreezing, lxdclient.StatusFrozen, lxdclient.StatusThawed, lxdclient.StatusStopping, lxdclient.StatusStopped:
		jujuStatus = status.Empty
	default:
		jujuStatus = status.Empty
	}
	return instance.InstanceStatus{
		Status:  jujuStatus,
		Message: instStatus,
	}
}

// OpenPorts implements instance.Instance.OpenPorts.
func (lxd *lxdInstance) OpenPorts(machineId string, rules []network.IngressRule) error {
	return fmt.Errorf("not implemented")
}

// ClosePorts implements instance.Instance.ClosePorts.
func (lxd *lxdInstance) ClosePorts(machineId string, rules []network.IngressRule) error {
	return fmt.Errorf("not implemented")
}

// IngressRules implements instance.Instance.IngressRules.
func (lxd *lxdInstance) IngressRules(machineId string) ([]network.IngressRule, error) {
	return nil, fmt.Errorf("not implemented")
}

// Add a string representation of the id.
func (lxd *lxdInstance) String() string {
	return fmt.Sprintf("lxd:%s", lxd.id)
}
