// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// TODO(wallyworld) - move to instancepoller_test
package instancepoller

import (
	stdtesting "testing"

	coretesting "github.com/juju/1.25-upgrade/juju2/testing"
)

func TestPackage(t *stdtesting.T) {
	coretesting.MgoTestPackage(t)
}
