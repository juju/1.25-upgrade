// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package openstack

import (
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/environs/tools"
)

const (
	providerType = "openstack"
)

func init() {
	environs.RegisterProvider(providerType, providerInstance)

	environs.RegisterImageDataSourceFunc("keystone catalog", getKeystoneImageSource)
	tools.RegisterToolsDataSourceFunc("keystone catalog", getKeystoneToolsSource)
}
