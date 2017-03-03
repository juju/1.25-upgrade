// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"os"

	"github.com/juju/1.25-upgrade/juju1/cmd/juju/commands"
	components "github.com/juju/1.25-upgrade/juju1/component/all"
	// Import the providers.
	_ "github.com/juju/1.25-upgrade/juju1/provider/all"
	"github.com/juju/1.25-upgrade/juju1/utils"
)

func init() {
	utils.Must(components.RegisterForClient())
}

func main() {
	commands.Main(os.Args)
}
