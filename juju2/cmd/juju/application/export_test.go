// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package application

import (
	"github.com/juju/cmd"
	"gopkg.in/juju/charmrepo.v2-unstable/csclient"
	"gopkg.in/macaroon-bakery.v1/httpbakery"

	"github.com/juju/1.25-upgrade/juju2/api"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
	"github.com/juju/1.25-upgrade/juju2/resource/resourceadapters"
)

func NewUpgradeCharmCommandForTest(
	store jujuclient.ClientStore,
	apiOpen api.OpenFunc,
	deployResources resourceadapters.DeployResourcesFunc,
	resolveCharm ResolveCharmFunc,
	newCharmAdder NewCharmAdderFunc,
	newCharmClient func(api.Connection) CharmClient,
	newCharmUpgradeClient func(api.Connection) CharmUpgradeClient,
	newModelConfigGetter func(api.Connection) ModelConfigGetter,
	newResourceLister func(api.Connection) (ResourceLister, error),
) cmd.Command {
	cmd := &upgradeCharmCommand{
		DeployResources:       deployResources,
		ResolveCharm:          resolveCharm,
		NewCharmAdder:         newCharmAdder,
		NewCharmClient:        newCharmClient,
		NewCharmUpgradeClient: newCharmUpgradeClient,
		NewModelConfigGetter:  newModelConfigGetter,
		NewResourceLister:     newResourceLister,
	}
	cmd.SetClientStore(store)
	cmd.SetAPIOpen(apiOpen)
	return modelcmd.Wrap(cmd)
}

// NewAddUnitCommandForTest returns an AddUnitCommand with the api provided as specified.
func NewAddUnitCommandForTest(api serviceAddUnitAPI) cmd.Command {
	return modelcmd.Wrap(&addUnitCommand{
		api: api,
	})
}

// NewAddRelationCommandForTest returns an AddRelationCommand with the api provided as specified.
func NewAddRelationCommandForTest(api ApplicationAddRelationAPI) modelcmd.ModelCommand {
	cmd := &addRelationCommand{newAPIFunc: func() (ApplicationAddRelationAPI, error) {
		return api, nil
	}}
	return modelcmd.Wrap(cmd)
}

// NewRemoveRelationCommandForTest returns an RemoveRelationCommand with the api provided as specified.
func NewRemoveRelationCommandForTest(api ApplicationDestroyRelationAPI) modelcmd.ModelCommand {
	cmd := &removeRelationCommand{newAPIFunc: func() (ApplicationDestroyRelationAPI, error) {
		return api, nil
	}}
	return modelcmd.Wrap(cmd)
}

// NewConsumeCommandForTest returns a ConsumeCommand with the specified api.
func NewConsumeCommandForTest(store jujuclient.ClientStore, api applicationConsumeAPI) cmd.Command {
	c := &consumeCommand{api: api}
	c.SetClientStore(store)
	return modelcmd.Wrap(c)
}

type Patcher interface {
	PatchValue(dest, value interface{})
}

func PatchNewCharmStoreClient(s Patcher, url string) {
	s.PatchValue(&newCharmStoreClient, func(bakeryClient *httpbakery.Client) *csclient.Client {
		return csclient.New(csclient.Params{
			URL:          url,
			BakeryClient: bakeryClient,
		})
	})
}
