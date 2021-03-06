// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"github.com/juju/1.25-upgrade/juju2/state/imagestorage"
)

var (
	imageStorageNewStorage = imagestorage.NewStorage
)

// ImageStorage returns a new imagestorage.Storage
// that stores image metadata.
func (st *State) ImageStorage() imagestorage.Storage {
	return imageStorageNewStorage(st.session, st.ModelUUID())
}
