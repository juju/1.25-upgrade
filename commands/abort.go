// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"fmt"

	"github.com/juju/cmd"
	"github.com/juju/errors"

	agent1 "github.com/juju/1.25-upgrade/juju1/agent"
	agent2 "github.com/juju/1.25-upgrade/juju2/agent"
	"github.com/juju/1.25-upgrade/juju2/api/migrationtarget"
)

var abortDoc = `

The abort command undoes the actions of previous import and
upgrade-agents commands.

It removes any imported model on the target controller as long as the
model hasn't yet been activated. It also rolls back the agent upgrade
on machines in the environment: removing Juju 2 tools, setting
symlinks back to the previous tools and reverting changes to agent
configurations.

`

func newAbortCommand() cmd.Command {
	command := &abortCommand{}
	command.remoteCommand = "abort-impl"
	command.needsController = true
	return wrap(command)
}

type abortCommand struct {
	baseClientCommand
}

func (c *abortCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "abort",
		Args:    "<environment name> <controller name>",
		Purpose: "undoes the import and upgrade-agents commands",
		Doc:     abortDoc,
	}
}

func (c *abortCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var abortImplDoc = `

abort-impl must be executed on an API server machine of a 1.25
environment.

The command will roll back the effects of previous import and
upgrade-agents commands.

`

func newAbortImplCommand() cmd.Command {
	return &abortImplCommand{
		baseRemoteCommand{needsController: true},
	}
}

type abortImplCommand struct {
	baseRemoteCommand
}

func (c *abortImplCommand) Init(args []string) error {
	args, err := c.baseRemoteCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}

	return cmd.CheckEmpty(args)
}

func (c *abortImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "abort-impl",
		Purpose: "controller aspect of abort",
		Doc:     abortImplDoc,
	}
}

func (c *abortImplCommand) Run(ctx *cmd.Context) error {
	modelErr := c.abortImport(ctx)
	if modelErr != nil {
		// We still want to rollback the agent upgrades, so just
		// report this and continue.
		logger.Errorf("failed to abort model: %s", modelErr.Error())
	}

	machines, err := loadMachines()
	if err != nil {
		return errors.Annotate(err, "unable to get addresses for machines")
	}
	targets := flatMachineExecTargets(machines...)
	results, err := parallelExec(targets, "python3 ~/1.25-agent-upgrade/agent-upgrade.py rollback")
	if err != nil {
		return errors.Trace(err)
	}
	if err := reportResults(ctx, "rollback", machines, results); err != nil {
		return errors.Trace(err)
	}

	// If there were no problems rolling back agents but aborting the
	// model failed, reflect that in the return code.
	return errors.Trace(modelErr)
}

func (c *abortImplCommand) abortImport(ctx *cmd.Context) error {
	conn, err := c.getControllerConnection()
	if err != nil {
		return errors.Annotate(err, "getting controller connection")
	}
	defer conn.Close()
	targetAPI := migrationtarget.NewClient(conn)

	modelUUID, err := getModelUUIDEitherVersion()
	if err != nil {
		return errors.Annotate(err, "getting model UUID")
	}

	err = targetAPI.Abort(modelUUID)
	if err != nil {
		return errors.Annotate(err, "aborting new model")
	}
	fmt.Fprintf(ctx.Stdout, "model %q aborted\n", modelUUID)
	return nil
}

func getModelUUIDEitherVersion() (string, error) {
	tag, err := getCurrentMachineTag(dataDir)
	if err != nil {
		return "", errors.Trace(err)
	}
	configPath := agent2.ConfigPath(dataDir, tag)
	// Try both formats of config - this command might be run before
	// or after upgrading the agent config.
	uuid, err := getModelUUIDVersion2(configPath)
	if err == nil {
		return uuid, nil
	}
	return getModelUUIDVersion1(configPath)
}

func getModelUUIDVersion2(path string) (string, error) {
	config, err := agent2.ReadConfig(path)
	if err != nil {
		return "", errors.Trace(err)
	}
	return config.Model().Id(), nil
}

func getModelUUIDVersion1(path string) (string, error) {
	config, err := agent1.ReadConfig(path)
	if err != nil {
		return "", errors.Trace(err)
	}
	return config.Environment().Id(), nil
}
