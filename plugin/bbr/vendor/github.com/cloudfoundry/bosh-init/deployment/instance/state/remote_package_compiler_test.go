package state_test

import (
	. "github.com/cloudfoundry/bosh-init/deployment/instance/state"

	mock_agentclient "github.com/cloudfoundry/bosh-init/agentclient/mocks"
	mock_blobstore "github.com/cloudfoundry/bosh-init/blobstore/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	biagentclient "github.com/cloudfoundry/bosh-agent/agentclient"
	biindex "github.com/cloudfoundry/bosh-init/index"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bistatepkg "github.com/cloudfoundry/bosh-init/state/pkg"
)

var _ = Describe("RemotePackageCompiler", describeRemotePackageCompiler)

func describeRemotePackageCompiler() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		packageRepo bistatepkg.CompiledPackageRepo

		pkgDependency *birelpkg.Package
		pkg           *birelpkg.Package

		mockBlobstore   *mock_blobstore.MockBlobstore
		mockAgentClient *mock_agentclient.MockAgentClient

		archivePath = "fake-archive-path"

		remotePackageCompiler bistatepkg.Compiler

		compiledPackages map[bistatepkg.CompiledPackageRecord]*birelpkg.Package

		expectBlobstoreAdd *gomock.Call
		expectAgentCompile *gomock.Call
	)

	BeforeEach(func() {
		mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)
		mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)

		index := biindex.NewInMemoryIndex()
		packageRepo = bistatepkg.NewCompiledPackageRepo(index)
		remotePackageCompiler = NewRemotePackageCompiler(mockBlobstore, mockAgentClient, packageRepo)

		pkgDependency = &birelpkg.Package{
			Name:        "fake-package-name-dep",
			Fingerprint: "fake-package-fingerprint-dep",
		}

		pkg = &birelpkg.Package{
			Name:         "fake-package-name",
			Fingerprint:  "fake-package-fingerprint",
			SHA1:         "fake-source-package-sha1",
			ArchivePath:  archivePath,
			Dependencies: []*birelpkg.Package{pkgDependency},
		}

		depRecord1 := bistatepkg.CompiledPackageRecord{
			BlobID:   "fake-compiled-package-blob-id-dep",
			BlobSHA1: "fake-compiled-package-sha1-dep",
		}

		compiledPackages = map[bistatepkg.CompiledPackageRecord]*birelpkg.Package{
			depRecord1: pkgDependency,
		}
	})

	JustBeforeEach(func() {
		// add compiled packages to the repo
		for record, dependency := range compiledPackages {
			err := packageRepo.Save(*dependency, record)
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

	Describe("Compile", func() {
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

			record, found, err := packageRepo.Find(*pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(record).To(Equal(compiledPackageRecord))
		})

		Context("when the dependencies are not in the repo", func() {
			BeforeEach(func() {
				compiledPackages = map[bistatepkg.CompiledPackageRecord]*birelpkg.Package{}
			})

			It("returns an error", func() {
				_, _, err := remotePackageCompiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Remote compilation failure: Package 'fake-package-name/fake-package-fingerprint' requires package 'fake-package-name-dep/fake-package-fingerprint-dep', but it has not been compiled"))
			})
		})

		Context("when package belongs to a compiled release", func() {
			BeforeEach(func() {
				pkg.Stemcell = "ubuntu/fake"
			})

			AfterEach(func() {
				pkg.Stemcell = ""
			})

			It("should skip compilation", func() {
				compiledPackageRecord, isAlreadyCompiled, err := remotePackageCompiler.Compile(pkg)

				expectAgentCompile.Times(0)

				Expect(err).ToNot(HaveOccurred())
				Expect(isAlreadyCompiled).To(Equal(true))
				Expect(compiledPackageRecord.BlobID).To(Equal("fake-source-package-blob-id"))
				Expect(compiledPackageRecord.BlobSHA1).To(Equal(pkg.SHA1))
			})
		})
	})
}
