// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package all

// Register all the available providers.
import (
	_ "github.com/juju/1.25-upgrade/juju1/provider/azure"
	_ "github.com/juju/1.25-upgrade/juju1/provider/cloudsigma"
	_ "github.com/juju/1.25-upgrade/juju1/provider/ec2"
	_ "github.com/juju/1.25-upgrade/juju1/provider/gce"
	_ "github.com/juju/1.25-upgrade/juju1/provider/joyent"
	_ "github.com/juju/1.25-upgrade/juju1/provider/local"
	_ "github.com/juju/1.25-upgrade/juju1/provider/maas"
	_ "github.com/juju/1.25-upgrade/juju1/provider/manual"
	_ "github.com/juju/1.25-upgrade/juju1/provider/openstack"
	_ "github.com/juju/1.25-upgrade/juju1/provider/vsphere"
)
