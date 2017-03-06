// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"os"

	"github.com/juju/loggo"

	"github.com/juju/1.25-upgrade/juju2/cmd/juju/commands"
	components "github.com/juju/1.25-upgrade/juju2/component/all"
	// Import the providers.
	_ "github.com/juju/1.25-upgrade/juju2/provider/all"
)

var log = loggo.GetLogger("juju.cmd.juju")

func init() {
	if err := components.RegisterForClient(); err != nil {
		log.Criticalf("unable to register client components: %v", err)
		os.Exit(1)
	}
}

func main() {
	os.Exit(commands.Main(os.Args))
}
