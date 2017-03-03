// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common_test

import (
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju1/apiserver/common"
	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju1/state/multiwatcher"
)

type machineSuite struct{}

var _ = gc.Suite(&machineSuite{})

func (s *machineSuite) TestMachineJobFromParams(c *gc.C) {
	var tests = []struct {
		name multiwatcher.MachineJob
		want state.MachineJob
		err  string
	}{{
		name: multiwatcher.JobHostUnits,
		want: state.JobHostUnits,
	}, {
		name: multiwatcher.JobManageEnviron,
		want: state.JobManageEnviron,
	}, {
		name: multiwatcher.JobManageNetworking,
		want: state.JobManageNetworking,
	}, {
		name: multiwatcher.JobManageStateDeprecated,
		want: state.JobManageStateDeprecated,
	}, {
		name: "invalid",
		want: -1,
		err:  `invalid machine job "invalid"`,
	}}
	for _, test := range tests {
		got, err := common.MachineJobFromParams(test.name)
		if err != nil {
			c.Check(err, gc.ErrorMatches, test.err)
		}
		c.Check(got, gc.Equals, test.want)
	}
}
