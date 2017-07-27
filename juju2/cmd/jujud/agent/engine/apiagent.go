// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package engine

import (
	worker "gopkg.in/juju/worker.v1"

	"github.com/juju/1.25-upgrade/juju2/agent"
	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/worker/dependency"
)

// Many manifolds completely depend on an agent and an API connection; this
// type configures them.
type AgentAPIManifoldConfig struct {
	AgentName     string
	APICallerName string
}

// AgentAPIStartFunc encapsulates the behaviour that varies among AgentAPIManifolds.
type AgentAPIStartFunc func(agent.Agent, base.APICaller) (worker.Worker, error)

// AgentAPIManifold returns a dependency.Manifold that calls the supplied start
// func with the API and agent resources defined in the config (once those
// resources are present).
func AgentAPIManifold(config AgentAPIManifoldConfig, start AgentAPIStartFunc) dependency.Manifold {
	return dependency.Manifold{
		Inputs: []string{
			config.AgentName,
			config.APICallerName,
		},
		Start: func(context dependency.Context) (worker.Worker, error) {
			var agent agent.Agent
			if err := context.Get(config.AgentName, &agent); err != nil {
				return nil, err
			}
			var apiCaller base.APICaller
			if err := context.Get(config.APICallerName, &apiCaller); err != nil {
				return nil, err
			}
			return start(agent, apiCaller)
		},
	}
}
