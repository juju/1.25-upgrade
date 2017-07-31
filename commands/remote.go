// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
)

func remoteMD5Sum(plugin, address string) (string, error) {
	pluginBase := filepath.Base(plugin)

	result, err := runViaSSH(
		address,
		fmt.Sprintf("md5sum %s | cut -f 1 -d ' '\n", pluginBase),
		"")

	if err != nil {
		return "", errors.Annotate(err, "getting md5sum")
	}

	if result.Code != 0 {
		return "", errors.Errorf("getting md5: %q, %q", result.Stdout, result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func localMD5Sum(plugin string) (string, error) {
	content, err := ioutil.ReadFile(plugin)
	if err != nil {
		return "", errors.Annotate(err, "opening plugin")
	}
	bytes := md5.Sum(content)
	return fmt.Sprintf("%x", bytes), nil
}

func updateRemotePlugin(plugin, address string) error {
	scp := exec.Command("scp", "-C", plugin, fmt.Sprintf("ubuntu@%s:~", address))
	copyResult, err := scp.CombinedOutput()
	if err != nil {
		return errors.Annotate(err, "copying command to environment")
	}
	logger.Debugf("scp: %s", copyResult)
	return nil
}

func checkUpdatePlugin(ctx *cmd.Context, plugin, address string) error {
	ctx.Infof("checking remote plugin")
	local, err := localMD5Sum(plugin)
	if err != nil {
		return errors.Annotate(err, "generating local md5sum")
	}
	ctx.Verbosef("local: %q", local)

	remote, err := remoteMD5Sum(plugin, address)
	if err != nil {
		return errors.Annotate(err, "generating remote md5sum")
	}
	ctx.Verbosef("remote: %q", remote)

	if local != remote {
		ctx.Infof("updating remote plugin")
		return updateRemotePlugin(plugin, address)
	}
	return nil
}
