// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package hostkeyreporter

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/api/base"
	apihostkeyreporter "github.com/juju/1.25-upgrade/juju2/api/hostkeyreporter"
	"github.com/juju/1.25-upgrade/juju2/worker"
)

func NewFacade(apiCaller base.APICaller) (Facade, error) {
	return apihostkeyreporter.NewFacade(apiCaller), nil
}

func NewWorker(config Config) (worker.Worker, error) {
	worker, err := New(config)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return worker, nil
}
