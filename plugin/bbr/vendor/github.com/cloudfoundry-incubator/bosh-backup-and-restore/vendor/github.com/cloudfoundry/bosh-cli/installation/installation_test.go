package installation_test

import (
	"errors"
	"log"

	. "github.com/cloudfoundry/bosh-cli/installation"

	biinstallmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	mock_registry "github.com/cloudfoundry/bosh-cli/registry/mocks"
	test_support_mocks "github.com/cloudfoundry/bosh-cli/test_support/mocks"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Installation", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		manifest                  biinstallmanifest.Manifest
		mockRegistryServerManager *mock_registry.MockServerManager
		mockRegistryServer        *mock_registry.MockServer

		target       Target
		installedJob InstalledJob
	)

	var newInstalation = func() Installation {
		return NewInstallation(target, installedJob, manifest, mockRegistryServerManager)
	}

	BeforeEach(func() {
		manifest = biinstallmanifest.Manifest{}

		mockRegistryServerManager = mock_registry.NewMockServerManager(mockCtrl)
		mockRegistryServer = mock_registry.NewMockServer(mockCtrl)

		target = NewTarget("fake-installation-path")

		installedJob = NewInstalledJob(RenderedJobRef{Name: "cpi"}, "fake-job-path")
	})

	Describe("WithRunningRegistry", func() {
		var (
			startCall, spyCall, stopCall *gomock.Call
			spy                          *test_support_mocks.MockSpy
			logBuffer                    *gbytes.Buffer
			logger                       boshlog.Logger
			fakeStage                    *fakebiui.FakeStage
		)

		BeforeEach(func() {
			manifest.Registry = biinstallmanifest.Registry{
				Username: "fake-username",
				Password: "fake-password",
				Host:     "fake-host",
				Port:     123,
			}

			spy = test_support_mocks.NewMockSpy(mockCtrl)
			startCall = mockRegistryServerManager.EXPECT().Start("fake-username", "fake-password", "fake-host", 123).Return(mockRegistryServer, nil)
			spyCall = spy.EXPECT().Record()
			stopCall = mockRegistryServer.EXPECT().Stop().Return(nil)

			logBuffer = gbytes.NewBuffer()
			goLogger := log.New(logBuffer, "", log.LstdFlags)
			logger = boshlog.New(boshlog.LevelWarn, goLogger, goLogger)
			fakeStage = fakebiui.NewFakeStage()
		})

		It("starts the registry before calling the function", func() {
			spyCall.After(startCall)

			err := newInstalation().WithRunningRegistry(logger, fakeStage, func() error {
				spy.Record()
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("performs stages for starting and stopping the registry", func() {
			err := newInstalation().WithRunningRegistry(logger, fakeStage, func() error {
				spy.Record()
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStage.PerformCalls).To(ContainElement(&fakebiui.PerformCall{
				Name: "Starting registry",
			}))
			Expect(fakeStage.PerformCalls).To(ContainElement(&fakebiui.PerformCall{
				Name: "Stopping registry",
			}))
		})

		Context("the function succeeds", func() {
			It("stops the registry and returns nil", func() {
				stopCall.After(spyCall)

				err := newInstalation().WithRunningRegistry(logger, fakeStage, func() error {
					spy.Record()
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("the function fails", func() {
			It("stops the registry and returns the error", func() {
				stopCall.After(spyCall)
				err := newInstalation().WithRunningRegistry(logger, fakeStage, func() error {
					spy.Record()
					return errors.New("blarg!")
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when starting registry fails", func() {
			It("returns an error and doesn't call the function", func() {
				startCall.Return(mockRegistryServer, errors.New("registry-start-error"))
				spyCall.Times(0)
				stopCall.Times(0)

				err := newInstalation().WithRunningRegistry(logger, fakeStage, func() error {
					spy.Record()
					return nil
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Starting registry: registry-start-error"))
			})
		})

		Context("when stopping registry fails", func() {
			Context("when the function fails", func() {
				It("logs a warning and returns the function error", func() {
					stopCall.Return(errors.New("registry-stop-error"))

					err := newInstalation().WithRunningRegistry(logger, fakeStage, func() error {
						spy.Record()
						return errors.New("blarg!")
					})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("blarg!"))

					Expect(logBuffer).To(gbytes.Say("registry-stop-error"))
				})
			})

			Context("when the function succeeds", func() {
				It("logs a warning and returns nil", func() {
					err := newInstalation().WithRunningRegistry(logger, fakeStage, func() error {
						spy.Record()
						return nil
					})
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})

	Describe("StartRegistry", func() {
		Context("when registry config is not empty", func() {
			BeforeEach(func() {
				manifest.Registry = biinstallmanifest.Registry{
					Username: "fake-username",
					Password: "fake-password",
					Host:     "fake-host",
					Port:     123,
				}
			})

			It("starts the registry", func() {
				mockRegistryServerManager.EXPECT().Start("fake-username", "fake-password", "fake-host", 123).Return(mockRegistryServer, nil)

				err := newInstalation().StartRegistry()
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when starting registry fails", func() {
				BeforeEach(func() {
					mockRegistryServerManager.EXPECT().Start("fake-username", "fake-password", "fake-host", 123).Return(nil, errors.New("fake-registry-start-error"))
				})

				It("returns an error", func() {
					err := newInstalation().StartRegistry()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-registry-start-error"))
				})
			})
		})

		Context("when registry config is empty", func() {
			BeforeEach(func() {
				manifest.Registry = biinstallmanifest.Registry{}
			})

			It("does not start the registry", func() {
				err := newInstalation().StartRegistry()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("StopRegistry", func() {
		Context("when registry has been started", func() {
			var installation Installation

			BeforeEach(func() {
				manifest.Registry = biinstallmanifest.Registry{
					Username: "fake-username",
					Password: "fake-password",
					Host:     "fake-host",
					Port:     123,
				}

				installation = newInstalation()

				mockRegistryServerManager.EXPECT().Start("fake-username", "fake-password", "fake-host", 123).Return(mockRegistryServer, nil)
				err := installation.StartRegistry()
				Expect(err).ToNot(HaveOccurred())
			})

			It("stops the registry", func() {
				mockRegistryServer.EXPECT().Stop()

				err := installation.StopRegistry()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when registry is configured but has not been started", func() {
			BeforeEach(func() {
				manifest.Registry = biinstallmanifest.Registry{
					Username: "fake-username",
					Password: "fake-password",
					Host:     "fake-host",
					Port:     123,
				}
			})

			It("returns an error", func() {
				err := newInstalation().StopRegistry()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Registry must be started before it can be stopped"))
			})
		})

		Context("when registry config is empty", func() {
			BeforeEach(func() {
				manifest.Registry = biinstallmanifest.Registry{}
			})

			It("does not stop the registry", func() {
				err := newInstalation().StopRegistry()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
