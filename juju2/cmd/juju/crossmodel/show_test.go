// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package crossmodel_test

import (
	"github.com/juju/cmd"
	"github.com/juju/cmd/cmdtesting"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"

	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/cmd/juju/crossmodel"
)

type showSuite struct {
	BaseCrossModelSuite
	mockAPI *mockShowAPI
}

var _ = gc.Suite(&showSuite{})

func (s *showSuite) SetUpTest(c *gc.C) {
	s.BaseCrossModelSuite.SetUpTest(c)

	s.mockAPI = &mockShowAPI{
		desc: "IBM DB2 Express Server Edition is an entry level database system",
	}
}

func (s *showSuite) runShow(c *gc.C, args ...string) (*cmd.Context, error) {
	return cmdtesting.RunCommand(c, crossmodel.NewShowEndpointsCommandForTest(s.store, s.mockAPI), args...)
}

func (s *showSuite) TestShowNoUrl(c *gc.C) {
	s.assertShowError(c, nil, ".*must specify endpoint URL.*")
}

func (s *showSuite) TestShowDifferentController(c *gc.C) {
	s.assertShowError(c, []string{"different:user/model.offer"}, `showing endpoints from another controller "different" not supported`)
}

func (s *showSuite) TestShowApiError(c *gc.C) {
	s.mockAPI.msg = "fail"
	s.assertShowError(c, []string{"fred/model.db2"}, ".*fail.*")
}

func (s *showSuite) TestShowURLError(c *gc.C) {
	s.assertShowError(c, []string{"fred/model.foo/db2"}, "application offer URL has invalid form.*")
}

func (s *showSuite) TestShowYaml(c *gc.C) {
	s.assertShow(
		c,
		[]string{"fred/model.db2", "--format", "yaml"},
		`
fred/model.db2:
  access: consume
  endpoints:
    db2:
      interface: http
      role: requirer
    log:
      interface: http
      role: provider
  description: IBM DB2 Express Server Edition is an entry level database system
`[1:],
	)
}

func (s *showSuite) TestShowTabular(c *gc.C) {
	s.assertShow(
		c,
		[]string{"fred/model.db2", "--format", "tabular"},
		`
URL             Access   Description                                 Endpoint  Interface  Role
fred/model.db2  consume  IBM DB2 Express Server Edition is an entry  db2       http       requirer
                         level database system                       log       http       provider

`[1:],
	)
}

func (s *showSuite) TestShowTabularExactly180Desc(c *gc.C) {
	s.mockAPI.desc = s.mockAPI.desc + s.mockAPI.desc + s.mockAPI.desc[:52]
	s.assertShow(
		c,
		[]string{"fred/model.db2", "--format", "tabular"},
		`
URL             Access   Description                                   Endpoint  Interface  Role
fred/model.db2  consume  IBM DB2 Express Server Edition is an entry    db2       http       requirer
                         level database systemIBM DB2 Express Server   log       http       provider
                         Edition is an entry level database systemIBM                       
                         DB2 Express Server Edition is an entry level                       
                         dat                                                                

`[1:],
	)
}

func (s *showSuite) TestShowTabularMoreThan180Desc(c *gc.C) {
	s.mockAPI.desc = s.mockAPI.desc + s.mockAPI.desc + s.mockAPI.desc
	s.assertShow(
		c,
		[]string{"fred/model.db2", "--format", "tabular"},
		`
URL             Access   Description                                   Endpoint  Interface  Role
fred/model.db2  consume  IBM DB2 Express Server Edition is an entry    db2       http       requirer
                         level database systemIBM DB2 Express Server   log       http       provider
                         Edition is an entry level database systemIBM                       
                         DB2 Express Server Edition is an entry level                       
                         ...                                                                

`[1:],
	)
}

func (s *showSuite) assertShow(c *gc.C, args []string, expected string) {
	context, err := s.runShow(c, args...)
	c.Assert(err, jc.ErrorIsNil)

	obtained := cmdtesting.Stdout(context)
	c.Assert(obtained, gc.Matches, expected)
}

func (s *showSuite) assertShowError(c *gc.C, args []string, expected string) {
	_, err := s.runShow(c, args...)
	c.Assert(err, gc.ErrorMatches, expected)
}

type mockShowAPI struct {
	msg, desc string
}

func (s mockShowAPI) Close() error {
	return nil
}

func (s mockShowAPI) ApplicationOffer(url string) (params.ApplicationOffer, error) {
	if s.msg != "" {
		return params.ApplicationOffer{}, errors.New(s.msg)
	}

	return params.ApplicationOffer{
		OfferName:              "hosted-db2",
		OfferURL:               "fred/model.db2",
		ApplicationDescription: s.desc,
		Endpoints: []params.RemoteEndpoint{
			{Name: "log", Interface: "http", Role: charm.RoleProvider},
			{Name: "db2", Interface: "http", Role: charm.RoleRequirer},
		},
		Access: "consume",
	}, nil
}
