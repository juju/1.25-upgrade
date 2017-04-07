// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/kardianos/osext"

	"github.com/juju/1.25-upgrade/juju1/environs/configstore"
)

type baseClientCommand struct {
	cmd.CommandBase

	info configstore.EnvironInfo

	name    string
	address string
	plugin  string

	remoteCommand string
}

// Init will grab the first arg as the environment name.
// Validation of the name is also done here.
func (c *baseClientCommand) init(args []string) ([]string, error) {
	// Make sure we can work out our own location.
	if plugin, err := osext.Executable(); err != nil {
		return args, errors.Annotate(err, "finding plugin location")
	} else {
		c.plugin = plugin
	}

	if len(args) == 0 {
		return args, errors.Errorf("no environment name specified")
	}
	c.name, args = args[0], args[1:]

	if err := c.loadInfo(); err != nil {
		return args, err
	}

	return args, nil
}

func (c *baseClientCommand) loadInfo() error {
	store, err := configstore.Default()
	if err != nil {
		return errors.Annotate(err, "cannot get default config store")
	}

	// Look to open the .jenv file.
	info, err := store.ReadInfo(c.name)
	if err != nil {
		return errors.Annotate(err, "loading environment info")
	}

	if !info.Initialized() {
		return errors.Errorf("environment %q not initialized", c.name)
	}

	c.info = info

	// Grab the first address
	addresses := info.APIEndpoint().Addresses
	address := addresses[0]
	c.address = address[:strings.LastIndex(address, ":")]

	return nil
}

func (c *baseClientCommand) Run(ctx *cmd.Context) error {
	if err := checkUpdatePlugin(ctx, c.plugin, c.address); err != nil {
		return errors.Annotate(err, "checking remote plugin")
	}

	pluginBase := filepath.Base(c.plugin)

	debug := ""
	if logger.IsDebugEnabled() {
		debug = "--debug"
	}

	result, err := runViaSSH(
		c.address,
		fmt.Sprintf("./%s %s %s\n", pluginBase, c.remoteCommand, debug),
		"")

	if err != nil {
		return errors.Annotatef(err, "running %s via SSH", c.remoteCommand)
	}

	fmt.Fprintf(ctx.Stdout, result.Stdout)
	fmt.Fprintf(ctx.Stderr, result.Stderr)

	if result.Code != 0 {
		return &cmd.RcPassthroughError{result.Code}
	}

	return nil
}
