// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package uniter

import (
	"github.com/juju/1.25-upgrade/juju2/apiserver/common"
	"github.com/juju/1.25-upgrade/juju2/apiserver/facade"
	"github.com/juju/1.25-upgrade/juju2/apiserver/meterstatus"
)

var (
	GetZone = &getZone

	_ meterstatus.MeterStatus = (*UniterAPI)(nil)
)

type StorageStateInterface storageStateInterface

func NewStorageAPI(
	st StorageStateInterface,
	resources facade.Resources,
	accessUnit common.GetAuthFunc,
) (*StorageAPI, error) {
	return newStorageAPI(storageStateInterface(st), resources, accessUnit)
}
