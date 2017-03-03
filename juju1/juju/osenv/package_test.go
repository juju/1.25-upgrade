// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package osenv_test

import (
	stdtesting "testing"

	"github.com/juju/testing"
	gc "gopkg.in/check.v1"

	coretesting "github.com/juju/1.25-upgrade/juju1/testing"
)

func Test(t *stdtesting.T) {
	gc.TestingT(t)
}

type importSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&importSuite{})

func (*importSuite) TestDependencies(c *gc.C) {
	c.Assert(coretesting.FindJujuCoreImports(c, "github.com/juju/1.25-upgrade/juju1/juju/osenv"),
		gc.HasLen, 0)
}
