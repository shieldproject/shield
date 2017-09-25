// +build !windows

package net

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

// cmdRoutesSearcher uses `route -n` command to list routes
// which routes in a same format on Ubuntu and CentOS
type cmdRoutesSearcher struct {
	runner boshsys.CmdRunner
}

func NewRoutesSearcher(runner boshsys.CmdRunner) RoutesSearcher {
	return cmdRoutesSearcher{runner}
}

func (s cmdRoutesSearcher) SearchRoutes() ([]Route, error) {
	var routes []Route

	stdout, _, _, err := s.runner.RunCommand("route", "-n")
	if err != nil {
		return routes, bosherr.WrapError(err, "Running route")
	}

	for i, routeEntry := range strings.Split(stdout, "\n") {
		if i < 2 { // first two lines are informational
			continue
		}

		if routeEntry == "" {
			continue
		}

		routeFields := strings.Fields(routeEntry)

		routes = append(routes, Route{
			Destination:   routeFields[0],
			Gateway:       routeFields[1],
			InterfaceName: routeFields[7],
		})
	}

	return routes, nil
}
