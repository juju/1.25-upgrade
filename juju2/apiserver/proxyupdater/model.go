// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package proxyupdater

import (
	"github.com/juju/1.25-upgrade/juju2/apiserver/facade"
	"github.com/juju/1.25-upgrade/juju2/state"
)

// NewAPI creates a new API server-side facade with a state.State backing.
func NewAPI(st *state.State, res facade.Resources, auth facade.Authorizer) (*ProxyUpdaterAPI, error) {
	return NewAPIWithBacking(&stateShim{st: st}, res, auth)
}
