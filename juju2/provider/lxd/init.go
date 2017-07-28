// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxd

import (
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/provider/lxd/lxdnames"
)

func init() {
	environs.RegisterProvider(lxdnames.ProviderType, NewProvider())
}
