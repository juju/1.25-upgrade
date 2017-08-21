// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
)

var rollbackAgentsDoc = `

The rollback-agents command rolls back the actions of a previous
upgrade-agents command on the machines in a Juju 1.25 environment.

It removes the installed Juju 2 tools, sets symlinks back to the
previous version and undoes the changes to agent configurations.

`

func newRollbackAgentsCommand() cmd.Command {
	command := &rollbackAgentsCommand{}
	command.remoteCommand = "rollback-agents-impl"
	return wrap(command)
}

type rollbackAgentsCommand struct {
	baseClientCommand
}

func (c *rollbackAgentsCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "rollback-agents",
		Args:    "<environment name>",
		Purpose: "rollback a previous upgrade-agents in the specified environment",
		Doc:     rollbackAgentsDoc,
	}
}

func (c *rollbackAgentsCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var rollbackAgentsImplDoc = `

rollback-agents-impl must be executed on an API server machine of a 1.25
environment.

The command will roll back the effects of a previous upgrade-agents
command.

`

func newRollbackAgentsImplCommand() cmd.Command {
	return &rollbackAgentsImplCommand{}
}

type rollbackAgentsImplCommand struct {
	baseRemoteCommand
}

func (c *rollbackAgentsImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "rollback-agents-impl",
		Purpose: "controller aspect of rollback-agents",
		Doc:     rollbackAgentsImplDoc,
	}
}

func (c *rollbackAgentsImplCommand) Run(ctx *cmd.Context) error {
	machines, err := c.loadMachines()
	if err != nil {
		return errors.Annotate(err, "unable to get addresses for machines")
	}
	targets := flatMachineExecTargets(machines...)
	results, err := parallelExec(targets, "python3 ~/1.25-agent-upgrade/agent-upgrade.py rollback")
	if err != nil {
		return errors.Trace(err)
	}

	var badMachines []string
	for i, res := range results {
		if res.Code != 0 {
			logger.Errorf("failed to rollback on machine %s: exited with %d", machines[i].ID, res.Code)
			badMachines = append(badMachines, machines[i].ID)
		}
	}

	if len(badMachines) > 0 {
		plural := "s"
		if len(badMachines) == 1 {
			plural = ""
		}
		return errors.Errorf("rollback failed on machine%s %s",
			plural, strings.Join(badMachines, ", "))
	}

	return nil
}

func (c *rollbackAgentsImplCommand) loadMachines() ([]FlatMachine, error) {
	data, err := ioutil.ReadFile(path.Join(toolsDir, "saved-machines.json"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	var machines []FlatMachine
	err = json.Unmarshal(data, &machines)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return machines, nil
}
