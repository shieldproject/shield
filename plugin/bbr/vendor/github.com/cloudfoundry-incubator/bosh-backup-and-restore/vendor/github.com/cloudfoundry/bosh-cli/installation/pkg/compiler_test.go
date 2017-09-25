package pkg_test

import (
	"errors"
	"path/filepath"

	fakeblobstore "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-cli/installation/blobextract/fakeblobextract"
	. "github.com/cloudfoundry/bosh-cli/installation/pkg"
	birelpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	bistatepkg "github.com/cloudfoundry/bosh-cli/state/pkg"
	mock_state_package "github.com/cloudfoundry/bosh-cli/state/pkg/mocks"
)

var _ = Describe("PackageCompiler", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		logger                  boshlog.Logger
		compiler                bistatepkg.Compiler
		runner                  *fakesys.FakeCmdRunner
		pkg                     *birelpkg.Package
		fs                      *fakesys.FakeFileSystem
		compressor              *fakecmd.FakeCompressor
		packagesDir             string
		blobstore               *fakeblobstore.FakeDigestBlobstore
		mockCompiledPackageRepo *mock_state_package.MockCompiledPackageRepo

		fakeExtractor *fakeblobextract.FakeExtractor

		dependency1 *birelpkg.Package
		dependency2 *birelpkg.Package
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
		packagesDir = "fake-packages-dir"
		runner = fakesys.NewFakeCmdRunner()
		fs = fakesys.NewFakeFileSystem()
		compressor = fakecmd.NewFakeCompressor()

		fakeExtractor = &fakeblobextract.FakeExtractor{}

		blobstore = &fakeblobstore.FakeDigestBlobstore{}
		digest := boshcrypto.MustParseMultipleDigest("fakefingerprint")

		blobstore.CreateReturns("fake-blob-id", digest, nil)

		mockCompiledPackageRepo = mock_state_package.NewMockCompiledPackageRepo(mockCtrl)

		dependency1 = birelpkg.NewPackage(NewResource("pkg-dep1-name", "", nil), nil)
		dependency2 = birelpkg.NewPackage(NewResource("pkg-dep2-name", "", nil), nil)
		pkg = birelpkg.NewExtractedPackage(NewResource("pkg1-name", "", nil), []string{"pkg-dep1-name", "pkg-dep2-name"}, "/pkg-dir", fs)
		pkg.AttachDependencies([]*birelpkg.Package{dependency1, dependency2})

		compiler = NewPackageCompiler(
			runner,
			packagesDir,
			fs,
			compressor,
			blobstore,
			mockCompiledPackageRepo,
			fakeExtractor,
			logger,
		)
	})

	Describe("Compile", func() {
		var (
			compiledPackageTarballPath string
			installPath                string

			dep1 bistatepkg.CompiledPackageRecord
			dep2 bistatepkg.CompiledPackageRecord

			expectFind *gomock.Call
			expectSave *gomock.Call
		)

		BeforeEach(func() {
			installPath = filepath.Join(packagesDir, "pkg1-name")
			compiledPackageTarballPath = filepath.Join(packagesDir, "new-tarball.tgz")
		})

		JustBeforeEach(func() {
			expectFind = mockCompiledPackageRepo.EXPECT().Find(pkg).Return(bistatepkg.CompiledPackageRecord{}, false, nil).AnyTimes()

			dep1 = bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-dependency-blobstore-id-1",
				BlobSHA1: "fake-dependency-sha1-1",
			}
			mockCompiledPackageRepo.EXPECT().Find(dependency1).Return(dep1, true, nil).AnyTimes()

			dep2 = bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-dependency-blobstore-id-2",
				BlobSHA1: "fake-dependency-sha1-2",
			}
			mockCompiledPackageRepo.EXPECT().Find(dependency2).Return(dep2, true, nil).AnyTimes()

			// packaging file created when source is extracted
			fs.WriteFileString("/pkg-dir/packaging", "")

			compressor.CompressFilesInDirTarballPath = compiledPackageTarballPath

			record := bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-blob-id",
				BlobSHA1: "fakefingerprint",
			}
			expectSave = mockCompiledPackageRepo.EXPECT().Save(pkg, record).AnyTimes()
		})

		Context("when the compiled package repo already has the package", func() {
			JustBeforeEach(func() {
				compiledPkgRecord := bistatepkg.CompiledPackageRecord{
					BlobSHA1: "fakefingerprint",
				}
				expectFind.Return(compiledPkgRecord, true, nil).Times(1)
			})

			It("skips the compilation", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(runner.RunComplexCommands)).To(Equal(0))
			})
		})

		It("installs all the dependencies for the package", func() {
			_, _, err := compiler.Compile(pkg)
			Expect(err).ToNot(HaveOccurred())

			blobstoreID, sha1, jobPath := fakeExtractor.ExtractArgsForCall(0)
			Expect(blobstoreID).To(Equal(dep1.BlobID))
			Expect(sha1).To(Equal(dep1.BlobSHA1))
			Expect(jobPath).To(Equal(filepath.Join(packagesDir, "pkg-dep1-name")))

			blobstoreID, sha1, jobPath = fakeExtractor.ExtractArgsForCall(1)
			Expect(blobstoreID).To(Equal(dep2.BlobID))
			Expect(sha1).To(Equal(dep2.BlobSHA1))
			Expect(jobPath).To(Equal(filepath.Join(packagesDir, "pkg-dep2-name")))
		})

		It("runs the packaging script in package extractedPath dir", func() {
			_, _, err := compiler.Compile(pkg)
			Expect(err).ToNot(HaveOccurred())

			expectedCmd := boshsys.Command{
				Name: "bash",
				Args: []string{"-x", "packaging"},
				Env: map[string]string{
					"BOSH_COMPILE_TARGET": "/pkg-dir",
					"BOSH_INSTALL_TARGET": installPath,
					"BOSH_PACKAGE_NAME":   "pkg1-name",
					"BOSH_PACKAGES_DIR":   packagesDir,
					"PATH":                "/usr/local/bin:/usr/bin:/bin",
				},
				UseIsolatedEnv: true,
				WorkingDir:     "/pkg-dir",
			}

			Expect(runner.RunComplexCommands).To(HaveLen(1))
			Expect(runner.RunComplexCommands[0]).To(Equal(expectedCmd))
		})

		It("compresses the compiled package", func() {
			_, _, err := compiler.Compile(pkg)
			Expect(err).ToNot(HaveOccurred())

			Expect(compressor.CompressFilesInDirDir).To(Equal(installPath))
			Expect(compressor.CleanUpTarballPath).To(Equal(compiledPackageTarballPath))
		})

		It("moves the compressed package to a blobstore", func() {
			_, _, err := compiler.Compile(pkg)
			Expect(err).ToNot(HaveOccurred())

			Expect(blobstore.CreateArgsForCall(0)).To(Equal(compiledPackageTarballPath))
		})

		It("stores the compiled package blobID and fingerprint into the compile package repo", func() {
			expectSave.Times(1)

			_, _, err := compiler.Compile(pkg)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the repo record", func() {
			record, _, err := compiler.Compile(pkg)
			Expect(err).ToNot(HaveOccurred())

			Expect(record).To(Equal(bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-blob-id",
				BlobSHA1: "fakefingerprint",
			}))
		})

		It("cleans up the packages dir", func() {
			_, _, err := compiler.Compile(pkg)
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists(packagesDir)).To(BeFalse())
		})

		Context("when dependency installation fails", func() {
			JustBeforeEach(func() {
				fakeExtractor.ExtractReturns(errors.New("fake-install-error"))
			})

			It("returns an error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-install-error"))
			})
		})

		Context("when the packaging script does not exist", func() {
			JustBeforeEach(func() {
				err := fs.RemoveAll(filepath.Join("/", "pkg-dir", "packaging"))
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Packaging script for package 'pkg1-name' not found"))
			})
		})

		Context("when the packaging script fails", func() {
			JustBeforeEach(func() {
				fakeResult := fakesys.FakeCmdResult{
					ExitStatus: 1,
					Error:      errors.New("fake-error"),
				}
				runner.AddCmdResult("bash -x packaging", fakeResult)
			})

			It("returns error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Compiling package"))
				Expect(err.Error()).To(ContainSubstring("fake-error"))
			})
		})

		Context("when compression fails", func() {
			JustBeforeEach(func() {
				compressor.CompressFilesInDirErr = errors.New("fake-compression-error")
			})

			It("returns error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Compressing compiled package"))
			})
		})

		Context("when adding to blobstore fails", func() {
			JustBeforeEach(func() {
				blobstore.CreateReturns("", boshcrypto.MultipleDigest{}, errors.New("fake-error"))
			})

			It("returns error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Creating blob"))
				Expect(err.Error()).To(ContainSubstring("fake-error"))
			})
		})

		Context("when saving to the compiled package repo fails", func() {
			JustBeforeEach(func() {
				expectSave.Return(errors.New("fake-error")).Times(1)
			})

			It("returns error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Saving compiled package"))
				Expect(err.Error()).To(ContainSubstring("fake-error"))
			})
		})

		Context("when creating packages dir fails", func() {
			JustBeforeEach(func() {
				fs.RegisterMkdirAllError(installPath, errors.New("fake-error"))
			})

			It("returns error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Creating package install dir"))
				Expect(err.Error()).To(ContainSubstring("fake-error"))
			})
		})

		Context("when finding compiled package in the repo fails", func() {
			JustBeforeEach(func() {
				expectFind.Return(bistatepkg.CompiledPackageRecord{}, false, errors.New("fake-error")).Times(1)
			})

			It("returns an error", func() {
				_, _, err := compiler.Compile(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Attempting to find compiled package 'pkg1-name'"))
				Expect(err.Error()).To(ContainSubstring("fake-error"))
			})
		})
	})
})
