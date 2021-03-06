// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package application

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/api/application"
	"github.com/juju/1.25-upgrade/juju2/cmd/juju/block"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
)

// NewRemoveUnitCommand returns a command which removes an application's units.
func NewRemoveUnitCommand() modelcmd.ModelCommand {
	return modelcmd.Wrap(&removeUnitCommand{})
}

// removeUnitCommand is responsible for destroying application units.
type removeUnitCommand struct {
	modelcmd.ModelCommandBase
	UnitNames []string
}

const removeUnitDoc = `
Remove application units from the model.

Units of a application are numbered in sequence upon creation. For example, the
fourth unit of wordpress will be designated "wordpress/3". These identifiers
can be supplied in a space delimited list to remove unwanted units from the
model.

Juju will also remove the machine if the removed unit was the only unit left
on that machine (including units in containers).

Removing all units of a application is not equivalent to removing the
application itself; for that, the ` + "`juju remove-application`" + ` command
is used.

Examples:

    juju remove-unit wordpress/2 wordpress/3 wordpress/4

See also:
    remove-application
`

func (c *removeUnitCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "remove-unit",
		Args:    "<unit> [...]",
		Purpose: "Remove application units from the model.",
		Doc:     removeUnitDoc,
	}
}

func (c *removeUnitCommand) Init(args []string) error {
	c.UnitNames = args
	if len(c.UnitNames) == 0 {
		return errors.Errorf("no units specified")
	}
	for _, name := range c.UnitNames {
		if !names.IsValidUnit(name) {
			return errors.Errorf("invalid unit name %q", name)
		}
	}
	return nil
}

func (c *removeUnitCommand) getAPI() (removeApplicationAPI, int, error) {
	root, err := c.NewAPIRoot()
	if err != nil {
		return nil, -1, errors.Trace(err)
	}
	version := root.BestFacadeVersion("Application")
	return application.NewClient(root), version, nil
}

// Run connects to the environment specified on the command line and destroys
// units therein.
func (c *removeUnitCommand) Run(ctx *cmd.Context) error {
	client, apiVersion, err := c.getAPI()
	if err != nil {
		return err
	}
	defer client.Close()

	if apiVersion < 4 {
		return c.removeUnitsDeprecated(ctx, client)
	}
	return c.removeUnits(ctx, client)
}

// TODO(axw) 2017-03-16 #1673323
// Drop this in Juju 3.0.
func (c *removeUnitCommand) removeUnitsDeprecated(ctx *cmd.Context, client removeApplicationAPI) error {
	err := client.DestroyUnitsDeprecated(c.UnitNames...)
	return block.ProcessBlockedError(err, block.BlockRemove)
}

func (c *removeUnitCommand) removeUnits(ctx *cmd.Context, client removeApplicationAPI) error {
	results, err := client.DestroyUnits(c.UnitNames...)
	if err != nil {
		return block.ProcessBlockedError(err, block.BlockRemove)
	}
	anyFailed := false
	for i, name := range c.UnitNames {
		result := results[i]
		if result.Error != nil {
			anyFailed = true
			ctx.Infof("removing unit %s failed: %s", name, result.Error)
			continue
		}
		ctx.Infof("removing unit %s", name)
		for _, entity := range result.Info.DestroyedStorage {
			storageTag, err := names.ParseStorageTag(entity.Tag)
			if err != nil {
				logger.Warningf("%s", err)
				continue
			}
			ctx.Infof("- will remove %s", names.ReadableString(storageTag))
		}
		for _, entity := range result.Info.DetachedStorage {
			storageTag, err := names.ParseStorageTag(entity.Tag)
			if err != nil {
				logger.Warningf("%s", err)
				continue
			}
			ctx.Infof("- will detach %s", names.ReadableString(storageTag))
		}
	}
	if anyFailed {
		return cmd.ErrSilent
	}
	return nil
}
