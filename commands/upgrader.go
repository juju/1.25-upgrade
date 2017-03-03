// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"os"

	"github.com/juju/cmd"
	"github.com/juju/version"
)

var upgraderVersion = version.MustParse("0.1.0")

// NewUpgradeCommand returns the supercommand for the various upgrade
// commands.
func NewUpgradeCommand(ctx *cmd.Context) cmd.Command {
	upgrader := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name: "juju 1.25-upgrade",
		Doc:  "some docs",
		Log: &cmd.Log{
			DefaultConfig: os.Getenv("JUJU_LOGGING_CONFIG"),
		},
		Version: upgraderVersion.String(),
	})
	registerCommands(upgrader)
	return upgrader
}

func registerCommands(super *cmd.SuperCommand) {
	super.Register(newVerifySourceCommand())
}
