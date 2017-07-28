// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package crossmodel

import (
	"github.com/juju/1.25-upgrade/juju2/api/applicationoffers"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
)

// ApplicationOffersCommandBase is a base for various cross model commands.
type ApplicationOffersCommandBase struct {
	modelcmd.ControllerCommandBase
}

// NewApplicationOffersAPI returns an application offers api for the root api endpoint
// that the command returns.
func (c *ApplicationOffersCommandBase) NewApplicationOffersAPI() (*applicationoffers.Client, error) {
	root, err := c.NewAPIRoot()
	if err != nil {
		return nil, err
	}
	return applicationoffers.NewClient(root), nil
}
