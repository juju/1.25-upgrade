// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
)

var rollbackAgentsDoc = `

The rollback-agents command rolls back the actions of a previous
upgrade-agents command on the machines in a Juju 1.25 environment.

It removes the installed Juju 2 tools, sets symlinks back to the
previous version and undoes the changes to agent configurations.

`

func newRollbackAgentsCommand() cmd.Command {
	command := &rollbackAgentsCommand{}
	command.remoteCommand = "rollback-agents-impl"
	return wrap(command)
}

type rollbackAgentsCommand struct {
	baseClientCommand
}

func (c *rollbackAgentsCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "rollback-agents",
		Args:    "<environment name>",
		Purpose: "rollback a previous upgrade-agents in the specified environment",
		Doc:     rollbackAgentsDoc,
	}
}

func (c *rollbackAgentsCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var rollbackAgentsImplDoc = `

rollback-agents-impl must be executed on an API server machine of a 1.25
environment.

The command will roll back the effects of a previous upgrade-agents
command.

`

func newRollbackAgentsImplCommand() cmd.Command {
	return &rollbackAgentsImplCommand{}
}

type rollbackAgentsImplCommand struct {
	baseRemoteCommand
}

func (c *rollbackAgentsImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "rollback-agents-impl",
		Purpose: "controller aspect of rollback-agents",
		Doc:     rollbackAgentsImplDoc,
	}
}

func (c *rollbackAgentsImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	machines, err := getMachines(st)
	if err != nil {
		return errors.Annotate(err, "unable to get addresses for machines")
	}

	results := parallelCall(machines, "python3 ~/1.25-agent-upgrade/agent-upgrade.py rollback")

	var badMachines []string
	for _, res := range results {
		if res.Error != nil {
			logger.Errorf("failed to rollback on machine %s: %s", res.MachineID, res.Error)
			badMachines = append(badMachines, res.MachineID)
		}
	}

	if len(badMachines) > 0 {
		plural := "s"
		if len(badMachines) == 1 {
			plural = ""
		}
		return errors.Errorf("rollback failed on machine%s %s",
			plural, strings.Join(badMachines, ", "))
	}

	return nil
}
