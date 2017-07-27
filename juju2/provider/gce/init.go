// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package gce

import "github.com/juju/1.25-upgrade/juju2/environs"

const (
	providerType = "gce"
)

func init() {
	environs.RegisterProvider(providerType, providerInstance)
}
