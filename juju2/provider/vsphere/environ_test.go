// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build !gccgo

package vsphere_test

import (
	"os"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/environs"
	envtesting "github.com/juju/1.25-upgrade/juju2/environs/testing"
	"github.com/juju/1.25-upgrade/juju2/feature"
	"github.com/juju/1.25-upgrade/juju2/juju/osenv"
	"github.com/juju/1.25-upgrade/juju2/provider/vsphere"
	"github.com/juju/1.25-upgrade/juju2/testing"
)

type environSuite struct {
	vsphere.BaseSuite
}

var _ = gc.Suite(&environSuite{})

func (s *environSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
}

func (s *environSuite) TestBootstrap(c *gc.C) {
	s.PatchValue(&vsphere.Bootstrap, func(ctx environs.BootstrapContext, env environs.Environ, args environs.BootstrapParams,
	) (*environs.BootstrapResult, error) {
		return nil, errors.New("Bootstrap called")
	})

	os.Setenv(osenv.JujuFeatureFlagEnvKey, feature.VSphereProvider)
	_, err := s.Env.Bootstrap(nil, environs.BootstrapParams{ControllerConfig: testing.FakeControllerConfig()})
	c.Assert(err, gc.ErrorMatches, "Bootstrap called")
}

func (s *environSuite) TestDestroy(c *gc.C) {
	s.PatchValue(&vsphere.DestroyEnv, func(env environs.Environ) error {
		return errors.New("Destroy called")
	})
	err := s.Env.Destroy()
	c.Assert(err, gc.ErrorMatches, "Destroy called")
}

func (s *environSuite) TestPrepareForBootstrap(c *gc.C) {
	err := s.Env.PrepareForBootstrap(envtesting.BootstrapContext(c))
	c.Check(err, jc.ErrorIsNil)
}

func (s *environSuite) TestSupportsNetworking(c *gc.C) {
	_, ok := environs.SupportsNetworking(s.Env)
	c.Assert(ok, jc.IsFalse)
}
