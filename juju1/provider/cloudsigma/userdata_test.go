// Copyright 2015 Canonical Ltd.
// Copyright 2015 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package cloudsigma_test

import (
	"encoding/base64"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju1/provider/cloudsigma"
	"github.com/juju/1.25-upgrade/juju1/testing"
	"github.com/juju/1.25-upgrade/juju1/version"
)

type UserdataSuite struct{ testing.BaseSuite }

var _ = gc.Suite(&UserdataSuite{})

func (s *UserdataSuite) TestCloudSigmaUnix(c *gc.C) {
	renderer := cloudsigma.CloudSigmaRenderer{}
	data := []byte("test")
	result, err := renderer.EncodeUserdata(data, version.Ubuntu)
	c.Assert(err, jc.ErrorIsNil)
	expected := base64.StdEncoding.EncodeToString(data)
	c.Assert(string(result), jc.DeepEquals, expected)

	data = []byte("test")
	result, err = renderer.EncodeUserdata(data, version.CentOS)
	c.Assert(err, jc.ErrorIsNil)
	expected = base64.StdEncoding.EncodeToString(data)
	c.Assert(string(result), jc.DeepEquals, expected)
}

func (s *UserdataSuite) TestCloudSigmaUnknownOS(c *gc.C) {
	renderer := cloudsigma.CloudSigmaRenderer{}
	result, err := renderer.EncodeUserdata(nil, version.Windows)
	c.Assert(result, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, "Cannot encode userdata for OS: Windows")
}
