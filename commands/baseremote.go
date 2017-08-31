// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"encoding/base64"
	"encoding/json"

	"gopkg.in/macaroon.v1"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/mongo"
	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju2/api"
)

const dataDir = "/var/lib/juju"

type baseRemoteCommand struct {
	cmd.CommandBase

	needsController bool

	controllerInfo *api.Info
}

type Info struct {
	Addrs       []string
	SNIHostName string
	CACert      string
	Tag         string
	Password    string
	Macaroons   []macaroon.Slice
}

func (c *baseRemoteCommand) init(args []string) ([]string, error) {
	if c.needsController {
		if len(args) == 0 {
			return args, errors.Errorf("missing controller info")
		}

		bytes, err := base64.StdEncoding.DecodeString(args[0])
		if err != nil {
			return args, errors.Annotate(err, "decoding controller info")
		}
		var info Info
		err = json.Unmarshal(bytes, &info)
		if err != nil {
			return args, errors.Annotate(err, "unmarshalling controller info")
		}
		tag, err := names.ParseTag(info.Tag)
		if err != nil {
			return args, errors.Annotate(err, "parsing tag")
		}
		c.controllerInfo = &api.Info{
			Addrs:       info.Addrs,
			SNIHostName: info.SNIHostName,
			CACert:      info.CACert,
			Tag:         tag,
			Password:    info.Password,
			Macaroons:   info.Macaroons,
		}

		args = args[1:]
	}
	return args, nil
}

func (c *baseRemoteCommand) getControllerConnection() (api.Connection, error) {
	return api.Open(c.controllerInfo, api.DefaultDialOpts())
}

func getState() (*state.State, error) {
	tag, err := getCurrentMachineTag(dataDir)
	if err != nil {
		return nil, errors.Annotate(err, "finding machine tag")
	}

	logger.Infof("current machine tag: %s", tag)

	config, err := getConfig(tag)
	if err != nil {
		return nil, errors.Annotate(err, "loading agent config")
	}

	mongoInfo, available := config.MongoInfo()
	if !available {
		return nil, errors.New("mongo info not available from agent config")
	}
	st, err := state.Open(config.Environment(), mongoInfo, mongo.DefaultDialOpts(), environs.NewStatePolicy())
	if err != nil {
		return nil, errors.Annotate(err, "opening state connection")
	}
	return st, nil
}
