// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"
	"gopkg.in/yaml.v2"
)

var dumpSourceDB = `
Dump the contents of the remote DB.
`

func newDumpSourceDBCommand() cmd.Command {
	command := &dumpSourceDBCommand{}
	command.remoteCommand = "dump-source-db-impl"
	return wrap(command)
}

type dumpSourceDBCommand struct {
	baseClientCommand
}

func (c *dumpSourceDBCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "dump-source-db",
		Args:    "<environment name>",
		Purpose: "dump the DB contents",
		Doc:     dumpSourceDB,
	}
}

func (c *dumpSourceDBCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

func newDumpSourceDBImplCommand() cmd.Command {
	command := &dumpSourceDBImpl{}
	return command
}

type dumpSourceDBImpl struct {
	baseRemoteCommand
}

func (c *dumpSourceDBImpl) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "dump-source-db-impl",
		Purpose: "dump the DB contents",
		Doc:     "",
	}
}

func (c *dumpSourceDBImpl) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	data, err := st.DumpAll()
	if err != nil {
		return errors.Annotate(err, "dumping state collections")
	}

	// Check for LXC containers
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return errors.Annotate(err, "marshalling data")
	}

	_, err = ctx.GetStdout().Write(bytes)
	if err != nil {
		return errors.Annotate(err, "writing yaml data")
	}

	return nil
}
