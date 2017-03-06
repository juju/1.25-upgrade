// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"flag"
	"os"
	"runtime"
	stdtesting "testing"

	cmdtesting "github.com/juju/1.25-upgrade/juju2/cmd/testing"
	_ "github.com/juju/1.25-upgrade/juju2/provider/dummy"
	"github.com/juju/1.25-upgrade/juju2/testing"
)

func TestPackage(t *stdtesting.T) {
	if runtime.GOARCH == "386" {
		t.Skipf("skipping package for %v/%v, see http://pad.lv/1425569", runtime.GOOS, runtime.GOARCH)
	}
	testing.MgoTestPackage(t)
}

// Reentrancy point for testing (something as close as possible to) the juju
// tool itself.
func TestRunMain(t *stdtesting.T) {
	if *cmdtesting.FlagRunMain {
		os.Exit(Main(flag.Args()))
	}
}
