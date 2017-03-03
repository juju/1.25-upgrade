// Copyright 2015 Canonical Ltd.
// Copyright 2015 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package joyent

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/version"
)

type JoyentRenderer struct{}

func (JoyentRenderer) EncodeUserdata(udata []byte, vers version.OSType) ([]byte, error) {
	switch vers {
	case version.Ubuntu, version.CentOS:
		return udata, nil
	default:
		return nil, errors.Errorf("Cannot encode userdata for OS: %s", vers)
	}
}
