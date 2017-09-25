package compiler_test

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakebc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection/fakes"
	boshmodels "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	fakepackages "github.com/cloudfoundry/bosh-agent/agent/applier/packages/fakes"
	fakecmdrunner "github.com/cloudfoundry/bosh-agent/agent/cmdrunner/fakes"
	. "github.com/cloudfoundry/bosh-agent/agent/compiler"
	fakeblobstore "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

type FakeCompileDirProvider struct {
	Dir string
}

func (cdp FakeCompileDirProvider) CompileDir() string { return cdp.Dir }

func getCompileArgs() (Package, []boshmodels.Package) {
	pkg := Package{
		BlobstoreID: "blobstore_id",
		Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "sha1")),
		Name:        "pkg_name",
		Version:     "pkg_version",
	}

	pkgDeps := []boshmodels.Package{
		{
			Name:    "first_dep_name",
			Version: "first_dep_version",
			Source: boshmodels.Source{
				Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "first_dep_sha1")),
				BlobstoreID: "first_dep_blobstore_id",
			},
		},
		{
			Name:    "sec_dep_name",
			Version: "sec_dep_version",
			Source: boshmodels.Source{
				Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "sec_dep_sha1")),
				BlobstoreID: "sec_dep_blobstore_id",
			},
		},
	}

	return pkg, pkgDeps
}

