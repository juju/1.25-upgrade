// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package uniter_test

import (
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/worker/uniter/hook"
	"github.com/juju/1.25-upgrade/juju2/worker/uniter/operation"
	"github.com/juju/1.25-upgrade/juju2/worker/uniter/relation"
	"github.com/juju/1.25-upgrade/juju2/worker/uniter/remotestate"
	"github.com/juju/1.25-upgrade/juju2/worker/uniter/resolver"
	"github.com/juju/1.25-upgrade/juju2/worker/uniter/storage"
)

type dummyRelations struct {
	relation.Relations
}

func (*dummyRelations) NextHook(_ resolver.LocalState, _ remotestate.Snapshot) (hook.Info, error) {
	return hook.Info{}, resolver.ErrNoOperation
}

type dummyStorageAccessor struct {
	storage.StorageAccessor
}

func (*dummyStorageAccessor) UnitStorageAttachments(_ names.UnitTag) ([]params.StorageAttachmentId, error) {
	return nil, nil
}

type nopResolver struct{}

func (nopResolver) NextOp(resolver.LocalState, remotestate.Snapshot, operation.Factory) (operation.Operation, error) {
	return nil, resolver.ErrNoOperation
}
