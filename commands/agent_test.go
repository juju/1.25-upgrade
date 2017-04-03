// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"os"
	"path/filepath"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type machineTagSuite struct{}

var _ = gc.Suite(&machineTagSuite{})

func (*machineTagSuite) TestMissingDirs(c *gc.C) {
	datadir := c.MkDir()
	_, err := getCurrentMachineTag(datadir)
	c.Assert(err, gc.ErrorMatches, "no machines found")
}

func (*machineTagSuite) TestMachine(c *gc.C) {
	datadir := c.MkDir()
	err := os.MkdirAll(filepath.Join(datadir, "agents", "machine-42"), 0755)
	c.Assert(err, jc.ErrorIsNil)
	tag, err := getCurrentMachineTag(datadir)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(tag.Id(), gc.Equals, "42")
}
