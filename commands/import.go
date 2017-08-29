// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/description"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"

	"github.com/juju/1.25-upgrade/juju2/api/migrationtarget"
	coretools "github.com/juju/1.25-upgrade/juju2/tools"
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
		baseClientCommand: baseClientCommand{
			needsController: true,
			remoteCommand:   "import-impl",
		},
	})
}

type importCommand struct {
	baseClientCommand

	keepBroken bool
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

func (c *importCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseClientCommand.SetFlags(f)
	f.BoolVar(&c.keepBroken, "keep-broken", false, "Keep a failed import")
}

func (c *importCommand) Run(ctx *cmd.Context) error {
	if c.keepBroken {
		c.extraOptions = append(c.extraOptions, "--keep-broken")
	}
	return c.baseClientCommand.Run(ctx)
}

var importImplDoc = `

import-impl must be run on an API server machine for a 1.25
environment.

It will convert the environment into the Juju 2.2.3 import format and
import it as a model under the target controller.

`

func newImportImplCommand() cmd.Command {
	return &importImplCommand{
		baseRemoteCommand: baseRemoteCommand{needsController: true},
	}
}

type importImplCommand struct {
	baseRemoteCommand

	keepBroken bool
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

func (c *importImplCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseRemoteCommand.SetFlags(f)
	f.BoolVar(&c.keepBroken, "keep-broken", false, "Keep a failed import")
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
	model, err := exportModel(st)
	if err != nil {
		return errors.Annotate(err, "exporting")
	}

	// We need to update the tools in the exported model to match the
	// ones we'll put on the agents.
	tw := newToolsWrangler(conn)
	err = updateToolsInModel(model, tw)
	if err != nil {
		return errors.Trace(err)
	}
	err = writeModel(ctx, model)
	if err != nil {
		return errors.Trace(err)
	}

	bytes, err := description.Serialize(model)
	if err != nil {
		return errors.Annotate(err, "serializing model representation")
	}
	logger.Debugf("importing model to target controller %s", conn.ControllerTag().Id())
	err = targetAPI.Import(bytes)
	// We want to try to clean up the model in the target even if
	// there's an error importing - that can still leave the model
	// around.
	defer func() {
		if err != nil && !c.keepBroken {
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

func updateToolsInModel(model description.Model, tw *toolsWrangler) error {
	for _, machine := range model.Machines() {
		metadata, err := tw.metadata(seriesArchFromAgentTools(machine.Tools()))
		if err != nil {
			return errors.Trace(err)
		}
		machine.SetTools(agentToolsFromTools(metadata))
	}
	for _, app := range model.Applications() {
		for _, unit := range app.Units() {
			metadata, err := tw.metadata(seriesArchFromAgentTools(unit.Tools()))
			if err != nil {
				return errors.Trace(err)
			}
			unit.SetTools(agentToolsFromTools(metadata))
		}
	}
	return nil
}

func seriesArchFromAgentTools(t description.AgentTools) string {
	return t.Version().Series + "-" + t.Version().Arch
}

func agentToolsFromTools(t *coretools.Tools) description.AgentToolsArgs {
	return description.AgentToolsArgs{
		Version: t.Version,
		URL:     t.URL,
		Size:    t.Size,
		SHA256:  t.SHA256,
	}
}
