// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package resourceadapters

import (
	"github.com/juju/errors"
	"gopkg.in/juju/names.v2"

	csclient "github.com/juju/1.25-upgrade/juju2/charmstore"
	"github.com/juju/1.25-upgrade/juju2/resource"
	"github.com/juju/1.25-upgrade/juju2/resource/charmstore"
	corestate "github.com/juju/1.25-upgrade/juju2/state"
)

// resourceOpener is an implementation of server.ResourceOpener.
type resourceOpener struct {
	st     *corestate.State
	res    corestate.Resources
	userID names.Tag
	unit   *corestate.Unit
	closer func() error
}

// OpenResource implements server.ResourceOpener.
func (ro *resourceOpener) OpenResource(name string) (o resource.Opened, err error) {
	defer func() {
		if err != nil {
			ro.closer()
		}
	}()

	if ro.unit == nil {
		return resource.Opened{}, errors.Errorf("missing unit")
	}
	svc, err := ro.unit.Application()
	if err != nil {
		return resource.Opened{}, errors.Trace(err)
	}
	cURL, _ := ro.unit.CharmURL()
	id := csclient.CharmID{
		URL:     cURL,
		Channel: svc.Channel(),
	}

	csOpener := newCharmstoreOpener(ro.st)
	client, err := csOpener.NewClient()
	if err != nil {
		return resource.Opened{}, errors.Trace(err)
	}

	cache := &charmstoreEntityCache{
		st:            ro.res,
		userID:        ro.userID,
		unit:          ro.unit,
		applicationID: ro.unit.ApplicationName(),
	}

	res, reader, err := charmstore.GetResource(charmstore.GetResourceArgs{
		Client:  client,
		Cache:   cache,
		CharmID: id,
		Name:    name,
	})
	if err != nil {
		return resource.Opened{}, errors.Trace(err)
	}

	opened := resource.Opened{
		Resource:   res,
		ReadCloser: reader,
		Closer:     ro.closer,
	}
	return opened, nil
}
