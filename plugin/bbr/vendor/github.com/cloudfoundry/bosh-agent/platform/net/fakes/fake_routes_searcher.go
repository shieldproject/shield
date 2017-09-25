package fakes

import (
	boshnet "github.com/cloudfoundry/bosh-agent/platform/net"
)

type FakeRoutesSearcher struct {
	SearchRoutesRoutes []boshnet.Route
	SearchRoutesErr    error
}

func (s *FakeRoutesSearcher) SearchRoutes() ([]boshnet.Route, error) {
	return s.SearchRoutesRoutes, s.SearchRoutesErr
}
