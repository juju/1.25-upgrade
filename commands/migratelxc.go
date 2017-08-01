// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/lxc/lxd/shared/api"
	"golang.org/x/sync/errgroup"

	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju2/instance"
)

var migrateLXCDoc = ` 
The purpose of the migrate-lxc command is to migrate all of the
LXC containers in a 1.25 environment to LXD.
`

func newMigrateLXCCommand() cmd.Command {
	command := &migrateLXCCommand{}
	command.remoteCommand = "migrate-lxc-impl"
	return wrap(command)
}

type migrateLXCCommand struct {
	baseClientCommand
	migrateDir string
}

func (c *migrateLXCCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "migrate-lxc",
		Args:    "<environment name>",
		Purpose: "migrate of all the LXC containers in the specified environment to LXD",
		Doc:     migrateLXCDoc,
	}
}

func (c *migrateLXCCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
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
}

func (c *migrateLXCImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "migrate-lxc-impl",
		Purpose: "controller aspect of migrate-lxc",
		Doc:     migrateLXCImplDoc,
	}
}

func (c *migrateLXCImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	// Collect LXC container machines by host.
	lxcByHost, err := getLXCContainersFromState(st)
	if err != nil {
		return errors.Trace(err)
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
	lxcToMigrateByHost := getLXCContainersToMigrate(lxcByHost, lxdByHost, containerNames)
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
func getLXDContainersFromMachines(hosts []*state.Machine) (map[*state.Machine]map[string]*api.Container, error) {
	var group errgroup.Group
	lxdContainers := make([]map[string]*api.Container, len(hosts))
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
	lxdByHost := make(map[*state.Machine]map[string]*api.Container)
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
			newName, err := namespace.Hostname(container.Id())
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
	lxcByHost map[*state.Machine][]*state.Machine,
	lxdByHost map[*state.Machine]map[string]*api.Container,
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
				logger.Debugf(
					"LXC container %q already migrated to LXD (%q)",
					container.Id(), names.oldName,
				)
				continue
			}
			if lxdContainers[names.newName] != nil {
				// migrated and renamed
				logger.Debugf(
					"LXC container %q already migrated to LXD (%q)",
					container.Id(), names.newName,
				)
				continue
			}
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
	lxdByHost map[*state.Machine]map[string]*api.Container,
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
