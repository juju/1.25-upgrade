// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package joyent

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/constraints"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/environs/instances"
)

var _ environs.InstanceTypesFetcher = (*joyentEnviron)(nil)

// InstanceTypes implements InstanceTypesFetcher
func (env *joyentEnviron) InstanceTypes(c constraints.Value) (instances.InstanceTypesWithCostMetadata, error) {
	iTypes, err := env.listInstanceTypes()
	if err != nil {
		return instances.InstanceTypesWithCostMetadata{}, errors.Trace(err)
	}
	iTypes, err = instances.MatchingInstanceTypes(iTypes, "", c)
	if err != nil {
		return instances.InstanceTypesWithCostMetadata{}, errors.Trace(err)
	}
	return instances.InstanceTypesWithCostMetadata{InstanceTypes: iTypes}, nil
}
