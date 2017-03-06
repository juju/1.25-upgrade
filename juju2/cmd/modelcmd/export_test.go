// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelcmd

import "github.com/juju/1.25-upgrade/juju2/jujuclient"

// NewModelCommandBase returns a new ModelCommandBase with the given client
// store, controller name, and model name.
func NewModelCommandBase(store jujuclient.ClientStore, controller, model string) *ModelCommandBase {
	return &ModelCommandBase{
		store:          store,
		controllerName: controller,
		modelName:      model,
	}
}
