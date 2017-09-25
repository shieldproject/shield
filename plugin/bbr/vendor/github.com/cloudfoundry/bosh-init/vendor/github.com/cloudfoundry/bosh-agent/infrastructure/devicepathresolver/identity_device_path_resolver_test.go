package devicepathresolver_test

import (
	. "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IdentityDevicePathResolver", func() {
	var (
		identityDevicePathResolver DevicePathResolver
	)

	BeforeEach(func() {
		identityDevicePathResolver = NewIdentityDevicePathResolver()
	})

	Context("when path is not provided", func() {
		It("returns an error", func() {
			diskSettings := boshsettings.DiskSettings{}
			_, _, err := identityDevicePathResolver.GetRealDevicePath(diskSettings)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path is missing"))
		})
	})
})
