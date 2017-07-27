// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import "github.com/juju/1.25-upgrade/juju2/state"

type stateShim struct {
	*state.State
}

func (s *stateShim) MachineSeries(id string) (string, error) {
	return "xenial", nil
}
