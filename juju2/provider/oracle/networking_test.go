// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oracle_test

import (
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/provider/oracle"
	oracletesting "github.com/juju/1.25-upgrade/juju2/provider/oracle/testing"
	"github.com/juju/1.25-upgrade/juju2/testing"
)

type networkingSuite struct{}

var _ = gc.Suite(&networkingSuite{})

func (n networkingSuite) TestDeleteMachineVnicSet(c *gc.C) {
	environ, err := oracle.NewOracleEnviron(
		&oracle.EnvironProvider{},
		environs.OpenParams{
			Config: testing.ModelConfig(c),
		},
		oracletesting.DefaultEnvironAPI,
		&advancingClock,
	)

	c.Assert(err, gc.IsNil)
	c.Assert(environ, gc.NotNil)

	err = environ.DeleteMachineVnicSet("0")
	c.Assert(err, gc.IsNil)
}
