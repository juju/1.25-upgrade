// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"text/template"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/utils/set"
	"github.com/juju/utils/ssh"
	"github.com/juju/version"
	"golang.org/x/sync/errgroup"

	"github.com/juju/1.25-upgrade/juju2/api"
)

//go:generate go run ../juju2/generate/filetoconst/filetoconst.go agentUpgradeScript agent-upgrade.py agentupgrade_script.go 2017 commands

var upgradeAgentsDoc = `

The purpose of the upgrade-agents command is to upgrade the agents on the 1.25
environment to the version used by the controller.

This command updates the tools symlinks for the agents, and updates their
agent config files to specify the correct version, along with the CA Cert and
addresses of the controller.

`

func newUpgradeAgentsCommand() cmd.Command {
	return wrap(&upgradeAgentsCommand{
		baseClientCommand{
			needsController: true,
			remoteCommand:   "upgrade-agents-impl",
		},
	})
}

type upgradeAgentsCommand struct {
	baseClientCommand
}

func (c *upgradeAgentsCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "upgrade-agents",
		Args:    "<environment name> <controller name>",
		Purpose: "upgrade all the agents for the specified environment",
		Doc:     upgradeAgentsDoc,
	}
}

func (c *upgradeAgentsCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var upgradeAgentsImplDoc = `

upgrade-agents-impl must be executed on an API server machine of a 1.25
environment.

The command will get a list of all the machines, and their addresses, and then
ssh to all the machines to upgrade the various agents on those machines.

`

func newUpgradeAgentsImplCommand() cmd.Command {
	return &upgradeAgentsImplCommand{
		baseRemoteCommand{needsController: true},
	}
}

type upgradeAgentsImplCommand struct {
	baseRemoteCommand
}

func (c *upgradeAgentsImplCommand) Init(args []string) error {
	args, err := c.baseRemoteCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}

	return cmd.CheckEmpty(args)
}

func (c *upgradeAgentsImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "upgrade-agents-impl",
		Purpose: "controller aspect of upgrade-agents",
		Doc:     upgradeAgentsImplDoc,
	}
}

func (c *upgradeAgentsImplCommand) Run(ctx *cmd.Context) error {
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

	// Make a dir to put the downloaded tools into.
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return errors.Trace(err)
	}

	// Save machine addresses so that we don't need to be able to talk
	// to the database to rollback the agent upgrades.
	if err := c.saveMachines(machines); err != nil {
		return errors.Annotate(err, "saving machine addresses")
	}

	conn, err := c.getControllerConnection()
	if err != nil {
		return errors.Annotate(err, "getting controller connection")
	}
	defer conn.Close()

	ver, _ := conn.ServerVersion()
	fmt.Fprintf(ctx.Stdout, "Controller version: %s\n", ver)
	fmt.Fprintf(ctx.Stdout, "Controller addresses: %#v\n", conn.APIHostPorts())
	fmt.Fprintf(ctx.Stdout, "Controller UUID: %s\n", conn.ControllerTag().Id())

	// Emit the upgrade script for pushing to other machines.
	scriptPath, err := c.writeUpgradeScript(&scriptConfig{
		ControllerTag:  conn.ControllerTag().String(),
		ControllerInfo: c.controllerInfo,
		Version:        ver,
	})
	if err != nil {
		return errors.Trace(err)
	}

	// Get a list of all the architectures and series for the machines?
	toolsNeeded := set.NewStrings()
	for _, m := range machines {
		toolsNeeded.Add(seriesArch(m))
	}

	// Get the tools from the controller.
	tw := newToolsWrangler(conn)
	for _, seriesArch := range toolsNeeded.SortedValues() {
		if err := tw.getTools(seriesArch); err != nil {
			return errors.Trace(err)
		}
	}

	err = c.pushTools(ctx, ver, scriptPath, machines)
	if err != nil {
		return errors.Trace(err)
	}

	targets := flatMachineExecTargets(machines...)
	results, err := parallelExec(targets, "apt-get install --yes python3 python3-yaml; python3 ~/1.25-agent-upgrade/agent-upgrade.py")
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(reportResults(ctx, "upgrade", machines, results))
}

func (c *upgradeAgentsImplCommand) saveMachines(machines []FlatMachine) error {
	fileData, err := json.Marshal(machines)
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(writeFile(
		path.Join(toolsDir, "saved-machines.json"),
		0644,
		bytes.NewBuffer(fileData)))
}

func (c *upgradeAgentsImplCommand) pushTools(ctx *cmd.Context, ver version.Number, scriptPath string, machines []FlatMachine) error {
	var group errgroup.Group
	for i := range machines {
		machine := machines[i]
		group.Go(func() error {
			return errors.Annotatef(
				c.pushToolsToMachine(ctx, ver, scriptPath, machine),
				"machine %s", machine.ID)
		})
	}
	logger.Debugf("waiting for copies to finish")
	return group.Wait()
}

func (c *upgradeAgentsImplCommand) pushToolsToMachine(ctx *cmd.Context, ver version.Number, scriptPath string, machine FlatMachine) error {
	logger.Debugf("making target dir for machine %s", machine.ID)
	rc, err := runViaSSH(
		machine.Address,
		"rm -rf 1.25-agent-upgrade; mkdir 1.25-agent-upgrade; chown ubuntu:ubuntu 1.25-agent-upgrade",
		withSystemIdentity())
	if err != nil {
		return errors.Trace(err)
	}
	if rc != 0 {
		return &cmd.RcPassthroughError{Code: rc}
	}
	toolsPath := toolsFilePath(ver, seriesArch(machine))
	options := defaultSSHOptions()
	options.SetIdentities(systemIdentity)
	logger.Debugf("copying upgrade script and %s to machine %s", toolsPath, machine.ID)
	args := []string{toolsPath, scriptPath, fmt.Sprintf("ubuntu@%s:~/1.25-agent-upgrade/", machine.Address)}
	err = ssh.Copy(args, &options)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *upgradeAgentsImplCommand) writeUpgradeScript(config *scriptConfig) (string, error) {
	tmpl, err := template.New("upgrade-script").Parse(agentUpgradeScript)
	if err != nil {
		return "", errors.Trace(err)
	}
	var script bytes.Buffer
	err = tmpl.Execute(&script, config)
	if err != nil {
		return "", errors.Trace(err)
	}
	scriptPath := path.Join(toolsDir, "agent-upgrade.py")
	err = writeFile(scriptPath, 0644, &script)
	if err != nil {
		return "", errors.Trace(err)
	}
	return scriptPath, nil
}
func removeAll(dir string) {
	err := os.RemoveAll(dir)
	if err == nil || os.IsNotExist(err) {
		return
	}
	logger.Errorf("cannot remove %q: %v", dir, err)
}

func seriesArch(machine FlatMachine) string {
	binary := version.MustParseBinary(machine.Tools)
	return fmt.Sprintf("%s-%s", binary.Series, binary.Arch)
}

type scriptConfig struct {
	ControllerInfo *api.Info
	ControllerTag  string
	Version        version.Number
}
