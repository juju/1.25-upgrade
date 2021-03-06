// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package crossmodel_test

import (
	"testing"

	gc "gopkg.in/check.v1"

	"github.com/juju/1.25-upgrade/juju2/jujuclient"
	jujutesting "github.com/juju/1.25-upgrade/juju2/testing"
)

func TestAll(t *testing.T) {
	gc.TestingT(t)
}

type BaseCrossModelSuite struct {
	jujutesting.BaseSuite

	store *jujuclient.MemStore
}

func (s *BaseCrossModelSuite) SetUpTest(c *gc.C) {
	// Set up the current controller, and write just enough info
	// so we don't try to refresh
	controllerName := "test-master"
	s.store = jujuclient.NewMemStore()
	s.store.CurrentControllerName = controllerName
	s.store.Controllers[controllerName] = jujuclient.ControllerDetails{}
	s.store.Models[controllerName] = &jujuclient.ControllerModels{
		CurrentModel: "fred/test",
		Models: map[string]jujuclient.ModelDetails{
			"bob/test":  {"test-uuid"},
			"bob/prod":  {"prod-uuid"},
			"fred/test": {"fred-uuid"},
		},
	}
	s.store.Accounts[controllerName] = jujuclient.AccountDetails{
		User: "bob",
	}
}
