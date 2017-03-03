// Copyright 2014 Canonical Ltd. All rights reserved.
// Licensed under the AGPLv3, see LICENCE file for details.

package authentication

import (
	"github.com/juju/1.25-upgrade/juju1/apiserver/common"
	"github.com/juju/1.25-upgrade/juju1/state"
)

// FindEntityAuthenticator looks up the authenticator for the entity identified tag.
func FindEntityAuthenticator(entity state.Entity) (EntityAuthenticator, error) {
	switch entity.(type) {
	case *state.Machine, *state.Unit:
		return &AgentAuthenticator{}, nil
	case *state.User:
		return &UserAuthenticator{}, nil
	}

	return nil, common.ErrBadRequest
}
