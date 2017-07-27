// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package maas

import (
	"github.com/juju/1.25-upgrade/juju2/environs"
)

const (
	providerType = "maas"
)

func init() {
	environs.RegisterProvider(providerType, MaasEnvironProvider{GetCapabilities: getCapabilities})
}
