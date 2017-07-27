// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package internal_test

import (
	"io"
	"time"

	"github.com/juju/testing"
	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/resource"
	"github.com/juju/1.25-upgrade/juju2/resource/resourcetesting"
)

func newResource(c *gc.C, stub *testing.Stub, name, content string) (resource.Resource, io.ReadCloser) {
	opened := resourcetesting.NewResource(c, stub, name, "a-application", content)
	res := opened.Resource
	if content != "" {
		return res, opened.ReadCloser
	}
	res.Username = ""
	res.Timestamp = time.Time{}
	return res, nil
}
