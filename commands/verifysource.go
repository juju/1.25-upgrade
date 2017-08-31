// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"strings"

	_ "github.com/juju/1.25-upgrade/juju2/provider/maas"
	"github.com/juju/cmd"
	"github.com/juju/description"
	"github.com/juju/errors"
	"golang.org/x/sync/errgroup"
)

var verifySourceDoc = `
The purpose of the verify-source command is to check connectivity, status, and
viability of a 1.25 juju environment for migration into a Juju 2.x controller.

`

func newVerifySourceCommand() cmd.Command {
	command := &verifySourceCommand{}
	command.remoteCommand = "verify-source-impl"
	return wrap(command)
}

type verifySourceCommand struct {
	baseClientCommand
}

func (c *verifySourceCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "verify-source",
		Args:    "<environment name>",
		Purpose: "check a 1.25 environment for migration suitability",
		Doc:     verifySourceDoc,
	}
}

func (c *verifySourceCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var verifySourceImplDoc = `

verify-source-impl must be executed on an API server machine of a 1.25
environment.

The command will check the export of the environment into the 2.0 model
format.

`

func newVerifySourceImplCommand() cmd.Command {
	return &verifySourceImplCommand{}
}

type verifySourceImplCommand struct {
	baseRemoteCommand
}

func (c *verifySourceImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "verify-source-impl",
		Purpose: "check the database export for migration suitability",
		Doc:     verifySourceImplDoc,
	}
}

func (c *verifySourceImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	// Check that the LXC containers can be migrated to LXD.
	opts := MigrateLXCOptions{DryRun: true}
	byHost, err := getLXCContainersFromState(st)
	if err != nil {
		return errors.Trace(err)
	}
	var group errgroup.Group
	for host, containers := range byHost {
		containerNames := make([]string, len(containers))
		for i, container := range containers {
			containerNames[i] = container.Id()
		}
		logger.Debugf("dry-running LXC migration for %s", strings.Join(containerNames, ", "))
		host, containers := host, containers // copy for closure
		group.Go(func() error {
			err := MigrateLXC(containers, host, opts)
			return errors.Annotatef(err, "dry-running LXC migration for host %q", host.Id())
		})
	}
	if err := group.Wait(); err != nil {
		return errors.Annotate(err, "dry-running LXC migration")
	}

	model, err := exportModel(st)
	if err != nil {
		return errors.Annotate(err, "exporting model")
	}
	return errors.Annotate(writeModel(ctx, model), "writing model")
}

func writeModel(ctx *cmd.Context, model description.Model) error {
	bytes, err := description.Serialize(model)
	if err != nil {
		return errors.Annotate(err, "serializing model representation")
	}

	_, err = ctx.GetStdout().Write(bytes)
	if err != nil {
		return errors.Annotate(err, "writing model representation")
	}

	return nil
}
