// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package allocate

import (
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
	"github.com/juju/cmd"
)

func NewAllocateCommandForTest(api apiClient, store jujuclient.ClientStore) cmd.Command {
	c := &allocateCommand{api: api}
	c.SetClientStore(store)
	return modelcmd.Wrap(c)
}
