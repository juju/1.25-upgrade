// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package agent_test

import (
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/api/agent"
	apitesting "github.com/juju/1.25-upgrade/juju2/api/testing"
	jujutesting "github.com/juju/1.25-upgrade/juju2/juju/testing"
)

type modelSuite struct {
	jujutesting.JujuConnSuite
	*apitesting.ModelWatcherTests
}

var _ = gc.Suite(&modelSuite{})

func (s *modelSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)

	stateAPI, _ := s.OpenAPIAsNewMachine(c)

	agentAPI := agent.NewState(stateAPI)
	c.Assert(agentAPI, gc.NotNil)

	s.ModelWatcherTests = apitesting.NewModelWatcherTests(
		agentAPI, s.BackingState,
	)
}
