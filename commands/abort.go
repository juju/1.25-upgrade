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
	// Abort import, roll back agent upgrade and undo provider tag
	// changes. We want to attempt to do all of these steps, even if a
	// preceding one fails.
	modelErr := c.abortImport(ctx)
	if modelErr != nil {
		logger.Errorf("aborting model failed: %s", modelErr.Error())
	}

	rollbackErr := c.rollbackAgents(ctx)
	if rollbackErr != nil {
		logger.Errorf("rolling back agent upgrades failed: %s", rollbackErr.Error())
	}

	// This is a bit funny - if the agent upgrade failed we might not
	// be able to open a state to talk to the environ
	// provider. Although if the rollback failed because it was
	// already done that would be ok.
	tagErr := c.downgradeTags(ctx)
	if tagErr != nil {
		logger.Errorf("downgrading tags failed: %s", tagErr.Error())
	}

	if modelErr != nil || rollbackErr != nil || tagErr != nil {
		return errors.Errorf("at least one error occurred aborting the upgrade")
	}
	return nil
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

func (c *abortImplCommand) rollbackAgents(ctx *cmd.Context) error {
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
	return nil
}

func (c *abortImplCommand) downgradeTags(ctx *cmd.Context) error {
	st, err := getState()
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	err = downgradeTags(st)
	if err != nil {
		return errors.Trace(err)
	}
	fmt.Fprintf(ctx.Stdout, "tags downgraded\n")
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
