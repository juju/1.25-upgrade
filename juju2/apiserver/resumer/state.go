// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package resumer

import (
	"github.com/juju/1.25-upgrade/juju2/state"
)

type stateInterface interface {
	ResumeTransactions() error
}

type stateShim struct {
	*state.State
}

var getState = func(st *state.State) stateInterface {
	return stateShim{st}
}
