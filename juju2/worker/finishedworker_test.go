// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package worker_test

import (
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/worker.v1"

	jworker "github.com/juju/1.25-upgrade/juju2/worker"
)

type FinishedSuite struct{}

var _ = gc.Suite(&FinishedSuite{})

func (s *FinishedSuite) TestFinishedWorker(c *gc.C) {
	// Pretty dumb test if interface is implemented
	// and Wait() returns nil.
	var fw worker.Worker = jworker.FinishedWorker{}
	c.Assert(fw.Wait(), gc.IsNil)
}
