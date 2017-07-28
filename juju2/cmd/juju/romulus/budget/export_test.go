// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package budget

import (
	"github.com/juju/cmd"

	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
)

func NewBudgetCommandForTest(api apiClient, store jujuclient.ClientStore) cmd.Command {
	c := &budgetCommand{api: api}
	c.SetClientStore(store)
	return modelcmd.Wrap(c)
}
