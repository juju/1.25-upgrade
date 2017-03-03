// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package gce

import (
	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/storage/provider/registry"
)

const (
	providerType = "gce"
)

func init() {
	environs.RegisterProvider(providerType, providerInstance)

	// Register the GCE specific providers.
	registry.RegisterProvider(storageProviderType, &storageProvider{})

	// Inform the storage provider registry about the GCE providers.
	registry.RegisterEnvironStorageProviders(providerType, storageProviderType)
}
