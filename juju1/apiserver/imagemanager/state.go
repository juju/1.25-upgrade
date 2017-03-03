// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package imagemanager

import (
	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju1/state/imagestorage"
)

type stateInterface interface {
	ImageStorage() imagestorage.Storage
}

type stateShim struct {
	*state.State
}

func (s stateShim) ImageStorage() imagestorage.Storage {
	return s.State.ImageStorage()
}
