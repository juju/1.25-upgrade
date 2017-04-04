// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/mongo"
	"github.com/juju/1.25-upgrade/juju1/state"
)

const dataDir = "/var/lib/juju"

type baseRemoteCommand struct {
	cmd.CommandBase
}

func (c *baseRemoteCommand) getState(ctx *cmd.Context) (*state.State, error) {
	tag, err := getCurrentMachineTag(dataDir)
	if err != nil {
		return nil, errors.Annotate(err, "finding machine tag")
	}

	ctx.Infof("current machine tag: %s", tag)

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
