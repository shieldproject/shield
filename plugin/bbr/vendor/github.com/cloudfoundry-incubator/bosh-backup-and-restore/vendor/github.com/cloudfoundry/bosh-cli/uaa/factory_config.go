package uaa

import (
	gonet "net"
	gourl "net/url"
	"strconv"
	"strings"

	"github.com/cloudfoundry/bosh-cli/common/net"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type Config struct {
	net.ClientFactoryConfig

	Path string
}

func NewConfigFromURL(url string) (Config, error) {
	if len(url) == 0 {
		return Config{}, bosherr.Error("Expected non-empty UAA URL")
	}

	parsedURL, err := gourl.Parse(url)
	if err != nil {
		return Config{}, bosherr.WrapErrorf(err, "Parsing UAA URL '%s'", url)
	}

	host := parsedURL.Host
	port := 443
	path := parsedURL.Path

	if len(host) == 0 {
		host = url
		path = ""
	}

	if strings.Contains(host, ":") {
		var portStr string

		host, portStr, err = gonet.SplitHostPort(host)
		if err != nil {
			return Config{}, bosherr.WrapErrorf(
				err, "Extracting host/port from URL '%s'", url)
		}

		port, err = strconv.Atoi(portStr)
		if err != nil {
			return Config{}, bosherr.WrapErrorf(
				err, "Extracting port from URL '%s'", url)
		}
	}

	if len(host) == 0 {
		return Config{}, bosherr.Errorf("Expected to extract host from URL '%s'", url)
	}

	return Config{
		ClientFactoryConfig: net.ClientFactoryConfig{
			Host: host,
			Port: port,
		},
		Path: path,
	}, nil
}
