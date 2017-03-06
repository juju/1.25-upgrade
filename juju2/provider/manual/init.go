// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package manual

import "github.com/juju/1.25-upgrade/juju2/environs"

const (
	providerType = "manual"
)

func init() {
	p := manualProvider{}
	environs.RegisterProvider(providerType, p, "null")
}
