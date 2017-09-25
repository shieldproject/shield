package cmd_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bicmd "github.com/cloudfoundry/bosh-init/cmd"

	mock_httpagent "github.com/cloudfoundry/bosh-agent/agentclient/http/mocks"
	mock_agentclient "github.com/cloudfoundry/bosh-init/agentclient/mocks"
	mock_blobstore "github.com/cloudfoundry/bosh-init/blobstore/mocks"
	mock_cloud "github.com/cloudfoundry/bosh-init/cloud/mocks"
	mock_deployment "github.com/cloudfoundry/bosh-init/deployment/mocks"
	mock_install "github.com/cloudfoundry/bosh-init/installation/mocks"
	mock_release "github.com/cloudfoundry/bosh-init/release/mocks"
	"github.com/golang/mock/gomock"

	biconfig "github.com/cloudfoundry/bosh-init/config"
	bicpirel "github.com/cloudfoundry/bosh-init/cpi/release"
	biinstall "github.com/cloudfoundry/bosh-init/installation"
	biinstallmanifest "github.com/cloudfoundry/bosh-init/installation/manifest"
	bitarball "github.com/cloudfoundry/bosh-init/installation/tarball"
	birel "github.com/cloudfoundry/bosh-init/release"
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	biui "github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"

	"errors"

	fakecmd "github.com/cloudfoundry/bosh-init/cmd/fakes"
	fakebicrypto "github.com/cloudfoundry/bosh-init/crypto/fakes"
	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
	fakeui "github.com/cloudfoundry/bosh-init/ui/fakes"
	fakebihttpclient "github.com/cloudfoundry/bosh-utils/httpclient/fakes"
)

