// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/state"
)

// TagUpgrader will be implemented by providers to enable us to update
// the specific tags they store. Making upgrade and downgrade methods
// on the same interface so we can be sure that we only upgrade tags
// when we know we can downgrade them again.
type TagUpgrader interface {
	// UpgradeTags replaces juju-env-uuid tags with juju-model-uuid on
	// any resources that need updating.
	UpgradeTags(controllerUUID string) error
	// Downgrade tags converts juju-model-uuid tags back to
	// juju-env-uuid, and removes any juju-controller-uuid tags.
	DowngradeTags() error
}

func getTagUpgrader(st *state.State) (TagUpgrader, error) {
	e, err := st.Environment()
	if err != nil {
		return nil, errors.Trace(err)
	}
	config, err := e.Config()
	if err != nil {
		return nil, errors.Trace(err)
	}
	env, err := environs.New(config)
	if err != nil {
		return nil, errors.Trace(err)
	}
	upgrader, ok := env.(TagUpgrader)
	if !ok {
		return nil, errors.Errorf("%q (type=%s) environ doesn't support upgrading tags", e.Name(), config.Type())
	}
	return upgrader, nil
}

func upgradeTags(st *state.State, controllerUUID string) error {
	upgrader, err := getTagUpgrader(st)
	if err != nil {
		return errors.Trace(err)
	}
	return upgrader.UpgradeTags(controllerUUID)
}

func downgradeTags(st *state.State) error {
	upgrader, err := getTagUpgrader(st)
	if err != nil {
		return errors.Trace(err)
	}
	return upgrader.DowngradeTags()
}
