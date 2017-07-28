// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package gui

import (
	"github.com/juju/cmd"

	"github.com/juju/1.25-upgrade/juju2/api"
	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
)

var (
	ClientGet      = &clientGet
	WebbrowserOpen = &webbrowserOpen

	ClientGUIArchives      = &clientGUIArchives
	ClientSelectGUIVersion = &clientSelectGUIVersion
	ClientUploadGUIArchive = &clientUploadGUIArchive
	GUIFetchMetadata       = &guiFetchMetadata
)

func NewGUICommandForTest(getGUIVersions func(connection api.Connection) ([]params.GUIArchiveVersion, error)) cmd.Command {
	return modelcmd.Wrap(&guiCommand{
		getGUIVersions: getGUIVersions,
	})
}
