// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package gate

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/worker"
)

func NewFlagWorker(gate Waiter) (worker.Worker, error) {
	worker, err := NewFlag(gate)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return worker, nil
}
