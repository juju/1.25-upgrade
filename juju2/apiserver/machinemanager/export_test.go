// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package machinemanager

import (
	"github.com/juju/1.25-upgrade/juju2/apiserver/facade"
	"github.com/juju/1.25-upgrade/juju2/state"
)

type StateInterface stateInterface

type Patcher interface {
	PatchValue(ptr, value interface{})
}

func PatchState(p Patcher, st StateInterface) {
	p.PatchValue(&getState, func(*state.State) stateInterface {
		return st
	})
}

func NewMachineManagerTestingAPI(st stateInterface, authorizer facade.Authorizer) MachineManagerAPI {
	return MachineManagerAPI{
		st:         st,
		authorizer: authorizer,
	}
}

var InstanceTypes = instanceTypes
