package net_test

import (
	gonet "net"

	"github.com/cloudfoundry/bosh-agent/platform/net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Windows Route Searcher", func() {
	Describe("SeachRoutes", func() {
		It("returns all default routes with a gateway", func() {
			routeSearcher := net.NewRoutesSearcher(nil)
			routes, err := routeSearcher.SearchRoutes()
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).ToNot(HaveLen(0))

			for _, route := range routes {
				Expect(route.IsDefault()).To(BeTrue())
				gatewayIP := gonet.ParseIP(route.Gateway)
				Expect(gatewayIP.To4()).NotTo(BeNil())
				Expect(route.InterfaceName).NotTo(BeEmpty())
			}
		})
	})
})
