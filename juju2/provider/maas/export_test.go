// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package maas

import (
	"github.com/juju/gomaasapi"

	"github.com/juju/1.25-upgrade/juju2/cloudconfig/cloudinit"
	"github.com/juju/1.25-upgrade/juju2/environs"
)

var (
	ShortAttempt = &shortAttempt
)

func GetMAASClient(env environs.Environ) *gomaasapi.MAASObject {
	return env.(*maasEnviron).getMAASClient()
}

func NewCloudinitConfig(env environs.Environ, hostname, series string) (cloudinit.CloudConfig, error) {
	return env.(*maasEnviron).newCloudinitConfig(hostname, series)
}
