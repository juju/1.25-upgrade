// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package applicationoffers_test

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/apiserver/applicationoffers"
	"github.com/juju/1.25-upgrade/juju2/apiserver/common"
	"github.com/juju/1.25-upgrade/juju2/apiserver/testing"
	jujucrossmodel "github.com/juju/1.25-upgrade/juju2/core/crossmodel"
	"github.com/juju/1.25-upgrade/juju2/permission"
	coretesting "github.com/juju/1.25-upgrade/juju2/testing"
)

const (
	addOffersBackendCall  = "addOffersCall"
	listOffersBackendCall = "listOffersCall"
)

type baseSuite struct {
	resources  *common.Resources
	authorizer *testing.FakeAuthorizer

	mockState         *mockState
	mockStatePool     *mockStatePool
	applicationOffers *mockApplicationOffers
}

func (s *baseSuite) SetUpTest(c *gc.C) {
	s.resources = common.NewResources()
	s.authorizer = &testing.FakeAuthorizer{
		Tag:      names.NewUserTag("read"),
		AdminTag: names.NewUserTag("admin"),
	}

	s.mockState = &mockState{
		modelUUID:         coretesting.ModelTag.Id(),
		users:             set.NewStrings(),
		applicationOffers: make(map[string]jujucrossmodel.ApplicationOffer),
		accessPerms:       make(map[offerAccess]permission.Access),
	}
	s.mockStatePool = &mockStatePool{map[string]applicationoffers.Backend{s.mockState.modelUUID: s.mockState}}
}

func (s *baseSuite) addApplication(c *gc.C, name string) jujucrossmodel.ApplicationOffer {
	return jujucrossmodel.ApplicationOffer{
		OfferName:              "offer-" + name,
		ApplicationName:        name,
		Endpoints:              map[string]charm.Relation{"db": {Name: "db"}},
		ApplicationDescription: "applicaion description",
	}
}

func (s *baseSuite) setupOffers(c *gc.C, filterAppName string) {
	applicationName := "test"
	offerName := "hosted-db2"

	anOffer := jujucrossmodel.ApplicationOffer{
		OfferName:              offerName,
		ApplicationName:        applicationName,
		ApplicationDescription: "description",
		Endpoints:              map[string]charm.Relation{"db": {Name: "db2"}},
	}

	s.applicationOffers.listOffers = func(filters ...jujucrossmodel.ApplicationOfferFilter) ([]jujucrossmodel.ApplicationOffer, error) {
		c.Assert(filters, gc.HasLen, 1)
		c.Assert(filters[0], jc.DeepEquals, jujucrossmodel.ApplicationOfferFilter{
			OfferName:       offerName,
			ApplicationName: filterAppName,
		})
		return []jujucrossmodel.ApplicationOffer{anOffer}, nil
	}
	ch := &mockCharm{meta: &charm.Meta{Description: "A pretty popular blog engine"}}
	s.mockState.applications = map[string]applicationoffers.Application{
		"test": &mockApplication{charm: ch, curl: charm.MustParseURL("db2-2")},
	}
	s.mockState.model = &mockModel{uuid: coretesting.ModelTag.Id(), name: "prod", owner: "fred"}
	s.mockState.connStatus = &mockConnectionStatus{count: 5}
}
