// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package status

import (
	stdtesting "testing"

	"github.com/juju/1.25-upgrade/juju1/testing"
)

func TestPackage(t *stdtesting.T) {
	testing.MgoTestPackage(t)
}
