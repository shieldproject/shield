package cmd_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	fakebihttpclient "github.com/cloudfoundry/bosh-utils/httpclient/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	"github.com/cppforlife/go-patch/patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	mock_httpagent "github.com/cloudfoundry/bosh-agent/agentclient/http/mocks"
	mock_agentclient "github.com/cloudfoundry/bosh-cli/agentclient/mocks"
	mock_blobstore "github.com/cloudfoundry/bosh-cli/blobstore/mocks"
	bicloud "github.com/cloudfoundry/bosh-cli/cloud"
	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"
	mock_cloud "github.com/cloudfoundry/bosh-cli/cloud/mocks"
	bicmd "github.com/cloudfoundry/bosh-cli/cmd"
	biconfig "github.com/cloudfoundry/bosh-cli/config"
	mock_config "github.com/cloudfoundry/bosh-cli/config/mocks"
	bicpirel "github.com/cloudfoundry/bosh-cli/cpi/release"
	"github.com/cloudfoundry/bosh-cli/crypto"
	"github.com/cloudfoundry/bosh-cli/deployment"
	bideplmanifest "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	fakebideplmanifest "github.com/cloudfoundry/bosh-cli/deployment/manifest/manifestfakes"
	fakebideplval "github.com/cloudfoundry/bosh-cli/deployment/manifest/manifestfakes"
	mock_deployment "github.com/cloudfoundry/bosh-cli/deployment/mocks"
	bidepltpl "github.com/cloudfoundry/bosh-cli/deployment/template"
	fakebidepltpl "github.com/cloudfoundry/bosh-cli/deployment/template/templatefakes"
	fakebivm "github.com/cloudfoundry/bosh-cli/deployment/vm/fakes"
	mock_vm "github.com/cloudfoundry/bosh-cli/deployment/vm/mocks"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
	biinstall "github.com/cloudfoundry/bosh-cli/installation"
	biinstallmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	fakebiinstallmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest/fakes"
	mock_install "github.com/cloudfoundry/bosh-cli/installation/mocks"
	bitarball "github.com/cloudfoundry/bosh-cli/installation/tarball"
	mock_registry "github.com/cloudfoundry/bosh-cli/registry/mocks"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	bireljob "github.com/cloudfoundry/bosh-cli/release/job"
	birelmanifest "github.com/cloudfoundry/bosh-cli/release/manifest"
	fakebirel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	birelsetmanifest "github.com/cloudfoundry/bosh-cli/release/set/manifest"
	fakebirelsetmanifest "github.com/cloudfoundry/bosh-cli/release/set/manifest/fakes"
	bistemcell "github.com/cloudfoundry/bosh-cli/stemcell"
	mock_stemcell "github.com/cloudfoundry/bosh-cli/stemcell/mocks"
	fakebistemcell "github.com/cloudfoundry/bosh-cli/stemcell/stemcellfakes"
	biui "github.com/cloudfoundry/bosh-cli/ui"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("CreateEnvCmd", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {
		var (
			command          *bicmd.CreateEnvCmd
			fs               *fakesys.FakeFileSystem
			stdOut           *gbytes.Buffer
			stdErr           *gbytes.Buffer
			userInterface    biui.UI
			digestCalculator crypto.DigestCalculator
			manifestSHA      string

			mockDeployer              *mock_deployment.MockDeployer
			mockInstaller             *mock_install.MockInstaller
			mockInstallerFactory      *mock_install.MockInstallerFactory
			releaseReader             *fakerel.FakeReader
			releaseManager            biinstall.ReleaseManager
			mockRegistryServerManager *mock_registry.MockServerManager
			mockRegistryServer        *mock_registry.MockServer
			mockAgentClient           *mock_agentclient.MockAgentClient
			mockAgentClientFactory    *mock_httpagent.MockAgentClientFactory
			mockCloudFactory          *mock_cloud.MockFactory

			cpiRelease *fakebirel.FakeRelease
			logger     boshlog.Logger

			mockBlobstoreFactory *mock_blobstore.MockFactory
			mockBlobstore        *mock_blobstore.MockBlobstore

			mockVMManagerFactory       *mock_vm.MockManagerFactory
			fakeVMManager              *fakebivm.FakeManager
			fakeStemcellExtractor      *fakebistemcell.FakeExtractor
			mockStemcellManager        *mock_stemcell.MockManager
			fakeStemcellManagerFactory *fakebistemcell.FakeManagerFactory

			fakeReleaseSetParser              *fakebirelsetmanifest.FakeParser
			fakeInstallationParser            *fakebiinstallmanifest.FakeParser
			fakeDeploymentParser              *fakebideplmanifest.FakeParser
			fakeDeploymentTemplateFactory     *fakebidepltpl.FakeDeploymentTemplateFactory
			mockLegacyDeploymentStateMigrator *mock_config.MockLegacyDeploymentStateMigrator
			setupDeploymentStateService       biconfig.DeploymentStateService
			fakeDeploymentValidator           *fakebideplval.FakeValidator

			directorID          = "generated-director-uuid"
			fakeUUIDGenerator   *fakeuuid.FakeGenerator
			configUUIDGenerator *fakeuuid.FakeGenerator

			fakeStage *fakebiui.FakeStage

			deploymentManifestPath string
			deploymentStatePath    string
			cpiReleaseTarballPath  string
			stemcellTarballPath    string
			extractedStemcell      bistemcell.ExtractedStemcell

			expectDeploy *gomock.Call

			mbusURL = "http://fake-mbus-user:fake-mbus-password@fake-mbus-endpoint"

			releaseSetManifest     birelsetmanifest.Manifest
			template               bidepltpl.DeploymentTemplate
			boshDeploymentManifest bideplmanifest.Manifest
			installationManifest   biinstallmanifest.Manifest
			cloud                  bicloud.Cloud

			cloudStemcell bistemcell.CloudStemcell

			defaultCreateEnvOpts bicmd.CreateEnvOpts

			expectLegacyMigrate        *gomock.Call
			expectStemcellUpload       *gomock.Call
			expectStemcellDeleteUnused *gomock.Call
			expectInstall              *gomock.Call
			expectNewCloud             *gomock.Call
		)

		BeforeEach(func() {
			logger = boshlog.NewLogger(boshlog.LevelNone)
			stdOut = gbytes.NewBuffer()
			stdErr = gbytes.NewBuffer()
			userInterface = biui.NewWriterUI(stdOut, stdErr, logger)
			fs = fakesys.NewFakeFileSystem()
			fs.EnableStrictTempRootBehavior()
			deploymentManifestPath = filepath.Join("/", "path", "to", "manifest.yml")
			deploymentStatePath = filepath.Join("/", "path", "to", "manifest-state.json")
			fs.RegisterOpenFile(deploymentManifestPath, &fakesys.FakeFile{
				Stats: &fakesys.FakeFileStats{FileType: fakesys.FakeFileTypeFile},
			})

			fs.WriteFileString(deploymentManifestPath, "")

			mockDeployer = mock_deployment.NewMockDeployer(mockCtrl)
			mockInstaller = mock_install.NewMockInstaller(mockCtrl)
			mockInstallerFactory = mock_install.NewMockInstallerFactory(mockCtrl)

			releaseReader = &fakerel.FakeReader{}
			releaseManager = biinstall.NewReleaseManager(logger)

			mockRegistryServerManager = mock_registry.NewMockServerManager(mockCtrl)
			mockRegistryServer = mock_registry.NewMockServer(mockCtrl)

			mockAgentClientFactory = mock_httpagent.NewMockAgentClientFactory(mockCtrl)
			mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)
			mockAgentClientFactory.EXPECT().NewAgentClient(gomock.Any(), gomock.Any()).Return(mockAgentClient).AnyTimes()

			mockCloudFactory = mock_cloud.NewMockFactory(mockCtrl)

			mockBlobstoreFactory = mock_blobstore.NewMockFactory(mockCtrl)
			mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)
			mockBlobstoreFactory.EXPECT().Create(mbusURL, gomock.Any()).Return(mockBlobstore, nil).AnyTimes()

			mockVMManagerFactory = mock_vm.NewMockManagerFactory(mockCtrl)
			fakeVMManager = fakebivm.NewFakeManager()
			mockVMManagerFactory.EXPECT().NewManager(gomock.Any(), mockAgentClient).Return(fakeVMManager).AnyTimes()

			fakeStemcellExtractor = fakebistemcell.NewFakeExtractor()
			mockStemcellManager = mock_stemcell.NewMockManager(mockCtrl)
			fakeStemcellManagerFactory = fakebistemcell.NewFakeManagerFactory()

			fakeReleaseSetParser = fakebirelsetmanifest.NewFakeParser()
			fakeInstallationParser = fakebiinstallmanifest.NewFakeParser()
			fakeDeploymentParser = &fakebideplmanifest.FakeParser{}
			fakeDeploymentTemplateFactory = &fakebidepltpl.FakeDeploymentTemplateFactory{}

			mockLegacyDeploymentStateMigrator = mock_config.NewMockLegacyDeploymentStateMigrator(mockCtrl)

			configUUIDGenerator = &fakeuuid.FakeGenerator{}
			configUUIDGenerator.GeneratedUUID = directorID
			setupDeploymentStateService = biconfig.NewFileSystemDeploymentStateService(fs, configUUIDGenerator, logger, biconfig.DeploymentStatePath(deploymentManifestPath, ""))

			fakeDeploymentValidator = fakebideplval.NewFakeValidator()

			fakeStage = fakebiui.NewFakeStage()

			algos := []boshcrypto.Algorithm{boshcrypto.DigestAlgorithmSHA1}
			digestCalculator = crypto.NewDigestCalculator(fs, algos)
			fakeUUIDGenerator = &fakeuuid.FakeGenerator{}

			var err error
			manifestSHA = "ed173647f91a1001fa3859cb7857b0318794a7e92b40412146a93bebfb052218c91c0299e7b495470bf67b462722b807e8db7b9df3b59866451efcf4ae9e27a4"
			Expect(err).ToNot(HaveOccurred())

			cpiReleaseTarballPath = filepath.Join("/", "release", "tarball", "path")

			stemcellTarballPath = filepath.Join("/", "stemcell", "tarball", "path")
			extractedStemcell = bistemcell.NewExtractedStemcell(
				bistemcell.Manifest{
					Name:            "fake-stemcell-name",
					Version:         "fake-stemcell-version",
					SHA1:            "fake-stemcell-sha1",
					CloudProperties: biproperty.Map{},
				},
				"fake-extracted-path",
				nil,
				fs,
			)

			// create input files
			fs.WriteFileString(cpiReleaseTarballPath, "")
			fs.WriteFileString(stemcellTarballPath, "")

			// deployment exists
			fs.WriteFileString(deploymentManifestPath, "")

			// deployment is valid
			fakeDeploymentValidator.SetValidateBehavior([]fakebideplval.ValidateOutput{
				{Err: nil},
			})
			fakeDeploymentValidator.SetValidateReleaseJobsBehavior([]fakebideplval.ValidateReleaseJobsOutput{
				{Err: nil},
			})

			// stemcell exists
			fs.WriteFile(stemcellTarballPath, []byte{})

			releaseSetManifest = birelsetmanifest.Manifest{
				Releases: []birelmanifest.ReleaseRef{
					{
						Name: "fake-cpi-release-name",
						URL:  "file://" + cpiReleaseTarballPath,
					},
				},
			}

			// parsed CPI deployment manifest
			installationManifest = biinstallmanifest.Manifest{
				Template: biinstallmanifest.ReleaseJobRef{
					Name:    "fake-cpi-release-job-name",
					Release: "fake-cpi-release-name",
				},
				Mbus: mbusURL,
			}

			// parsed BOSH deployment manifest
			boshDeploymentManifest = bideplmanifest.Manifest{
				Name: "fake-deployment-name",
				Jobs: []bideplmanifest.Job{
					{
						Name: "fake-job-name",
					},
				},
				ResourcePools: []bideplmanifest.ResourcePool{
					{
						Stemcell: bideplmanifest.StemcellRef{
							URL: "file://" + stemcellTarballPath,
						},
					},
				},
			}
			fakeDeploymentTemplateFactory.NewDeploymentTemplateFromPathReturns(template, nil)
			fakeDeploymentParser.ParseReturns(boshDeploymentManifest, nil)

			// parsed/extracted CPI release
			cpiRelease = &fakebirel.FakeRelease{}
			cpiRelease.NameReturns("fake-cpi-release-name")
			cpiRelease.VersionReturns("1.0")

			job := bireljob.NewJob(NewResource("fake-cpi-release-job-name", "job-fp", nil))
			job.Templates = map[string]string{"templates/cpi.erb": "bin/cpi"}
			cpiRelease.JobsReturns([]*bireljob.Job{job})
			cpiRelease.FindJobByNameStub = func(jobName string) (bireljob.Job, bool) {
				for _, job := range cpiRelease.Jobs() {
					if job.Name() == jobName {
						return *job, true
					}
				}
				return bireljob.Job{}, false
			}

			releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
				Expect(path).To(Equal(cpiReleaseTarballPath))
				return cpiRelease, nil
			}

			cloud = bicloud.NewCloud(fakebicloud.NewFakeCPICmdRunner(), "fake-director-id", logger)
			cloudStemcell = fakebistemcell.NewFakeCloudStemcell(
				"fake-stemcell-cid", "fake-stemcell-name", "fake-stemcell-version")

			defaultCreateEnvOpts = bicmd.CreateEnvOpts{
				Args: bicmd.CreateEnvArgs{
					Manifest: bicmd.FileBytesWithPathArg{Path: deploymentManifestPath},
				},
			}
		})

		JustBeforeEach(func() {
			doGet := func(deploymentManifestPath string, statePath string, deploymentVars boshtpl.Variables, deploymentOp patch.Op) bicmd.DeploymentPreparer {
				deploymentStateService := biconfig.NewFileSystemDeploymentStateService(fs, configUUIDGenerator, logger, biconfig.DeploymentStatePath(deploymentManifestPath, statePath))
				deploymentRepo := biconfig.NewDeploymentRepo(deploymentStateService)
				releaseRepo := biconfig.NewReleaseRepo(deploymentStateService, fakeUUIDGenerator)
				stemcellRepo := biconfig.NewStemcellRepo(deploymentStateService, fakeUUIDGenerator)
				deploymentRecord := deployment.NewRecord(deploymentRepo, releaseRepo, stemcellRepo)

				fakeHTTPClient := fakebihttpclient.NewFakeHTTPClient()
				tarballCache := bitarball.NewCache("fake-base-path", fs, logger)
				tarballProvider := bitarball.NewProvider(tarballCache, fs, fakeHTTPClient, 1, 0, logger)

				cpiInstaller := bicpirel.CpiInstaller{
					ReleaseManager:   releaseManager,
					InstallerFactory: mockInstallerFactory,
					Validator:        bicpirel.NewValidator(),
				}
				releaseFetcher := biinstall.NewReleaseFetcher(tarballProvider, releaseReader, releaseManager)
				stemcellFetcher := bistemcell.Fetcher{
					TarballProvider:   tarballProvider,
					StemcellExtractor: fakeStemcellExtractor,
				}
				releaseSetAndInstallationManifestParser := bicmd.ReleaseSetAndInstallationManifestParser{
					ReleaseSetParser:   fakeReleaseSetParser,
					InstallationParser: fakeInstallationParser,
				}

				deploymentManifestParser := bicmd.NewDeploymentManifestParser(
					fakeDeploymentParser,
					fakeDeploymentValidator,
					releaseManager,
					fakeDeploymentTemplateFactory,
				)

				fakeInstallationUUIDGenerator := &fakeuuid.FakeGenerator{}
				fakeInstallationUUIDGenerator.GeneratedUUID = "fake-installation-id"
				targetProvider := biinstall.NewTargetProvider(
					deploymentStateService,
					fakeInstallationUUIDGenerator,
					filepath.Join("fake-install-dir"),
				)
				tempRootConfigurator := bicmd.NewTempRootConfigurator(fs)

				return bicmd.NewDeploymentPreparer(
					userInterface,
					logger,
					"deployCmd",
					deploymentStateService,
					mockLegacyDeploymentStateMigrator,
					releaseManager,
					deploymentRecord,
					mockCloudFactory,
					fakeStemcellManagerFactory,
					mockAgentClientFactory,
					mockVMManagerFactory,
					mockBlobstoreFactory,
					mockDeployer,
					deploymentManifestPath,
					deploymentVars,
					deploymentOp,
					cpiInstaller,
					releaseFetcher,
					stemcellFetcher,
					releaseSetAndInstallationManifestParser,
					deploymentManifestParser,
					tempRootConfigurator,
					targetProvider,
				)
			}

			command = bicmd.NewCreateEnvCmd(userInterface, doGet)

			expectLegacyMigrate = mockLegacyDeploymentStateMigrator.EXPECT().MigrateIfExists(filepath.Join("/", "path", "to", "bosh-deployments.yml")).AnyTimes()

			fakeStemcellExtractor.SetExtractBehavior(stemcellTarballPath, extractedStemcell, nil)

			fakeStemcellManagerFactory.SetNewManagerBehavior(cloud, mockStemcellManager)

			expectStemcellUpload = mockStemcellManager.EXPECT().Upload(extractedStemcell, fakeStage).Return(cloudStemcell, nil).AnyTimes()

			expectStemcellDeleteUnused = mockStemcellManager.EXPECT().DeleteUnused(fakeStage).AnyTimes()

			fakeReleaseSetParser.ParseManifest = releaseSetManifest
			template := bidepltpl.NewDeploymentTemplate([]byte("--- {\"test\":true}"))
			fakeDeploymentTemplateFactory.NewDeploymentTemplateFromPathReturns(template, nil)
			fakeDeploymentParser.ParseReturns(boshDeploymentManifest, nil)
			fakeInstallationParser.ParseManifest = installationManifest

			installationPath := filepath.Join("fake-install-dir", "fake-installation-id")
			target := biinstall.NewTarget(installationPath)

			installedJob := biinstall.NewInstalledJob(
				biinstall.RenderedJobRef{
					Name: "fake-cpi-release-job-name",
				},
				filepath.Join(target.JobsPath(), "fake-cpi-release-job-name"),
			)

			mockInstallerFactory.EXPECT().NewInstaller(target).Return(mockInstaller).AnyTimes()

			installation := biinstall.NewInstallation(target, installedJob, installationManifest, mockRegistryServerManager)

			expectInstall = mockInstaller.EXPECT().Install(installationManifest, gomock.Any()).Do(func(_ interface{}, stage biui.Stage) {
				Expect(fakeStage.SubStages).To(ContainElement(stage))
			}).Return(installation, nil).AnyTimes()
			mockInstaller.EXPECT().Cleanup(installation).AnyTimes()

			mockDeployment := mock_deployment.NewMockDeployment(mockCtrl)

			expectDeploy = mockDeployer.EXPECT().Deploy(
				cloud,
				boshDeploymentManifest,
				cloudStemcell,
				installationManifest.Registry,
				fakeVMManager,
				mockBlobstore,
				gomock.Any(),
			).Do(func(_, _, _, _, _, _ interface{}, stage biui.Stage) {
				Expect(fakeStage.SubStages).To(ContainElement(stage))
			}).Return(mockDeployment, nil).AnyTimes()

			expectNewCloud = mockCloudFactory.EXPECT().NewCloud(installation, directorID).Return(cloud, nil).AnyTimes()
		})

		Describe("prints the deployment manifest and state file", func() {
			It("prints the deployment manifest", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).NotTo(HaveOccurred())
				Expect(stdOut).To(gbytes.Say("Deployment manifest: '" + regexp.QuoteMeta(filepath.Join("/", "path", "to", "manifest.yml")) + "'"))
			})

			Context("when state file is NOT specified", func() {
				It("prints the default state file path", func() {
					err := command.Run(fakeStage, defaultCreateEnvOpts)
					Expect(err).NotTo(HaveOccurred())
					Expect(stdOut).To(gbytes.Say("Deployment state: '" + regexp.QuoteMeta(filepath.Join("/", "path", "to", "manifest-state.json")) + "'"))
				})
			})

			Context("when state file is specified", func() {
				It("prints specified state file path", func() {
					createEnvOptsWithStatePath := bicmd.CreateEnvOpts{
						StatePath: filepath.Join("/", "specified", "path", "to", "cool-state.json"),
						Args: bicmd.CreateEnvArgs{
							Manifest: bicmd.FileBytesWithPathArg{Path: deploymentManifestPath},
						},
					}

					err := command.Run(fakeStage, createEnvOptsWithStatePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(stdOut).To(gbytes.Say("Deployment state: '" + regexp.QuoteMeta(filepath.Join("/", "specified", "path", "to", "cool-state.json")) + "'"))
				})
			})
		})

		It("does not migrate the legacy bosh-deployments.yml if manifest-state.json exists", func() {
			err := fs.WriteFileString(deploymentStatePath, "{}")
			Expect(err).ToNot(HaveOccurred())

			expectLegacyMigrate.Times(0)

			err = command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeInstallationParser.ParsePath).To(Equal(deploymentManifestPath))
		})

		It("migrates the legacy bosh-deployments.yml if manifest-state.json does not exist", func() {
			err := fs.RemoveAll(deploymentStatePath)
			Expect(err).ToNot(HaveOccurred())

			expectLegacyMigrate.Return(true, nil).Times(1)

			err = command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeInstallationParser.ParsePath).To(Equal(deploymentManifestPath))

			Expect(stdOut).To(gbytes.Say("Deployment manifest: '" + regexp.QuoteMeta(filepath.Join("/", "path", "to", "manifest.yml")) + "'"))
			Expect(stdOut).To(gbytes.Say("Deployment state: '" + regexp.QuoteMeta(filepath.Join("/", "path", "to", "manifest-state.json")) + "'"))
			Expect(stdOut).To(gbytes.Say("Migrated legacy deployments file: '" + regexp.QuoteMeta(filepath.Join("/", "path", "to", "bosh-deployments.yml")) + "'"))
		})

		It("sets the temp root", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(fs.TempRootPath).To(Equal(filepath.Join("fake-install-dir", "fake-installation-id", "tmp")))
		})

		Context("when setting the temp root fails", func() {
			It("returns an error", func() {
				fs.ChangeTempRootErr = errors.New("fake ChangeTempRootErr")
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Setting temp root: fake ChangeTempRootErr"))
			})
		})

		It("parses the installation manifest", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeInstallationParser.ParsePath).To(Equal(deploymentManifestPath))
		})

		It("parses the deployment manifest", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			actualManifestPath := fakeDeploymentTemplateFactory.NewDeploymentTemplateFromPathArgsForCall(0)
			Expect(actualManifestPath).To(Equal(deploymentManifestPath))

			actualInterpolatedTemplate, actualPath := fakeDeploymentParser.ParseArgsForCall(0)
			Expect(actualInterpolatedTemplate.Content()).To(Equal([]byte("test: true\n")))
			Expect(actualPath).To(Equal(deploymentManifestPath))
		})

		It("validates bosh deployment manifest", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeDeploymentValidator.ValidateInputs).To(Equal([]fakebideplval.ValidateInput{
				{Manifest: boshDeploymentManifest, ReleaseSetManifest: releaseSetManifest},
			}))
		})

		It("validates jobs in manifest refer to job in releases", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeDeploymentValidator.ValidateReleaseJobsInputs).To(Equal([]fakebideplval.ValidateReleaseJobsInput{
				{Manifest: boshDeploymentManifest, ReleaseManager: releaseManager},
			}))
		})

		It("logs validating stages", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStage.PerformCalls[0]).To(Equal(&fakebiui.PerformCall{
				Name: "validating",
				Stage: &fakebiui.FakeStage{
					PerformCalls: []*fakebiui.PerformCall{
						{Name: "Validating release 'fake-cpi-release-name'"},
						{Name: "Validating cpi release"},
						{Name: "Validating deployment manifest"},
						{Name: "Validating stemcell"},
					},
				},
			}))
		})

		It("installs the CPI locally", func() {
			expectInstall.Times(1)
			expectNewCloud.Times(1)

			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
		})

		It("adds a new 'installing CPI' event logger stage", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStage.PerformCalls[1]).To(Equal(&fakebiui.PerformCall{
				Name:  "installing CPI",
				Stage: &fakebiui.FakeStage{}, // mock installer doesn't add sub-stages
			}))
		})

		It("adds a new 'Starting registry' event logger stage", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStage.PerformCalls[2]).To(Equal(&fakebiui.PerformCall{
				Name: "Starting registry",
			}))
		})

		Context("when the registry is configured", func() {
			BeforeEach(func() {
				installationManifest.Registry = biinstallmanifest.Registry{
					Username: "fake-username",
					Password: "fake-password",
					Host:     "fake-host",
					Port:     123,
				}
			})

			It("starts & stops the registry", func() {
				mockRegistryServerManager.EXPECT().Start("fake-username", "fake-password", "fake-host", 123).Return(mockRegistryServer, nil)
				mockRegistryServer.EXPECT().Stop()

				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("deletes the extracted CPI release", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(cpiRelease.CleanUpCallCount()).To(Equal(1))
		})

		It("extracts the stemcell", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeStemcellExtractor.ExtractInputs).To(Equal([]fakebistemcell.ExtractInput{
				{TarballPath: stemcellTarballPath},
			}))
		})

		It("uploads the stemcell", func() {
			expectStemcellUpload.Times(1)

			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("adds a new 'deploying' event logger stage", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStage.PerformCalls[3]).To(Equal(&fakebiui.PerformCall{
				Name:  "deploying",
				Stage: &fakebiui.FakeStage{}, // mock deployer doesn't add sub-stages
			}))
		})

		It("deploys", func() {
			expectDeploy.Times(1)

			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
		})

		It("updates the deployment record", func() {
			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())

			deploymentState, err := setupDeploymentStateService.Load()
			Expect(err).ToNot(HaveOccurred())

			Expect(deploymentState.CurrentManifestSHA).To(Equal(manifestSHA))
			Expect(deploymentState.Releases).To(Equal([]biconfig.ReleaseRecord{
				{
					ID:      "fake-uuid-0",
					Name:    cpiRelease.Name(),
					Version: cpiRelease.Version(),
				},
			}))
		})

		It("deletes unused stemcells", func() {
			expectStemcellDeleteUnused.Times(1)

			err := command.Run(fakeStage, defaultCreateEnvOpts)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when deployment has not changed", func() {
			JustBeforeEach(func() {
				previousDeploymentState := biconfig.DeploymentState{
					DirectorID:        directorID,
					CurrentReleaseIDs: []string{"my-release-id-1"},
					Releases: []biconfig.ReleaseRecord{{
						ID:      "my-release-id-1",
						Name:    cpiRelease.Name(),
						Version: cpiRelease.Version(),
					}},
					CurrentStemcellID: "my-stemcellRecordID",
					Stemcells: []biconfig.StemcellRecord{{
						ID:      "my-stemcellRecordID",
						Name:    cloudStemcell.Name(),
						Version: cloudStemcell.Version(),
					}},
					CurrentManifestSHA: manifestSHA,
				}

				err := setupDeploymentStateService.Save(previousDeploymentState)
				Expect(err).ToNot(HaveOccurred())
			})

			It("skips deploy", func() {
				expectDeploy.Times(0)

				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).NotTo(HaveOccurred())
				Expect(stdOut).To(gbytes.Say("No deployment, stemcell or release changes. Skipping deploy."))
			})
		})

		Context("when parsing the cpi deployment manifest fails", func() {
			JustBeforeEach(func() {
				manifest := bideplmanifest.Manifest{}
				err := bosherr.Error("fake-parse-error")
				fakeDeploymentParser.ParseReturns(manifest, err)
			})

			It("returns error", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Parsing deployment manifest"))
				Expect(err.Error()).To(ContainSubstring("fake-parse-error"))
				parsePath := fakeDeploymentTemplateFactory.NewDeploymentTemplateFromPathArgsForCall(0)
				Expect(parsePath).To(Equal(deploymentManifestPath))
			})
		})

		Context("when the cpi release does not contain a 'cpi' job", func() {
			BeforeEach(func() {
				cpiRelease.JobsReturns([]*bireljob.Job{
					bireljob.NewJob(NewResource("not-cpi", "job-fp", nil)),
				})
			})

			It("returns error", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Invalid CPI release 'fake-cpi-release-name': CPI release must contain specified job 'fake-cpi-release-job-name'"))
			})
		})

		Context("when multiple releases are given", func() {
			var (
				otherReleaseTarballPath string
				otherRelease            *fakebirel.FakeRelease
			)

			BeforeEach(func() {
				otherReleaseTarballPath = filepath.Join("/", "path", "to", "other-release.tgz")
				fs.WriteFileString(otherReleaseTarballPath, "")

				otherRelease = &fakebirel.FakeRelease{}
				otherRelease.NameReturns("other-release")
				otherRelease.VersionReturns("1234")
				otherRelease.JobsReturns([]*bireljob.Job{
					bireljob.NewJob(NewResource("not-cpi", "job-fp", nil)),
				})

				releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
					switch path {
					case cpiReleaseTarballPath:
						return cpiRelease, nil
					case otherReleaseTarballPath:
						return otherRelease, nil
					default:
						panic(fmt.Sprintf("Unexpected release reading for path '%s'", path))
					}
				}

				releaseSetManifest = birelsetmanifest.Manifest{
					Releases: []birelmanifest.ReleaseRef{
						{
							Name: "fake-cpi-release-name",
							URL:  "file://" + cpiReleaseTarballPath,
						},
						{
							Name: "other-release",
							URL:  "file://" + otherReleaseTarballPath,
						},
					},
				}
			})

			It("installs the CPI release locally", func() {
				expectInstall.Times(1)
				expectNewCloud.Times(1)

				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).NotTo(HaveOccurred())
			})

			It("updates the deployment record", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).NotTo(HaveOccurred())

				deploymentState, err := setupDeploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentState.CurrentManifestSHA).To(Equal(manifestSHA))
				Expect(deploymentState.Releases).To(Equal([]biconfig.ReleaseRecord{
					{
						ID:      "fake-uuid-0",
						Name:    cpiRelease.Name(),
						Version: cpiRelease.Version(),
					},
					{
						ID:      "fake-uuid-1",
						Name:    otherRelease.Name(),
						Version: otherRelease.Version(),
					},
				}))
			})

			Context("when one of the releases in the deployment has changed", func() {
				JustBeforeEach(func() {
					olderReleaseVersion := "1233"
					Expect(otherRelease.Version()).ToNot(Equal(olderReleaseVersion))
					previousDeploymentState := biconfig.DeploymentState{
						DirectorID:        directorID,
						CurrentReleaseIDs: []string{"existing-release-id-1", "existing-release-id-2"},
						Releases: []biconfig.ReleaseRecord{
							{
								ID:      "existing-release-id-1",
								Name:    cpiRelease.Name(),
								Version: cpiRelease.Version(),
							},
							{
								ID:      "existing-release-id-2",
								Name:    otherRelease.Name(),
								Version: olderReleaseVersion,
							},
						},
						CurrentStemcellID: "my-stemcellRecordID",
						Stemcells: []biconfig.StemcellRecord{{
							ID:      "my-stemcellRecordID",
							Name:    cloudStemcell.Name(),
							Version: cloudStemcell.Version(),
						}},
						CurrentManifestSHA: manifestSHA,
					}

					err := setupDeploymentStateService.Save(previousDeploymentState)
					Expect(err).ToNot(HaveOccurred())
				})

				It("updates the deployment record, clearing out unused releases", func() {
					err := command.Run(fakeStage, defaultCreateEnvOpts)
					Expect(err).NotTo(HaveOccurred())

					deploymentState, err := setupDeploymentStateService.Load()
					Expect(err).ToNot(HaveOccurred())

					Expect(deploymentState.CurrentManifestSHA).To(Equal(manifestSHA))
					keys := []string{}
					ids := []string{}
					for _, releaseRecord := range deploymentState.Releases {
						keys = append(keys, fmt.Sprintf("%s-%s", releaseRecord.Name, releaseRecord.Version))
						ids = append(ids, releaseRecord.ID)
					}
					Expect(deploymentState.CurrentReleaseIDs).To(ConsistOf(ids))
					Expect(keys).To(ConsistOf([]string{
						fmt.Sprintf("%s-%s", cpiRelease.Name(), cpiRelease.Version()),
						fmt.Sprintf("%s-%s", otherRelease.Name(), otherRelease.Version()),
					}))
				})
			})

			Context("when the deployment has not changed", func() {
				JustBeforeEach(func() {
					previousDeploymentState := biconfig.DeploymentState{
						DirectorID:        directorID,
						CurrentReleaseIDs: []string{"my-release-id-1", "my-release-id-2"},
						Releases: []biconfig.ReleaseRecord{
							{
								ID:      "my-release-id-1",
								Name:    cpiRelease.Name(),
								Version: cpiRelease.Version(),
							},
							{
								ID:      "my-release-id-2",
								Name:    otherRelease.Name(),
								Version: otherRelease.Version(),
							},
						},
						CurrentStemcellID: "my-stemcellRecordID",
						Stemcells: []biconfig.StemcellRecord{{
							ID:      "my-stemcellRecordID",
							Name:    cloudStemcell.Name(),
							Version: cloudStemcell.Version(),
						}},
						CurrentManifestSHA: manifestSHA,
					}

					err := setupDeploymentStateService.Save(previousDeploymentState)
					Expect(err).ToNot(HaveOccurred())
				})

				It("skips deploy", func() {
					expectDeploy.Times(0)

					err := command.Run(fakeStage, defaultCreateEnvOpts)
					Expect(err).NotTo(HaveOccurred())
					Expect(stdOut).To(gbytes.Say("No deployment, stemcell or release changes. Skipping deploy."))
				})
			})
		})

		Context("when release name does not match the name in release tarball", func() {
			BeforeEach(func() {
				releaseSetManifest.Releases = []birelmanifest.ReleaseRef{
					{
						Name: "fake-other-cpi-release-name",
						URL:  "file://" + cpiReleaseTarballPath,
					},
				}
			})

			It("returns an error", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Release name 'fake-other-cpi-release-name' does not match the name in release tarball 'fake-cpi-release-name'"))
			})
		})

		Context("When the stemcell tarball does not exist", func() {
			JustBeforeEach(func() {
				fakeStemcellExtractor.SetExtractBehavior(stemcellTarballPath, extractedStemcell, errors.New("no-stemcell-there"))
			})

			It("returns error", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no-stemcell-there"))

				performCall := fakeStage.PerformCalls[0].Stage.PerformCalls[3]
				Expect(performCall.Name).To(Equal("Validating stemcell"))
				Expect(performCall.Error.Error()).To(ContainSubstring("no-stemcell-there"))
			})
		})

		Context("when release file does not exist", func() {
			BeforeEach(func() {
				releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
					Expect(path).To(Equal(cpiReleaseTarballPath))
					return nil, errors.New("not there")
				}
			})

			It("returns error", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not there"))

				performCall := fakeStage.PerformCalls[0].Stage.PerformCalls[0]
				Expect(performCall.Name).To(Equal("Validating release 'fake-cpi-release-name'"))
				Expect(performCall.Error.Error()).To(ContainSubstring("not there"))
			})
		})

		Context("when the deployment state file does not exist", func() {
			BeforeEach(func() {
				fs.RemoveAll(deploymentStatePath)
			})

			It("creates a deployment state", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).ToNot(HaveOccurred())

				deploymentState, err := setupDeploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentState.DirectorID).To(Equal(directorID))
			})
		})

		Context("when the deployment manifest is invalid", func() {
			BeforeEach(func() {
				fakeDeploymentValidator.SetValidateBehavior([]fakebideplval.ValidateOutput{
					{Err: bosherr.Error("fake-deployment-validation-error")},
				})
			})

			It("returns err", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-deployment-validation-error"))
			})

			It("logs the failed event log", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())

				performCall := fakeStage.PerformCalls[0].Stage.PerformCalls[2]
				Expect(performCall.Name).To(Equal("Validating deployment manifest"))
				Expect(performCall.Error.Error()).To(Equal("Validating deployment manifest: fake-deployment-validation-error"))
			})
		})

		Context("when validating jobs fails", func() {
			BeforeEach(func() {
				fakeDeploymentValidator.SetValidateReleaseJobsBehavior([]fakebideplval.ValidateReleaseJobsOutput{
					{Err: bosherr.Error("fake-jobs-validation-error")},
				})
			})

			It("returns err", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-jobs-validation-error"))
			})

			It("logs the failed event log", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())

				performCall := fakeStage.PerformCalls[0].Stage.PerformCalls[2]
				Expect(performCall.Name).To(Equal("Validating deployment manifest"))
				Expect(performCall.Error.Error()).To(Equal("Validating deployment jobs refer to jobs in release: fake-jobs-validation-error"))
			})
		})

		Context("when uploading stemcell fails", func() {
			JustBeforeEach(func() {
				expectStemcellUpload.Return(nil, bosherr.Error("fake-upload-error"))
			})

			It("returns an error", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-upload-error"))
			})
		})

		Context("when deploy fails", func() {
			BeforeEach(func() {
				mockDeployer.EXPECT().Deploy(
					cloud,
					boshDeploymentManifest,
					cloudStemcell,
					installationManifest.Registry,
					fakeVMManager,
					mockBlobstore,
					gomock.Any(),
				).Return(nil, errors.New("fake-deploy-error")).AnyTimes()

				previousDeploymentState := biconfig.DeploymentState{
					CurrentReleaseIDs: []string{"my-release-id-1"},
					Releases: []biconfig.ReleaseRecord{{
						ID:      "my-release-id-1",
						Name:    cpiRelease.Name(),
						Version: cpiRelease.Version(),
					}},
					CurrentManifestSHA: "fake-manifest-sha",
				}

				setupDeploymentStateService.Save(previousDeploymentState)
			})

			It("clears the deployment record", func() {
				err := command.Run(fakeStage, defaultCreateEnvOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-deploy-error"))

				deploymentState, err := setupDeploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentState.CurrentManifestSHA).To(Equal(""))
				Expect(deploymentState.Releases).To(Equal([]biconfig.ReleaseRecord{}))
				Expect(deploymentState.CurrentReleaseIDs).To(Equal([]string{}))
			})
		})
	})
})
