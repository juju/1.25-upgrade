// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package azure

import (
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/storage"
)

func ForceVolumeSourceTokenRefresh(vs storage.VolumeSource) error {
	return ForceTokenRefresh(vs.(*azureVolumeSource).env)
}

func ForceTokenRefresh(env environs.Environ) error {
	return env.(*azureEnviron).authorizer.refresh()
}
