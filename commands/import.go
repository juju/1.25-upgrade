// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"io"
	"io/ioutil"

	"github.com/juju/cmd"
	"github.com/juju/description"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"github.com/juju/utils/set"
	charmv5 "gopkg.in/juju/charm.v5"
	charmv6 "gopkg.in/juju/charm.v6-unstable"

	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju1/state/storage"
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
	allTools, err := updateToolsInModel(model, tw)
	if err != nil {
		return errors.Trace(err)
	}
	if logger.IsDebugEnabled() {
		err = writeModel(ctx, model)
		if err != nil {
			return errors.Trace(err)
		}
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

	// Sanity check - ask the target controller whether the machines
	// match what it expects.
	checkResults, err := targetAPI.CheckMachines(model.Tag().Id())
	if err != nil {
		return errors.Annotate(err, "sanity checking machines in imported model")
	}
	if len(checkResults) > 0 {
		for _, err := range checkResults {
			logger.Errorf(err.Error())
		}
		return errors.Errorf("machine sanity check failed in imported model")
	}

	for _, seriesArch := range allTools {
		err = tw.uploadTools(model.Tag().Id(), seriesArch)
		if err != nil {
			return errors.Annotatef(err, "uploading tools %q to target controller", seriesArch)
		}
	}

	usedCharms := set.NewStrings()
	for _, app := range model.Applications() {
		usedCharms.Add(app.CharmURL())
	}
	return errors.Trace(transferCharms(st, usedCharms.SortedValues(), targetAPI))
}

func updateToolsInModel(model description.Model, tw *toolsWrangler) ([]string, error) {
	allTools := set.NewStrings()
	for _, machine := range model.Machines() {
		seriesArch := seriesArchFromAgentTools(machine.Tools())
		allTools.Add(seriesArch)
		metadata, err := tw.metadata(seriesArch)
		if err != nil {
			return nil, errors.Trace(err)
		}
		machine.SetTools(agentToolsFromTools(metadata))
	}
	for _, app := range model.Applications() {
		for _, unit := range app.Units() {
			seriesArch := seriesArchFromAgentTools(unit.Tools())
			allTools.Add(seriesArch)
			metadata, err := tw.metadata(seriesArch)
			if err != nil {
				return nil, errors.Trace(err)
			}
			unit.SetTools(agentToolsFromTools(metadata))
		}
	}
	return allTools.Values(), nil
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

func transferCharms(st *state.State, charms []string, targetAPI *migrationtarget.Client) error {
	store := storage.NewStorage(st.EnvironUUID(), st.MongoSession())
	for _, curl := range charms {
		logger.Debugf("uploading charm %q", curl)
		if err := transferCharm(st, store, curl, targetAPI); err != nil {
			return errors.Annotatef(err, "uploading charm %q", curl)
		}
	}
	return nil
}

func transferCharm(st *state.State, store storage.Storage, curlString string, targetAPI *migrationtarget.Client) error {
	curl, err := charmv5.ParseURL(curlString)
	if err != nil {
		return errors.Trace(err)
	}
	ch, err := st.Charm(curl)
	if err != nil {
		return errors.Trace(err)
	}
	reader, _, err := store.Get(ch.StoragePath())
	if err != nil {
		return errors.Trace(err)
	}
	defer reader.Close()

	localFile, err := ioutil.TempFile("", "charm-"+ch.URL().Name)
	if err != nil {
		return errors.Trace(err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, reader)
	if err != nil {
		return errors.Trace(err)
	}
	_, err = localFile.Seek(0, io.SeekStart)
	if err != nil {
		return errors.Trace(err)
	}
	_, err = targetAPI.UploadCharm(st.EnvironUUID(), toV6(curl), localFile)
	return errors.Trace(err)
}

func toV6(v5 *charmv5.URL) *charmv6.URL {
	return &charmv6.URL{
		Schema:   v5.Schema,
		User:     v5.User,
		Name:     v5.Name,
		Revision: v5.Revision,
		Series:   v5.Series,
	}
}
