// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/1.25-upgrade/juju2/api/keymanager"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
)

type SSHKeysBase struct {
	modelcmd.ModelCommandBase
}

// NewKeyManagerClient returns a keymanager client for the root api endpoint
// that the environment command returns.
func (c *SSHKeysBase) NewKeyManagerClient() (*keymanager.Client, error) {
	root, err := c.NewAPIRoot()
	if err != nil {
		return nil, err
	}
	return keymanager.NewClient(root), nil
}
