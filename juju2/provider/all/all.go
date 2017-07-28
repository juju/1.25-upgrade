// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package all

// Register all the available providers.
import (
	_ "github.com/juju/1.25-upgrade/juju2/provider/azure"
	_ "github.com/juju/1.25-upgrade/juju2/provider/cloudsigma"
	_ "github.com/juju/1.25-upgrade/juju2/provider/ec2"
	_ "github.com/juju/1.25-upgrade/juju2/provider/gce"
	_ "github.com/juju/1.25-upgrade/juju2/provider/joyent"
	_ "github.com/juju/1.25-upgrade/juju2/provider/maas"
	_ "github.com/juju/1.25-upgrade/juju2/provider/manual"
	_ "github.com/juju/1.25-upgrade/juju2/provider/openstack"
	_ "github.com/juju/1.25-upgrade/juju2/provider/oracle"
	_ "github.com/juju/1.25-upgrade/juju2/provider/rackspace"
	_ "github.com/juju/1.25-upgrade/juju2/provider/vsphere"
)
