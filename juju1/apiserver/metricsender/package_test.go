// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package metricsender_test

import (
	stdtesting "testing"

	"github.com/juju/1.25-upgrade/juju1/testing"
)

func TestPackage(t *stdtesting.T) {
	testing.MgoTestPackage(t)
}
