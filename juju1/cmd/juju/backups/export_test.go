// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"github.com/juju/cmd"

	"github.com/juju/1.25-upgrade/juju1/cmd/envcmd"
	"github.com/juju/1.25-upgrade/juju1/environs"
)

const (
	NotSet          = notset
	DownloadWarning = downloadWarning
)

var (
	NewAPIClient = &newAPIClient
)

func RestoreCommandForTest(environTestFunc func() (environs.Environ, error)) cmd.Command {
	c := &RestoreCommand{environFunc: environTestFunc}
	return envcmd.Wrap(c)
}
