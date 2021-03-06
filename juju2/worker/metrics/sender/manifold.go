// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package sender

import (
	"time"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"gopkg.in/juju/names.v2"
	"gopkg.in/juju/worker.v1"

	"github.com/juju/1.25-upgrade/juju2/agent"
	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/api/metricsadder"
	jworker "github.com/juju/1.25-upgrade/juju2/worker"
	"github.com/juju/1.25-upgrade/juju2/worker/dependency"
	"github.com/juju/1.25-upgrade/juju2/worker/metrics/spool"
	"github.com/juju/1.25-upgrade/juju2/worker/uniter"
)

var (
	logger               = loggo.GetLogger("juju.worker.metrics.sender")
	newMetricAdderClient = func(apiCaller base.APICaller) metricsadder.MetricsAdderClient {
		return metricsadder.NewClient(apiCaller)
	}
	period = time.Minute * 5
)

// ManifoldConfig defines configuration of a metric sender manifold.
type ManifoldConfig struct {
	AgentName       string
	APICallerName   string
	MetricSpoolName string
}

// Manifold creates a metric sender manifold.
func Manifold(config ManifoldConfig) dependency.Manifold {
	return dependency.Manifold{
		Inputs: []string{
			config.AgentName,
			config.APICallerName,
			config.MetricSpoolName,
		},
		Start: func(context dependency.Context) (worker.Worker, error) {
			var apicaller base.APICaller
			var factory spool.MetricFactory
			err := context.Get(config.APICallerName, &apicaller)
			if err != nil {
				return nil, errors.Trace(err)
			}
			err = context.Get(config.MetricSpoolName, &factory)
			if err != nil {
				return nil, errors.Trace(err)
			}
			var agent agent.Agent
			if err := context.Get(config.AgentName, &agent); err != nil {
				return nil, err
			}
			agentConfig := agent.CurrentConfig()
			tag := agentConfig.Tag()
			unitTag, ok := tag.(names.UnitTag)
			if !ok {
				return nil, errors.Errorf("expected a unit tag, got %v", tag)
			}
			paths := uniter.NewWorkerPaths(agentConfig.DataDir(), unitTag, "metrics-send")

			client := newMetricAdderClient(apicaller)

			s, err := newSender(client, factory, paths.State.BaseDir, unitTag.String())
			if err != nil {
				return nil, errors.Trace(err)
			}
			return spool.NewPeriodicWorker(s.Do, period, jworker.NewTimer, s.stop), nil
		},
	}
}
