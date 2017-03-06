// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiaddressupdater

import (
	"github.com/juju/errors"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/agent"
	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/api/machiner"
	"github.com/juju/1.25-upgrade/juju2/api/uniter"
	"github.com/juju/1.25-upgrade/juju2/cmd/jujud/agent/engine"
	"github.com/juju/1.25-upgrade/juju2/worker"
	"github.com/juju/1.25-upgrade/juju2/worker/dependency"
)

// ManifoldConfig defines the names of the manifolds on which a Manifold will depend.
type ManifoldConfig engine.AgentAPIManifoldConfig

// Manifold returns a dependency manifold that runs an API address updater worker,
// using the resource names defined in the supplied config.
func Manifold(config ManifoldConfig) dependency.Manifold {
	typedConfig := engine.AgentAPIManifoldConfig(config)
	return engine.AgentAPIManifold(typedConfig, newWorker)
}

// newWorker trivially wraps NewAPIAddressUpdater for use in a engine.AgentAPIManifold.
// It's not tested at the moment, because the scaffolding necessary is too
// unwieldy/distracting to introduce at this point.
var newWorker = func(a agent.Agent, apiCaller base.APICaller) (worker.Worker, error) {
	tag := a.CurrentConfig().Tag()

	// TODO(fwereade): use appropriate facade!
	var facade APIAddresser
	switch apiTag := tag.(type) {
	case names.UnitTag:
		facade = uniter.NewState(apiCaller, apiTag)
	case names.MachineTag:
		facade = machiner.NewState(apiCaller)
	default:
		return nil, errors.Errorf("expected a unit or machine tag; got %q", tag)
	}

	setter := agent.APIHostPortsSetter{a}
	w, err := NewAPIAddressUpdater(facade, setter)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return w, nil
}
