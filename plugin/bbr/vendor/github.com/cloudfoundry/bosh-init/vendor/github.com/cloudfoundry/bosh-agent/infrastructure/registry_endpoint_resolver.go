package infrastructure

import (
	"fmt"
	"net/url"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type registryEndpointResolver struct {
	delegate DNSResolver
}

func NewRegistryEndpointResolver(resolver DNSResolver) DNSResolver {
	return registryEndpointResolver{
		delegate: resolver,
	}
}

func (r registryEndpointResolver) LookupHost(dnsServers []string, endpoint string) (string, error) {
	registryURL, err := url.Parse(endpoint)
	if err != nil {
		return "", bosherr.WrapError(err, "Parsing registry named endpoint")
	}

	registryHostAndPort := strings.Split(registryURL.Host, ":")
	registryIP, err := r.delegate.LookupHost(dnsServers, registryHostAndPort[0])
	if err != nil {
		return "", bosherr.WrapError(err, "Looking up registry")
	}

	if len(registryHostAndPort) == 2 {
		registryURL.Host = fmt.Sprintf("%s:%s", registryIP, registryHostAndPort[1])
	} else {
		registryURL.Host = registryIP
	}

	return registryURL.String(), nil
}
