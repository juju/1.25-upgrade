// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"
)

var verifySourceDoc = `
The purpose of the verify-source command is to check connectivity, status, and
viability of a 1.25 juju environment for migration into a Juju 2.x controller.

`

func newVerifySourceCommand() cmd.Command {
	return &verifySourceCommand{}
}

type verifySourceCommand struct {
	cmd.CommandBase

	name string
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
	if len(args) == 0 {
		return errors.Errorf("no environment name specified")
	}
	c.name, args = args[0], args[1:]
	return cmd.CheckEmpty(args)
}

func (c *verifySourceCommand) Run(ctx *cmd.Context) error {
	return errors.Errorf("wat")
}
