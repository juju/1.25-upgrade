// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common_test

import (
	"fmt"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/apiserver/common"
	"github.com/juju/1.25-upgrade/juju2/controller"
	"github.com/juju/1.25-upgrade/juju2/environs/config"
	"github.com/juju/1.25-upgrade/juju2/provider/dummy"
	"github.com/juju/1.25-upgrade/juju2/testing"
)

type controllerConfigSuite struct {
	testing.BaseSuite

	testingEnvConfig *config.Config
}

var _ = gc.Suite(&controllerConfigSuite{})

type fakeControllerAccessor struct {
	controllerConfigError error
}

func (f *fakeControllerAccessor) ControllerConfig() (controller.Config, error) {
	if f.controllerConfigError != nil {
		return nil, f.controllerConfigError
	}
	return map[string]interface{}{
		controller.ControllerUUIDKey: testing.ControllerTag.Id(),
		controller.CACertKey:         testing.CACert,
		controller.APIPort:           4321,
		controller.StatePort:         1234,
	}, nil
}

func (s *controllerConfigSuite) TearDownTest(c *gc.C) {
	dummy.Reset(c)
	s.BaseSuite.TearDownTest(c)
}

func (*controllerConfigSuite) TestControllerConfigSuccess(c *gc.C) {
	cc := common.NewControllerConfig(
		&fakeControllerAccessor{},
	)
	result, err := cc.ControllerConfig()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(map[string]interface{}(result.Config), jc.DeepEquals, map[string]interface{}{
		"ca-cert":         testing.CACert,
		"controller-uuid": "deadbeef-1bad-500d-9000-4b1d0d06f00d",
		"state-port":      1234,
		"api-port":        4321,
	})
}

func (*controllerConfigSuite) TestControllerConfigFetchError(c *gc.C) {
	cc := common.NewControllerConfig(
		&fakeControllerAccessor{
			controllerConfigError: fmt.Errorf("pow"),
		},
	)
	_, err := cc.ControllerConfig()
	c.Assert(err, gc.ErrorMatches, "pow")
}
