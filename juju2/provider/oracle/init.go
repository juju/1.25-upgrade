// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oracle

import "github.com/juju/1.25-upgrade/juju2/environs"

func init() {
	environs.RegisterProvider(providerType, &EnvironProvider{})
}
