// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"

	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
)

// TODO(ericsnow) Replace all this with a better registry mechanism,
// likely over in the cmd repo.

var (
	registeredCommands    []func() cmd.Command
	registeredEnvCommands []func() modelcmd.ModelCommand
)

// RegisterCommand adds the provided func to the set of those that will
// be called when the juju command runs. Each returned command will be
// registered with the "juju" supercommand.
func RegisterCommand(newCommand func() cmd.Command) {
	registeredCommands = append(registeredCommands, newCommand)
}

// RegisterEnvCommand adds the provided func to the set of those that will
// be called when the juju command runs. Each returned command will be
// wrapped in envCmdWrapper, which is what gets registered with the
// "juju" supercommand.
func RegisterEnvCommand(newCommand func() modelcmd.ModelCommand) {
	registeredEnvCommands = append(registeredEnvCommands, newCommand)
}
