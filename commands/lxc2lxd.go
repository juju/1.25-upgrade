// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/juju/errors"
	"github.com/lxc/lxd/shared/api"

	"github.com/juju/1.25-upgrade/juju1/state"
)

//go:generate go run ../juju2/generate/filetoconst/filetoconst.go LXCMigrationScript lxc-to-lxd lxc2lxd_script.go 2017 commands

type MigrateLXCOptions struct {
	DryRun     bool
	MoveRootfs bool
}

// MigrateLXC changes the LXC containers into LXD containers.
func MigrateLXC(containers []*state.Machine, host *state.Machine, opts MigrateLXCOptions) error {
	var args []string
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.MoveRootfs {
		args = append(args, "--move-rootfs")
	}
	for _, container := range containers {
		// The LXC container name is recorded as the instance ID.
		instanceId, err := container.InstanceId()
		if err != nil {
			return errors.Trace(err)
		}
		args = append(args, string(instanceId))
	}

	var buf bytes.Buffer
	buf.WriteString(`
mkdir -p /var/lib/juju/1.25-upgrade/scripts
cat << 'EOF' > /var/lib/juju/1.25-upgrade/scripts/lxc-to-lxd
`)
	buf.WriteString(LXCMigrationScript)
	buf.WriteString("\nEOF\n")
	buf.WriteString("python3 /var/lib/juju/1.25-upgrade/scripts/lxc-to-lxd ")
	buf.WriteString(strings.Join(args, " "))

	hostAddr, err := getMachineAddress(host)
	if err != nil {
		return errors.Trace(err)
	}
	rc, err := runViaSSH(
		hostAddr,
		buf.String(),
		withSystemIdentity(),
		// write lxc-to-lxd output to stderr
		withStdout(os.Stderr),
	)
	if err != nil {
		return errors.Trace(err)
	}
	if rc != 0 {
		return errors.Errorf("lxc-to-lxd exited %d", rc)
	}
	return nil
}

// StopLXCContainer stops the specified LXC container machine.
func StopLXCContainer(container, host *state.Machine) error {
	hostAddr, err := getMachineAddress(host)
	if err != nil {
		return errors.Trace(err)
	}
	instanceId, err := container.InstanceId()
	if err != nil {
		return errors.Trace(err)
	}
	rc, err := runViaSSH(
		hostAddr,
		"lxc-stop -n "+string(instanceId),
		withSystemIdentity(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	if rc != 0 && rc != 2 {
		return errors.Errorf("lxc-stop exited %d", rc)
	}
	return nil
}

// BackupLXCContainer backups up the specified container as an archive,
// written to the given writer.
func BackupLXCContainer(container, host *state.Machine, out io.Writer) error {
	hostAddr, err := getMachineAddress(host)
	if err != nil {
		return errors.Trace(err)
	}
	instanceId, err := container.InstanceId()
	if err != nil {
		return errors.Trace(err)
	}
	rc, err := runViaSSH(
		hostAddr,
		"tar -C /var/lib/lxc -cJ "+string(instanceId),
		withSystemIdentity(),
		withStdout(out),
	)
	if err != nil {
		return errors.Trace(err)
	}
	if rc != 0 {
		return errors.Errorf("backup of LXC container exited %d", rc)
	}
	return nil
}

// StartLXCContainer starts the specified LXD container machine.
func StartLXDContainers(containerNames []string, host *state.Machine) error {
	hostAddr, err := getMachineAddress(host)
	if err != nil {
		return errors.Trace(err)
	}
	rc, err := runViaSSH(
		hostAddr,
		"lxc start "+strings.Join(containerNames, " "),
		withSystemIdentity(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	if rc != 0 {
		return errors.Errorf("lxc start exited %d", rc)
	}
	return nil
}

// RenameLXDContainer renames a LXD container on the given host.
func RenameLXDContainer(newName, oldName string, host *state.Machine) error {
	hostAddr, err := getMachineAddress(host)
	if err != nil {
		return errors.Trace(err)
	}
	rc, err := runViaSSH(
		hostAddr,
		fmt.Sprintf("lxc move %s %s", oldName, newName),
		withSystemIdentity(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	if rc != 0 {
		return errors.Errorf("renaming LXD container exited %d", rc)
	}
	return nil
}

// SetLXDContainerConfig sets the config for a LXD container on the given host.
func SetLXDContainerConfig(containerName, key, value string, host *state.Machine) error {
	hostAddr, err := getMachineAddress(host)
	if err != nil {
		return errors.Trace(err)
	}
	rc, err := runViaSSH(
		hostAddr,
		fmt.Sprintf("lxc config set %s %s %s", containerName, key, value),
		withSystemIdentity(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	if rc != 0 {
		return errors.Errorf("setting LXD container config exited %d", rc)
	}
	return nil
}

// ListLXDContainers lists the LXD containers on the given host.
func ListLXDContainers(host *state.Machine) (map[string]*lxdContainer, error) {
	hostAddr, err := getMachineAddress(host)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var buf bytes.Buffer
	rc, err := runViaSSH(
		hostAddr,
		// NOTE(axw) older versions of the lxc version
		// do not support --format=yaml.
		"lxc list --format=json",
		withSystemIdentity(),
		withStdout(&buf),
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if rc != 0 {
		return nil, errors.Errorf("listing LXD containers exited %d", rc)
	}

	var lxcList []*lxdContainer
	if err := json.Unmarshal(buf.Bytes(), &lxcList); err != nil {
		return nil, errors.Trace(err)
	}
	containers := make(map[string]*lxdContainer)
	for _, item := range lxcList {
		containers[item.Name] = item
	}
	return containers, nil
}

type lxdContainer struct {
	*api.Container
	State     *api.ContainerState     `json:"state" yaml:"state"`
	Snapshots []api.ContainerSnapshot `json:"snapshots" yaml:"snapshots"`
}
