// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"github.com/juju/loggo"
	"github.com/juju/testing"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju1/cloudconfig/instancecfg"
	"github.com/juju/1.25-upgrade/juju1/cmd/envcmd"
	"github.com/juju/1.25-upgrade/juju1/environs"
	"github.com/juju/1.25-upgrade/juju1/instance"
	"github.com/juju/1.25-upgrade/juju1/provider/common"
	coretesting "github.com/juju/1.25-upgrade/juju1/testing"
	"github.com/juju/1.25-upgrade/juju1/utils/ssh"
)

var logger = loggo.GetLogger("juju.environs.testing")

// DisableFinishBootstrap disables common.FinishBootstrap so that tests
// do not attempt to SSH to non-existent machines. The result is a function
// that restores finishBootstrap.
func DisableFinishBootstrap() func() {
	f := func(environs.BootstrapContext, ssh.Client, instance.Instance, *instancecfg.InstanceConfig) error {
		logger.Warningf("provider/common.FinishBootstrap is disabled")
		return nil
	}
	return testing.PatchValue(&common.FinishBootstrap, f)
}

// BootstrapContext creates a simple bootstrap execution context.
func BootstrapContext(c *gc.C) environs.BootstrapContext {
	return envcmd.BootstrapContext(coretesting.Context(c))
}
