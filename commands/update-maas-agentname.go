// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"fmt"
	"time"

	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/instance"
	"github.com/juju/cmd"
	"github.com/juju/errors"
)

var updateMAASAgentNameDoc = ` 
The purpose of the update-maas-agentname command is to update the agent_name
config for a Juju 1.25 MAAS environment. The agents should be running the 1.25
binary.
`

func newUpdateMAASAgentNameCommand() cmd.Command {
	command := &updateMAASAgentNameCommand{}
	command.remoteCommand = "update-maas-agentname-impl"
	return wrap(command)
}

type updateMAASAgentNameCommand struct {
	baseClientCommand
}

func (c *updateMAASAgentNameCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "update-maas-agentname",
		Args:    "<environment name>",
		Purpose: "updates the MAAS agent name for nodes corresponding to machines in the environment",
		Doc:     updateMAASAgentNameDoc,
	}
}

func (c *updateMAASAgentNameCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var updateMAASAgentNameImplDoc = `

update-maas-agentname-impl must be executed on an API server machine of a 1.25
environment.

The command will print a psql command to be run on the MAAS region controller
in order to update the agent_name field of the nodes for this Juju environment.
The agent_name will be updated to the environment UUID. Once the nodes are all
updated, the maas-agent-name environment config will be updated to match.

`

func newUpdateMAASAgentNameImplCommand() cmd.Command {
	return &updateMAASAgentNameImplCommand{}
}

type updateMAASAgentNameImplCommand struct {
	baseRemoteCommand
}

func (c *updateMAASAgentNameImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "update-maas-agentname-impl",
		Purpose: "controller aspect of update-maas-agentname",
		Doc:     updateMAASAgentNameImplDoc,
	}
}

func (c *updateMAASAgentNameImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	machines, err := getMachines(st)
	if err != nil {
		return errors.Annotate(err, "getting machines from state")
	}
	var instanceIds []instance.Id
	for _, m := range machines {
		if m.HostAddress == "" && m.InstanceID != "" {
			instanceIds = append(instanceIds, instance.Id(m.InstanceID))
		}
	}

	cfg, err := st.EnvironConfig()
	if err != nil {
		return errors.Annotate(err, "getting environ config")
	}
	attrs := cfg.AllAttrs()
	envUUID, _ := cfg.UUID()
	oldAgentName, ok := attrs["maas-agent-name"]
	if !ok {
		return errors.New("maas-agent-name is missing from the environ config")
	}
	if oldAgentName == envUUID {
		ctx.Infof("MAAS agent name already updated, nothing to do.")
		return nil
	}

	// Print out the command for the user to run on the MAAS region controller.
	sqlCommand := fmt.Sprintf(`
UPDATE maasserver_node
SET agent_name='%s' WHERE agent_name='%s'
`,
		envUUID, oldAgentName,
	)
	psqlCommand := fmt.Sprintf(`sudo -u postgres psql maasdb -c "%s"`, sqlCommand)
	printedCommand := false
	lastWaitingMessage := time.Time{}
	printPSQLCommandOnce := func() {
		if printedCommand {
			return
		}
		printedCommand = true
		ctx.Infof("Updating MAAS agent name from %q to %q", oldAgentName, envUUID)
		ctx.Infof(
			"In another shell, execute the following command on the MAAS region controller:\n\n%s",
			psqlCommand,
		)
		ctx.Infof("")
	}

	// Update the config in-memory so we can list the instances. If they
	// aren't found with the new maas-agent-name config, then either they
	// were never in the DB, or the user hasn't run the psql command yet.
	updateAttrs := map[string]interface{}{"maas-agent-name": envUUID}
	cfg, err = cfg.Apply(updateAttrs)
	if err != nil {
		return errors.Annotate(err, "updating environ config")
	}
	env, err := environs.New(cfg)
	if err != nil {
		return errors.Trace(err)
	}
	for {
		_, err := env.Instances(instanceIds)
		if err == nil {
			// All done.
			break
		}
		switch errors.Cause(err) {
		case environs.ErrPartialInstances:
		case environs.ErrNoInstances:
		default:
			return errors.Annotate(err, "listing instances")
		}
		printPSQLCommandOnce()
		if time.Since(lastWaitingMessage) > 30*time.Second {
			ctx.Infof("Waiting for database command to be executed...")
			lastWaitingMessage = time.Now()
		}
		time.Sleep(5 * time.Second)
	}

	// Finally, update the environ config in the database.
	ctx.Infof("Done.")
	return errors.Annotate(
		st.UpdateEnvironConfig(updateAttrs, nil, nil),
		"updating environ config in the database",
	)
}
