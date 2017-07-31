// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"bytes"
	"os"
	"strings"

	"github.com/juju/description"
	"github.com/juju/errors"
)

//go:generate go run ../juju2/generate/filetoconst/filetoconst.go LXCMigrationScript ../../../lxc/lxd/scripts/lxc-to-lxd lxc2lxd_script.go 2017 commands

type MigrateLXCOptions struct {
	DryRun     bool
	MoveRootfs bool
}

func MigrateLXC(containers []description.Machine, host description.Machine, opts MigrateLXCOptions) error {
	var args []string
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.MoveRootfs {
		args = append(args, "--move-rootfs")
	}
	for _, container := range containers {
		args = append(args,
			// The LXC container name is recorded as the instance ID.
			container.Instance().InstanceId(),
		)
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

	hostAddr := host.PreferredPrivateAddress()
	rc, err := runViaSSH(
		hostAddr.Value(),
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
