package net

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type DNSValidator interface {
	Validate([]string) error
}

type dnsValidator struct {
	fs boshsys.FileSystem
}

func NewDNSValidator(fs boshsys.FileSystem) DNSValidator {
	return &dnsValidator{
		fs: fs,
	}
}

func (d *dnsValidator) Validate(dnsServers []string) error {
	if len(dnsServers) == 0 {
		return nil
	}

	resolvConfContents, err := d.fs.ReadFileString("/etc/resolv.conf")
	if err != nil {
		return bosherr.WrapError(err, "Reading /etc/resolv.conf")
	}

	for _, dnsServer := range dnsServers {
		if strings.Contains(resolvConfContents, dnsServer) {
			return nil
		}
	}

	return bosherr.WrapError(err, "None of the DNS servers that were specified in the manifest were found in /etc/resolv.conf.")
}
