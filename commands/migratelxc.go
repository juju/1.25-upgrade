// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"context"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"github.com/juju/utils"
	"github.com/juju/utils/set"
	"golang.org/x/sync/errgroup"

	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju2/instance"
)

var migrateLXCDoc = ` 
The purpose of the migrate-lxc command is to migrate the
LXC containers in a 1.25 environment to LXD.

If --match is specified, it is treated as a regular expression for
matching container names. Only containers whose names match will
be migrated.

If --dry-run is specified, then no migration will actually be
performed, nor will the containers be stopped.
`

func newMigrateLXCCommand() cmd.Command {
	command := &migrateLXCCommand{}
	command.remoteCommand = "migrate-lxc-impl"
	return wrap(command)
}

type migrateLXCCommand struct {
	baseClientCommand
	dryRun bool
	match  string
}

func (c *migrateLXCCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "migrate-lxc",
		Args:    "<environment name>",
		Purpose: "migrate of all the LXC containers in the specified environment to LXD",
		Doc:     migrateLXCDoc,
	}
}

func (c *migrateLXCCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseClientCommand.SetFlags(f)
	f.BoolVar(&c.dryRun, "dry-run", false, "perform a dry run, without making any changes")
	f.StringVar(&c.match, "match", "", "regular expression for matching LXC container IDs to migrate")
}

func (c *migrateLXCCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

func (c *migrateLXCCommand) Run(ctx *cmd.Context) error {
	if c.match != "" {
		c.extraOptions = append(c.extraOptions, utils.ShQuote("--match="+c.match))
	}
	if c.dryRun {
		c.extraOptions = append(c.extraOptions, "--dry-run")
	}
	return c.baseClientCommand.Run(ctx)
}

var migrateLXCImplDoc = `

migrate-lxc-impl must be executed on an API server machine of a 1.25
environment.

The command will migrate all LXC containers in the environment to LXD.

`

func newMigrateLXCImplCommand() cmd.Command {
	return &migrateLXCImplCommand{}
}

type migrateLXCImplCommand struct {
	baseRemoteCommand
	dryRun bool
	match  string
}

func (c *migrateLXCImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "migrate-lxc-impl",
		Purpose: "controller aspect of migrate-lxc",
		Doc:     migrateLXCImplDoc,
	}
}

func (c *migrateLXCImplCommand) SetFlags(f *gnuflag.FlagSet) {
	c.baseRemoteCommand.SetFlags(f)
	f.BoolVar(&c.dryRun, "dry-run", false, "perform a dry run, without making any changes")
	f.StringVar(&c.match, "match", "", "regular expression for matching LXC container IDs to migrate")
}

func (c *migrateLXCImplCommand) Run(ctx *cmd.Context) error {
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

	// List LXD containers on each host, so we know which LXC
	// containers still need migrating. This is required to make
	// the migrate-lxc command idempotent/resumable.
	hosts := make([]*state.Machine, 0, len(lxcByHost))
	for host := range lxcByHost {
		hosts = append(hosts, host)
	}
	lxdByHost, err := getLXDContainersFromMachines(hosts)
	if err != nil {
		return errors.Trace(err)
	}

	// Determine which LXC containers still need to be migrated,
	// by looking for LXD containers of the same (or new) names.
	lxcToMigrateByHost := getLXCContainersToMigrate(
		ctx, lxcByHost, lxdByHost, containerNames,
	)

	if c.dryRun {
		return nil
	}

	if err := stopLXCContainers(lxcToMigrateByHost); err != nil {
		return errors.Annotate(err, "stopping LXC containers")
	}
	if err := migrateLXCContainers(lxcToMigrateByHost); err != nil {
		return errors.Annotate(err, "migrating LXC containers")
	}

	// Rename the LXD containers and set metadata.
	if err := renameLXDContainers(lxcByHost, lxdByHost, containerNames, environUUID); err != nil {
		return errors.Annotate(err, "renaming LXD containers")
	}

	// Start the LXD containers back up, so the other upgrade
	// commands (upgrade agents, etc.) can work. The agents must
	// be stopped.
	if err := startLXDContainers(lxcByHost, lxdByHost, containerNames); err != nil {
		return errors.Annotate(err, "starting LXD containers")
	}
	if err := waitContainersReady(
		"lxd", lxcByHost, containerNames,
		5*time.Minute, // should be long enough for anyone
	); err != nil {
		return errors.Annotate(err, "waiting for LXD containers to have addresses")
	}
	if err := stopContainerAgents(ctx, st, lxcByHost); err != nil {
		return errors.Annotate(err, "stopping Juju agents in LXD containers")
	}

	return nil
}

