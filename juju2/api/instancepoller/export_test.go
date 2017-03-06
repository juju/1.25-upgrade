// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package instancepoller

import (
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
)

func NewMachine(caller base.APICaller, tag names.MachineTag, life params.Life) *Machine {
	facade := base.NewFacadeCaller(caller, instancePollerFacade)
	return &Machine{facade, tag, life}
}

var NewStringsWatcher = &newStringsWatcher
