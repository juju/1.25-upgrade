// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package environment_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju1/apiserver/common"
	commontesting "github.com/juju/1.25-upgrade/juju1/apiserver/common/testing"
	"github.com/juju/1.25-upgrade/juju1/apiserver/environment"
	apiservertesting "github.com/juju/1.25-upgrade/juju1/apiserver/testing"
	"github.com/juju/1.25-upgrade/juju1/juju/testing"
	"github.com/juju/1.25-upgrade/juju1/state"
)

type environmentSuite struct {
	testing.JujuConnSuite
	*commontesting.EnvironWatcherTest

	authorizer apiservertesting.FakeAuthorizer
	resources  *common.Resources

	machine0 *state.Machine
	api      *environment.EnvironmentAPI
}

var _ = gc.Suite(&environmentSuite{})

func (s *environmentSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)

	var err error
	s.machine0, err = s.State.AddMachine("quantal", state.JobHostUnits, state.JobManageEnviron)
	c.Assert(err, jc.ErrorIsNil)

	s.authorizer = apiservertesting.FakeAuthorizer{
		Tag: s.machine0.Tag(),
	}
	s.resources = common.NewResources()
	s.AddCleanup(func(_ *gc.C) { s.resources.StopAll() })

	s.api, err = environment.NewEnvironmentAPI(
		s.State,
		s.resources,
		s.authorizer,
	)
	c.Assert(err, jc.ErrorIsNil)
	s.EnvironWatcherTest = commontesting.NewEnvironWatcherTest(
		s.api, s.State, s.resources, commontesting.NoSecrets)
}
