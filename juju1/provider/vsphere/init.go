// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build !gccgo

package vsphere

import (
	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/storage/provider/registry"
)

const (
	providerType = "vsphere"
)

func init() {
	environs.RegisterProvider(providerType, providerInstance)
	registry.RegisterEnvironStorageProviders(providerType)
}
