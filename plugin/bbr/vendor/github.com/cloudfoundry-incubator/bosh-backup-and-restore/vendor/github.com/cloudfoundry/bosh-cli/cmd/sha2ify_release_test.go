package cmd_test

import (
	. "github.com/cloudfoundry/bosh-cli/cmd"

	. "github.com/cloudfoundry/bosh-cli/release/resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshrel "github.com/cloudfoundry/bosh-cli/release"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"

	fakecrypto "github.com/cloudfoundry/bosh-cli/crypto/fakes"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	fakefu "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	fakes2 "github.com/cloudfoundry/bosh-utils/system/fakes"

	"github.com/cloudfoundry/bosh-cli/crypto/fakes"
	"github.com/cloudfoundry/bosh-cli/release/license"
	"github.com/cloudfoundry/bosh-utils/errors"
)

var _ = Describe("Sha2ifyRelease", func() {

	var (
		releaseReader                *fakerel.FakeReader
		ui                           *fakeui.FakeUI
		fmv                          *fakefu.FakeMover
		releaseWriter                *fakerel.FakeWriter
		command                      Sha2ifyReleaseCmd
		args                         Sha2ifyReleaseArgs
		fakeDigestCalculator         *fakes.FakeDigestCalculator
		releaseWriterTempDestination string
		fs                           *fakes2.FakeFileSystem
	)

	BeforeEach(func() {
		releaseReader = &fakerel.FakeReader{}
		releaseWriter = &fakerel.FakeWriter{}
		ui = &fakeui.FakeUI{}
		fmv = &fakefu.FakeMover{}
		fs = fakes2.NewFakeFileSystem()

		fakeDigestCalculator = fakes.NewFakeDigestCalculator()
		command = NewSha2ifyReleaseCmd(releaseReader, releaseWriter, fakeDigestCalculator, fmv, fs, ui)
	})
	var fakeSha128Release *fakerel.FakeRelease

	job1ResourcePath := "/job-resource-1-path"
	pkg1ResourcePath := "/pkg-resource-1-path"
	compiledPackage1ResourcePath := "/compiled-pkg-resource-path"
	licenseResourcePath := "/license-resource-path"
	fileContentSha1 := "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"

	createFakeFileWithKnownSha1 := func() *fakes2.FakeFile {
		return &fakes2.FakeFile{
			Contents: []byte("hello world"),
		}
	}

	BeforeEach(func() {
		args = Sha2ifyReleaseArgs{
			Path:        "/some/release_128.tgz",
			Destination: FileArg{ExpandedPath: "/some/release_256.tgz"},
		}

		fs.RegisterOpenFile(job1ResourcePath, createFakeFileWithKnownSha1())
		fs.RegisterOpenFile(pkg1ResourcePath, createFakeFileWithKnownSha1())
		fs.RegisterOpenFile(compiledPackage1ResourcePath, createFakeFileWithKnownSha1())
		fs.RegisterOpenFile(licenseResourcePath, createFakeFileWithKnownSha1())

		fakeSha128Release = &fakerel.FakeRelease{}
		jobSha128 := boshjob.NewJob(NewResourceWithBuiltArchive("job-resource-1", "job-sha128-fp", job1ResourcePath, fileContentSha1))
		packageSha128 := boshpkg.NewPackage(NewResourceWithBuiltArchive("pkg-resource-1", "pkg-sha128-fp", pkg1ResourcePath, fileContentSha1), nil)
		compiledPackageSha128 := boshpkg.NewCompiledPackageWithArchive("compiledpkg-resource-1", "compiledpkg-sha128-fp", "1", compiledPackage1ResourcePath, fileContentSha1, nil)

		fakeSha128Release.JobsReturns([]*boshjob.Job{jobSha128})
		fakeSha128Release.PackagesReturns([]*boshpkg.Package{packageSha128})
		fakeSha128Release.LicenseReturns(license.NewLicense(NewResourceWithBuiltArchive("license-resource-path", "lic-sha128-fp", licenseResourcePath, fileContentSha1)))
		fakeSha128Release.CompiledPackagesReturns([]*boshpkg.CompiledPackage{compiledPackageSha128})

		fakeSha128Release.CopyWithStub = func(jobs []*boshjob.Job, pkgs []*boshpkg.Package, lic *license.License, compiledPackages []*boshpkg.CompiledPackage) boshrel.Release {
			fakeSha256Release := &fakerel.FakeRelease{}
			fakeSha256Release.NameReturns("custom-name")
			fakeSha256Release.VersionReturns("custom-ver")
			fakeSha256Release.CommitHashWithMarkReturns("commit")
			fakeSha256Release.JobsReturns(jobs)
			fakeSha256Release.PackagesReturns(pkgs)
			fakeSha256Release.LicenseReturns(lic)
			fakeSha256Release.CompiledPackagesReturns(compiledPackages)
			return fakeSha256Release
		}

		fakeDigestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
			job1ResourcePath:             {DigestStr: "sha256:jobsha256"},
			pkg1ResourcePath:             {DigestStr: "sha256:pkgsha256"},
			licenseResourcePath:          {DigestStr: "sha256:licsha256"},
			compiledPackage1ResourcePath: {DigestStr: "sha256:compiledpkgsha256"},
		})

		releaseReader.ReadReturns(fakeSha128Release, nil)
		releaseWriterTempDestination = "/some/temp/release_256.tgz"
		releaseWriter.WriteReturns(releaseWriterTempDestination, nil)
	})

	Context("Given a valid sha128 release tar", func() {
		It("Should convert it to a sha256 release tar", func() {
			err := command.Run(args)
			Expect(err).ToNot(HaveOccurred())

			Expect(releaseReader.ReadCallCount()).ToNot(Equal(0))

			readPathArg := releaseReader.ReadArgsForCall(0)
			Expect(readPathArg).To(Equal("/some/release_128.tgz"))

			Expect(releaseWriter.WriteCallCount()).To(Equal(1))
			sha2ifyRelease, _ := releaseWriter.WriteArgsForCall(0)

			Expect(sha2ifyRelease).NotTo(BeNil())

			Expect(sha2ifyRelease.License()).ToNot(BeNil())
			Expect(sha2ifyRelease.License().ArchiveSHA1()).To(Equal("sha256:licsha256"))

			Expect(sha2ifyRelease.Jobs()).To(HaveLen(1))
			Expect(sha2ifyRelease.Jobs()[0].ArchiveSHA1()).To(Equal("sha256:jobsha256"))

			Expect(sha2ifyRelease.Packages()).To(HaveLen(1))
			Expect(sha2ifyRelease.Packages()[0].ArchiveSHA1()).To(Equal("sha256:pkgsha256"))

			Expect(sha2ifyRelease.CompiledPackages()).To(HaveLen(1))
			Expect(sha2ifyRelease.CompiledPackages()[0].ArchiveSHA1()).To(Equal("sha256:compiledpkgsha256"))

			Expect(fmv.MoveCallCount()).To(Equal(1))

			src, dst := fmv.MoveArgsForCall(0)
			Expect(src).To(Equal(releaseWriterTempDestination))
			Expect(dst).To(Equal(args.Destination.ExpandedPath))

			Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
				Header: []boshtbl.Header{
					boshtbl.NewHeader("Name"),
					boshtbl.NewHeader("Version"),
					boshtbl.NewHeader("Commit Hash"),
					boshtbl.NewHeader("Archive"),
				},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("custom-name"),
						boshtbl.NewValueString("custom-ver"),
						boshtbl.NewValueString("commit"),
						boshtbl.NewValueString("/some/release_256.tgz"),
					},
				},
				Transpose: true,
			}))
		})

		Context("when unable to write the sha256 tarball", func() {
			BeforeEach(func() {
				releaseWriter.WriteReturns("", errors.Error("disaster"))
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("disaster"))
			})
		})

		Context("when rehashing a licence fails", func() {
			BeforeEach(func() {
				fakeDigestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
					job1ResourcePath:             {DigestStr: "sha256:jobsha256"},
					pkg1ResourcePath:             {DigestStr: "sha256:pkgsha256"},
					compiledPackage1ResourcePath: {DigestStr: "sha256:compiledpkgsha256"},
					licenseResourcePath:          {Err: errors.Error("Unknown algorithm")},
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unknown algorithm"))
			})
		})

		Context("when rehashing compiled packages fails", func() {
			BeforeEach(func() {
				fakeDigestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
					job1ResourcePath:             {DigestStr: "sha256:jobsha256"},
					pkg1ResourcePath:             {DigestStr: "sha256:pkgsha256"},
					compiledPackage1ResourcePath: {Err: errors.Error("Unknown algorithm")},
					licenseResourcePath:          {DigestStr: "sha256:licsha256"},
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unknown algorithm"))
			})
		})

		Context("when rehashing packages fails", func() {
			BeforeEach(func() {
				fakeDigestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
					job1ResourcePath:             {DigestStr: "sha256:jobsha256"},
					pkg1ResourcePath:             {Err: errors.Error("Unknown algorithm")},
					compiledPackage1ResourcePath: {DigestStr: "sha256:compiledpkgsha256"},
					licenseResourcePath:          {DigestStr: "sha256:licsha256"},
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unknown algorithm"))
			})
		})

		Context("when rehashing jobs fails", func() {
			BeforeEach(func() {
				fakeDigestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
					job1ResourcePath:             {Err: errors.Error("Unknown algorithm")},
					pkg1ResourcePath:             {DigestStr: "sha256:pkgsha256"},
					compiledPackage1ResourcePath: {DigestStr: "sha256:compiledpkgsha256"},
					licenseResourcePath:          {DigestStr: "sha256:licsha256"},
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unknown algorithm"))
			})
		})

		Context("when no licence is provided", func() {
			BeforeEach(func() {
				fakeSha128Release.LicenseReturns(nil)
				fakeDigestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
					job1ResourcePath:             {DigestStr: "sha256:jobsha256"},
					pkg1ResourcePath:             {DigestStr: "sha256:pkgsha256"},
					compiledPackage1ResourcePath: {DigestStr: "sha256:compiledpkgsha256"},
				})
			})

			It("should not return an error", func() {
				err := command.Run(args)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When unable to move sha2fyied release to destination", func() {
			BeforeEach(func() {
				fmv.MoveReturns(errors.Error("disaster"))
			})

			It("Should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("disaster"))
			})
		})
	})

	Context("Given an invalid sha128 release tar", func() {
		Context("Given a job that does not verify", func() {
			BeforeEach(func() {
				fs.RegisterOpenFile(job1ResourcePath, &fakes2.FakeFile{
					Contents: []byte("content that does not match expected sha1"),
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Expected stream to have digest"))

			})
		})

		Context("Given a package that does not verify", func() {
			BeforeEach(func() {
				fs.RegisterOpenFile(pkg1ResourcePath, &fakes2.FakeFile{
					Contents: []byte("content that does not match expected sha1"),
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Expected stream to have digest"))

			})
		})

		Context("Given a compiled package that does not verify", func() {
			BeforeEach(func() {
				fs.RegisterOpenFile(compiledPackage1ResourcePath, &fakes2.FakeFile{
					Contents: []byte("content that does not match expected sha1"),
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Expected stream to have digest"))

			})
		})

		Context("Given a license that does not verify", func() {
			BeforeEach(func() {
				fs.RegisterOpenFile(licenseResourcePath, &fakes2.FakeFile{
					Contents: []byte("content that does not match expected sha1"),
				})
			})

			It("should return an error", func() {
				err := command.Run(args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Expected stream to have digest"))

			})
		})
	})

	Context("Given a bad file path", func() {
		BeforeEach(func() {
			args = Sha2ifyReleaseArgs{
				Path:        "/some/release_128.tgz",
				Destination: FileArg{ExpandedPath: "/some/release_256.tgz"},
			}

			releaseReader.ReadReturns(nil, errors.Error("disaster"))
		})

		It("Should return an error", func() {
			err := command.Run(args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("disaster"))
		})
	})
})
