// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common_test

import (
	"fmt"

	"github.com/juju/cmd/cmdtesting"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/apiserver/common"
	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	apiservertesting "github.com/juju/1.25-upgrade/juju2/apiserver/testing"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
	"github.com/juju/1.25-upgrade/juju2/environs/bootstrap"
	"github.com/juju/1.25-upgrade/juju2/environs/config"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
	"github.com/juju/1.25-upgrade/juju2/provider/dummy"
	"github.com/juju/1.25-upgrade/juju2/state"
	"github.com/juju/1.25-upgrade/juju2/testing"
)

type environWatcherSuite struct {
	testing.BaseSuite

	testingEnvConfig *config.Config
}

var _ = gc.Suite(&environWatcherSuite{})

type fakeModelAccessor struct {
	modelConfig      *config.Config
	modelConfigError error
}

func (*fakeModelAccessor) WatchForModelConfigChanges() state.NotifyWatcher {
	return apiservertesting.NewFakeNotifyWatcher()
}

func (f *fakeModelAccessor) ModelConfig() (*config.Config, error) {
	if f.modelConfigError != nil {
		return nil, f.modelConfigError
	}
	return f.modelConfig, nil
}

func (s *environWatcherSuite) TearDownTest(c *gc.C) {
	dummy.Reset(c)
	s.BaseSuite.TearDownTest(c)
}

func (s *environWatcherSuite) TestWatchSuccess(c *gc.C) {
	resources := common.NewResources()
	s.AddCleanup(func(_ *gc.C) { resources.StopAll() })
	e := common.NewModelWatcher(
		&fakeModelAccessor{},
		resources,
		nil,
	)
	result, err := e.WatchForModelConfigChanges()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result, gc.DeepEquals, params.NotifyWatchResult{"1", nil})
	c.Assert(resources.Count(), gc.Equals, 1)
}

func (*environWatcherSuite) TestModelConfigSuccess(c *gc.C) {
	authorizer := apiservertesting.FakeAuthorizer{
		Tag:        names.NewMachineTag("0"),
		Controller: true,
	}
	testingEnvConfig := testingEnvConfig(c)
	e := common.NewModelWatcher(
		&fakeModelAccessor{modelConfig: testingEnvConfig},
		nil,
		authorizer,
	)
	result, err := e.ModelConfig()
	c.Assert(err, jc.ErrorIsNil)
	// Make sure we can read the secret attribute (i.e. it's not masked).
	c.Check(result.Config["secret"], gc.Equals, "pork")
	c.Check(map[string]interface{}(result.Config), jc.DeepEquals, testingEnvConfig.AllAttrs())
}

func (*environWatcherSuite) TestModelConfigFetchError(c *gc.C) {
	authorizer := apiservertesting.FakeAuthorizer{
		Tag:        names.NewMachineTag("0"),
		Controller: true,
	}
	e := common.NewModelWatcher(
		&fakeModelAccessor{
			modelConfigError: fmt.Errorf("pow"),
		},
		nil,
		authorizer,
	)
	_, err := e.ModelConfig()
	c.Assert(err, gc.ErrorMatches, "pow")
}

func testingEnvConfig(c *gc.C) *config.Config {
	env, err := bootstrap.Prepare(
		modelcmd.BootstrapContext(cmdtesting.Context(c)),
		jujuclient.NewMemStore(),
		bootstrap.PrepareParams{
			ControllerConfig: testing.FakeControllerConfig(),
			ControllerName:   "dummycontroller",
			ModelConfig:      dummy.SampleConfig(),
			Cloud:            dummy.SampleCloudSpec(),
			AdminSecret:      "admin-secret",
		},
	)
	c.Assert(err, jc.ErrorIsNil)
	return env.Config()
}
