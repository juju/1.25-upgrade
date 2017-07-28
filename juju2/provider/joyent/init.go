// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package joyent

import "github.com/juju/1.25-upgrade/juju2/environs"

const (
	providerType = "joyent"
)

func init() {
	environs.RegisterProvider(providerType, providerInstance)
}
