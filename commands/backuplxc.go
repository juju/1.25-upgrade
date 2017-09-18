// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"github.com/juju/utils"
	"golang.org/x/sync/errgroup"

	"github.com/juju/1.25-upgrade/juju1/state"
)

var backupLXCDoc = ` 
The purpose of the backup-lxc command is to create a backup of the
LXC containers in a 1.25 environment.

If --match is specified, it is treated as a regular expression for
matching container names. Only containers whose names match will
be backed up.

If --dry-run is specified, then no backups will be created, nor
will the containers be stopped.
`

func newBackupLXCCommand() cmd.Command {
	command := &backupLXCCommand{}
	command.remoteCommand = "backup-lxc-impl"
	return wrap(command)
}

type backupLXCCommand struct {
	baseClientCommand
	backupDir string
	dryRun    bool
	match     string
}

func (c *backupLXCCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "backup-lxc",
		Args:    "<environment name> <backup dir>",
		Purpose: "create a backup of LXC containers for the specified environment",
		Doc:     backupLXCDoc,
	}
}

func (c *backupLXCCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	if len(args) == 0 {
		return errors.New("no backup directory specified")
	}
	c.backupDir, args = args[0], args[1:]
	return cmd.CheckEmpty(args)
}

func (c *backupLXCCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseClientCommand.SetFlags(f)
	f.BoolVar(&c.dryRun, "dry-run", false, "perform a dry run, without making any changes")
	f.StringVar(&c.match, "match", "", "regular expression for matching LXC container IDs to back up")
}

func (c *backupLXCCommand) Run(ctx *cmd.Context) error {
	if _, err := os.Stat(c.backupDir); err != nil {
		return errors.Annotate(err, "checking backup dir")
	}

	match := func(string) bool { return true }
	if c.match != "" {
		matchRE, err := regexp.Compile(c.match)
		if err != nil {
			return errors.Annotate(err, "parsing --match")
		}
		match = matchRE.MatchString
	}

	if err := c.prepareRemote(ctx); err != nil {
		return errors.Trace(err)
	}

	// Get a listing of all of the LXC containers in the environment.
	lxcContainers, err := getLXCContainerList(&c.baseClientCommand)
	if err != nil {
		return errors.Annotate(err, "getting LXC container list")
	}

	doBackup := func(containerName, outpath string) error {
		temp := outpath + ".tmp"
		f, err := os.Create(temp)
		if err != nil {
			return errors.Annotate(err, "creating output file")
		}
		rc, err := runViaSSH(
			c.address,
			c.getRemoteCommand(c.remoteCommand, containerName),
			withStdout(f),
		)
		f.Close()
		if err != nil {
			return errors.Annotatef(err, "running %s via SSH", c.remoteCommand)
		}
		if rc != 0 {
			return errors.Errorf("creating LXC backup exited %d", rc)
		}
		return utils.ReplaceFile(temp, outpath)
	}

	// Create a backup of each container matching --match,
	// or all machines if --match isn't specified.
	var group errgroup.Group
	for _, container := range lxcContainers {
		containerName := container.Id
		if !match(containerName) {
			ctx.Infof("Skipping non-matching container %q", containerName)
			continue
		}
		outpath := filepath.Join(c.backupDir, container.InstanceId+".tar.xz")
		ctx.Infof("Backing up container %q to %s", containerName, outpath)
		if c.dryRun {
			continue
		}
		group.Go(func() error {
			return errors.Annotatef(
				doBackup(containerName, outpath),
				"backing up %q to %s",
				containerName, outpath,
			)
		})
	}
	return group.Wait()
}

func getLXCContainerList(c *baseClientCommand) ([]lxcContainer, error) {
	// Get a listing of all of the LXC containers in the environment.
	var buf bytes.Buffer
	rc, err := runViaSSH(
		c.address,
		c.getRemoteCommand(c.remoteCommand),
		withStdout(&buf),
	)
	if err != nil {
		return nil, errors.Annotatef(err, "running %s via SSH", c.remoteCommand)
	}
	if rc != 0 {
		return nil, &cmd.RcPassthroughError{rc}
	}
	var lxcContainers lxcContainerList
	if err := json.Unmarshal(buf.Bytes(), &lxcContainers); err != nil {
		return nil, errors.Trace(err)
	}
	return lxcContainers.Containers, nil
}

type lxcContainerList struct {
	Containers []lxcContainer `json:"containers"`
}

type lxcContainer struct {
	Id         string
	InstanceId string
}

var backupLXCImplDoc = `

backup-lxc-impl must be executed on an API server machine of a 1.25
environment.

The command will get a list of all the LXC containers when run without
arguments. When run with the name of a container, the command will
SSH to the container's host, stop the container, send an archive
of the container over stdout, and then start the container again.

`

func newBackupLXCImplCommand() cmd.Command {
	return &backupLXCImplCommand{}
}

type backupLXCImplCommand struct {
	baseRemoteCommand
	containerName string
}

func (c *backupLXCImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "backup-lxc-impl",
		Args:    "[container]",
		Purpose: "controller aspect of backup-lxc",
		Doc:     backupLXCImplDoc,
	}
}

func (c *backupLXCImplCommand) Init(args []string) error {
	if len(args) > 0 {
		c.containerName, args = args[0], args[1:]
	}
	return cmd.CheckEmpty(args)
}

func (c *backupLXCImplCommand) Run(ctx *cmd.Context) error {
	st, err := getState()
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	if c.containerName == "" {
		// Output a listing of LXC containers.
		return listLXCContainers(ctx, st)
	}

	containerMachine, err := st.Machine(c.containerName)
	if err != nil {
		return errors.Annotate(err, "getting container machine")
	}
	parentId, _ := containerMachine.ParentId()
	hostMachine, err := st.Machine(parentId)
	if err != nil {
		return errors.Annotate(err, "getting host machine")
	}

	logger.Debugf("stopping LXC container %q", c.containerName)
	if err := StopLXCContainer(containerMachine, hostMachine); err != nil {
		return errors.Annotate(err, "stopping LXC container")
	}
	logger.Debugf("creating backup of LXC container %q", c.containerName)
	if err := BackupLXCContainer(containerMachine, hostMachine, ctx.GetStdout()); err != nil {
		return errors.Annotate(err, "backing up LXC container")
	}
	logger.Debugf("restarting LXC container %q", c.containerName)
	if err := StartLXCContainer(containerMachine, hostMachine); err != nil {
		return errors.Annotate(err, "starting LXC container")
	}
	return nil
}

func listLXCContainers(ctx *cmd.Context, st *state.State) error {
	// Output a listing of LXC containers.
	var lxcContainers lxcContainerList
	machines, err := st.AllMachines()
	if err != nil {
		return errors.Annotate(err, "getting machines")
	}
	for _, m := range machines {
		if m.ContainerType() != "lxc" {
			continue
		}
		instanceId, err := m.InstanceId()
		if err != nil {
			return errors.Annotate(err, "getting container instance ID")
		}
		lxcContainers.Containers = append(lxcContainers.Containers, lxcContainer{
			Id:         m.Id(),
			InstanceId: string(instanceId),
		})
	}
	return json.NewEncoder(ctx.GetStdout()).Encode(&lxcContainers)
}
