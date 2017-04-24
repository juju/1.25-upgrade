// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/macaroon-bakery.v1/httpbakery"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/kardianos/osext"

	"github.com/juju/1.25-upgrade/juju1/environs/configstore"
	"github.com/juju/1.25-upgrade/juju2/api"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
)

type baseClientCommand struct {
	cmd.CommandBase

	needsController bool

	info configstore.EnvironInfo

	name    string
	address string
	plugin  string

	controller modelcmd.ControllerCommandBase

	remoteCommand string
	remoteArgs    string
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

	if c.needsController {
		if len(args) == 0 {
			return args, errors.Errorf("no controller name specified")
		}
		c.controller.SetClientStore(jujuclient.NewFileClientStore())
		if err := c.controller.SetControllerName(args[0]); err != nil {
			return args, errors.Trace(err)
		}
		if err := c.setRemoteControllerInfo(); err != nil {
			return args, errors.Trace(err)
		}
		args = args[1:]
	}

	if err := c.loadInfo(); err != nil {
		return args, err
	}

	return args, nil
}

func (c *baseClientCommand) setRemoteControllerInfo() error {
	// Read the controller info and pass it as extra args to the
	// remote command.
	cinfo, err := c.GetControllerAPIInfo()
	if err != nil {
		return errors.Trace(err)
	}

	info := Info{
		Addrs:       cinfo.Addrs,
		SNIHostName: cinfo.SNIHostName,
		CACert:      cinfo.CACert,
		Tag:         cinfo.Tag.String(),
		Password:    cinfo.Password,
		Macaroons:   cinfo.Macaroons,
	}

	// Lets serialize to yaml, then base64 encode it.
	bytes, err := json.Marshal(info)
	if err != nil {
		return errors.Trace(err)
	}
	logger.Debugf("info: %s", bytes)
	c.remoteArgs = base64.StdEncoding.EncodeToString(bytes)
	return nil
}

func (c *baseClientCommand) GetControllerAPIInfo() (*api.Info, error) {
	info, err := c.controller.GetControllerAPIInfo(
		c.controller.ClientStore(),
		c.controller.ControllerName())
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Put the macaroons in.
	apiContext, err := c.controller.APIContext()
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Connect to the target controller, ensuring up-to-date macaroons,
	// and return the macaroons in the cookie jar for the controller.
	//
	// TODO(axw,mjs) add a controller API that returns a macaroon that
	// may be used for the sole purpose of migration.
	api, err := c.controller.NewAPIRoot()
	if err != nil {
		return nil, errors.Annotate(err, "connecting to target controller")
	}
	defer api.Close()
	info.Macaroons = httpbakery.MacaroonsForURL(apiContext.Jar, api.CookieURL())
	return info, nil
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
		fmt.Sprintf("./%s %s %s %s\n", pluginBase, c.remoteCommand, c.remoteArgs, debug),
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
