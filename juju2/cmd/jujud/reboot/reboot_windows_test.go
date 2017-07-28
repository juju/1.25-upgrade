// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package reboot_test

import (
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/cmd/jujud/reboot"
)

const (
	rebootBin  = "shutdown.exe"
	rebootTime = "15"
)

func (s *RebootSuite) rebootCommandParams(c *gc.C) []string {
	return []string{
		"-f",
		"-r",
		"-t",
		rebootTime,
	}
}

func (s *RebootSuite) shutdownCommandParams(c *gc.C) []string {
	return []string{
		"-f",
		"-s",
		"-t",
		rebootTime,
	}
}

func (s *RebootSuite) TestRebootNoContainers(c *gc.C) {
	w, err := reboot.NewRebootWaiter(s.st, s.acfg)
	c.Assert(err, jc.ErrorIsNil)
	expectedRebootParams := s.rebootCommandParams(c)

	err = w.ExecuteReboot(params.ShouldReboot)
	c.Assert(err, jc.ErrorIsNil)
	testing.AssertEchoArgs(c, rebootBin, expectedRebootParams...)
}

func (s *RebootSuite) TestShutdownNoContainers(c *gc.C) {
	w, err := reboot.NewRebootWaiter(s.st, s.acfg)
	c.Assert(err, jc.ErrorIsNil)
	expectedShutdownParams := s.shutdownCommandParams(c)

	err = w.ExecuteReboot(params.ShouldShutdown)
	c.Assert(err, jc.ErrorIsNil)
	testing.AssertEchoArgs(c, rebootBin, expectedShutdownParams...)
}