// getLXCContainersFromState returns a map of host machines
// to LXC containers contained within them. Hosts without
// LXC containers are not included in the map.
func getLXCContainersFromState(st *state.State) (map[*state.Machine][]*state.Machine, error) {
	// Collect LXC container machines by host.
	hosts := make(map[string]*state.Machine)
	byHost := make(map[*state.Machine][]*state.Machine)
	machines, err := st.AllMachines()
	if err != nil {
		return nil, errors.Annotate(err, "getting machines")
	}
	for _, m := range machines {
		if m.ContainerType() != "lxc" {
			continue
		}
		parentId, _ := m.ParentId()
		host, ok := hosts[parentId]
		if !ok {
			var err error
			host, err = st.Machine(parentId)
			if err != nil {
				return nil, errors.Annotate(err, "getting host machine")
			}
			hosts[parentId] = host
		}
		byHost[host] = append(byHost[host], m)
	}
	return byHost, nil
}

// getLXDContainersFromMachines returns a map of host machines
// to LXD containers contained within them. Hosts without LXD
// containers are not included in the map.
func getLXDContainersFromMachines(hosts []*state.Machine) (map[*state.Machine]map[string]*lxdContainer, error) {
	var group errgroup.Group
	lxdContainers := make([]map[string]*lxdContainer, len(hosts))
	for i, host := range hosts {
		i, host := i, host // copy for closure
		group.Go(func() error {
			containers, err := ListLXDContainers(host)
			if err != nil {
				return errors.Annotatef(err,
					"listing LXD containers for host %q",
					host.Id(),
				)
			}
			lxdContainers[i] = containers
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, errors.Annotate(err, "listing LXD containers")
	}
	lxdByHost := make(map[*state.Machine]map[string]*lxdContainer)
	for i, containers := range lxdContainers {
		lxdByHost[hosts[i]] = containers
	}
	return lxdByHost, nil
}

type containerNames struct {
	// oldName is the LXC container name used by Juju 1.25.
	oldName string

	// newName is the LXD container name used by Juju 2.x.
	newName string
}

// getContainerNames returns the old (LXC) and new (LXD) container names
// for the container entries recorded in state.
func getContainerNames(
	byHost map[*state.Machine][]*state.Machine,
	environUUID string,
) (map[*state.Machine]containerNames, error) {
	namespace, err := instance.NewNamespace(environUUID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	names := make(map[*state.Machine]containerNames)
	for _, containers := range byHost {
		for _, container := range containers {
			// The recorded instance ID is the old
			// LXC container name.
			instanceId, err := container.InstanceId()
			if err != nil {
				return nil, errors.Trace(err)
			}
			oldName := string(instanceId)

			// The new name, expected by Juju 2.x, uses a model
			// UUID namespace.
			newName, err := namespace.Hostname(strings.Replace(
				container.Id(), "lxc", "lxd", 1))
			if err != nil {
				return nil, errors.Trace(err)
			}

			names[container] = containerNames{
				oldName: oldName,
				newName: newName,
			}
		}
	}
	return names, nil
}

// getLXCContainersToMigrate returns the LXC containers to be migrated,
// grouped by host.
func getLXCContainersToMigrate(
	ctx *cmd.Context,
	lxcByHost map[*state.Machine][]*state.Machine,
	lxdByHost map[*state.Machine]map[string]*lxdContainer,
	containerNames map[*state.Machine]containerNames,
) map[*state.Machine][]*state.Machine {
	lxcToMigrateByHost := make(map[*state.Machine][]*state.Machine)
	for host, containers := range lxcByHost {
		lxdContainers := lxdByHost[host]
		nonMigrated := make([]*state.Machine, 0, len(containers))
		for _, container := range containers {
			names := containerNames[container]
			if lxdContainers[names.oldName] != nil {
				// migrated but not yet renamed
				ctx.Infof(
					"LXC container %q already migrated to LXD (%q)",
					container.Id(), names.oldName,
				)
				continue
			}
			if lxdContainers[names.newName] != nil {
				// migrated and renamed
				ctx.Infof(
					"LXC container %q already migrated to LXD (%q)",
					container.Id(), names.newName,
				)
				continue
			}
			ctx.Infof(
				"Migrating LXC container %q to LXD (%q)",
				container.Id(), names.newName,
			)
			nonMigrated = append(nonMigrated, container)
		}
		if len(nonMigrated) > 0 {
			lxcToMigrateByHost[host] = nonMigrated
		}
	}
	return lxcToMigrateByHost
}

// stopLXCContainers stops all of the LXC containers.
func stopLXCContainers(lxcByHost map[*state.Machine][]*state.Machine) error {
	var group errgroup.Group
	for host, containers := range lxcByHost {
		for _, container := range containers {
			logger.Debugf("stopping LXC container %q", container.Id())
			host, container := host, container // copy for closure
			group.Go(func() error {
				return errors.Annotatef(
					StopLXCContainer(container, host),
					"stopping LXC container %q",
					container.Id(),
				)
			})
		}
	}
	return group.Wait()
}

// migrateLXCContainers migrates all of the LXC containers to LXD.
func migrateLXCContainers(lxcByHost map[*state.Machine][]*state.Machine) error {
	opts := MigrateLXCOptions{
		// TODO(axw) option to copy rootfs?
		MoveRootfs: true,
	}
	var group errgroup.Group
	for host, containers := range lxcByHost {
		containerNames := make([]string, len(containers))
		for i, container := range containers {
			containerNames[i] = container.Id()
		}
		logger.Debugf("migrating LXC containers: %s", strings.Join(containerNames, ", "))
		host, containers := host, containers // copy for closure
		group.Go(func() error {
			return errors.Annotatef(
				MigrateLXC(containers, host, opts),
				"migrating LXC containers: %s", strings.Join(containerNames, ", "),
			)
		})
	}
	return group.Wait()
}

// renameLXDContainers renames all of the LXD containers to the new name,
// if they aren't already named as such.
func renameLXDContainers(
	lxcByHost map[*state.Machine][]*state.Machine,
	lxdByHost map[*state.Machine]map[string]*lxdContainer,
	containerNames map[*state.Machine]containerNames,
	environUUID string,
) error {
	renameLXDContainer := func(newName, oldName string, host *state.Machine) error {
		if err := SetLXDContainerConfig(oldName, "user.juju-model", environUUID, host); err != nil {
			return errors.Trace(err)
		}
		return RenameLXDContainer(newName, oldName, host)
	}
	var group errgroup.Group
	for host, containers := range lxcByHost {
		// lxcByHost contains all of the containers recorded in state,
		// whether or not they've been migrated. We filter out the
		// ones that were already migrated and renamed in a previous
		// session.
		lxdContainers := lxdByHost[host]
		for _, container := range containers {
			names := containerNames[container]
			if lxdContainers[names.newName] != nil {
				// Already renamed.
				continue
			}
			host := host // copy for closure
			group.Go(func() error {
				return errors.Annotatef(
					renameLXDContainer(names.newName, names.oldName, host),
					"renaming LXD container %q to %q",
					names.oldName, names.newName,
				)
			})
		}
	}
	return group.Wait()
}

// startLXDContainers starts the LXD containers that have
// just been migrated from LXC, or were already migrated
// but not yet started.
func startLXDContainers(
	lxcByHost map[*state.Machine][]*state.Machine,
	lxdByHost map[*state.Machine]map[string]*lxdContainer,
	containerNames map[*state.Machine]containerNames,
) error {
	running := make(set.Strings) // keyed by LXD container name
	for _, containers := range lxdByHost {
		for name, container := range containers {
			if container.IsActive() {
				running.Add(name)
			}
		}
	}
	var group errgroup.Group
	for host, containers := range lxcByHost {
		// By this stage, all of the LXC containers
		// have been migrated to LXD and renamed.
		// If they don't feature in "lxdByHost",
		// then they were migrated by this process,
		// and they're known to not be running.
		//
		// NOTE(axw) lxcByHost is based on the
		// container machines recorded in state,
		// so it will contain records for LXC
		// containers that have already been
		// migrated to LXD.
		toStart := make([]string, 0, len(containers))
		for _, container := range containers {
			newName := containerNames[container].newName
			if !running.Contains(newName) {
				toStart = append(toStart, newName)
			}
		}
		if len(toStart) == 0 {
			logger.Debugf("no LXD containers to start on %q", host.Id())
			continue
		}
		logger.Debugf("starting LXD containers on %q: %q", host.Id(), toStart)
		host := host // copy for closure
		group.Go(func() error {
			return errors.Annotatef(
				StartLXDContainers(toStart, host),
				"starting LXD containers on %q: %q",
				host.Id(), toStart,
			)
		})
	}
	return group.Wait()
}

// waitContainersReady waits for the containers to have addresses, as
// recorded in the state database, and be ready to accept SSH
// connections.
func waitContainersReady(
	containerType string,
	lxcByHost map[*state.Machine][]*state.Machine,
	containerNames map[*state.Machine]containerNames,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	logger.Debugf("waiting for %s containers to be ready for SSH connections", containerType)
	for host, containers := range lxcByHost {
		hostAddr, err := getMachineAddress(host)
		if err != nil {
			return errors.Trace(err)
		}
		for _, container := range containers {
			containerAddr, err := getMachineAddress(container)
			if err != nil {
				return errors.Trace(err)
			}
			var containerName string
			if containerType == "lxd" {
				containerName = containerNames[container].newName
			} else {
				containerName = containerNames[container].oldName
			}
			group.Go(func() error {
				return errors.Annotatef(
					waitContainerReady(ctx, containerName, containerAddr, hostAddr),
					"waiting for %q to be ready for SSH connections",
					containerName,
				)
			})
		}
	}
	return group.Wait()
}

func waitContainerReady(ctx context.Context, containerName, containerAddr, hostAddr string) error {
	const interval = time.Second // time to wait between checks
	for {
		logger.Debugf(
			"waiting for %q to be ready for SSH connections via %q",
			containerName, containerAddr,
		)
		if rc, err := runViaSSH(
			containerAddr, "/bin/true",
			withSystemIdentity(),
			withProxyCommandForHost(hostAddr),
			withStdout(ioutil.Discard),
			withStderr(ioutil.Discard),
		); err == nil && rc == 0 {
			return nil
		}
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// stopContainerAgents stops the Juju agents running inside
// LXC or LXD containers. The containers are expected to be running.
func stopContainerAgents(
	ctx *cmd.Context,
	st *state.State,
	lxcByHost map[*state.Machine][]*state.Machine,
) error {
	// Stop the Juju agents on the containers.
	var flatMachines []FlatMachine
	for _, containers := range lxcByHost {
		for _, container := range containers {
			fm, err := makeFlatMachine(st, container)
			if err != nil {
				return errors.Trace(err)
			}
			flatMachines = append(flatMachines, fm)
		}
	}

	logger.Debugf("stopping Juju agents running in container machines")
	_, err := agentServiceCommand(ctx, flatMachines, "stop")
	return errors.Trace(err)
}
