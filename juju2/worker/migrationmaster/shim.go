// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package migrationmaster

import (
	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/api/migrationmaster"
	"github.com/juju/1.25-upgrade/juju2/api/watcher"
	"github.com/juju/1.25-upgrade/juju2/worker"
	"github.com/juju/errors"
)

func NewFacade(apiCaller base.APICaller) (Facade, error) {
	facade := migrationmaster.NewClient(apiCaller, watcher.NewNotifyWatcher)
	return facade, nil
}

func NewWorker(config Config) (worker.Worker, error) {
	worker, err := New(config)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return worker, nil
}
