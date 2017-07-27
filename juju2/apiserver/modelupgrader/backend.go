// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelupgrader

import (
	"github.com/juju/errors"
	"github.com/juju/1.25-upgrade/juju2/cloud"
	"github.com/juju/1.25-upgrade/juju2/state"
	"gopkg.in/juju/names.v2"
)

type Backend interface {
	Cloud(string) (cloud.Cloud, error)
	GetModel(names.ModelTag) (Model, error)
}

type Model interface {
	Cloud() string
	EnvironVersion() int
	SetEnvironVersion(int) error
}

func NewStateBackend(st *state.State) Backend {
	return stateBackend{st}
}

type stateBackend struct {
	*state.State
}

func (s stateBackend) GetModel(tag names.ModelTag) (Model, error) {
	m, err := s.State.GetModel(tag)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return m, nil
}
