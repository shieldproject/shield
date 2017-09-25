package state_test

import (
	. "github.com/cloudfoundry/bosh-init/deployment/instance/state"

	mock_blobstore "github.com/cloudfoundry/bosh-init/blobstore/mocks"
	mock_deployment_release "github.com/cloudfoundry/bosh-init/deployment/release/mocks"
	mock_state_job "github.com/cloudfoundry/bosh-init/state/job/mocks"
	mock_template "github.com/cloudfoundry/bosh-init/templatescompiler/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	biac "github.com/cloudfoundry/bosh-agent/agentclient"
	bias "github.com/cloudfoundry/bosh-agent/agentclient/applyspec"
	bideplmanifest "github.com/cloudfoundry/bosh-init/deployment/manifest"
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bistatejob "github.com/cloudfoundry/bosh-init/state/job"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"

	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("Builder", describeBuilder)

func describeBuilder() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		logger boshlog.Logger

		mockReleaseJobResolver *mock_deployment_release.MockJobResolver
		mockDependencyCompiler *mock_state_job.MockDependencyCompiler
		mockJobListRenderer    *mock_template.MockJobListRenderer
		mockCompressor         *mock_template.MockRenderedJobListCompressor
		mockBlobstore          *mock_blobstore.MockBlobstore

		stateBuilder Builder
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)

		mockReleaseJobResolver = mock_deployment_release.NewMockJobResolver(mockCtrl)
		mockDependencyCompiler = mock_state_job.NewMockDependencyCompiler(mockCtrl)
		mockJobListRenderer = mock_template.NewMockJobListRenderer(mockCtrl)
		mockCompressor = mock_template.NewMockRenderedJobListCompressor(mockCtrl)
		mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)
	})

	Describe("BuildInitialState", func() {
		var (
			jobName            string
			instanceID         int
			deploymentManifest bideplmanifest.Manifest
		)

		BeforeEach(func() {
			jobName = "fake-deployment-job-name"
			instanceID = 0

			deploymentManifest = bideplmanifest.Manifest{
				Name: "fake-deployment-name",
				Jobs: []bideplmanifest.Job{
					{
						Name: "fake-deployment-job-name",
						Networks: []bideplmanifest.JobNetwork{
							{
								Name:      "fake-network-name",
								StaticIPs: []string{"1.2.3.4"},
							},
						},
						Templates: []bideplmanifest.ReleaseJobRef{
							{
								Name:    "fake-release-job-name",
								Release: "fake-release-name",
								Properties: &biproperty.Map{
									"fake-template-property": "fake-template-property-value",
								},
							},
						},
						Properties: biproperty.Map{
							"fake-job-property": "fake-job-property-value",
						},
					},
				},
				Networks: []bideplmanifest.Network{
					{
						Name: "fake-network-name",
						Type: "fake-network-type",
						CloudProperties: biproperty.Map{
							"fake-network-cloud-property": "fake-network-cloud-property-value",
						},
					},
				},
				Properties: biproperty.Map{
					"fake-job-property": "fake-global-property-value", //overridden by job property value
				},
			}

			stateBuilder = NewBuilder(
				mockReleaseJobResolver,
				mockDependencyCompiler,
				mockJobListRenderer,
				mockCompressor,
				mockBlobstore,
				logger,
			)
		})

		It("generates an initial apply spec", func() {
			state, err := stateBuilder.BuildInitialState(jobName, instanceID, deploymentManifest)
			Expect(err).ToNot(HaveOccurred())

			Expect(state.ToApplySpec()).To(Equal(bias.ApplySpec{
				Deployment: "fake-deployment-name",
				Index:      0,
				Job: bias.Job{
					Name:      "fake-deployment-job-name",
					Templates: []bias.Blob{},
				},
				Packages: map[string]bias.Blob{},
				Networks: map[string]interface{}{
					"fake-network-name": map[string]interface{}{
						"type":    "fake-network-type",
						"default": []bideplmanifest.NetworkDefault{"dns", "gateway"},
						"ip":      "1.2.3.4",
						"cloud_properties": biproperty.Map{
							"fake-network-cloud-property": "fake-network-cloud-property-value",
						},
					},
				},
			}))
		})
	})

	Describe("Build", func() {
		var (
			mockRenderedJobList        *mock_template.MockRenderedJobList
			mockRenderedJobListArchive *mock_template.MockRenderedJobListArchive

			jobName            string
			instanceID         int
			deploymentManifest bideplmanifest.Manifest
			fakeStage          *fakebiui.FakeStage

			releasePackageLibyaml *birelpkg.Package
			releasePackageRuby    *birelpkg.Package
			releasePackageCPI     *birelpkg.Package

			agentState biac.AgentState
			expectedIP string

			expectCompile *gomock.Call
		)

		BeforeEach(func() {
			mockRenderedJobList = mock_template.NewMockRenderedJobList(mockCtrl)
			mockRenderedJobListArchive = mock_template.NewMockRenderedJobListArchive(mockCtrl)

			jobName = "fake-deployment-job-name"
			instanceID = 0
			expectedIP = "1.2.3.4"

			deploymentManifest = bideplmanifest.Manifest{
				Name: "fake-deployment-name",
				Jobs: []bideplmanifest.Job{
					{
						Name: "fake-deployment-job-name",
						Networks: []bideplmanifest.JobNetwork{
							{
								Name:      "fake-network-name",
								StaticIPs: []string{"1.2.3.4"},
							},
						},
						Templates: []bideplmanifest.ReleaseJobRef{
							{
								Name:    "fake-release-job-name",
								Release: "fake-release-name",
								Properties: &biproperty.Map{
									"fake-template-property": "fake-template-property-value",
								},
							},
						},
						Properties: biproperty.Map{
							"fake-job-property": "fake-job-property-value",
						},
					},
				},
				Networks: []bideplmanifest.Network{
					{
						Name: "fake-network-name",
						Type: "fake-network-type",
						CloudProperties: biproperty.Map{
							"fake-network-cloud-property": "fake-network-cloud-property-value",
						},
					},
				},
				Properties: biproperty.Map{
					"fake-job-property": "fake-global-property-value", //overridden by job property value
				},
			}

			agentState = biac.AgentState{
				NetworkSpecs: map[string]biac.NetworkSpec{
					"fake-network-name": biac.NetworkSpec{
						IP: "1.2.3.5",
					},
				},
			}

			fakeStage = fakebiui.NewFakeStage()

			stateBuilder = NewBuilder(
				mockReleaseJobResolver,
				mockDependencyCompiler,
				mockJobListRenderer,
				mockCompressor,
				mockBlobstore,
				logger,
			)

			releasePackageLibyaml = &birelpkg.Package{
				Name:         "libyaml",
				Fingerprint:  "fake-package-source-fingerprint-libyaml",
				SHA1:         "fake-package-source-sha1-libyaml",
				Dependencies: []*birelpkg.Package{},
				ArchivePath:  "fake-package-archive-path-libyaml", // only required by compiler...
			}
			releasePackageRuby = &birelpkg.Package{
				Name:         "ruby",
				Fingerprint:  "fake-package-source-fingerprint-ruby",
				SHA1:         "fake-package-source-sha1-ruby",
				Dependencies: []*birelpkg.Package{releasePackageLibyaml},
				ArchivePath:  "fake-package-archive-path-ruby", // only required by compiler...
			}
			releasePackageCPI = &birelpkg.Package{
				Name:         "cpi",
				Fingerprint:  "fake-package-source-fingerprint-cpi",
				SHA1:         "fake-package-source-sha1-cpi",
				Dependencies: []*birelpkg.Package{releasePackageRuby},
				ArchivePath:  "fake-package-archive-path-cpi", // only required by compiler...
			}
		})

		JustBeforeEach(func() {
			releaseJob := bireljob.Job{
				Name:        "fake-release-job-name",
				Fingerprint: "fake-release-job-source-fingerprint",
				Packages:    []*birelpkg.Package{releasePackageCPI, releasePackageRuby},
			}
			mockReleaseJobResolver.EXPECT().Resolve("fake-release-job-name", "fake-release-name").Return(releaseJob, nil)

			releaseJobs := []bireljob.Job{releaseJob}
			compiledPackageRefs := []bistatejob.CompiledPackageRef{
				{
					Name:        "libyaml",
					Version:     "fake-package-source-fingerprint-libyaml",
					BlobstoreID: "fake-package-compiled-archive-blob-id-libyaml",
					SHA1:        "fake-package-compiled-archive-sha1-libyaml",
				},
				{
					Name:        "ruby",
					Version:     "fake-package-source-fingerprint-ruby",
					BlobstoreID: "fake-package-compiled-archive-blob-id-ruby",
					SHA1:        "fake-package-compiled-archive-sha1-ruby",
				},
				{
					Name:        "cpi",
					Version:     "fake-package-source-fingerprint-cpi",
					BlobstoreID: "fake-package-compiled-archive-blob-id-cpi",
					SHA1:        "fake-package-compiled-archive-sha1-cpi",
				},
			}
			expectCompile = mockDependencyCompiler.EXPECT().Compile(releaseJobs, fakeStage).Return(compiledPackageRefs, nil).AnyTimes()

			releaseJobProperties := map[string]*biproperty.Map{
				"fake-release-job-name": &biproperty.Map{
					"fake-template-property": "fake-template-property-value",
				},
			}

			jobProperties := biproperty.Map{
				"fake-job-property": "fake-job-property-value",
			}
			globalProperties := biproperty.Map{
				"fake-job-property": "fake-global-property-value",
			}

			mockJobListRenderer.EXPECT().Render(releaseJobs, releaseJobProperties, jobProperties, globalProperties, "fake-deployment-name", expectedIP).Return(mockRenderedJobList, nil)

			mockRenderedJobList.EXPECT().DeleteSilently()

			mockCompressor.EXPECT().Compress(mockRenderedJobList).Return(mockRenderedJobListArchive, nil)

			mockRenderedJobListArchive.EXPECT().DeleteSilently()

			mockRenderedJobListArchive.EXPECT().Path().Return("fake-rendered-job-list-archive-path")
			mockRenderedJobListArchive.EXPECT().SHA1().Return("fake-rendered-job-list-archive-sha1")
			mockRenderedJobListArchive.EXPECT().Fingerprint().Return("fake-rendered-job-list-fingerprint")

			mockBlobstore.EXPECT().Add("fake-rendered-job-list-archive-path").Return("fake-rendered-job-list-archive-blob-id", nil)
		})

		It("compiles the dependencies of the jobs", func() {
			expectCompile.Times(1)

			_, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
			Expect(err).ToNot(HaveOccurred())
		})

		It("builds a new instance state with zero-to-many networks", func() {
			state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
			Expect(err).ToNot(HaveOccurred())

			Expect(state.NetworkInterfaces()).To(ContainElement(NetworkRef{
				Name: "fake-network-name",
				Interface: map[string]interface{}{
					"type":    "fake-network-type",
					"default": []bideplmanifest.NetworkDefault{"dns", "gateway"},
					"ip":      "1.2.3.4",
					"cloud_properties": biproperty.Map{
						"fake-network-cloud-property": "fake-network-cloud-property-value",
					},
				},
			}))
			Expect(state.NetworkInterfaces()).To(HaveLen(1))
		})

		Context("dynamic network without IP address", func() {
			Context("single network", func() {
				BeforeEach(func() {
					deploymentManifest.Jobs[0].Networks[0].StaticIPs = nil
					deploymentManifest.Networks[0].Type = "dynamic"
					expectedIP = "1.2.3.5"
				})

				It("should not fail", func() {
					state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
					Expect(err).ToNot(HaveOccurred())

					Expect(state.NetworkInterfaces()).To(ContainElement(NetworkRef{
						Name: "fake-network-name",
						Interface: map[string]interface{}{
							"type":    "dynamic",
							"default": []bideplmanifest.NetworkDefault{"dns", "gateway"},
							"cloud_properties": biproperty.Map{
								"fake-network-cloud-property": "fake-network-cloud-property-value",
							},
						},
					}))
					Expect(state.NetworkInterfaces()).To(HaveLen(1))
				})
			})

			Context("multiple networks", func() {
				BeforeEach(func() {
					expectedIP = "1.2.3.6"
					deploymentManifest.Networks = append(
						deploymentManifest.Networks,
						bideplmanifest.Network{
							Name: "fake-dynamic-network-name",
							Type: "dynamic",
							CloudProperties: biproperty.Map{
								"fake-network-cloud-property": "fake-network-cloud-property-value",
							},
						},
					)
					deploymentManifest.Jobs[0].Networks = append(
						deploymentManifest.Jobs[0].Networks,
						bideplmanifest.JobNetwork{
							Name:     "fake-dynamic-network-name",
							Defaults: []bideplmanifest.NetworkDefault{bideplmanifest.NetworkDefaultDNS, bideplmanifest.NetworkDefaultGateway},
						},
					)
					agentState.NetworkSpecs["fake-dynamic-network-name"] = biac.NetworkSpec{
						IP: "1.2.3.6",
					}
				})

				It("should not fail", func() {
					state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
					Expect(err).ToNot(HaveOccurred())

					Expect(state.NetworkInterfaces()).To(ContainElement(NetworkRef{
						Name: "fake-dynamic-network-name",
						Interface: map[string]interface{}{
							"type":    "dynamic",
							"default": []bideplmanifest.NetworkDefault{"dns", "gateway"},
							"cloud_properties": biproperty.Map{
								"fake-network-cloud-property": "fake-network-cloud-property-value",
							},
						},
					}))
					Expect(state.NetworkInterfaces()).To(HaveLen(2))
				})
			})
		})

		It("builds a new instance state with zero-to-many rendered jobs from one or more releases", func() {
			state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
			Expect(err).ToNot(HaveOccurred())

			Expect(state.RenderedJobs()).To(ContainElement(JobRef{
				Name:    "fake-release-job-name",
				Version: "fake-release-job-source-fingerprint",
			}))

			// multiple jobs are rendered in a single archive
			Expect(state.RenderedJobListArchive()).To(Equal(BlobRef{
				BlobstoreID: "fake-rendered-job-list-archive-blob-id",
				SHA1:        "fake-rendered-job-list-archive-sha1",
			}))
			Expect(state.RenderedJobs()).To(HaveLen(1))
		})

		It("prints ui stages for compiling packages and rendering job templates", func() {
			_, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				// compile stages not produced by mockDependencyCompiler
				{Name: "Rendering job templates"},
			}))
		})

		It("builds a new instance state with the compiled packages required by the release jobs", func() {
			state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
			Expect(err).ToNot(HaveOccurred())

			Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
				Name:    "cpi",
				Version: "fake-package-source-fingerprint-cpi",
				Archive: BlobRef{
					SHA1:        "fake-package-compiled-archive-sha1-cpi",
					BlobstoreID: "fake-package-compiled-archive-blob-id-cpi",
				},
			}))
			Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
				Name:    "ruby",
				Version: "fake-package-source-fingerprint-ruby",
				Archive: BlobRef{
					SHA1:        "fake-package-compiled-archive-sha1-ruby",
					BlobstoreID: "fake-package-compiled-archive-blob-id-ruby",
				},
			}))
		})

		It("builds a new instance state that includes transitively dependent compiled packages", func() {
			state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
			Expect(err).ToNot(HaveOccurred())

			Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
				Name:    "cpi",
				Version: "fake-package-source-fingerprint-cpi",
				Archive: BlobRef{
					SHA1:        "fake-package-compiled-archive-sha1-cpi",
					BlobstoreID: "fake-package-compiled-archive-blob-id-cpi",
				},
			}))
			Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
				Name:    "ruby",
				Version: "fake-package-source-fingerprint-ruby",
				Archive: BlobRef{
					SHA1:        "fake-package-compiled-archive-sha1-ruby",
					BlobstoreID: "fake-package-compiled-archive-blob-id-ruby",
				},
			}))
			Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
				Name:    "libyaml",
				Version: "fake-package-source-fingerprint-libyaml",
				Archive: BlobRef{
					SHA1:        "fake-package-compiled-archive-sha1-libyaml",
					BlobstoreID: "fake-package-compiled-archive-blob-id-libyaml",
				},
			}))
			Expect(state.CompiledPackages()).To(HaveLen(3))
		})

		Context("when multiple packages have the same dependency", func() {
			BeforeEach(func() {
				releasePackageRuby.Dependencies = append(releasePackageRuby.Dependencies, releasePackageLibyaml)
			})

			It("does not recompile dependant packages", func() {
				state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
				Expect(err).ToNot(HaveOccurred())

				Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
					Name:    "cpi",
					Version: "fake-package-source-fingerprint-cpi",
					Archive: BlobRef{
						SHA1:        "fake-package-compiled-archive-sha1-cpi",
						BlobstoreID: "fake-package-compiled-archive-blob-id-cpi",
					},
				}))
				Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
					Name:    "ruby",
					Version: "fake-package-source-fingerprint-ruby",
					Archive: BlobRef{
						SHA1:        "fake-package-compiled-archive-sha1-ruby",
						BlobstoreID: "fake-package-compiled-archive-blob-id-ruby",
					},
				}))
				Expect(state.CompiledPackages()).To(ContainElement(PackageRef{
					Name:    "libyaml",
					Version: "fake-package-source-fingerprint-libyaml",
					Archive: BlobRef{
						SHA1:        "fake-package-compiled-archive-sha1-libyaml",
						BlobstoreID: "fake-package-compiled-archive-blob-id-libyaml",
					},
				}))
				Expect(state.CompiledPackages()).To(HaveLen(3))
			})
		})

		It("builds an instance state that can be converted to an ApplySpec", func() {
			state, err := stateBuilder.Build(jobName, instanceID, deploymentManifest, fakeStage, agentState)
			Expect(err).ToNot(HaveOccurred())

			Expect(state.ToApplySpec()).To(Equal(bias.ApplySpec{
				Deployment: "fake-deployment-name",
				Index:      0,
				Networks: map[string]interface{}{
					"fake-network-name": map[string]interface{}{
						"type":    "fake-network-type",
						"default": []bideplmanifest.NetworkDefault{"dns", "gateway"},
						"ip":      "1.2.3.4",
						"cloud_properties": biproperty.Map{
							"fake-network-cloud-property": "fake-network-cloud-property-value",
						},
					},
				},
				Job: bias.Job{
					Name: "fake-deployment-job-name",
					Templates: []bias.Blob{
						{
							Name:    "fake-release-job-name",
							Version: "fake-release-job-source-fingerprint",
						},
					},
				},
				Packages: map[string]bias.Blob{
					"cpi": bias.Blob{
						Name:        "cpi",
						Version:     "fake-package-source-fingerprint-cpi",
						SHA1:        "fake-package-compiled-archive-sha1-cpi",
						BlobstoreID: "fake-package-compiled-archive-blob-id-cpi",
					},
					"ruby": bias.Blob{
						Name:        "ruby",
						Version:     "fake-package-source-fingerprint-ruby",
						SHA1:        "fake-package-compiled-archive-sha1-ruby",
						BlobstoreID: "fake-package-compiled-archive-blob-id-ruby",
					},
					"libyaml": bias.Blob{
						Name:        "libyaml",
						Version:     "fake-package-source-fingerprint-libyaml",
						SHA1:        "fake-package-compiled-archive-sha1-libyaml",
						BlobstoreID: "fake-package-compiled-archive-blob-id-libyaml",
					},
				},
				RenderedTemplatesArchive: bias.RenderedTemplatesArchiveSpec{
					BlobstoreID: "fake-rendered-job-list-archive-blob-id",
					SHA1:        "fake-rendered-job-list-archive-sha1",
				},
				ConfigurationHash: "fake-rendered-job-list-fingerprint",
			}))
		})
	})
}
