// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

// This file imports all of the facades so they get registered at runtime.
// When adding a new facade implementation, import it here so that its init()
// function will get called to register it.
//
// TODO(fwereade): this is silly. We should be declaring our full API in *one*
// place, not scattering it across packages and depending on magic import lists.
import (
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/action" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/agent"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/agenttools"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/annotations" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/application" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/applicationscaler"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/backups" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/block"   // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/bundle"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/charmrevisionupdater"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/charms" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/cleaner"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/client"     // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/cloud"      // ModelUser Read
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/controller" // ModelUser Admin (although some methods check for read only)
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/crossmodel"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/deployer"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/discoverspaces"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/diskmanager"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/firewaller"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/highavailability" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/hostkeyreporter"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/imagemanager" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/imagemetadata"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/instancepoller"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/keymanager" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/keyupdater"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/leadership"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/lifeflag"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/logfwd"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/logger"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/machine"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/machineactions"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/machinemanager" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/machineundertaker"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/meterstatus"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/metricsadder"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/metricsdebug" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/metricsmanager"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/migrationflag"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/migrationmaster"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/migrationminion"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/migrationtarget" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/modelconfig"     // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/modelmanager"    // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/provisioner"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/proxyupdater"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/reboot"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/remoterelations"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/resumer"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/retrystrategy"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/singular"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/spaces"    // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/sshclient" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/statushistory"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/storage" // ModelUser Write
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/storageprovisioner"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/subnets"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/undertaker"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/unitassigner"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/uniter"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/upgrader"
	_ "github.com/juju/1.25-upgrade/juju2/apiserver/usermanager"
)
