// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package factory

import (
	"github.com/juju/1.25-upgrade/juju2/container"
	"github.com/juju/1.25-upgrade/juju2/instance"
)

// DefaultNetworkBridge returns the correct network device name for the
// given container type. If there isn't a correct name or the type is
// unknown then an empty string is returned.
func DefaultNetworkBridge(cType instance.ContainerType) string {
	switch cType {
	case instance.LXD:
		return container.DefaultLxcBridge
	case instance.KVM:
		return container.DefaultKvmBridge
	default:
		return ""
	}
}
