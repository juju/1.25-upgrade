// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"github.com/juju/1.25-upgrade/juju1/service/systemd"
	"github.com/juju/1.25-upgrade/juju1/service/upstart"
	"github.com/juju/1.25-upgrade/juju1/service/windows"
)

var _ Service = (*upstart.Service)(nil)
var _ Service = (*windows.Service)(nil)
var _ Service = (*systemd.Service)(nil)
