// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package imagemetadataworker_test

import (
	"testing"

	gc "gopkg.in/check.v1"

	apitesting "github.com/juju/1.25-upgrade/juju2/api/base/testing"
	"github.com/juju/1.25-upgrade/juju2/api/imagemetadata"
	coretesting "github.com/juju/1.25-upgrade/juju2/testing"
)

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

type baseMetadataSuite struct {
	coretesting.BaseSuite
	apiCalled bool
}

func (s *baseMetadataSuite) ImageClient(done chan struct{}) *imagemetadata.Client {
	closer := apitesting.APICallerFunc(func(objType string, version int, id, request string, a, result interface{}) error {
		s.apiCalled = false
		if request == "UpdateFromPublishedImages" {
			s.apiCalled = true
			close(done)
			return nil
		}
		return nil
	})

	return imagemetadata.NewClient(closer)
}
