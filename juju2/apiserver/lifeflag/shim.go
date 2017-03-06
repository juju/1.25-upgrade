// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package lifeflag

import (
	"github.com/juju/1.25-upgrade/juju2/apiserver/common"
	"github.com/juju/1.25-upgrade/juju2/apiserver/facade"
	"github.com/juju/1.25-upgrade/juju2/state"
)

func init() {
	common.RegisterStandardFacade(
		"LifeFlag", 1,
		func(st *state.State, resources facade.Resources, authorizer facade.Authorizer) (*Facade, error) {
			return NewFacade(st, resources, authorizer)
		},
	)
}
