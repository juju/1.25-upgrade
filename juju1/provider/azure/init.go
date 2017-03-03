// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package azure

import (
	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/storage/provider/registry"
)

const (
	providerType = "azure"
)

func init() {
	environs.RegisterProvider(providerType, azureEnvironProvider{})

	// Register the Azure storage provider.
	registry.RegisterProvider(storageProviderType, &azureStorageProvider{})
	registry.RegisterEnvironStorageProviders(providerType, storageProviderType)
}
