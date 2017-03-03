// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rsyslog

import (
	"github.com/juju/1.25-upgrade/juju1/agent"
	"github.com/juju/1.25-upgrade/juju1/api/base"
	"github.com/juju/1.25-upgrade/juju1/api/rsyslog"
	"github.com/juju/1.25-upgrade/juju1/feature"
	"github.com/juju/1.25-upgrade/juju1/worker"
	"github.com/juju/1.25-upgrade/juju1/worker/dependency"
	"github.com/juju/1.25-upgrade/juju1/worker/util"
	"github.com/juju/utils/featureflag"
)

// ManifoldConfig defines the names of the manifolds on which a
// Manifold will depend.
type ManifoldConfig util.AgentApiManifoldConfig

// Manifold returns a dependency manifold that runs an rsyslog
// worker, using the resource names defined in the supplied config.
func Manifold(config ManifoldConfig) dependency.Manifold {
	return util.AgentApiManifold(util.AgentApiManifoldConfig(config), newWorker)
}

// newWorker exists to wrap NewRsyslogConfigWorker in a format convenient for an
// AgentApiManifold.
// TODO(fwereade) 2015-05-11 Eventually, the method should be the sole accessible
// package factory function -- as part of the manifold -- and all tests should
// thus be routed through it.
var newWorker = func(a agent.Agent, apiCaller base.APICaller) (worker.Worker, error) {
	if featureflag.Enabled(feature.DisableRsyslog) {
		logger.Warningf("rsyslog manifold disabled by feature flag")
		return nil, dependency.ErrMissing
	}

	agentConfig := a.CurrentConfig()
	tag := agentConfig.Tag()
	namespace := agentConfig.Value(agent.Namespace)
	addrs, err := agentConfig.APIAddresses()
	if err != nil {
		return nil, err
	}
	return NewRsyslogConfigWorker(
		rsyslog.NewState(apiCaller),
		RsyslogModeForwarding,
		tag,
		namespace,
		addrs,
		agentConfig.DataDir(),
	)
}
