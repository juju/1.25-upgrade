// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package all

// Register all the available providers.
import (
	_ "github.com/juju/1.25-upgrade/juju2/provider/lxd"
)
