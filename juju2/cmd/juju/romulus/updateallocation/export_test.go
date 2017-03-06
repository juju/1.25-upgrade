// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package updateallocation

import (
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
	"github.com/juju/cmd"
)

func NewUpdateAllocateCommandForTest(api apiClient, store jujuclient.ClientStore) cmd.Command {
	c := &updateAllocationCommand{api: api}
	c.SetClientStore(store)
	return modelcmd.Wrap(c)
}
