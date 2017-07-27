// Copyright 2014 Canonical Ltd. All rights reserved.
// Licensed under the AGPLv3, see LICENCE file for details.

package authentication

import (
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/state"
)

// EntityAuthenticator is the interface all entity authenticators need to implement
// to authenticate juju entities.
type EntityAuthenticator interface {
	// Authenticate authenticates the given entity
	Authenticate(entityFinder EntityFinder, tag names.Tag, req params.LoginRequest) (state.Entity, error)
}

// EntityFinder finds the entity described by the tag.
type EntityFinder interface {
	FindEntity(tag names.Tag) (state.Entity, error)
}
