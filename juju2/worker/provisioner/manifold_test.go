// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provisioner_test

import (
	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/agent"
	"github.com/juju/1.25-upgrade/juju2/api/base"
	apitesting "github.com/juju/1.25-upgrade/juju2/api/base/testing"
	apiprovisioner "github.com/juju/1.25-upgrade/juju2/api/provisioner"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/worker/dependency"
	dt "github.com/juju/1.25-upgrade/juju2/worker/dependency/testing"
	"github.com/juju/1.25-upgrade/juju2/worker/provisioner"
)

type ManifoldSuite struct {
	testing.IsolationSuite
	stub testing.Stub
}

var _ = gc.Suite(&ManifoldSuite{})

func (s *ManifoldSuite) makeManifold() dependency.Manifold {
	fakeNewProvFunc := func(
		apiSt *apiprovisioner.State,
		agentConf agent.Config,
		environ environs.Environ,
	) (provisioner.Provisioner, error) {
		s.stub.AddCall("NewProvisionerFunc")
		return struct{ provisioner.Provisioner }{}, nil
	}
	return provisioner.Manifold(provisioner.ManifoldConfig{
		AgentName:          "agent",
		APICallerName:      "api-caller",
		EnvironName:        "environ",
		NewProvisionerFunc: fakeNewProvFunc,
	})
}

func (s *ManifoldSuite) TestManifold(c *gc.C) {
	manifold := s.makeManifold()
	c.Check(manifold.Inputs, jc.SameContents, []string{"agent", "api-caller", "environ"})
	c.Check(manifold.Output, gc.IsNil)
	c.Check(manifold.Start, gc.NotNil)
}

func (s *ManifoldSuite) TestMissingAgent(c *gc.C) {
	manifold := s.makeManifold()
	w, err := manifold.Start(dt.StubContext(nil, map[string]interface{}{
		"agent":      dependency.ErrMissing,
		"api-caller": struct{ base.APICaller }{},
		"environ":    struct{ environs.Environ }{},
	}))
	c.Check(w, gc.IsNil)
	c.Check(errors.Cause(err), gc.Equals, dependency.ErrMissing)
}

func (s *ManifoldSuite) TestMissingAPICaller(c *gc.C) {
	manifold := s.makeManifold()
	w, err := manifold.Start(dt.StubContext(nil, map[string]interface{}{
		"agent":      struct{ agent.Agent }{},
		"api-caller": dependency.ErrMissing,
		"environ":    struct{ environs.Environ }{},
	}))
	c.Check(w, gc.IsNil)
	c.Check(errors.Cause(err), gc.Equals, dependency.ErrMissing)
}

func (s *ManifoldSuite) TestMissingEnviron(c *gc.C) {
	manifold := s.makeManifold()
	w, err := manifold.Start(dt.StubContext(nil, map[string]interface{}{
		"agent":      struct{ agent.Agent }{},
		"api-caller": struct{ base.APICaller }{},
		"environ":    dependency.ErrMissing,
	}))
	c.Check(w, gc.IsNil)
	c.Check(errors.Cause(err), gc.Equals, dependency.ErrMissing)
}

func (s *ManifoldSuite) TestStarts(c *gc.C) {
	manifold := s.makeManifold()
	w, err := manifold.Start(dt.StubContext(nil, map[string]interface{}{
		"agent":      new(fakeAgent),
		"api-caller": apitesting.APICallerFunc(nil),
		"environ":    struct{ environs.Environ }{},
	}))
	c.Check(w, gc.NotNil)
	c.Check(err, jc.ErrorIsNil)
	s.stub.CheckCallNames(c, "NewProvisionerFunc")
}

type fakeAgent struct {
	agent.Agent
}

func (a *fakeAgent) CurrentConfig() agent.Config {
	return nil
}
