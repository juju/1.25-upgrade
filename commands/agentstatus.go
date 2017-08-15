// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/state"
)

var agentStatusDoc = ` 
The purpose of the agent-status command is to check the status of all the
agents of a 1.25 environment. The agents may be running the 1.25 binary, or a
2.x binary. The command will return the status of the agent, and what tools
they are currently set to use.

`

func newAgentStatusCommand() cmd.Command {
	command := &agentStatusCommand{}
	command.remoteCommand = "agent-status-impl"
	return wrap(command)
}

type agentStatusCommand struct {
	baseClientCommand
}

func (c *agentStatusCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "agent-status",
		Args:    "<environment name>",
		Purpose: "show the status of all the agents for the specified environment",
		Doc:     agentStatusDoc,
	}
}

func (c *agentStatusCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var agentStatusImplDoc = `

agent-status-impl must be executed on an API server machine of a 1.25
environment.

The command will get a list of all the machines, and their addresses, and then
ssh to all the machines to check on the status of the various agents on those
machines.

`

func newAgentStatusImplCommand() cmd.Command {
	return &agentStatusImplCommand{}
}

type agentStatusImplCommand struct {
	baseRemoteCommand
}

func (c *agentStatusImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "agent-status-impl",
		Purpose: "controller aspect of agent-status",
		Doc:     agentStatusImplDoc,
	}
}

func (c *agentStatusImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	// Here we always use the 1.25 environment to get all of the machine
	// addresses. We then use those to ssh into every one of those machine
	// and run the service status script against all the agents.
	machines, err := getMachines(st)
	if err != nil {
		return errors.Annotate(err, "unable to get addresses for machines")
	}

	// The information is then gathered and parsed and formatted here before
	// the data is passed back to the caller.
	return printServiceStatus(ctx, machines)
}

func getMachines(st *state.State) ([]FlatMachine, error) {
	machines, err := st.AllMachines()
	if err != nil {
		return nil, errors.Annotate(err, "getting 1.25 machines")
	}
	result := make([]FlatMachine, len(machines))
	for i, m := range machines {
		fm, err := makeFlatMachine(st, m)
		if err != nil {
			return nil, errors.Trace(err)
		}
		logger.Debugf("%d: %#v", i, fm)
		result[i] = fm
	}
	return result, nil
}

func makeFlatMachine(st *state.State, m *state.Machine) (FlatMachine, error) {
	address, err := getMachineAddress(m)
	if err != nil {
		return FlatMachine{}, errors.Annotatef(err, "address for machine %q", m.Id())
	}
	fm := FlatMachine{
		Model:   st.EnvironUUID(),
		Series:  m.Series(),
		ID:      m.Id(),
		Address: address,
	}
	if instanceId, err := m.InstanceId(); err == nil {
		fm.InstanceID = string(instanceId)
	} else if !errors.IsNotProvisioned(err) {
		return FlatMachine{}, errors.Trace(err)
	}
	if tools, err := m.AgentTools(); err == nil {
		fm.Tools = tools.Version.String()
	} else if !errors.IsNotFound(err) {
		return FlatMachine{}, errors.Trace(err)
	}
	if parentId, ok := m.ParentId(); ok {
		host, err := st.Machine(parentId)
		if err != nil {
			return FlatMachine{}, errors.Trace(err)
		}
		hostAddress, err := getMachineAddress(host)
		if err != nil {
			return FlatMachine{}, errors.Trace(err)
		}
		fm.HostAddress = hostAddress
	}
	return fm, nil
}

func getMachineAddress(m *state.Machine) (string, error) {
	// Start with the private address, which is more likely to be set
	// fallback to the public address, and error out if they are both missing.
	private, err := m.PrivateAddress()
	if err == nil {
		return private.Value, nil
	}
	public, err := m.PublicAddress()
	if err != nil {
		return "", errors.New("no private nor public address")
	}
	return public.Value, nil
}
