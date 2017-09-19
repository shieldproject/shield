package release_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-cli/cpi/release"
	biinstallation "github.com/cloudfoundry/bosh-cli/installation"
	biinstallationmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	"github.com/cloudfoundry/bosh-cli/installation/mocks"
	mock_install "github.com/cloudfoundry/bosh-cli/installation/mocks"
	"github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Installer", func() {
	Describe("WithInstalledCpiRelease", func() {
		var (
			mockCtrl             *gomock.Controller
			mockInstaller        *mocks.MockInstaller
			mockInstallerFactory *mock_install.MockInstallerFactory
			installationManifest biinstallationmanifest.Manifest
			installStage         *fakeui.FakeStage
			installation         *mocks.MockInstallation
			expectInstall        *gomock.Call
			expectCleanup        *gomock.Call
			target               biinstallation.Target
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		BeforeEach(func() {
			mockInstaller = mocks.NewMockInstaller(mockCtrl)
			mockInstallerFactory = mock_install.NewMockInstallerFactory(mockCtrl)

			installationManifest = biinstallationmanifest.Manifest{}
			installStage = fakeui.NewFakeStage()
			installation = mocks.NewMockInstallation(mockCtrl)

			target = biinstallation.NewTarget("fake-installation-path")
			mockInstallerFactory.EXPECT().NewInstaller(target).Return(mockInstaller).AnyTimes()
			expectInstall = mockInstaller.EXPECT().Install(installationManifest, gomock.Any())
			expectCleanup = mockInstaller.EXPECT().Cleanup(installation).Return(nil)
		})

		It("should install the CPI and call the passed in function with the installation", func() {
			cpiInstaller := release.CpiInstaller{
				InstallerFactory: mockInstallerFactory,
			}

			expectInstall.Return(installation, nil)

			var installationArgumentToFunction biinstallation.Installation
			err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
				installationArgumentToFunction = installation
				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(installationArgumentToFunction).ToNot(BeNil())
			Expect(installationArgumentToFunction).To(Equal(installation))

		})

		It("starts an 'installing CPI stage' and passes it to the installer", func() {
			cpiInstaller := release.CpiInstaller{
				InstallerFactory: mockInstallerFactory,
			}

			var stageForInstall ui.Stage
			expectInstall.Do(func(manifest biinstallationmanifest.Manifest, stage ui.Stage) {
				stageForInstall = stage
			}).Return(installation, nil)

			err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(stageForInstall).To(fakeui.BeASubstageOf(installStage))

			Expect(installStage.PerformCalls).To(ContainElement(
				&fakeui.PerformCall{
					Name:  "installing CPI",
					Stage: fakeui.NewFakeStage(),
				},
			))
		})

		It("cleans up the installation afterwards", func() {
			cpiInstaller := release.CpiInstaller{
				InstallerFactory: mockInstallerFactory,
			}

			cleanupCalled := false
			expectInstall.Return(installation, nil)
			expectCleanup.Times(1).Do(func(_ biinstallation.Installation) {
				cleanupCalled = true
			})
			err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
				Expect(cleanupCalled).To(BeFalse())
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("creates a stage for the cleanup", func() {
			cpiInstaller := release.CpiInstaller{
				InstallerFactory: mockInstallerFactory,
			}
			expectInstall.Return(installation, nil)

			err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(installStage.PerformCalls).To(ContainElement(
				&fakeui.PerformCall{
					Name: "Cleaning up rendered CPI jobs",
				},
			))

		})

		Context("when installing the cpi fails", func() {
			It("returns the error", func() {
				cpiInstaller := release.CpiInstaller{
					InstallerFactory: mockInstallerFactory,
				}

				expectInstall.Return(nil, errors.New("couldn't install that"))
				expectCleanup.Times(0)

				err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
					return nil
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("couldn't install that"))
			})
		})

		Context("when the passed in function returns an error", func() {
			It("returns the error", func() {
				cpiInstaller := release.CpiInstaller{
					InstallerFactory: mockInstallerFactory,
				}

				expectInstall.Return(installation, nil)

				err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
					return errors.New("My passed in function failed")
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("My passed in function failed"))
			})

			It("cleans up the installation", func() {
				cpiInstaller := release.CpiInstaller{
					InstallerFactory: mockInstallerFactory,
				}

				expectInstall.Return(installation, nil)
				expectCleanup.Times(1)

				err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
					return errors.New("My passed in function failed")
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when cleanup fails", func() {
			It("returns the error", func() {
				cpiInstaller := release.CpiInstaller{
					InstallerFactory: mockInstallerFactory,
				}

				expectInstall.Return(installation, nil)
				expectCleanup.Return(errors.New("cleanup failed"))

				err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
					return nil
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cleanup failed"))
			})

			It("returns the error from the function instead, if present", func() {
				cpiInstaller := release.CpiInstaller{
					InstallerFactory: mockInstallerFactory,
				}

				expectInstall.Return(installation, nil)
				expectCleanup.Return(errors.New("cleanup failed"))

				err := cpiInstaller.WithInstalledCpiRelease(installationManifest, target, installStage, func(installation biinstallation.Installation) error {
					return errors.New("My passed in function failed")
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("My passed in function failed"))
			})
		})
	})
})
