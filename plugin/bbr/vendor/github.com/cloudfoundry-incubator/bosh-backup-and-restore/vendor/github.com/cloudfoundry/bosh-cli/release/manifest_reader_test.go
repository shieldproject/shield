package release_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	boshlic "github.com/cloudfoundry/bosh-cli/release/license"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("ManifestReader", func() {
	var (
		fs     *fakesys.FakeFileSystem
		reader ManifestReader
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		fs.TempDirDir = "/release"

		logger := boshlog.NewLogger(boshlog.LevelNone)
		reader = NewManifestReader(fs, logger)
	})

	Describe("Read", func() {
		act := func() (Release, error) { return reader.Read("/release.yml") }

		Context("when manifest includes jobs and packages", func() {
			It("returns release with jobs and packages", func() {
				fs.WriteFileString("/release.yml", `---
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

				release, err := act()
				Expect(err).NotTo(HaveOccurred())

				job1 := boshjob.NewJob(NewExistingResource("job1", "job1-fp", "job1-sha"))
				job2 := boshjob.NewJob(NewExistingResource("job2", "job2-fp", "job2-sha"))

				pkg1 := boshpkg.NewPackage(NewExistingResource("pkg1", "pkg1-fp", "pkg1-sha"), []string{"pkg2"})
				pkg2 := boshpkg.NewPackage(NewExistingResource("pkg2", "pkg2-fp", "pkg2-sha"), nil)
				pkg1.AttachDependencies([]*boshpkg.Package{pkg2})

				Expect(release.Name()).To(Equal("release"))
				Expect(release.Version()).To(Equal("version"))
				Expect(release.CommitHashWithMark("*")).To(Equal("commit*"))
				Expect(release.Jobs()).To(Equal([]*boshjob.Job{job1, job2}))
				Expect(release.Packages()).To(Equal([]*boshpkg.Package{pkg2, pkg1}))
				Expect(release.CompiledPackages()).To(BeEmpty())
				Expect(release.IsCompiled()).To(BeFalse())
				Expect(release.License()).To(BeNil())
			})

			It("returns error if pkg's pkg dependencies cannot be satisfied", func() {
				fs.WriteFileString("/release.yml", `---
name: release
version: version
packages:
- name: pkg1
  version: pkg1-version
  fingerprint: pkg1-fp
  sha1: pkg1-sha
  dependencies: [pkg-with-other-name]
`)

				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					"Expected to find package 'pkg-with-other-name' since it's a dependency of package 'pkg1'"))
			})

			It("returns a release that can be cleaned up", func() {
				fs.WriteFileString("/release.yml", "")

				release, err := reader.Read("/release.yml")
				Expect(err).NotTo(HaveOccurred())

				Expect(release.CleanUp()).ToNot(HaveOccurred())
				Expect(fs.FileExists("/release.yml")).To(BeTrue())
			})
		})

		Context("when manifest includes jobs and compiled packages and license", func() {
			It("returns a release with jobs, compiled packages and license", func() {
				fs.WriteFileString("/release.yml", `---
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
`)

				fs.WriteFileString("/release/license.tgz", "license")

				job1 := boshjob.NewJob(NewExistingResource("job1", "job1-fp", "job1-sha"))
				job2 := boshjob.NewJob(NewExistingResource("job2", "job2-fp", "job2-sha"))

				compiledPkg1 := boshpkg.NewCompiledPackageWithoutArchive(
					"pkg1", "pkg1-fp", "pkg1-stemcell", "pkg1-sha", []string{"pkg2"})
				compiledPkg2 := boshpkg.NewCompiledPackageWithoutArchive(
					"pkg2", "pkg2-fp", "pkg2-stemcell", "pkg2-sha", nil)
				compiledPkg1.AttachDependencies([]*boshpkg.CompiledPackage{compiledPkg2})

				lic := boshlic.NewLicense(NewExistingResource("license", "lic-fp", "lic-sha"))

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
			})

			It("returns error if compiled pkg's compiled pkg dependencies cannot be satisfied", func() {
				fs.WriteFileString("/release.yml", `---
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
			})
		})

		Context("when the release manifest is invalid", func() {
			BeforeEach(func() {
				fs.WriteFileString("/release.yml", "-")
			})

			It("returns an error when the YAML in unparseable", func() {
				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Parsing release manifest"))
			})

			It("returns an error when the release manifest is missing", func() {
				fs.RemoveAll("/release.yml")

				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Reading manifest"))
			})
		})
	})
})
