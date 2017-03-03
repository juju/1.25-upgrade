// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package local

import (
	"github.com/juju/1.25-upgrade/juju1/environs"
	storageprovider "github.com/juju/1.25-upgrade/juju1/storage/provider"
	"github.com/juju/1.25-upgrade/juju1/storage/provider/registry"
)

const (
	providerType = "local"
)

func init() {
	environs.RegisterProvider(providerType, providerInstance)

	// TODO(wallyworld) - sort out policy for allowing loop provider
	registry.RegisterEnvironStorageProviders(
		providerType,
		storageprovider.HostLoopProviderType,
	)
}
