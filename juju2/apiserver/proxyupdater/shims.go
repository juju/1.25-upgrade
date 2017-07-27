// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package proxyupdater

import (
	"github.com/juju/1.25-upgrade/juju2/environs/config"
	"github.com/juju/1.25-upgrade/juju2/network"
	"github.com/juju/1.25-upgrade/juju2/state"
)

// stateShim forwards and adapts state.State methods to Backend
type stateShim struct {
	st *state.State
}

func (s *stateShim) ModelConfig() (*config.Config, error) {
	return s.st.ModelConfig()
}

func (s *stateShim) APIHostPorts() ([][]network.HostPort, error) {
	return s.st.APIHostPorts()
}

func (s *stateShim) WatchAPIHostPorts() state.NotifyWatcher {
	return s.st.WatchAPIHostPorts()
}

func (s *stateShim) WatchForModelConfigChanges() state.NotifyWatcher {
	return s.st.WatchForModelConfigChanges()
}
