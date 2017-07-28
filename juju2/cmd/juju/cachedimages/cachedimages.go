// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cachedimages

import (
	"github.com/juju/1.25-upgrade/juju2/api/imagemanager"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
)

// CachedImagesCommandBase is a helper base structure that has a method to get the
// image manager client.
type CachedImagesCommandBase struct {
	modelcmd.ModelCommandBase
}

// NewImagesManagerClient returns a imagemanager client for the root api endpoint
// that the environment command returns.
func (c *CachedImagesCommandBase) NewImagesManagerClient() (*imagemanager.Client, error) {
	root, err := c.NewAPIRoot()
	if err != nil {
		return nil, err
	}
	return imagemanager.NewClient(root), nil
}
