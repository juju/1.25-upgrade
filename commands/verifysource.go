// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"net"
	"strings"

	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju2/apiserver/common/networkingcommon"
	"github.com/juju/1.25-upgrade/juju2/cloud"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/environs/config"
	"github.com/juju/1.25-upgrade/juju2/instance"
	_ "github.com/juju/1.25-upgrade/juju2/provider/maas"
	"github.com/juju/cmd"
	"github.com/juju/description"
	"github.com/juju/errors"
	"golang.org/x/sync/errgroup"
)

var verifySourceDoc = `
The purpose of the verify-source command is to check connectivity, status, and
viability of a 1.25 juju environment for migration into a Juju 2.x controller.

`

func newVerifySourceCommand() cmd.Command {
	command := &verifySourceCommand{}
	command.remoteCommand = "verify-source-impl"
	return wrap(command)
}

type verifySourceCommand struct {
	baseClientCommand
}

func (c *verifySourceCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "verify-source",
		Args:    "<environment name>",
		Purpose: "check a 1.25 environment for migration suitability",
		Doc:     verifySourceDoc,
	}
}

func (c *verifySourceCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var verifySourceImplDoc = `

verify-source-impl must be executed on an API server machine of a 1.25
environment.

The command will check the export of the environment into the 2.0 model
format.

`

func newVerifySourceImplCommand() cmd.Command {
	return &verifySourceImplCommand{}
}

type verifySourceImplCommand struct {
	baseRemoteCommand
}

func (c *verifySourceImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "verify-source-impl",
		Purpose: "check the database export for migration suitability",
		Doc:     verifySourceImplDoc,
	}
}

func (c *verifySourceImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	// Check that the LXC containers can be migrated to LXD.
	opts := MigrateLXCOptions{DryRun: true}
	byHost, err := getLXCContainersFromState(st)
	if err != nil {
		return errors.Trace(err)
	}
	var group errgroup.Group
	for host, containers := range byHost {
		containerNames := make([]string, len(containers))
		for i, container := range containers {
			containerNames[i] = container.Id()
		}
		logger.Debugf("dry-running LXC migration for %s", strings.Join(containerNames, ", "))
		host, containers := host, containers // copy for closure
		group.Go(func() error {
			err := MigrateLXC(containers, host, opts)
			return errors.Annotatef(err, "dry-running LXC migration for host %q", host.Id())
		})
	}
	if err := group.Wait(); err != nil {
		return errors.Annotate(err, "dry-running LXC migration")
	}

	return errors.Annotate(writeModel(ctx, st), "exporting model")
}

func exportModel(st *state.State) ([]byte, error) {
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
	bytes, err := description.Serialize(model)
	if err != nil {
		return nil, errors.Annotate(err, "serializing model representation")
	}
	return bytes, nil
}

func writeModel(ctx *cmd.Context, st *state.State) error {
	bytes, err := exportModel(st)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = ctx.GetStdout().Write(bytes)
	if err != nil {
		return errors.Annotate(err, "writing model representation")
	}

	return nil
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
