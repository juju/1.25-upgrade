// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"net"

	"github.com/juju/description"
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju2/apiserver/common/networkingcommon"
	"github.com/juju/1.25-upgrade/juju2/cloud"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/environs/config"
	"github.com/juju/1.25-upgrade/juju2/instance"
)

func exportModel(st *state.State) (description.Model, error) {
	model, err := st.Export()
	if err != nil {
		return nil, errors.Annotate(err, "exporting model representation")
	}

	if envCfg, err := st.EnvironConfig(); err != nil {
		return nil, errors.Trace(err)
	} else if envCfg.Type() == "maas" {
		// Juju 1.25 doesn't have complete link-layer device definitions
		// (it has network interfaces, but they lack some of the details)
		// or IP addresses. Query MAAS for those using the Juju 2.x code,
		// and fill in the blanks.
		if err := addMAASNetworkEntities(model, st); err != nil {
			return nil, errors.Annotate(err, "adding MAAS network entities")
		}
	}
	return model, nil
}

// addMAASNetworkEntities adds link-layer devices and IP addresses to
// the model description. These entities are not modeled by Juju 1.25,
// so we take the model from 1.25 and augment it by using the Juju 2.x
// MAAS provider code.
func addMAASNetworkEntities(model description.Model, st *state.State) error {
	envCfg, err := st.EnvironConfig()
	if err != nil {
		return errors.Trace(err)
	}
	attrs := envCfg.AllAttrs()
	cred := cloud.NewCredential(cloud.OAuth1AuthType, map[string]string{
		"maas-oauth": attrs["maas-oauth"].(string),
	})
	cloudSpec := environs.CloudSpec{
		Type:       "maas",
		Name:       model.Cloud(),
		Region:     model.CloudRegion(),
		Endpoint:   attrs["maas-server"].(string),
		Credential: &cred,
	}

	modelCfg, err := config.New(config.NoDefaults, model.Config())
	if err != nil {
		return errors.Trace(err)
	}
	env, err := environs.New(environs.OpenParams{cloudSpec, modelCfg})
	if err != nil {
		return errors.Trace(err)
	}
	netenv := env.(environs.Networking)

	// Add link-layer devices and IP addresses.
	for _, machine := range model.Machines() {
		inst := machine.Instance()
		if inst == nil {
			continue
		}
		instanceId := inst.InstanceId()
		if instanceId == "" {
			continue
		}
		interfaces, err := netenv.NetworkInterfaces(instance.Id(instanceId))
		if err != nil {
			return errors.Annotatef(err, "getting network interfaces for %q", instanceId)
		}

		networkConfig := networkingcommon.NetworkConfigFromInterfaceInfo(interfaces)
		devicesArgs, devicesAddrs := networkingcommon.NetworkConfigsToStateArgs(networkConfig)
		for _, d := range devicesArgs {
			model.AddLinkLayerDevice(description.LinkLayerDeviceArgs{
				Name:        d.Name,
				MTU:         d.MTU,
				ProviderID:  string(d.ProviderID),
				MachineID:   machine.Id(),
				Type:        string(d.Type),
				MACAddress:  d.MACAddress,
				IsAutoStart: d.IsAutoStart,
				IsUp:        d.IsUp,
				ParentName:  d.ParentName,
			})
		}
		for _, d := range devicesAddrs {
			ip, ipNet, err := net.ParseCIDR(d.CIDRAddress)
			if err != nil {
				return errors.Trace(err)
			}
			model.AddIPAddress(description.IPAddressArgs{
				ProviderID:       string(d.ProviderID),
				DeviceName:       d.DeviceName,
				MachineID:        machine.Id(),
				SubnetCIDR:       ipNet.String(),
				ConfigMethod:     string(d.ConfigMethod),
				Value:            ip.String(),
				DNSServers:       d.DNSServers,
				DNSSearchDomains: d.DNSSearchDomains,
				GatewayAddress:   d.GatewayAddress,
			})
		}
	}

	return nil
}
