// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package watcher

import (
	"gopkg.in/mgo.v2"

	"github.com/juju/1.25-upgrade/juju2/mongo"
)

func NewTestWatcher(changelog *mgo.Collection, iteratorFunc func() mongo.Iterator) *Watcher {
	return newWatcher(changelog, iteratorFunc)
}
