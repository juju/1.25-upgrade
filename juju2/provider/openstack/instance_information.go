// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package openstack

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/constraints"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/environs/instances"
)

var _ environs.InstanceTypesFetcher = (*Environ)(nil)

func (e *Environ) InstanceTypes(c constraints.Value) (instances.InstanceTypesWithCostMetadata, error) {
	result := instances.InstanceTypesWithCostMetadata{}
	return result, errors.NotSupportedf("InstanceTypes")
}
