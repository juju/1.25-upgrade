// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"
)

var startAgentsDoc = ` 
The purpose of the start-agents command is to start all the agents of a 1.25
environment. The agents may be running the 1.25 binary, or a 2.x binary.
`

func newStartAgentsCommand() cmd.Command {
	command := &startAgentsCommand{}
	command.remoteCommand = "start-agents-impl"
	return command
}

type startAgentsCommand struct {
	baseClientCommand
}

func (c *startAgentsCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "start-agents",
		Args:    "<environment name>",
		Purpose: "start all the agents for the specified environment",
		Doc:     startAgentsDoc,
	}
}

func (c *startAgentsCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var startAgentsImplDoc = `

start-agents-impl must be executed on an API server machine of a 1.25
environment.

The command will get a list of all the machines, and their addresses, and then
ssh to all the machines to start the various agents on those machines.

`

func newStartAgentsImplCommand() cmd.Command {
	return &startAgentsImplCommand{}
}

type startAgentsImplCommand struct {
	baseRemoteCommand
}

func (c *startAgentsImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "start-agents-impl",
		Purpose: "controller aspect of start-agents",
		Doc:     startAgentsImplDoc,
	}
}

func (c *startAgentsImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	// Here we always use the 1.25 environment to get all of the machine
	// addresses. We then use those to ssh into every one of those machine
	// and run the service status script against all the agents.
	machines, err := getMachines(st)
	if err != nil {
		return errors.Annotate(err, "unable to get addresses for machines")
	}

	serviceCommand(ctx, machines, "start")

	// The information is then gathered and parsed and formatted here before
	// the data is passed back to the caller.
	serviceStatus(ctx, machines)

	return nil
}
