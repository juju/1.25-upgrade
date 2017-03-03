// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

// This file imports all of the facades so they get registered at runtime.
// When adding a new facade implementation, import it here so that its init()
// function will get called to register it.
import (
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/action"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/addresser"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/agent"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/annotations"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/backups"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/block"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/charmrevisionupdater"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/charms"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/cleaner"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/client"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/deployer"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/diskmanager"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/environment"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/environmentmanager"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/firewaller"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/highavailability"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/imagemanager"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/imagemetadata"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/instancepoller"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/keymanager"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/keyupdater"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/leadership"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/logger"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/machine"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/machinemanager"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/metricsmanager"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/networker"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/provisioner"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/proxyupdater"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/reboot"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/resumer"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/rsyslog"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/service"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/spaces"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/storage"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/storageprovisioner"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/subnets"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/systemmanager"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/uniter"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/upgrader"
	_ "github.com/juju/1.25-upgrade/juju1/apiserver/usermanager"
)
