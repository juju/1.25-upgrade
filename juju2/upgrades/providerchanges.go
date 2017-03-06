// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrades

import (
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/provider"
	"github.com/juju/errors"
	"github.com/juju/version"
)

func upgradeProviderChanges(env environs.Environ, reader environConfigReader, ver version.Number) error {
	cfg, err := reader.ModelConfig()
	if err != nil {
		return errors.Annotate(err, "reading model config")
	}

	upgrader, ok := env.(provider.Upgradeable)
	if !ok {
		logger.Debugf("provider %q has no upgrades", cfg.Type())
		return nil
	}

	if err := upgrader.RunUpgradeStepsFor(ver); err != nil {
		return errors.Annotate(err, "running upgrade steps")
	}
	return nil
}
