// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package pubsub_test

import (
	"testing"

	coretesting "github.com/juju/1.25-upgrade/juju2/testing"
)

func TestPackage(t *testing.T) {
	coretesting.MgoTestPackage(t)
}
