// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package discoverspaces_test

import (
	"github.com/juju/utils/set"

	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/worker"
	"github.com/juju/1.25-upgrade/juju2/worker/discoverspaces"
	"github.com/juju/1.25-upgrade/juju2/worker/gate"
)

type fakeWorker struct {
	worker.Worker
}

type fakeAPICaller struct {
	base.APICaller
}

type fakeFacade struct {
	discoverspaces.Facade
}

type fakeEnviron struct {
	environs.NetworkingEnviron
}

func fakeNewName(_ string, _ set.Strings) string {
	panic("fake")
}

type fakeUnlocker struct {
	gate.Unlocker
}
