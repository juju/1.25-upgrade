// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/api/migrationtarget"
)

var importDoc = `

The import command converts the specified Juju 1.25 environment into
the Juju 2.2.3 import format and imports it as a model under the
target controller.

All the agents in the source environment should be stopped before
running the import command.

`

func newImportCommand() cmd.Command {
	return wrap(&importCommand{
		baseClientCommand{
			needsController: true,
			remoteCommand:   "import-impl",
		},
	})
}

type importCommand struct {
	baseClientCommand
}

func (c *importCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "import",
		Args:    "<environment name> <controller name>",
		Purpose: "import the specified environment as a model in the target controller",
		Doc:     importDoc,
	}
}

func (c *importCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var importImplDoc = `

import-impl must be run on an API server machine for a 1.25
environment.

It will convert the environment into the Juju 2.2.3 import format and
import it as a model under the target controller.

`

func newImportImplCommand() cmd.Command {
	return &importImplCommand{
		baseRemoteCommand{needsController: true},
	}
}

type importImplCommand struct {
	baseRemoteCommand
}

func (c *importImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "import-impl",
		Purpose: "controller-side command for the import command",
		Doc:     importImplDoc,
	}
}

func (c *importImplCommand) Init(args []string) error {
	args, err := c.baseRemoteCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}

	return cmd.CheckEmpty(args)
}

func (c *importImplCommand) Run(ctx *cmd.Context) (err error) {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	conn, err := c.getControllerConnection()
	if err != nil {
		return errors.Annotate(err, "getting controller connection")
	}
	defer conn.Close()
	targetAPI := migrationtarget.NewClient(conn)

	logger.Debugf("exporting model from source environmment %s", st.EnvironTag().Id())
	modelBytes, err := exportModel(st)
	if err != nil {
		return errors.Annotate(err, "exporting")
	}

	logger.Debugf("importing model to target controller %s", conn.ControllerTag().Id())
	err = targetAPI.Import(modelBytes)
	// We want to try to clean up the model in the target even if
	// there's an error importing - that can still leave the model
	// around.
	defer func() {
		if err != nil {
			logger.Debugf("cleaning up failed import")
			if cleanupErr := targetAPI.Abort(st.EnvironTag().Id()); cleanupErr != nil {
				logger.Errorf("cleanup failed: %s", cleanupErr)
			}
		}
	}()
	if err != nil {
		return errors.Annotate(err, "importing model on target controller")
	}

	return errors.Errorf("not finished")
}
