// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package dummy

import (
	dummystorage "github.com/juju/1.25-upgrade/juju1/storage/provider/dummy"
	"github.com/juju/1.25-upgrade/juju1/storage/provider/registry"
)

func init() {
	registry.RegisterEnvironStorageProviders("dummy", "dummy")
	registry.RegisterProvider("dummy", &dummystorage.StorageProvider{})
}
