// Copyright 2015 Canonical Ltd.
// Copyright 2015 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package cloudsigma

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/cloudconfig/providerinit/renderers"
	"github.com/juju/1.25-upgrade/juju1/version"
)

type CloudSigmaRenderer struct{}

func (CloudSigmaRenderer) EncodeUserdata(udata []byte, vers version.OSType) ([]byte, error) {
	switch vers {
	case version.Ubuntu, version.CentOS:
		return renderers.ToBase64(udata), nil
	default:
		return nil, errors.Errorf("Cannot encode userdata for OS: %s", vers)
	}
}
