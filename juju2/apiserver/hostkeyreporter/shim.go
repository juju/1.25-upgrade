// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package hostkeyreporter

import (
	"github.com/juju/1.25-upgrade/juju2/apiserver/facade"
	"github.com/juju/1.25-upgrade/juju2/state"
	"github.com/juju/errors"
)

// newFacade wraps New to express the supplied *state.State as a Backend.
func newFacade(st *state.State, res facade.Resources, auth facade.Authorizer) (*Facade, error) {
	facade, err := New(st, res, auth)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return facade, nil
}
