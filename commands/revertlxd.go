// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"github.com/juju/utils"
	"golang.org/x/sync/errgroup"

	"github.com/juju/1.25-upgrade/juju1/state"
)

var revertLXDDoc = `
The revert-lxd command reverts migrated LXD containers back to LXC as
part of aborting a 1.25 upgrade.

If --match is specified, it is treated as a regular expression for
matching container names. Only containers whose original (unmigrated)
names match will be reverted.

This command requires information from the source environment state
database, so it will only work after running the abort command if
upgrade-agents has been run.
`

func newRevertLXDCommand() cmd.Command {
	command := &revertLXDCommand{}
	command.remoteCommand = "revert-lxd-impl"
	return wrap(command)
}

type revertLXDCommand struct {
	baseClientCommand
	match string
}

func (c *revertLXDCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "revert-lxd",
		Args:    "<environment name>",
		Purpose: "revert migrated LXD containers back to LXC",
		Doc:     revertLXDDoc,
	}
}

func (c *revertLXDCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseClientCommand.SetFlags(f)
	f.StringVar(&c.match, "match", "", "regular expression for matching LXD container IDs to revert")
}

func (c *revertLXDCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

func (c *revertLXDCommand) Run(ctx *cmd.Context) error {
	if c.match != "" {
		c.extraOptions = append(c.extraOptions, utils.ShQuote("--match="+c.match))
	}
	return c.baseClientCommand.Run(ctx)
}

var revertLXDImplDoc = `

revert-lxd-impl must be executed on an API server machine of a 1.25
environment.

The command will revert matching migrated LXD containers in the
environment back to LXC.

`

func newRevertLXDImplCommand() cmd.Command {
	return &revertLXDImplCommand{}
}

type revertLXDImplCommand struct {
	baseRemoteCommand
	match string
}

func (c *revertLXDImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "revert-lxd-impl",
		Purpose: "controller aspect of revert-lxd",
		Doc:     revertLXDImplDoc,
	}
}

func (c *revertLXDImplCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseRemoteCommand.SetFlags(f)
	f.StringVar(&c.match, "match", "", "regular expression for matching LXD container IDs to revert")
}

func (c *revertLXDImplCommand) Run(ctx *cmd.Context) error {
	match := func(string) bool { return true }
	if c.match != "" {
		matchRE, err := regexp.Compile(c.match)
		if err != nil {
			return errors.Annotate(err, "parsing --match")
		}
		match = matchRE.MatchString
	}

	st, err := getState()
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	// Collect LXC container machines by host.
	lxcByHost, err := getLXCContainersFromState(st)
	if err != nil {
		return errors.Trace(err)
	}
	for host, containers := range lxcByHost {
		matching := make([]*state.Machine, 0, len(containers))
		for _, container := range containers {
			if !match(container.Id()) {
				ctx.Infof("Skipping non-matching container %q", container.Id())
				continue
			}
			matching = append(matching, container)
		}
		if len(matching) == 0 {
			delete(lxcByHost, host)
			continue
		}
		lxcByHost[host] = matching
	}

	environUUID := st.EnvironUUID()
	containerNames, err := getContainerNames(lxcByHost, environUUID)
	if err != nil {
		return errors.Trace(err)
	}

	// List LXD containers on each host, so we know which containers
	// still need reverting. This is required to make the revert-lxd
	// command idempotent/resumable.
	hosts := make([]*state.Machine, 0, len(lxcByHost))
	for host := range lxcByHost {
		hosts = append(hosts, host)
	}
	lxdByHost, err := getLXDContainersFromMachines(hosts)
	if err != nil {
		return errors.Trace(err)
	}

	// Determine which LXD containers still need to be reverted,
	// by looking for LXD containers of the same (or new) names.
	lxdToRevertByHost := getAllLXDContainersToRevert(
		ctx, lxcByHost, lxdByHost, containerNames,
	)

	if err := stopLXDContainers(lxdToRevertByHost); err != nil {
		return errors.Annotate(err, "stopping LXD containers")
	}
	if err := revertAllLXDContainers(lxdToRevertByHost); err != nil {
		return errors.Annotate(err, "reverting LXD containers")
	}

	// Start all not-running LXC containers back up. The agents must
	// be stopped.
	if err := startLXCContainers(lxcByHost); err != nil {
		return errors.Annotate(err, "starting LXC containers")
	}
	if err := waitContainersReady(
		"lxc", lxcByHost, containerNames,
		5*time.Minute, // should be long enough for anyone
	); err != nil {
		return errors.Annotate(err, "waiting for reverted LXC containers to have addresses")
	}
	if err := stopContainerAgents(ctx, st, lxcByHost); err != nil {
		return errors.Annotate(err, "stopping Juju agents in reverted LXC containers")
	}

	return nil
}

type containerMap map[string]*state.Machine

type hostContainerMap map[*state.Machine]containerMap

