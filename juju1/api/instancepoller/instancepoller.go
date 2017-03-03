// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package instancepoller

import (
	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/1.25-upgrade/juju1/api/base"
	"github.com/juju/1.25-upgrade/juju1/api/common"
	"github.com/juju/1.25-upgrade/juju1/api/watcher"
	"github.com/juju/1.25-upgrade/juju1/apiserver/params"
)

const instancePollerFacade = "InstancePoller"

// API provides access to the InstancePoller API facade.
type API struct {
	*common.EnvironWatcher

	facade base.FacadeCaller
}

// NewAPI creates a new client-side InstancePoller facade.
func NewAPI(caller base.APICaller) *API {
	if caller == nil {
		panic("caller is nil")
	}
	facadeCaller := base.NewFacadeCaller(caller, instancePollerFacade)
	return &API{
		EnvironWatcher: common.NewEnvironWatcher(facadeCaller),
		facade:         facadeCaller,
	}
}

// Machine provides access to methods of a state.Machine through the
// facade.
func (api *API) Machine(tag names.MachineTag) (*Machine, error) {
	life, err := common.Life(api.facade, tag)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Machine{api.facade, tag, life}, nil
}

var newStringsWatcher = watcher.NewStringsWatcher

// WatchEnvironMachines return a StringsWatcher reporting waiting for the
// environment configuration to change.
func (api *API) WatchEnvironMachines() (watcher.StringsWatcher, error) {
	var result params.StringsWatchResult
	err := api.facade.FacadeCall("WatchEnvironMachines", nil, &result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return newStringsWatcher(api.facade.RawAPICaller(), result), nil
}