var _ = Describe("DeploymentDeleter", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("DeleteDeployment", func() {
		var (
			fs                          *fakesys.FakeFileSystem
			logger                      boshlog.Logger
			releaseManager              birel.Manager
			mockCpiInstaller            *mock_install.MockInstaller
			mockCpiUninstaller          *mock_install.MockUninstaller
			mockInstallerFactory        *mock_install.MockInstallerFactory
			mockCloudFactory            *mock_cloud.MockFactory
			mockReleaseExtractor        *mock_release.MockExtractor
			fakeUUIDGenerator           *fakeuuid.FakeGenerator
			setupDeploymentStateService biconfig.DeploymentStateService
			fakeInstallation            *fakecmd.FakeInstallation

			fakeUI *fakeui.FakeUI

			mockBlobstoreFactory *mock_blobstore.MockFactory
			mockBlobstore        *mock_blobstore.MockBlobstore

			mockDeploymentManagerFactory *mock_deployment.MockManagerFactory
			mockDeploymentManager        *mock_deployment.MockManager
			mockDeployment               *mock_deployment.MockDeployment

			mockAgentClient        *mock_agentclient.MockAgentClient
			mockAgentClientFactory *mock_httpagent.MockAgentClientFactory
			mockCloud              *mock_cloud.MockCloud

			fakeStage *fakebiui.FakeStage

			directorID string

			deploymentManifestPath = "/deployment-dir/fake-deployment-manifest.yml"
			deploymentStatePath    string

			expectCPIExtractRelease *gomock.Call
			expectCPIInstall        *gomock.Call
			expectNewCloud          *gomock.Call

			mbusURL = "http://fake-mbus-user:fake-mbus-password@fake-mbus-endpoint"
		)

		var writeDeploymentManifest = func() {
			fs.WriteFileString(deploymentManifestPath, `---
name: test-release

releases:
- name: fake-cpi-release-name
  url: file:///fake-cpi-release.tgz

cloud_provider:
  template:
    name: fake-cpi-release-job-name
    release: fake-cpi-release-name
  mbus: http://fake-mbus-user:fake-mbus-password@fake-mbus-endpoint
`)
		}

		var writeCPIReleaseTarball = func() {
			fs.WriteFileString("/fake-cpi-release.tgz", "fake-tgz-content")
		}

		var allowCPIToBeExtracted = func() {
			cpiRelease := birel.NewRelease(
				"fake-cpi-release-name",
				"fake-cpi-release-version",
				[]bireljob.Job{
					{
						Name: "fake-cpi-release-job-name",
						Templates: map[string]string{
							"templates/cpi.erb": "bin/cpi",
						},
					},
				},
				[]*birelpkg.Package{},
				"fake-cpi-extracted-dir",
				fs,
				false,
			)

			expectCPIExtractRelease = mockReleaseExtractor.EXPECT().Extract("/fake-cpi-release.tgz").Do(func(_ string) {
				err := fs.MkdirAll("fake-cpi-extracted-dir", os.ModePerm)
				Expect(err).ToNot(HaveOccurred())
			}).Return(cpiRelease, nil).AnyTimes()
		}

		var allowCPIToBeInstalled = func() {
			installationManifest := biinstallmanifest.Manifest{
				Name: "test-release",
				Template: biinstallmanifest.ReleaseJobRef{
					Name:    "fake-cpi-release-job-name",
					Release: "fake-cpi-release-name",
				},
				Mbus:       mbusURL,
				Properties: biproperty.Map{},
			}

			target := biinstall.NewTarget(filepath.Join("fake-install-dir", "fake-installation-id"))
			mockInstallerFactory.EXPECT().NewInstaller(target).Return(mockCpiInstaller).AnyTimes()

			expectCPIInstall = mockCpiInstaller.EXPECT().Install(installationManifest, gomock.Any()).Do(func(_ biinstallmanifest.Manifest, stage biui.Stage) {
				Expect(fakeStage.SubStages).To(ContainElement(stage))
			}).Return(fakeInstallation, nil).AnyTimes()
			mockCpiInstaller.EXPECT().Cleanup(fakeInstallation).AnyTimes()

			expectNewCloud = mockCloudFactory.EXPECT().NewCloud(fakeInstallation, directorID).Return(mockCloud, nil).AnyTimes()
		}

		var newDeploymentDeleter = func() bicmd.DeploymentDeleter {
			releaseSetValidator := birelsetmanifest.NewValidator(logger)
			releaseSetParser := birelsetmanifest.NewParser(fs, logger, releaseSetValidator)
			installationValidator := biinstallmanifest.NewValidator(logger)
			installationParser := biinstallmanifest.NewParser(fs, fakeUUIDGenerator, logger, installationValidator)
			fakeHTTPClient := fakebihttpclient.NewFakeHTTPClient()
			tarballCache := bitarball.NewCache("fake-base-path", fs, logger)
			fakeSHA1Calculator := fakebicrypto.NewFakeSha1Calculator()
			tarballProvider := bitarball.NewProvider(tarballCache, fs, fakeHTTPClient, fakeSHA1Calculator, 1, 0, logger)
			deploymentStateService := biconfig.NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, biconfig.DeploymentStatePath(deploymentManifestPath))

			cpiInstaller := bicpirel.CpiInstaller{
				ReleaseManager:   releaseManager,
				InstallerFactory: mockInstallerFactory,
				Validator:        bicpirel.NewValidator(),
			}
			releaseFetcher := birel.NewFetcher(tarballProvider, mockReleaseExtractor, releaseManager)
			releaseSetAndInstallationManifestParser := bicmd.ReleaseSetAndInstallationManifestParser{
				ReleaseSetParser:   releaseSetParser,
				InstallationParser: installationParser,
			}
			fakeInstallationUUIDGenerator := &fakeuuid.FakeGenerator{}
			fakeInstallationUUIDGenerator.GeneratedUUID = "fake-installation-id"
			targetProvider := biinstall.NewTargetProvider(
				deploymentStateService,
				fakeInstallationUUIDGenerator,
				filepath.Join("fake-install-dir"),
			)

			tempRootConfigurator := bicmd.NewTempRootConfigurator(fs)

			return bicmd.NewDeploymentDeleter(
				fakeUI,
				"deleteCmd",
				logger,
				deploymentStateService,
				releaseManager,
				mockCloudFactory,
				mockAgentClientFactory,
				mockBlobstoreFactory,
				mockDeploymentManagerFactory,
				deploymentManifestPath,
				cpiInstaller,
				mockCpiUninstaller,
				releaseFetcher,
				releaseSetAndInstallationManifestParser,
				tempRootConfigurator,
				targetProvider,
			)
		}

		var expectDeleteAndCleanup = func(defaultUninstallerUsed bool) {
			mockDeploymentManagerFactory.EXPECT().NewManager(mockCloud, mockAgentClient, mockBlobstore).Return(mockDeploymentManager)
			mockDeploymentManager.EXPECT().FindCurrent().Return(mockDeployment, true, nil)

			gomock.InOrder(
				mockDeployment.EXPECT().Delete(gomock.Any()).Do(func(stage biui.Stage) {
					Expect(fakeStage.SubStages).To(ContainElement(stage))
				}),
				mockDeploymentManager.EXPECT().Cleanup(fakeStage),
			)
			if defaultUninstallerUsed {
				mockCpiUninstaller.EXPECT().Uninstall(gomock.Any()).Return(nil)
			}

		}

		var expectCleanup = func() {
			mockDeploymentManagerFactory.EXPECT().NewManager(mockCloud, mockAgentClient, mockBlobstore).Return(mockDeploymentManager).AnyTimes()
			mockDeploymentManager.EXPECT().FindCurrent().Return(nil, false, nil).AnyTimes()

			mockDeploymentManager.EXPECT().Cleanup(fakeStage)
			mockCpiUninstaller.EXPECT().Uninstall(gomock.Any()).Return(nil)
		}

		var expectValidationInstallationDeletionEvents = func() {
			Expect(fakeUI.Said).To(Equal([]string{
				"Deployment state: '/deployment-dir/fake-deployment-manifest-state.json'",
			}))

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				{
					Name: "validating",
					Stage: &fakebiui.FakeStage{
						PerformCalls: []*fakebiui.PerformCall{
							{Name: "Validating release 'fake-cpi-release-name'"},
							{Name: "Validating cpi release"},
						},
					},
				},
				{
					Name:  "installing CPI",
					Stage: &fakebiui.FakeStage{},
				},
				// mock installation.WithRegistryRunning doesn't add stages
				{
					Name:  "deleting deployment",
					Stage: &fakebiui.FakeStage{},
				},
				{
					Name: "Uninstalling local artifacts for CPI and deployment",
				},
				{
					Name: "Cleaning up rendered CPI jobs",
				},
				// mock deployment manager cleanup doesn't add sub-stages
			}))

			// installing steps handled by installer.Install()
			// deleting steps handled by deployment.Delete()
		}

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()
			fs.EnableStrictTempRootBehavior()
			logger = boshlog.NewLogger(boshlog.LevelNone)
			fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
			setupDeploymentStateService = biconfig.NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, biconfig.DeploymentStatePath(deploymentManifestPath))
			deploymentStatePath = biconfig.DeploymentStatePath(deploymentManifestPath)
			setupDeploymentStateService.Load()

			fakeUI = &fakeui.FakeUI{}

			fakeStage = fakebiui.NewFakeStage()

			mockCloud = mock_cloud.NewMockCloud(mockCtrl)
			mockCloudFactory = mock_cloud.NewMockFactory(mockCtrl)

			mockCpiInstaller = mock_install.NewMockInstaller(mockCtrl)
			mockCpiUninstaller = mock_install.NewMockUninstaller(mockCtrl)
			mockInstallerFactory = mock_install.NewMockInstallerFactory(mockCtrl)

			fakeInstallation = &fakecmd.FakeInstallation{}

			mockBlobstoreFactory = mock_blobstore.NewMockFactory(mockCtrl)
			mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)
			mockBlobstoreFactory.EXPECT().Create(mbusURL, gomock.Any()).Return(mockBlobstore, nil).AnyTimes()

			mockDeploymentManagerFactory = mock_deployment.NewMockManagerFactory(mockCtrl)
			mockDeploymentManager = mock_deployment.NewMockManager(mockCtrl)
			mockDeployment = mock_deployment.NewMockDeployment(mockCtrl)

			mockReleaseExtractor = mock_release.NewMockExtractor(mockCtrl)
			releaseManager = birel.NewManager(logger)

			mockAgentClientFactory = mock_httpagent.NewMockAgentClientFactory(mockCtrl)
			mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)

			mockAgentClientFactory.EXPECT().NewAgentClient(gomock.Any(), gomock.Any()).Return(mockAgentClient).AnyTimes()

			directorID = "fake-uuid-0"

			writeDeploymentManifest()
			writeCPIReleaseTarball()
		})

		JustBeforeEach(func() {
			allowCPIToBeExtracted()
		})

		Context("when the CPI installs", func() {
			JustBeforeEach(func() {
				allowCPIToBeInstalled()
			})

			Context("when the deployment state file does not exist", func() {
				BeforeEach(func() {
					err := fs.RemoveAll(deploymentStatePath)
					Expect(err).ToNot(HaveOccurred())
				})

				It("does not delete anything", func() {
					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeUI.Said).To(Equal([]string{
						"Deployment state: '/deployment-dir/fake-deployment-manifest-state.json'",
						"No deployment state file found.",
					}))
				})
			})

			Context("when the deployment has been deployed", func() {
				BeforeEach(func() {
					directorID = "fake-director-id"

					// create deployment manifest yaml file
					setupDeploymentStateService.Save(biconfig.DeploymentState{
						DirectorID: directorID,
					})
				})

				Context("when change temp root fails", func() {
					It("returns an error", func() {
						fs.ChangeTempRootErr = errors.New("fake ChangeTempRootErr")
						err := newDeploymentDeleter().DeleteDeployment(fakeStage)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("Setting temp root: fake ChangeTempRootErr"))
					})
				})

				It("sets the temp root", func() {
					expectDeleteAndCleanup(true)
					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).NotTo(HaveOccurred())
					Expect(fs.TempRootPath).To(Equal("fake-install-dir/fake-installation-id/tmp"))
				})

				It("extracts & install CPI release tarball", func() {
					expectDeleteAndCleanup(true)

					gomock.InOrder(
						expectCPIExtractRelease.Times(1),
						expectCPIInstall.Times(1),
						expectNewCloud.Times(1),
					)

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).NotTo(HaveOccurred())
				})

				It("deletes the extracted CPI release", func() {
					expectDeleteAndCleanup(true)

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).NotTo(HaveOccurred())
					Expect(fs.FileExists("fake-cpi-extracted-dir")).To(BeFalse())
				})

				It("deletes the deployment & cleans up orphans", func() {
					expectDeleteAndCleanup(true)

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeUI.Errors).To(BeEmpty())
				})

				It("deletes the local CPI installation", func() {
					expectDeleteAndCleanup(false)
					mockCpiUninstaller.EXPECT().Uninstall(gomock.Any()).Return(nil)

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})

				It("logs validating & deleting stages", func() {
					expectDeleteAndCleanup(true)

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).ToNot(HaveOccurred())

					expectValidationInstallationDeletionEvents()
				})

				It("deletes the local deployment state file", func() {
					expectDeleteAndCleanup(true)

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.FileExists(deploymentStatePath)).To(BeFalse())
				})

			})

			Context("when nothing has been deployed", func() {
				BeforeEach(func() {
					setupDeploymentStateService.Save(biconfig.DeploymentState{DirectorID: "fake-uuid-0"})
				})

				It("cleans up orphans, but does not delete any deployment", func() {
					expectCleanup()

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeUI.Errors).To(BeEmpty())
				})
			})
		})

		Context("when the CPI fails to Delete", func() {
			JustBeforeEach(func() {
				installationManifest := biinstallmanifest.Manifest{
					Name: "test-release",
					Template: biinstallmanifest.ReleaseJobRef{
						Name:    "fake-cpi-release-job-name",
						Release: "fake-cpi-release-name",
					},
					Mbus:       mbusURL,
					Properties: biproperty.Map{},
				}

				target := biinstall.NewTarget(filepath.Join("fake-install-dir", "fake-installation-id"))
				mockInstallerFactory.EXPECT().NewInstaller(target).Return(mockCpiInstaller).AnyTimes()

				fakeInstallation := &fakecmd.FakeInstallation{}

				expectCPIInstall = mockCpiInstaller.EXPECT().Install(installationManifest, gomock.Any()).Do(func(_ biinstallmanifest.Manifest, stage biui.Stage) {
					Expect(fakeStage.SubStages).To(ContainElement(stage))
				}).Return(fakeInstallation, nil).AnyTimes()
				mockCpiInstaller.EXPECT().Cleanup(fakeInstallation).AnyTimes()

				expectNewCloud = mockCloudFactory.EXPECT().NewCloud(fakeInstallation, directorID).Return(mockCloud, nil).AnyTimes()
			})

			Context("when the call to delete the deployment returns an error", func() {
				It("returns the error", func() {
					mockDeploymentManagerFactory.EXPECT().NewManager(mockCloud, mockAgentClient, mockBlobstore).Return(mockDeploymentManager)
					mockDeploymentManager.EXPECT().FindCurrent().Return(mockDeployment, true, nil)

					deleteError := bosherr.Error("delete error")

					mockDeployment.EXPECT().Delete(gomock.Any()).Return(deleteError)

					err := newDeploymentDeleter().DeleteDeployment(fakeStage)

					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
