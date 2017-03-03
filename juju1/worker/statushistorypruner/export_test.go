// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package statushistorypruner

import (
	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju1/worker"
)

func NewPruneWorker(st *state.State, params *HistoryPrunerParams, t worker.NewTimerFunc, psh pruneHistoryFunc) worker.Worker {
	w := &pruneWorker{
		st:     st,
		params: params,
		pruner: psh,
	}
	return worker.NewPeriodicWorker(w.doPruning, w.params.PruneInterval, t)
}
