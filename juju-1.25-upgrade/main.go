// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"os"

	"github.com/juju/cmd"
	"github.com/juju/loggo"

	"github.com/juju/1.25-upgrade/commands"
	"github.com/juju/1.25-upgrade/juju1/juju/osenv"
)

var logger = loggo.GetLogger("upgrader")

func main() {
	// initialize loggo
	_, err := loggo.ReplaceDefaultWriter(cmd.NewWarningWriter(os.Stderr))
	if err != nil {
		panic(err)
	}

	os.Exit(Run(os.Args))
}

func Run(args []string) int {
	ctx, err := cmd.DefaultContext()
	if err != nil {
		logger.Errorf("%v", err)
		return 2
	}

	// Check JUJU_HOME for 1.25
	osenv.SetJujuHome(osenv.JujuHomeDir())
	// Check JUJU_XDG_DATA_HOME for 2.x

	upgrader := commands.NewUpgradeCommand(ctx)
	return cmd.Main(upgrader, ctx, args[1:])
}
