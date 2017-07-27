// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrades

import "github.com/juju/1.25-upgrade/juju2/environs"

var (
	UpgradeOperations      = &upgradeOperations
	StateUpgradeOperations = &stateUpgradeOperations
)

type ModelConfigUpdater environConfigUpdater
type ModelConfigReader environConfigReader

func UpgradeModelConfig(
	reader ModelConfigReader,
	updater ModelConfigUpdater,
	registry environs.ProviderRegistry,
) error {
	return upgradeModelConfig(reader, updater, registry)
}
