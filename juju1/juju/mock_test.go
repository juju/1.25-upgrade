package juju_test

import (
	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/1.25-upgrade/juju1/api"
	"github.com/juju/1.25-upgrade/juju1/network"
)

type mockAPIState struct {
	api.Connection
	close func(api.Connection) error

	addr         string
	apiHostPorts [][]network.HostPort
	environTag   string
}

func (s *mockAPIState) Close() error {
	if s.close != nil {
		return s.close(s)
	}
	return nil
}

func (s *mockAPIState) Addr() string {
	return s.addr
}

func (s *mockAPIState) APIHostPorts() [][]network.HostPort {
	return s.apiHostPorts
}

func (s *mockAPIState) EnvironTag() (names.EnvironTag, error) {
	return names.ParseEnvironTag(s.environTag)
}

func (s *mockAPIState) ServerTag() (names.EnvironTag, error) {
	return names.EnvironTag{}, errors.NotImplementedf("ServerTag")
}

func panicAPIOpen(apiInfo *api.Info, opts api.DialOpts) (api.Connection, error) {
	panic("api.Open called unexpectedly")
}
