package release_test

import (
	"errors"
	"os"
	"path/filepath"

	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	fakejob "github.com/cloudfoundry/bosh-cli/release/job/jobfakes"
	boshlic "github.com/cloudfoundry/bosh-cli/release/license"
	boshman "github.com/cloudfoundry/bosh-cli/release/manifest"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	fakepkg "github.com/cloudfoundry/bosh-cli/release/pkg/pkgfakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("ArchiveReader", func() {
	var (
		jobReader  *fakejob.FakeArchiveReader
		pkgReader  *fakepkg.FakeArchiveReader
		fs         *fakesys.FakeFileSystem
		compressor *fakecmd.FakeCompressor
		reader     ArchiveReader
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		fs.TempDirDir = filepath.Join("/", "extracted", "release")

		compressor = fakecmd.NewFakeCompressor()
		logger := boshlog.NewLogger(boshlog.LevelNone)

		jobReader = &fakejob.FakeArchiveReader{}
		pkgReader = &fakepkg.FakeArchiveReader{}
		reader = NewArchiveReader(jobReader, pkgReader, compressor, fs, logger)
	})

	Describe("Read", func() {
		act := func() (Release, error) { return reader.Read(filepath.Join("/", "some", "release.tgz")) }

		Context("when the given release archive is a valid tar", func() {
			Context("when manifest that includes jobs and packages", func() {
				BeforeEach(func() {
					fs.WriteFileString(filepath.Join("/", "extracted", "release", "release.MF"), `---
name: release
version: version
commit_hash: commit
uncommitted_changes: true

jobs:
- name: job1
  version: job1-version
  fingerprint: job1-fp
  sha1: job1-sha
- name: job2
  version: job2-version
  fingerprint: job2-fp
  sha1: job2-sha

packages:
- name: pkg2
  version: pkg2-version
  fingerprint: pkg2-fp
  sha1: pkg2-sha
- name: pkg1
  version: pkg1-version
  fingerprint: pkg1-fp
  sha1: pkg1-sha
  dependencies: [pkg2]
`)
				})

				It("returns a release from the given tar file", func() {
					job1 := boshjob.NewJob(NewResource("job1", "job1-fp", nil))
					job1.PackageNames = []string{"pkg1"}
					job2 := boshjob.NewJob(NewResource("job2", "job2-fp", nil))

					pkg1 := boshpkg.NewPackage(NewResource("pkg1", "pkg1-fp", nil), []string{"pkg2"})
					pkg2 := boshpkg.NewPackage(NewResource("pkg2", "pkg2-fp", nil), nil)

					jobReader.ReadStub = func(jobRef boshman.JobRef, path string) (*boshjob.Job, error) {
						if jobRef.Name == "job1" {
							Expect(jobRef).To(Equal(boshman.JobRef{
								Name:        "job1",
								Version:     "job1-version",
								Fingerprint: "job1-fp",
								SHA1:        "job1-sha",
							}))
							Expect(path).To(Equal(filepath.Join("/", "extracted", "release", "jobs", "job1.tgz")))
							return job1, nil
						}
						if jobRef.Name == "job2" {
							Expect(jobRef).To(Equal(boshman.JobRef{
								Name:        "job2",
								Version:     "job2-version",
								Fingerprint: "job2-fp",
								SHA1:        "job2-sha",
							}))
							Expect(path).To(Equal(filepath.Join("/", "extracted", "release", "jobs", "job2.tgz")))
							return job2, nil
						}
						panic("Unexpected job")
					}

					pkgReader.ReadStub = func(pkgRef boshman.PackageRef, path string) (*boshpkg.Package, error) {
						if pkgRef.Name == "pkg1" {
							Expect(pkgRef).To(Equal(boshman.PackageRef{
								Name:         "pkg1",
								Version:      "pkg1-version",
								Fingerprint:  "pkg1-fp",
								SHA1:         "pkg1-sha",
								Dependencies: []string{"pkg2"},
							}))
							Expect(path).To(Equal(filepath.Join("/", "extracted", "release", "packages", "pkg1.tgz")))
							return pkg1, nil
						}
						if pkgRef.Name == "pkg2" {
							Expect(pkgRef).To(Equal(boshman.PackageRef{
								Name:        "pkg2",
								Version:     "pkg2-version",
								Fingerprint: "pkg2-fp",
								SHA1:        "pkg2-sha",
							}))
							Expect(path).To(Equal(filepath.Join("/", "extracted", "release", "packages", "pkg2.tgz")))
							return pkg2, nil
						}
						panic("Unexpected package")
					}

					release, err := act()
					Expect(err).NotTo(HaveOccurred())

					Expect(release.Name()).To(Equal("release"))
					Expect(release.Version()).To(Equal("version"))
					Expect(release.CommitHashWithMark("*")).To(Equal("commit*"))
					Expect(release.Jobs()).To(Equal([]*boshjob.Job{job1, job2}))
					Expect(release.Packages()).To(Equal([]*boshpkg.Package{pkg2, pkg1}))
					Expect(release.CompiledPackages()).To(BeEmpty())
					Expect(release.IsCompiled()).To(BeFalse())
					Expect(release.License()).To(BeNil())

					// job pkg dependencies are resolved
					Expect(job1.Packages).To(Equal([]boshpkg.Compilable{pkg1}))
					Expect(pkg1.Dependencies).To(Equal([]*boshpkg.Package{pkg2}))

					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeTrue())
				})

				It("returns errors for each invalid job and package", func() {
					jobReader.ReadReturns(nil, errors.New("job-err"))
					pkgReader.ReadReturns(nil, errors.New("pkg-err"))

					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Reading job 'job1' from archive"))
					Expect(err.Error()).To(ContainSubstring("Reading job 'job2' from archive"))
					Expect(err.Error()).To(ContainSubstring("Reading package 'pkg1' from archive"))
					Expect(err.Error()).To(ContainSubstring("Reading package 'pkg2' from archive"))

					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeFalse())
				})

				It("returns error if job's pkg dependencies cannot be satisfied", func() {
					job1 := boshjob.NewJob(NewResource("job1", "job1-fp", nil))
					job1.PackageNames = []string{"pkg-with-other-name"}
					jobReader.ReadReturns(job1, nil)

					pkg1 := boshpkg.NewPackage(NewResource("pkg1", "pkg1-fp", nil), nil)
					pkgReader.ReadReturns(pkg1, nil)

					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						"Expected to find package 'pkg-with-other-name' since it's a dependency of job 'job1'"))

					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeFalse())
				})

				It("returns error if pkg's pkg dependencies cannot be satisfied", func() {
					job1 := boshjob.NewJob(NewResource("job1", "job1-fp", nil))
					jobReader.ReadReturns(job1, nil)

					pkg1 := boshpkg.NewPackage(NewResource("pkg1", "pkg1-fp", nil), []string{"pkg-with-other-name"})
					pkgReader.ReadReturns(pkg1, nil)

					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						"Expected to find package 'pkg-with-other-name' since it's a dependency of package 'pkg1'"))

					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeFalse())
				})

				It("returns a release that can be cleaned up", func() {
					fs.WriteFileString(filepath.Join("/", "extracted", "release", "release.MF"), "")
					fs.MkdirAll(filepath.Join("/", "extracted", "release"), os.ModeDir)

					release, err := reader.Read("archive-path")
					Expect(err).NotTo(HaveOccurred())

					Expect(release.CleanUp()).ToNot(HaveOccurred())
					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeFalse())
				})

				It("returns error when cleaning up fails", func() {
					fs.WriteFileString(filepath.Join("/", "extracted", "release", "release.MF"), "")
					fs.RemoveAllStub = func(_ string) error { return errors.New("fake-err") }

					release, err := reader.Read("archive-path")
					Expect(err).NotTo(HaveOccurred())

					Expect(release.CleanUp()).To(Equal(errors.New("fake-err")))
				})
			})

			Context("when manifest that includes jobs and compiled packages and license", func() {
				BeforeEach(func() {
					fs.WriteFileString(filepath.Join("/", "extracted", "release", "release.MF"), `---
name: release
version: version
commit_hash: commit
uncommitted_changes: true

jobs:
- name: job1
  version: job1-version
  fingerprint: job1-fp
  sha1: job1-sha
- name: job2
  version: job2-version
  fingerprint: job2-fp
  sha1: job2-sha

compiled_packages:
- name: pkg2
  version: pkg2-version
  fingerprint: pkg2-fp
  stemcell: pkg2-stemcell
  sha1: pkg2-sha
- name: pkg1
  version: pkg1-version
  fingerprint: pkg1-fp
  stemcell: pkg1-stemcell
  sha1: pkg1-sha
  dependencies: [pkg2]

license:
  version: lic-version
  fingerprint: lic-fp
  sha1: lic-sha
`,
					)

					fs.WriteFileString(filepath.Join("/", "extracted", "release", "license.tgz"), "license")
				})

				It("returns a release from the given tar file", func() {
					job1 := boshjob.NewJob(NewResource("job1", "job1-fp", nil))
					job1.PackageNames = []string{"pkg1"}
					job2 := boshjob.NewJob(NewResource("job2", "job2-fp", nil))

					compiledPkg1 := boshpkg.NewCompiledPackageWithArchive(
						"pkg1", "pkg1-fp", "pkg1-stemcell",
						filepath.Join("/", "extracted", "release", "compiled_packages", "pkg1.tgz"), "pkg1-sha", []string{"pkg2"})
					compiledPkg2 := boshpkg.NewCompiledPackageWithArchive(
						"pkg2", "pkg2-fp", "pkg2-stemcell",
						filepath.Join("/", "extracted", "release", "compiled_packages", "pkg2.tgz"), "pkg2-sha", nil)
					compiledPkg1.AttachDependencies([]*boshpkg.CompiledPackage{compiledPkg2})

					lic := boshlic.NewLicense(NewResourceWithBuiltArchive(
						"license", "lic-fp", filepath.Join("/", "extracted", "release", "license.tgz"), "lic-sha"))

					jobReader.ReadStub = func(jobRef boshman.JobRef, path string) (*boshjob.Job, error) {
						if jobRef.Name == "job1" {
							Expect(jobRef).To(Equal(boshman.JobRef{
								Name:        "job1",
								Version:     "job1-version",
								Fingerprint: "job1-fp",
								SHA1:        "job1-sha",
							}))
							Expect(path).To(Equal(filepath.Join("/", "extracted", "release", "jobs", "job1.tgz")))
							return job1, nil
						}
						if jobRef.Name == "job2" {
							Expect(jobRef).To(Equal(boshman.JobRef{
								Name:        "job2",
								Version:     "job2-version",
								Fingerprint: "job2-fp",
								SHA1:        "job2-sha",
							}))
							Expect(path).To(Equal(filepath.Join("/", "extracted", "release", "jobs", "job2.tgz")))
							return job2, nil
						}
						panic("Unexpected job")
					}

					release, err := act()
					Expect(err).NotTo(HaveOccurred())

					Expect(release.Name()).To(Equal("release"))
					Expect(release.Version()).To(Equal("version"))
					Expect(release.CommitHashWithMark("*")).To(Equal("commit*"))
					Expect(release.Jobs()).To(Equal([]*boshjob.Job{job1, job2}))
					Expect(release.Packages()).To(BeEmpty())
					Expect(release.CompiledPackages()).To(Equal(
						[]*boshpkg.CompiledPackage{compiledPkg2, compiledPkg1}))
					Expect(release.IsCompiled()).To(BeTrue())
					Expect(release.License()).To(Equal(lic))

					// job pkg dependencies are resolved
					Expect(job1.Packages).To(Equal([]boshpkg.Compilable{compiledPkg1}))

					// compiled pkg dependencies are resolved
					Expect(compiledPkg1.Dependencies).To(Equal([]*boshpkg.CompiledPackage{compiledPkg2}))

					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeTrue())
				})

				It("returns errors for each invalid job", func() {
					jobReader.ReadReturns(nil, errors.New("job-err"))

					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Reading job 'job1' from archive"))
					Expect(err.Error()).To(ContainSubstring("Reading job 'job2' from archive"))

					Expect(fs.FileExists("/extracted/release")).To(BeFalse())
				})

				It("returns error if compiled pkg's compiled pkg dependencies cannot be satisfied", func() {
					fs.WriteFileString(filepath.Join("/", "extracted", "release", "release.MF"), `---
name: release
version: version

compiled_packages:
- name: pkg1
  version: pkg1-version
  fingerprint: pkg1-fp
  stemcell: pkg1-stemcell
  sha1: pkg1-sha
  dependencies: [pkg-with-other-name]
`)

					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						"Expected to find compiled package 'pkg-with-other-name' since it's a dependency of compiled package 'pkg1'"))

					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeFalse())
				})
			})

			Context("when the release manifest is invalid", func() {
				BeforeEach(func() {
					fs.WriteFileString(filepath.Join("/", "extracted", "release", "release.MF"), "-")
				})

				It("returns an error when the YAML in unparseable", func() {
					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Parsing release manifest"))
				})

				It("returns an error when the release manifest is missing", func() {
					fs.RemoveAll(filepath.Join("/", "extracted", "release", "release.MF"))
					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Reading manifest"))
				})

				It("deletes extracted release", func() {
					_, err := act()
					Expect(err).To(HaveOccurred())
					Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeFalse())
				})
			})
		})

		Context("when the release is not a valid tar", func() {
			BeforeEach(func() {
				compressor.DecompressFileToDirErr = errors.New("fake-error")
			})

			It("returns error", func() {
				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Extracting release"))
			})

			It("deletes extracted release", func() {
				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(fs.FileExists(filepath.Join("/", "extracted", "release"))).To(BeFalse())
			})
		})
	})
})
