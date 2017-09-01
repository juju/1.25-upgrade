// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"fmt"

	"github.com/juju/cmd"
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/agent"
	"github.com/juju/1.25-upgrade/juju2/api/migrationtarget"
)

var activateDoc = `

The activate command enables the newly-imported model in the target
controller. It should only be run after upgrading the agents (which
includes a check that each agent can connect to the new model's API).

`

func newActivateCommand() cmd.Command {
	command := &activateCommand{}
	command.remoteCommand = "activate-impl"
	command.needsController = true
	return wrap(command)
}

type activateCommand struct {
	baseClientCommand
}

func (c *activateCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "activate",
		Args:    "<environment name> <controller name>",
		Purpose: "activate the new model in the target controller",
		Doc:     activateDoc,
	}
}

func (c *activateCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var activateImplDoc = `

activate-impl must be executed on an API server machine of a 1.25
environment.

The command will roll back the effects of a previous upgrade-agents
command.

`

func newActivateImplCommand() cmd.Command {
	return &activateImplCommand{
		baseRemoteCommand{needsController: true},
	}
}

type activateImplCommand struct {
	baseRemoteCommand
}

func (c *activateImplCommand) Init(args []string) error {
	args, err := c.baseRemoteCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}

	return cmd.CheckEmpty(args)
}

func (c *activateImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "activate-impl",
		Purpose: "controller aspect of activate",
		Doc:     activateImplDoc,
	}
}

func (c *activateImplCommand) Run(ctx *cmd.Context) error {
	conn, err := c.getControllerConnection()
	if err != nil {
		return errors.Annotate(err, "getting controller connection")
	}
	defer conn.Close()
	targetAPI := migrationtarget.NewClient(conn)

	modelUUID, err := getModelUUID()
	if err != nil {
		return errors.Annotate(err, "getting model UUID")
	}

	err = targetAPI.Activate(modelUUID)
	if err != nil {
		return errors.Annotate(err, "activating new model")
	}
	fmt.Fprintf(ctx.Stdout, "model %q activated\n", modelUUID)
	return nil
}

func getModelUUID() (string, error) {
	tag, err := getCurrentMachineTag(dataDir)
	if err != nil {
		return "", errors.Trace(err)
	}
	// Use the juju2 agent code to read the config, since this should
	// be run after upgrading the agents.
	config, err := agent.ReadConfig(agent.ConfigPath(dataDir, tag))
	if err != nil {
		return "", errors.Trace(err)
	}
	return config.Model().Id(), nil
}
