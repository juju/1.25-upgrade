// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package dummy

import (
	"github.com/juju/1.25-upgrade/juju2/storage"
	dummystorage "github.com/juju/1.25-upgrade/juju2/storage/provider/dummy"
)

func StorageProviders() storage.ProviderRegistry {
	return dummystorage.StorageProviders()
}
