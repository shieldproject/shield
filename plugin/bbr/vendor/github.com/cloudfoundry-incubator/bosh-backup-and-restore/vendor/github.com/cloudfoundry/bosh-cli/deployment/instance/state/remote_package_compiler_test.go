package state_test

import (
	biagentclient "github.com/cloudfoundry/bosh-agent/agentclient"
	mock_agentclient "github.com/cloudfoundry/bosh-cli/agentclient/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mock_blobstore "github.com/cloudfoundry/bosh-cli/blobstore/mocks"
	. "github.com/cloudfoundry/bosh-cli/deployment/instance/state"
	biindex "github.com/cloudfoundry/bosh-cli/index"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	bistatepkg "github.com/cloudfoundry/bosh-cli/state/pkg"
)

var _ = Describe("RemotePackageCompiler", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		packageRepo bistatepkg.CompiledPackageRepo

		mockBlobstore   *mock_blobstore.MockBlobstore
		mockAgentClient *mock_agentclient.MockAgentClient

		archivePath = "fake-archive-path"

		remotePackageCompiler bistatepkg.Compiler

		expectBlobstoreAdd *gomock.Call
		expectAgentCompile *gomock.Call
	)

	BeforeEach(func() {
		mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)
		mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)

		index := biindex.NewInMemoryIndex()
		packageRepo = bistatepkg.NewCompiledPackageRepo(index)
		remotePackageCompiler = NewRemotePackageCompiler(mockBlobstore, mockAgentClient, packageRepo)
	})

	Describe("Compile", func() {
		Context("when package is not compiled", func() {
			var (
				pkgDependency *boshpkg.Package
				pkg           *boshpkg.Package

				compiledPackages map[bistatepkg.CompiledPackageRecord]*boshpkg.Package
			)

			BeforeEach(func() {
				pkgDependency = boshpkg.NewPackage(NewResource(
					"fake-package-name-dep", "fake-package-fingerprint-dep", nil), nil)

				pkg = boshpkg.NewPackage(NewResourceWithBuiltArchive(
					"fake-package-name", "fake-package-fingerprint", archivePath, "fake-source-package-sha1"), []string{"fake-package-name-dep"})
				pkg.AttachDependencies([]*boshpkg.Package{pkgDependency})

				depRecord1 := bistatepkg.CompiledPackageRecord{
					BlobID:   "fake-compiled-package-blob-id-dep",
					BlobSHA1: "fake-compiled-package-sha1-dep",
				}

				compiledPackages = map[bistatepkg.CompiledPackageRecord]*boshpkg.Package{
					depRecord1: pkgDependency,
				}
			})

			JustBeforeEach(func() {
				// add compiled packages to the repo
				for record, dependency := range compiledPackages {
					err := packageRepo.Save(dependency, record)
					Expect(err).ToNot(HaveOccurred())
				}

				packageSource := biagentclient.BlobRef{
					Name:        "fake-package-name",
					Version:     "fake-package-fingerprint",
					BlobstoreID: "fake-source-package-blob-id",
					SHA1:        "fake-source-package-sha1",
				}
				packageDependencies := []biagentclient.BlobRef{
					{
						Name:        "fake-package-name-dep",
						Version:     "fake-package-fingerprint-dep",
						BlobstoreID: "fake-compiled-package-blob-id-dep",
						SHA1:        "fake-compiled-package-sha1-dep",
					},
				}
				compiledPackageRef := biagentclient.BlobRef{
					Name:        "fake-package-name",
					Version:     "fake-package-version",
					BlobstoreID: "fake-compiled-package-blob-id",
					SHA1:        "fake-compiled-package-sha1",
				}

				expectBlobstoreAdd = mockBlobstore.EXPECT().Add(archivePath).Return("fake-source-package-blob-id", nil).AnyTimes()
				expectAgentCompile = mockAgentClient.EXPECT().CompilePackage(packageSource, packageDependencies).Return(compiledPackageRef, nil).AnyTimes()
			})

			It("uploads the package archive to the blobstore and then compiles the package with the agent", func() {
				gomock.InOrder(
					expectBlobstoreAdd.Times(1),
					expectAgentCompile.Times(1),
				)

				compiledPackageRecord, _, err := remotePackageCompiler.Compile(pkg)
				Expect(err).ToNot(HaveOccurred())
				Expect(compiledPackageRecord).To(Equal(bistatepkg.CompiledPackageRecord{
					BlobID:   "fake-compiled-package-blob-id",
					BlobSHA1: "fake-compiled-package-sha1",
				}))
			})

			It("saves the compiled package ref in the package repo", func() {
				compiledPackageRecord, _, err := remotePackageCompiler.Compile(pkg)
				Expect(err).ToNot(HaveOccurred())

				record, found, err := packageRepo.Find(pkg)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(record).To(Equal(compiledPackageRecord))
			})

			Context("when the dependencies are not in the repo", func() {
				BeforeEach(func() {
					compiledPackages = map[bistatepkg.CompiledPackageRecord]*boshpkg.Package{}
				})

				It("returns an error", func() {
					_, _, err := remotePackageCompiler.Compile(pkg)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Remote compilation failure: Package 'fake-package-name/fake-package-fingerprint' requires package 'fake-package-name-dep/fake-package-fingerprint-dep', but it has not been compiled"))
				})
			})
		})

		Context("when package is compiled", func() {
			var (
				pkgDependency *boshpkg.CompiledPackage
				pkg           *boshpkg.CompiledPackage
			)

			BeforeEach(func() {
				pkgDependency = boshpkg.NewCompiledPackageWithoutArchive(
					"fake-package-name-dep", "fake-package-fingerprint-dep", "", "", nil)

				pkg = boshpkg.NewCompiledPackageWithArchive(
					"fake-package-name", "fake-package-fingerprint", "", archivePath, "fake-source-package-sha1", []string{"fake-package-name-dep"})
				pkg.AttachDependencies([]*boshpkg.CompiledPackage{pkgDependency})
			})

			It("should skip compilation but still add blobstore package", func() {
				err := packageRepo.Save(pkgDependency, bistatepkg.CompiledPackageRecord{
					BlobID:   "fake-compiled-package-blob-id-dep",
					BlobSHA1: "fake-compiled-package-sha1-dep",
				})
				Expect(err).ToNot(HaveOccurred())

				expectBlobstoreAdd = mockBlobstore.EXPECT().Add(archivePath).Return("fake-source-package-blob-id", nil).AnyTimes()
				expectAgentCompile = mockAgentClient.EXPECT().CompilePackage(gomock.Any(), gomock.Any()).AnyTimes()

				compiledPackageRecord, isAlreadyCompiled, err := remotePackageCompiler.Compile(pkg)
				Expect(err).ToNot(HaveOccurred())
				Expect(isAlreadyCompiled).To(Equal(true))
				Expect(compiledPackageRecord).To(Equal(bistatepkg.CompiledPackageRecord{
					BlobID:   "fake-source-package-blob-id",
					BlobSHA1: "fake-source-package-sha1",
				}))

				expectAgentCompile.Times(0)
			})
		})
	})
})
