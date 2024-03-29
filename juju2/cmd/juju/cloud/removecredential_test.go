// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cloud_test

import (
	"strings"

	"github.com/juju/cmd/cmdtesting"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	jujucloud "github.com/juju/1.25-upgrade/juju2/cloud"
	"github.com/juju/1.25-upgrade/juju2/cmd/juju/cloud"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
	"github.com/juju/1.25-upgrade/juju2/testing"
)

type removeCredentialSuite struct {
	testing.BaseSuite
}

var _ = gc.Suite(&removeCredentialSuite{})

func (s *removeCredentialSuite) TestBadArgs(c *gc.C) {
	cmd := cloud.NewRemoveCredentialCommand()
	_, err := cmdtesting.RunCommand(c, cmd)
	c.Assert(err, gc.ErrorMatches, "Usage: juju remove-credential <cloud-name> <credential-name>")
	_, err = cmdtesting.RunCommand(c, cmd, "cloud", "credential", "extra")
	c.Assert(err, gc.ErrorMatches, `unrecognized args: \["extra"\]`)
}

func (s *removeCredentialSuite) TestMissingCredential(c *gc.C) {
	store := &jujuclient.MemStore{
		Credentials: map[string]jujucloud.CloudCredential{
			"aws": {
				AuthCredentials: map[string]jujucloud.Credential{
					"my-credential": jujucloud.NewCredential(jujucloud.AccessKeyAuthType, nil),
				},
			},
		},
	}
	cmd := cloud.NewRemoveCredentialCommandForTest(store)
	ctx, err := cmdtesting.RunCommand(c, cmd, "aws", "foo")
	c.Assert(err, jc.ErrorIsNil)
	output := cmdtesting.Stderr(ctx)
	output = strings.Replace(output, "\n", "", -1)
	c.Assert(output, gc.Equals, `No credential called "foo" exists for cloud "aws"`)
}

func (s *removeCredentialSuite) TestBadCloudName(c *gc.C) {
	cmd := cloud.NewRemoveCredentialCommandForTest(jujuclient.NewMemStore())
	ctx, err := cmdtesting.RunCommand(c, cmd, "somecloud", "foo")
	c.Assert(err, jc.ErrorIsNil)
	output := cmdtesting.Stderr(ctx)
	output = strings.Replace(output, "\n", "", -1)
	c.Assert(output, gc.Equals, `No credentials exist for cloud "somecloud"`)
}

func (s *removeCredentialSuite) TestRemove(c *gc.C) {
	store := &jujuclient.MemStore{
		Credentials: map[string]jujucloud.CloudCredential{
			"aws": {
				AuthCredentials: map[string]jujucloud.Credential{
					"my-credential":      jujucloud.NewCredential(jujucloud.AccessKeyAuthType, nil),
					"another-credential": jujucloud.NewCredential(jujucloud.AccessKeyAuthType, nil),
				},
			},
		},
	}
	cmd := cloud.NewRemoveCredentialCommandForTest(store)
	ctx, err := cmdtesting.RunCommand(c, cmd, "aws", "my-credential")
	c.Assert(err, jc.ErrorIsNil)
	output := cmdtesting.Stderr(ctx)
	output = strings.Replace(output, "\n", "", -1)
	c.Assert(output, gc.Equals, `Credential "my-credential" for cloud "aws" has been deleted.`)
	_, stillThere := store.Credentials["aws"].AuthCredentials["my-credential"]
	c.Assert(stillThere, jc.IsFalse)
	c.Assert(store.Credentials["aws"].AuthCredentials, gc.HasLen, 1)
}
