// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrades

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju1/state/utils"
)

var addAZToInstData = state.AddAvailabilityZoneToInstanceData

func addAvaililityZoneToInstanceData(context Context) error {
	err := addAZToInstData(context.State(), utils.AvailabilityZone)
	return errors.Trace(err)
}
