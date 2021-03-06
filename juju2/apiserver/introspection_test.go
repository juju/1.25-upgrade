// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"io/ioutil"
	"net/http"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/permission"
	"github.com/juju/1.25-upgrade/juju2/state"
)

type introspectionSuite struct {
	authHTTPSuite
	bob *state.User
}

var _ = gc.Suite(&introspectionSuite{})

func (s *introspectionSuite) SetUpTest(c *gc.C) {
	s.authHTTPSuite.SetUpTest(c)
	bob, err := s.BackingState.AddUser("bob", "", "hunter2", "admin")
	c.Assert(err, jc.ErrorIsNil)
	s.bob = bob
}

func (s *introspectionSuite) url(c *gc.C) string {
	url := s.baseURL(c)
	url.Path = "/introspection/navel"
	return url.String()
}

func (s *introspectionSuite) TestAccess(c *gc.C) {
	s.testAccess(c, "user-admin", "dummy-secret")

	_, err := s.BackingState.AddModelUser(
		s.BackingState.ModelTag().Id(),
		state.UserAccessSpec{
			User:      names.NewUserTag("bob"),
			CreatedBy: names.NewUserTag("admin"),
			Access:    permission.ReadAccess,
		},
	)
	c.Assert(err, jc.ErrorIsNil)
	s.testAccess(c, "user-bob", "hunter2")
}

func (s *introspectionSuite) testAccess(c *gc.C, tag, password string) {
	resp := s.sendRequest(c, httpRequestParams{
		method:   "GET",
		url:      s.url(c),
		tag:      tag,
		password: password,
	})
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	content, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(string(content), gc.Equals, "gazing")
}

func (s *introspectionSuite) TestAccessDenied(c *gc.C) {
	resp := s.sendRequest(c, httpRequestParams{
		method:   "GET",
		url:      s.url(c),
		tag:      "user-bob",
		password: "hunter2",
	})
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusForbidden)
}
