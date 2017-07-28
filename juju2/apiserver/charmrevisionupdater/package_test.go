// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrevisionupdater

import (
	stdtesting "testing"

	"github.com/juju/1.25-upgrade/juju2/testing"
)

func TestAll(t *stdtesting.T) {
	testing.MgoTestPackage(t)
}
