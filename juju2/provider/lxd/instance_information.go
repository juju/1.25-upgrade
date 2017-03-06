// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package lxd

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/constraints"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/environs/instances"
)

var _ environs.InstanceTypesFetcher = (*environ)(nil)

// InstanceTypes implements InstanceTypesFetcher
func (env *environ) InstanceTypes(c constraints.Value) (instances.InstanceTypesWithCostMetadata, error) {
	return instances.InstanceTypesWithCostMetadata{}, errors.NotSupportedf("InstanceTypes")
}
