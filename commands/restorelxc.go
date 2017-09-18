// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"golang.org/x/sync/errgroup"
)

var restoreLXCDoc = ` 
The purpose of the restore-lxc command is to restore a backup of
the LXC containers in a 1.25 environment.

If --match is specified, it is treated as a regular expression for
matching container names. Only containers whose names match will
be restored.

If --dry-run is specified, then no changes will take place.
`

func newRestoreLXCCommand() cmd.Command {
	command := &restoreLXCCommand{}
	command.remoteCommand = "restore-lxc-impl"
	return wrap(command)
}

type restoreLXCCommand struct {
	baseClientCommand
	backupDir string
	dryRun    bool
	match     string
}

func (c *restoreLXCCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "restore-lxc",
		Args:    "<environment name> <backup dir>",
		Purpose: "restore LXC containers for the specified environment",
		Doc:     restoreLXCDoc,
	}
}

func (c *restoreLXCCommand) Init(args []string) error {
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

func (c *restoreLXCCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseClientCommand.SetFlags(f)
	f.BoolVar(&c.dryRun, "dry-run", false, "perform a dry run, without making any changes")
	f.StringVar(&c.match, "match", "", "regular expression for matching LXC container IDs to restore")
}

func (c *restoreLXCCommand) Run(ctx *cmd.Context) error {
	if _, err := os.Stat(c.backupDir); err != nil {
		return errors.Annotate(err, "checking restore dir")
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

	doRestore := func(containerName, path string) error {
		f, err := os.Open(path)
		if err != nil {
			return errors.Trace(err)
		}
		rc, err := runViaSSH(
			c.address,
			c.getRemoteCommand(c.remoteCommand, containerName),
			withStdin(f),
		)
		f.Close()
		if err != nil {
			return errors.Annotatef(err, "running %s via SSH", c.remoteCommand)
		}
		if rc != 0 {
			return errors.Errorf("restoring LXC container exited %d", rc)
		}
		return nil
	}

	// Restore each container matching --match,
	// or all machines if --match isn't specified.
	var group errgroup.Group
	for _, container := range lxcContainers {
		containerName := container.Id
		if !match(containerName) {
			ctx.Infof("Skipping non-matching container %q", containerName)
			continue
		}
		path := filepath.Join(c.backupDir, container.InstanceId+".tar.xz")
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				ctx.Infof("Skipping container %q, missing backup file %q", containerName, path)
				continue
			}
		}
		ctx.Infof("Restoring container %q from %s", containerName, path)
		if c.dryRun {
			continue
		}
		group.Go(func() error {
			return errors.Annotatef(
				doRestore(containerName, path),
				"restoring %q from %s",
				containerName, path,
			)
		})
	}
	return group.Wait()
}

var restoreLXCImplDoc = `

restore-lxc-impl must be executed on an API server machine of a 1.25
environment.

The command will get a list of all the LXC containers when run without
arguments. When run with the name of a container, the command will
SSH to the container's host, ensure the container is not running,
stream the container's rootfs as a compressed tarball over stdin,
unpack it, and then start the container.

`

func newRestoreLXCImplCommand() cmd.Command {
	return &restoreLXCImplCommand{}
}

type restoreLXCImplCommand struct {
	baseRemoteCommand
	containerName string
}

func (c *restoreLXCImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "restore-lxc-impl",
		Args:    "[container]",
		Purpose: "controller aspect of restore-lxc",
		Doc:     restoreLXCImplDoc,
	}
}

func (c *restoreLXCImplCommand) Init(args []string) error {
	if len(args) > 0 {
		c.containerName, args = args[0], args[1:]
	}
	return cmd.CheckEmpty(args)
}

func (c *restoreLXCImplCommand) Run(ctx *cmd.Context) error {
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

	logger.Debugf("restoring LXC container %q", c.containerName)
	if err := RestoreLXCContainer(containerMachine, hostMachine, ctx.GetStdin()); err != nil {
		return errors.Annotate(err, "restoring LXC container")
	}
	logger.Debugf("restarting LXC container %q", c.containerName)
	if err := StartLXCContainer(containerMachine, hostMachine); err != nil {
		return errors.Annotate(err, "starting LXC container")
	}
	return nil
}
