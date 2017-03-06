// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package controller

import (
	"github.com/juju/1.25-upgrade/juju2/core/migration"
	"github.com/juju/1.25-upgrade/juju2/state"
)

type patcher interface {
	PatchValue(destination, source interface{})
}

func SetPrecheckResult(p patcher, err error) {
	p.PatchValue(&runMigrationPrechecks, func(*state.State, migration.TargetInfo) error {
		return err
	})
}
