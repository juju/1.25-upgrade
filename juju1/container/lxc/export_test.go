// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package lxc

import (
	"github.com/juju/1.25-upgrade/juju1/container"
)

var (
	ContainerConfigFilename = containerConfigFilename
	ContainerDirFilesystem  = containerDirFilesystem
	GenerateNetworkConfig   = generateNetworkConfig
	ParseConfigLine         = parseConfigLine
	UpdateContainerConfig   = updateContainerConfig
	ReorderNetworkConfig    = reorderNetworkConfig
	NetworkConfigTemplate   = networkConfigTemplate
	RestartSymlink          = restartSymlink
	ReleaseVersion          = &releaseVersion
	PreferFastLXC           = preferFastLXC
	WriteWgetTmpFile        = &writeWgetTmpFile
)

func GetCreateWithCloneValue(mgr container.Manager) bool {
	return mgr.(*containerManager).createWithClone
}

func WgetEnvironment(caCert []byte) ([]string, func(), error) {
	return wgetEnvironment(caCert)
}
