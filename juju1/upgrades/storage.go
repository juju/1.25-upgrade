// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrades

import (
	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju1/storage/poolmanager"
)

func addDefaultStoragePools(st *state.State) error {
	settings := state.NewStateSettings(st)
	return poolmanager.AddDefaultStoragePools(settings)
}
