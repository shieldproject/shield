package applyspec_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshassert "github.com/cloudfoundry/bosh-utils/assert"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

func init() {
	Describe("concreteV1Service", func() {
		var (
			fs       *fakesys.FakeFileSystem
			specPath = "/spec.json"
			service  V1Service
		)

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()
			service = NewConcreteV1Service(fs, specPath)
		})

		Describe("Get", func() {
			Context("when filesystem has a spec file", func() {
				BeforeEach(func() {
					fs.WriteFileString(specPath, `{"deployment":"fake-deployment-name"}`)
				})

				It("reads spec from filesystem", func() {
					spec, err := service.Get()
					Expect(err).ToNot(HaveOccurred())
					Expect(spec).To(Equal(V1ApplySpec{Deployment: "fake-deployment-name"}))
				})

				It("returns error if reading spec from filesystem errs", func() {
					fs.ReadFileError = errors.New("fake-read-error")

					spec, err := service.Get()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-read-error"))
					Expect(spec).To(Equal(V1ApplySpec{}))
				})
			})

			Context("when filesystem does not have a spec file", func() {
				It("reads spec from filesystem", func() {
					spec, err := service.Get()
					Expect(err).ToNot(HaveOccurred())
					Expect(spec).To(Equal(V1ApplySpec{}))
				})
			})
		})

		Describe("Set", func() {
			newSpec := V1ApplySpec{Deployment: "fake-deployment-name"}

			It("writes spec to filesystem", func() {
				err := service.Set(newSpec)
				Expect(err).ToNot(HaveOccurred())

				specPathStats := fs.GetFileTestStat(specPath)
				Expect(specPathStats).ToNot(BeNil())
				boshassert.MatchesJSONBytes(GinkgoT(), newSpec, specPathStats.Content)
			})

			It("returns error if writing spec to filesystem errs", func() {
				fs.WriteFileError = errors.New("fake-write-error")

				err := service.Set(newSpec)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-write-error"))
			})
		})

		Describe("PopulateDHCPNetworks", func() {
			var settings boshsettings.Settings
			var unresolvedSpec V1ApplySpec
			var staticSpec NetworkSpec
			var dhcpSpec NetworkSpec
			var manualSetting boshsettings.Network
			var dynamicSetting boshsettings.Network

			BeforeEach(func() {
				settings = boshsettings.Settings{
					Networks: boshsettings.Networks{},
				}
				manualSetting = boshsettings.Network{
					Type:    "manual",
					IP:      "fake-manual-ip",
					Netmask: "fake-manual-netmask",
					Gateway: "fake-manual-gateway",
					Mac:     "fake-manual-mac",
				}
				dynamicSetting = boshsettings.Network{
					Type:    "dynamic",
					IP:      "fake-dynamic-ip",
					Netmask: "fake-dynamic-netmask",
					Gateway: "fake-dynamic-gateway",
				}

				unresolvedSpec = V1ApplySpec{
					Deployment:   "fake-deployment",
					NetworkSpecs: map[string]NetworkSpec{},
				}
				staticSpec = NetworkSpec{
					Fields: map[string]interface{}{
						"ip":      "fake-net1-ip",
						"netmask": "fake-net1-netmask",
						"gateway": "fake-net1-gateway",
						"mac":     "fake-net1-mac",
					},
				}
				dhcpSpec = NetworkSpec{
					Fields: map[string]interface{}{
						"type":    NetworkSpecTypeDynamic,
						"ip":      "fake-net2-ip",
						"netmask": "fake-net2-netmask",
						"gateway": "fake-net2-gateway",
					},
				}

			})

			Context("when associated network is in settings", func() {
				Context("when there are no networks configured with DHCP", func() {
					BeforeEach(func() {
						settings.Networks["fake-net"] = manualSetting

						unresolvedSpec.NetworkSpecs["fake-net"] = staticSpec
					})

					It("returns spec without modifying any networks", func() {
						spec, err := service.PopulateDHCPNetworks(unresolvedSpec, settings)
						Expect(err).ToNot(HaveOccurred())
						Expect(spec).To(Equal(unresolvedSpec))
					})
				})

				Context("when there is network with name 'local' and ip 127.0.0.1", func() {
					BeforeEach(func() {
						unresolvedSpec.NetworkSpecs["local"] = NetworkSpec{
							Fields: map[string]interface{}{"ip": "127.0.0.1"},
						}
					})

					It("returns spec without modifying any networks", func() {
						spec, err := service.PopulateDHCPNetworks(unresolvedSpec, settings)
						Expect(err).ToNot(HaveOccurred())
						Expect(spec).To(Equal(unresolvedSpec))
					})
				})

				Context("when there are networks configured with DHCP", func() {
					BeforeEach(func() {
						settings.Networks["static-net1"] = manualSetting
						settings.Networks["dhcp-net2"] = dynamicSetting

						unresolvedSpec.NetworkSpecs["static-net1"] = staticSpec
						unresolvedSpec.NetworkSpecs["dhcp-net2"] = dhcpSpec
					})

					It("returns spec with networks modified via DHCP and keeps everything else the same", func() {
						spec, err := service.PopulateDHCPNetworks(unresolvedSpec, settings)
						Expect(err).ToNot(HaveOccurred())
						Expect(spec).To(Equal(V1ApplySpec{
							Deployment: "fake-deployment",
							NetworkSpecs: map[string]NetworkSpec{
								"static-net1": staticSpec,
								"dhcp-net2": NetworkSpec{
									Fields: map[string]interface{}{
										"type":    NetworkSpecTypeDynamic,
										"ip":      dynamicSetting.IP,
										"netmask": dynamicSetting.Netmask,
										"gateway": dynamicSetting.Gateway,
									},
								},
							},
						}))
					})
				})
			})

			Context("when associated network cannot be found in settings", func() {
				BeforeEach(func() {
					settings.Networks["net-present-in-settings"] = manualSetting

					unresolvedSpec.NetworkSpecs["net-present-in-settings"] = staticSpec
					unresolvedSpec.NetworkSpecs["net-not-present-in-settings"] = dhcpSpec
				})

				It("returns error", func() {
					spec, err := service.PopulateDHCPNetworks(unresolvedSpec, settings)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Network 'net-not-present-in-settings' is not found in settings"))
					Expect(spec).To(Equal(V1ApplySpec{}))
				})
			})
		})
	})
}
