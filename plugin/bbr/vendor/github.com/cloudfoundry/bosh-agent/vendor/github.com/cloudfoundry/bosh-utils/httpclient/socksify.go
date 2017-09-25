package httpclient

import (
	"net"
	"net/url"
	"os"

	goproxy "golang.org/x/net/proxy"
)

type DialFunc func(network, address string) (net.Conn, error)

func (f DialFunc) Dial(network, address string) (net.Conn, error) { return f(network, address) }

func SOCKS5DialFuncFromEnvironment(origDialer DialFunc) DialFunc {
	allProxy := os.Getenv("BOSH_ALL_PROXY")
	if len(allProxy) == 0 {
		return origDialer
	}

	proxyURL, err := url.Parse(allProxy)
	if err != nil {
		return origDialer
	}

	proxy, err := goproxy.FromURL(proxyURL, origDialer)
	if err != nil {
		return origDialer
	}

	noProxy := os.Getenv("no_proxy")
	if len(noProxy) == 0 {
		return proxy.Dial
	}

	perHost := goproxy.NewPerHost(proxy, origDialer)
	perHost.AddFromString(noProxy)

	return perHost.Dial
}
