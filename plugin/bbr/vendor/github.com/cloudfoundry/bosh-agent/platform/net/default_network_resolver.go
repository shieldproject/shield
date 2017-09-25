package net

import (
	gonet "net"

	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type defaultNetworkResolver struct {
	routesSearcher RoutesSearcher
	ipResolver     boship.Resolver
}

func NewDefaultNetworkResolver(
	routesSearcher RoutesSearcher,
	ipResolver boship.Resolver,
) boshsettings.DefaultNetworkResolver {
	return defaultNetworkResolver{
		routesSearcher: routesSearcher,
		ipResolver:     ipResolver,
	}
}

func (r defaultNetworkResolver) GetDefaultNetwork() (boshsettings.Network, error) {
	network := boshsettings.Network{}

	routes, err := r.routesSearcher.SearchRoutes()
	if err != nil {
		return network, bosherr.WrapError(err, "Searching routes")
	}

	if len(routes) == 0 {
		return network, bosherr.Error("No routes found")
	}

	for _, route := range routes {
		if !route.IsDefault() {
			continue
		}

		ip, err := r.ipResolver.GetPrimaryIPv4(route.InterfaceName)
		if err != nil {
			return network, bosherr.WrapErrorf(err, "Getting primary IPv4 for interface '%s'", route.InterfaceName)
		}

		return boshsettings.Network{
			IP:      ip.IP.String(),
			Netmask: gonet.IP(ip.Mask).String(),
			Gateway: route.Gateway,
		}, nil

	}

	return network, bosherr.Error("Failed to find default route")
}
