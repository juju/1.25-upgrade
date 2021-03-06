// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// This package exists solely to avoid circular imports.

package factory

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/container"
	"github.com/juju/1.25-upgrade/juju2/container/kvm"
	"github.com/juju/1.25-upgrade/juju2/container/lxd"
	"github.com/juju/1.25-upgrade/juju2/instance"
)

// NewContainerManager creates the appropriate container.Manager for the
// specified container type.
func NewContainerManager(forType instance.ContainerType, conf container.ManagerConfig) (container.Manager, error) {
	switch forType {
	case instance.LXD:
		return lxd.NewContainerManager(conf)
	case instance.KVM:
		return kvm.NewContainerManager(conf)
	}
	return nil, errors.Errorf("unknown container type: %q", forType)
}
