// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package utils_test

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/instance"
	"github.com/juju/1.25-upgrade/juju1/provider/common"
)

// fakeZonedEnv wraps an Environ (e.g. dummy) and implements ZonedEnviron.
type fakeZonedEnv struct {
	environs.Environ

	zones     []common.AvailabilityZone
	instZones []string
	err       error

	calls  []string
	idsArg []instance.Id
}

// AvailabilityZones implements ZonedEnviron.
func (e *fakeZonedEnv) AvailabilityZones() ([]common.AvailabilityZone, error) {
	e.calls = append(e.calls, "AvailabilityZones")
	return e.zones, errors.Trace(e.err)
}

// InstanceAvailabilityZoneNames implements ZonedEnviron.
func (e *fakeZonedEnv) InstanceAvailabilityZoneNames(ids []instance.Id) ([]string, error) {
	e.calls = append(e.calls, "InstanceAvailabilityZoneNames")
	e.idsArg = ids
	return e.instZones, errors.Trace(e.err)
}
