// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package instancepoller

import (
	"github.com/juju/names"

	"github.com/juju/1.25-upgrade/juju1/api/base"
	"github.com/juju/1.25-upgrade/juju1/apiserver/params"
)

func NewMachine(caller base.APICaller, tag names.MachineTag, life params.Life) *Machine {
	facade := base.NewFacadeCaller(caller, instancePollerFacade)
	return &Machine{facade, tag, life}
}

var NewStringsWatcher = &newStringsWatcher
