// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelworkermanager

import (
	"github.com/juju/errors"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/state"
)

type BackendShim struct {
	*state.State
}

func (s BackendShim) GetModel(tag names.ModelTag) (BackendModel, error) {
	m, err := s.State.GetModel(tag)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return m, nil
}
