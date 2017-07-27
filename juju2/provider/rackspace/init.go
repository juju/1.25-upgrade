// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rackspace

import (
	"gopkg.in/goose.v2/client"
	"gopkg.in/goose.v2/identity"

	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/provider/openstack"
)

const (
	providerType = "rackspace"
)

func init() {
	osProvider := &openstack.EnvironProvider{
		ProviderCredentials: Credentials{},
		Configurator:        &rackspaceConfigurator{},
		FirewallerFactory:   &firewallerFactory{},
		FlavorFilter:        openstack.FlavorFilterFunc(acceptRackspaceFlavor),
		NetworkingDecorator: rackspaceNetworkingDecorator{},
		ClientFromEndpoint: func(endpoint string) client.AuthenticatingClient {
			return client.NewClient(&identity.Credentials{URL: endpoint}, 0, nil)
		},
	}
	providerInstance = &environProvider{
		osProvider,
	}
	environs.RegisterProvider(providerType, providerInstance)
}