// getLXDContainersToRevert returns a map of the LXD containers to
// revert. The outermost key is the host machine. The inner map is lxd
// container name -> container state.Machine. (This lets us handle LXD
// containers that have been migrated uniformly whether or not they've
// been renamed.)
func getAllLXDContainersToRevert(
	ctx *cmd.Context,
	lxcByHost map[*state.Machine][]*state.Machine,
	lxdByHost map[*state.Machine]map[string]*lxdContainer,
	containerNames map[*state.Machine]containerNames,
) hostContainerMap {
	lxdToRevertByHost := make(hostContainerMap)
	for host, containers := range lxcByHost {
		lxdContainers := lxdByHost[host]
		toRevert := getLXDToRevertForHost(ctx, containers, lxdContainers, containerNames)
		if len(toRevert) > 0 {
			lxdToRevertByHost[host] = toRevert
		}
	}
	return lxdToRevertByHost
}

// getLXDToRevertForHost returns a map of (lxd container name,
// juju machine) for a specific host machine.
func getLXDToRevertForHost(
	ctx *cmd.Context,
	lxcMachines []*state.Machine,
	lxdContainers map[string]*lxdContainer,
	containerNames map[*state.Machine]containerNames,
) containerMap {
	toRevert := make(map[string]*state.Machine)
	for _, lxcMachine := range lxcMachines {
		names := containerNames[lxcMachine]
		if lxdContainers[names.oldName] != nil {
			// migrated but not yet renamed
			ctx.Infof(
				"LXC container %q was migrated to LXD (%q)",
				lxcMachine.Id(), names.oldName,
			)
			toRevert[names.oldName] = lxcMachine
		} else if lxdContainers[names.newName] != nil {
			// migrated and renamed
			ctx.Infof(
				"LXC container %q was migrated to LXD (%q)",
				lxcMachine.Id(), names.newName,
			)
			toRevert[names.newName] = lxcMachine
		} else {
			ctx.Infof(
				"LXC container %q hasn't been migrated to LXD (%q) - no revert needed",
				lxcMachine.Id(), names.newName,
			)
		}
	}
	return toRevert
}

func stopLXDContainers(toRevertByHost hostContainerMap) error {
	var group errgroup.Group
	for host, containers := range toRevertByHost {
		for lxdName := range containers {
			logger.Debugf("stopping LXD container %q", lxdName)
			host, lxdName := host, lxdName // copy for closure
			group.Go(func() error {
				return errors.Annotatef(
					StopLXDContainer(lxdName, host),
					"stopping LXD container %q",
					lxdName,
				)
			})
		}
	}
	return group.Wait()
}

// startLXCContainers will try to start all of the containers that
// should be on each machine - this will already have been filtered by
// the match expression. We don't bother checking for ones that are
// already running - it's ok if they're already started, since
// lxc-start still returns 0.
func startLXCContainers(lxcByHost map[*state.Machine][]*state.Machine) error {
	var group errgroup.Group
	for host, containers := range lxcByHost {
		for _, container := range containers {
			logger.Debugf("starting reverted LXC container %q", container.Id())
			host, container := host, container // copy for closure
			group.Go(func() error {
				return errors.Annotatef(
					StartLXCContainer(container, host),
					"starting LXC container %q",
					container.Id(),
				)
			})
		}
	}
	return group.Wait()
}

// revertLXDContainers converts the specified LXD containers back into
// LXC containers by:
// * moving the rootfs back into the LXC container,
// * undoing the config change for the LXC container,
// * deleting the LXD version of the container
func revertAllLXDContainers(toRevertByHost hostContainerMap) error {
	var group errgroup.Group
	for host, containers := range toRevertByHost {
		host, containers := host, containers // copy for closure
		var containerNames []string
		for name := range containers {
			containerNames = append(containerNames, name)
		}
		group.Go(func() error {
			return errors.Annotatef(
				revertLXDForHost(host, containers),
				"reverting LXD containers: %s", strings.Join(containerNames, ", "),
			)
		})
	}
	return group.Wait()
}

func revertLXDForHost(host *state.Machine, toRevert containerMap) error {
	address, err := getMachineAddress(host)
	if err != nil {
		return errors.Annotatef(err, "getting address for machine %q", host.Id())
	}

	parts := []string{revertFunction}
	for lxdName, lxcMachine := range toRevert {
		instanceId, err := lxcMachine.InstanceId()
		if err != nil {
			return errors.Annotatef(err, "getting instance id for %q", lxcMachine.Id())
		}
		parts = append(parts, fmt.Sprintf(revertCall, lxdName, instanceId))
	}
	script := strings.Join(parts, "\n")
	logger.Debugf("running revert script on %q:\n\n%s", host.Id(), script)
	rc, err := runViaSSH(address, script, withSystemIdentity())
	if err != nil {
		return errors.Annotatef(err, "running revert script for %q", host.Id())
	}
	if rc != 0 {
		return errors.Errorf("revert script exited %d", rc)
	}
	return nil
}

const (
	revertFunction = `
set -ex

LXC_BASE=/var/lib/lxc
LXD_BASE=/var/lib/lxd/containers

function revert-lxd() {
   source_lxd=$1
   target_lxc=$2
   echo reverting from $source_lxd to $target_lxc
   # remove the migrated line from the config
   sed '/^lxd.migrated=true$/d' -i $LXC_BASE/$target_lxc/config
   # move the rootfs back
   mv $LXD_BASE/$source_lxd/rootfs $LXC_BASE/$target_lxc/
   # remove the container from lxd
   lxc delete $source_lxd
}
`

	revertCall = "revert-lxd %q %q"
)
