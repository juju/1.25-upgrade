// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelupgrader

import (
	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/api/modelupgrader"
)

func NewFacade(apiCaller base.APICaller) (Facade, error) {
	facade := modelupgrader.NewClient(apiCaller)
	return facade, nil
}
