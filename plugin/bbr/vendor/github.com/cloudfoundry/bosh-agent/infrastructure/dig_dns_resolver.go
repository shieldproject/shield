package infrastructure

import (
	"errors"
	"fmt"
	"net"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const digDNSResolverLogTag = "Dig DNS Resolver"

type DigDNSResolver struct {
	runner boshsys.CmdRunner
	logger boshlog.Logger
}

func NewDigDNSResolver(runner boshsys.CmdRunner, logger boshlog.Logger) DigDNSResolver {
	return DigDNSResolver{
		runner: runner,
		logger: logger,
	}
}

func (res DigDNSResolver) LookupHost(dnsServers []string, host string) (string, error) {
	if host == "localhost" {
		return "127.0.0.1", nil
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return host, nil
	}

	var err error
	var ipString string

	if len(dnsServers) == 0 {
		err = errors.New("No DNS servers provided")
	}

	for _, dnsServer := range dnsServers {
		ipString, err = res.lookupHostWithDNSServer(dnsServer, host)
		if err == nil {
			return ipString, nil
		}
	}

	return "", err
}

func (res DigDNSResolver) lookupHostWithDNSServer(dnsServer string, host string) (ipString string, err error) {
	stdout, _, _, err := res.runner.RunCommand(
		"dig",
		fmt.Sprintf("@%s", dnsServer),
		host,
		"+short",
		"+time=1",
	)

	if err != nil {
		return "", bosherr.WrapError(err, "Shelling out to dig")
	}

	ipString = strings.Split(stdout, "\n")[0]
	ip := net.ParseIP(ipString)
	if ip == nil {
		return "", errors.New("Resolving host")
	}

	return ipString, nil
}
