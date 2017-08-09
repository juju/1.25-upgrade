// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"
)

var stopAgentsDoc = ` 
The purpose of the stop-agents command is to stop all the agents of a 1.25
environment. The agents may be running the 1.25 binary, or a 2.x binary.
`

func newStopAgentsCommand() cmd.Command {
	command := &stopAgentsCommand{}
	command.remoteCommand = "stop-agents-impl"
	return wrap(command)
}

type stopAgentsCommand struct {
	baseClientCommand
}

func (c *stopAgentsCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "stop-agents",
		Args:    "<environment name>",
		Purpose: "stop all the agents for the specified environment",
		Doc:     stopAgentsDoc,
	}
}

func (c *stopAgentsCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var stopAgentsImplDoc = `

stop-agents-impl must be executed on an API server machine of a 1.25
environment.

The command will get a list of all the machines, and their addresses, and then
ssh to all the machines to stop the various agents on those machines.

`

func newStopAgentsImplCommand() cmd.Command {
	return &stopAgentsImplCommand{}
}

type stopAgentsImplCommand struct {
	baseRemoteCommand
}

func (c *stopAgentsImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "stop-agents-impl",
		Purpose: "controller aspect of stop-agents",
		Doc:     stopAgentsImplDoc,
	}
}

func (c *stopAgentsImplCommand) Run(ctx *cmd.Context) error {
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

	if _, err := agentServiceCommand(ctx, machines, "stop"); err != nil {
		return errors.Annotate(err, "stopping agents")
	}

	// The information is then gathered and parsed and formatted here before
	// the data is passed back to the caller.
	return printServiceStatus(ctx, machines)
}
