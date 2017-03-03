// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ec2

import (
	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/storage/provider/registry"
)

const (
	providerType = "ec2"
)

func init() {
	environs.RegisterProvider(providerType, environProvider{})

	//Register the AWS specific providers.
	registry.RegisterProvider(EBS_ProviderType, &ebsProvider{})

	// Inform the storage provider registry about the AWS providers.
	registry.RegisterEnvironStorageProviders(providerType, EBS_ProviderType)
}