func init() {
	Describe("concreteCompiler", func() {
		var (
			compiler       Compiler
			compressor     *fakecmd.FakeCompressor
			blobstore      *fakeblobstore.FakeDigestBlobstore
			fs             *fakesys.FakeFileSystem
			runner         *fakecmdrunner.FakeFileLoggingCmdRunner
			packageApplier *fakepackages.FakeApplier
			packagesBc     *fakebc.FakeBundleCollection
		)

		BeforeEach(func() {
			compressor = fakecmd.NewFakeCompressor()
			blobstore = &fakeblobstore.FakeDigestBlobstore{}
			fs = fakesys.NewFakeFileSystem()
			runner = fakecmdrunner.NewFakeFileLoggingCmdRunner()
			packageApplier = fakepackages.NewFakeApplier()
			packagesBc = fakebc.NewFakeBundleCollection()

			compiler = NewConcreteCompiler(
				compressor,
				blobstore,
				fs,
				runner,
				FakeCompileDirProvider{Dir: "/fake-compile-dir"},
				packageApplier,
				packagesBc,
			)

			fs.MkdirAll("/fake-compile-dir", os.ModePerm)
			Expect(fs.WriteFileString("/tmp/compressed-compiled-package", "fake-contents")).ToNot(HaveOccurred())
		})

		Describe("Compile", func() {
			var (
				bundle  *fakebc.FakeBundle
				pkg     Package
				pkgDeps []boshmodels.Package
			)

			BeforeEach(func() {
				bundle = packagesBc.FakeGet(boshmodels.LocalPackage{
					Name:    "pkg_name",
					Version: "pkg_version",
				})

				bundle.InstallPath = "/fake-dir/data/packages/pkg_name/pkg_version"
				bundle.EnablePath = "/fake-dir/packages/pkg_name"

				compressor.CompressFilesInDirTarballPath = "/tmp/compressed-compiled-package"

				pkg, pkgDeps = getCompileArgs()
			})

			It("returns blob id and sha1 of created compiled package", func() {
				blobstore.CreateReturns("fake-blob-id", boshcrypto.MultipleDigest{}, nil)

				blobID, digest, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())

				Expect(blobID).To(Equal("fake-blob-id"))
				Expect(digest).To(Equal(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "978ad524a02039f261773fe93d94973ae7de6470")))
			})

			It("returns blob id and correct sha algo of created compiled package", func() {
				blobstore.CreateReturns("fake-blob-id", boshcrypto.MultipleDigest{}, nil)

				// Currently algo of source package is used for compilation pkg algo
				pkg.Sha1 = boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA256, "fakesha"))

				_, digest, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())
				// echo -n fake-contents|shasum -a 256
				Expect(digest.String()).To(Equal("sha256:d12d3a3ee8dcdc9e7ea3416fd618298ea50abde2cf434313c6c3edb213f441cd"))

				blobID, fingerprint := blobstore.GetArgsForCall(0)
				Expect(blobID).To(Equal("blobstore_id"))
				Expect(fingerprint).To(Equal(pkg.Sha1))
			})

			It("cleans up all packages before and after applying dependent packages", func() {
				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())
				Expect(packageApplier.ActionsCalled).To(Equal([]string{"KeepOnly", "Apply", "Apply", "KeepOnly"}))
				Expect(packageApplier.KeptOnlyPackages).To(BeEmpty())
			})

			It("returns an error if cleaning up packages fails", func() {
				packageApplier.KeepOnlyErr = errors.New("fake-keep-only-error")

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-keep-only-error"))
			})

			It("returns an error if removing compile target directory during uncompression fails", func() {
				fs.RemoveAllStub = func(path string) error {
					if path == "/fake-compile-dir/pkg_name" {
						return errors.New("fake-remove-error")
					}
					return nil
				}

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-remove-error"))
			})

			It("returns an error if creating compile target directory during uncompression fails", func() {
				fs.RemoveAllStub = func(path string) error {
					if path == "/fake-compile-dir/pkg_name" {
						return errors.New("fake-mkdir-error")
					}
					return nil
				}

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mkdir-error"))
			})

			It("returns an error if removing temporary compile target directory during uncompression fails", func() {
				fs.RemoveAllStub = func(path string) error {
					if path == "/fake-compile-dir/pkg_name-bosh-agent-unpack" {
						return errors.New("fake-remove-error")
					}
					return nil
				}

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-remove-error"))
			})

			It("returns an error if creating temporary compile target directory during uncompression fails", func() {
				fs.RegisterMkdirAllError("/fake-compile-dir/pkg_name-bosh-agent-unpack", errors.New("fake-mkdir-error"))

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mkdir-error"))
			})

			It("returns an error if target directory is empty during uncompression", func() {
				pkg.BlobstoreID = ""

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Blobstore ID for package '%s' is empty", pkg.Name))
			})

			It("installs dependent packages", func() {
				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())
				Expect(packageApplier.AppliedPackages).To(Equal(pkgDeps))
			})

			It("cleans up the compile directory", func() {
				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())
				Expect(fs.FileExists("/fake-compile-dir/pkg_name")).To(BeFalse())
			})

			It("installs, enables and later cleans up bundle", func() {
				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())
				Expect(bundle.ActionsCalled).To(Equal([]string{
					"InstallWithoutContents",
					"Enable",
					"Disable",
					"Uninstall",
				}))
			})

			It("returns an error if removing the compile directory fails", func() {
				callCount := 0
				fs.RemoveAllStub = func(path string) error {
					if path == "/fake-compile-dir/pkg_name" {
						callCount++
						if callCount > 1 {
							return errors.New("fake-remove-error")
						}
					}
					return nil
				}

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-remove-error"))
			})

			Context("when packaging script exists", func() {
				const packagingScriptContents = "hi"
				BeforeEach(func() {
					compressor.DecompressFileToDirCallBack = func() {
						filename := "/fake-compile-dir/pkg_name/" + PackagingScriptName
						fs.WriteFileString(filename, packagingScriptContents)
					}
				})

				It("runs packaging script ", func() {
					_, _, err := compiler.Compile(pkg, pkgDeps)
					Expect(err).ToNot(HaveOccurred())

					expectedCmd := boshsys.Command{
						Env: map[string]string{
							"BOSH_COMPILE_TARGET":  "/fake-compile-dir/pkg_name",
							"BOSH_INSTALL_TARGET":  "/fake-dir/packages/pkg_name",
							"BOSH_PACKAGE_NAME":    "pkg_name",
							"BOSH_PACKAGE_VERSION": "pkg_version",
						},
						WorkingDir: "/fake-compile-dir/pkg_name",
					}

					cmd := runner.RunCommands[0]
					if runtime.GOOS == "windows" {
						expectedCmd.Name = "powershell"
						expectedCmd.Args = []string{"-command", fmt.Sprintf(`"iex (get-content -raw %s)"`, PackagingScriptName)}
					} else {
						expectedCmd.Name = "bash"
						expectedCmd.Args = []string{"-x", PackagingScriptName}
					}

					Expect(cmd).To(Equal(expectedCmd))
					Expect(len(runner.RunCommands)).To(Equal(1))
					Expect(runner.RunCommandJobName).To(Equal("compilation"))
					Expect(runner.RunCommandTaskName).To(Equal(PackagingScriptName))
				})

				It("propagates the error from packaging script", func() {
					runner.RunCommandErr = errors.New("fake-packaging-error")

					_, _, err := compiler.Compile(pkg, pkgDeps)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-packaging-error"))
				})
			})

			It("does not run packaging script when script does not exist", func() {
				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())
				Expect(runner.RunCommands).To(BeEmpty())
			})

			It("compresses compiled package", func() {
				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())

				// archive was downloaded from the blobstore and decompress to this temp dir
				Expect(compressor.DecompressFileToDirDirs[0]).To(Equal("/fake-compile-dir/pkg_name-bosh-agent-unpack"))
				Expect(compressor.DecompressFileToDirTarballPaths[0]).To(BeEmpty())

				// contents were moved from the temp dir to the install/enable dir
				Expect(fs.RenameOldPaths[0]).To(Equal("/fake-compile-dir/pkg_name-bosh-agent-unpack"))
				Expect(fs.RenameNewPaths[0]).To(Equal("/fake-compile-dir/pkg_name"))

				// install path, presumably with your packaged code, was compressed
				installPath := "/fake-dir/data/packages/pkg_name/pkg_version"
				Expect(compressor.CompressFilesInDirDir).To(Equal(installPath))
			})

			It("uploads compressed package to blobstore", func() {
				compressor.CompressFilesInDirTarballPath = "/tmp/compressed-compiled-package"

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())
				Expect(blobstore.CreateArgsForCall(0)).To(Equal("/tmp/compressed-compiled-package"))
			})

			It("returs error if uploading compressed package fails", func() {
				blobstore.CreateReturns("", boshcrypto.MultipleDigest{}, errors.New("fake-create-err"))

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-create-err"))
			})

			It("cleans up compressed package after uploading it to blobstore", func() {
				var beforeCleanUpTarballPath, afterCleanUpTarballPath string

				blobstore.CreateStub = func(fileName string) (blobID string, digest boshcrypto.MultipleDigest, err error) {
					beforeCleanUpTarballPath = compressor.CleanUpTarballPath
					return "my-blob-id", boshcrypto.MultipleDigest{}, nil
				}

				_, _, err := compiler.Compile(pkg, pkgDeps)
				Expect(err).ToNot(HaveOccurred())

				// Compressed package is not cleaned up before blobstore upload
				Expect(beforeCleanUpTarballPath).To(Equal(""))

				// Deleted after it was uploaded
				afterCleanUpTarballPath = compressor.CleanUpTarballPath
				Expect(afterCleanUpTarballPath).To(Equal("/tmp/compressed-compiled-package"))
			})
		})
	})
}
