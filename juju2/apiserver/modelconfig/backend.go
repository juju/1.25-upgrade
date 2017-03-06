// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelconfig

import (
	"github.com/juju/1.25-upgrade/juju2/apiserver/common"
	"github.com/juju/1.25-upgrade/juju2/environs/config"
	"github.com/juju/1.25-upgrade/juju2/state"
	names "gopkg.in/juju/names.v2"
)

// Backend contains the state.State methods used in this package,
// allowing stubs to be created for testing.
type Backend interface {
	common.BlockGetter
	ControllerTag() names.ControllerTag
	ModelTag() names.ModelTag
	ModelConfigValues() (config.ConfigValues, error)
	UpdateModelConfig(map[string]interface{}, []string, state.ValidateConfigFunc) error
}

type stateShim struct {
	*state.State
}

// NewStateBackend creates a backend for the facade to use.
func NewStateBackend(st *state.State) Backend {
	return stateShim{st}
}
