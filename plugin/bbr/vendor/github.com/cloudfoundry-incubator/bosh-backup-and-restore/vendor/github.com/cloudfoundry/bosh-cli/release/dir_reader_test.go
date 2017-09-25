package release_test

import (
	"errors"
	"os"
	"path/filepath"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	fakejob "github.com/cloudfoundry/bosh-cli/release/job/jobfakes"
	boshlic "github.com/cloudfoundry/bosh-cli/release/license"
	fakelic "github.com/cloudfoundry/bosh-cli/release/license/licensefakes"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	fakepkg "github.com/cloudfoundry/bosh-cli/release/pkg/pkgfakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("DirReader", func() {
	var (
		jobReader *fakejob.FakeDirReader
		pkgReader *fakepkg.FakeDirReader
		licReader *fakelic.FakeDirReader
		fs        *fakesys.FakeFileSystem
		reader    DirReader
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		fs.TempDirDir = filepath.Join("/", "release")

		logger := boshlog.NewLogger(boshlog.LevelNone)

		jobReader = &fakejob.FakeDirReader{}
		pkgReader = &fakepkg.FakeDirReader{}
		licReader = &fakelic.FakeDirReader{}
		reader = NewDirReader(jobReader, pkgReader, licReader, fs, logger)
	})

	Describe("Read", func() {
		act := func() (Release, error) { return reader.Read(filepath.Join("/", "release")) }

		var (
			job1 *boshjob.Job
			job2 *boshjob.Job
			pkg1 *boshpkg.Package
			pkg2 *boshpkg.Package
			lic  *boshlic.License
		)

		BeforeEach(func() {
			fs.SetGlob(filepath.Join("/", "release", "jobs", "*"), []string{
				filepath.Join("/", "release", "jobs", "job2"),
				filepath.Join("/", "release", "jobs", "job1"),
			})

			fs.SetGlob(filepath.Join("/", "release", "packages", "*"), []string{
				filepath.Join("/", "release", "packages", "pkg2"),
				filepath.Join("/", "release", "packages", "pkg1"),
			})

			fs.MkdirAll(filepath.Join("/", "release", "jobs", "job2"), os.ModeDir)
			fs.MkdirAll(filepath.Join("/", "release", "jobs", "job1"), os.ModeDir)

			fs.MkdirAll(filepath.Join("/", "release", "packages", "pkg2"), os.ModeDir)
			fs.MkdirAll(filepath.Join("/", "release", "packages", "pkg1"), os.ModeDir)

			job1 = boshjob.NewJob(NewResource("job1", "job1-fp", nil))
			job1.PackageNames = []string{"pkg1"}
			job2 = boshjob.NewJob(NewResource("job2", "job2-fp", nil))

			pkg1 = boshpkg.NewPackage(NewResource("pkg1", "pkg1-fp", nil), []string{"pkg2"})
			pkg2 = boshpkg.NewPackage(NewResource("pkg2", "pkg2-fp", nil), nil)

			lic = boshlic.NewLicense(NewResource("lic", "lic-fp", nil))

			jobReader.ReadStub = func(path string) (*boshjob.Job, error) {
				if path == filepath.Join("/", "release", "jobs", "job1") {
					return job1, nil
				}
				if path == filepath.Join("/", "release", "jobs", "job2") {
					return job2, nil
				}
				panic("Unexpected job")
			}

			pkgReader.ReadStub = func(path string) (*boshpkg.Package, error) {
				if path == filepath.Join("/", "release", "packages", "pkg1") {
					return pkg1, nil
				}
				if path == filepath.Join("/", "release", "packages", "pkg2") {
					return pkg2, nil
				}
				panic("Unexpected package")
			}

			licReader.ReadStub = func(path string) (*boshlic.License, error) {
				if path == filepath.Join("/", "release") {
					return lic, nil
				}
				panic("Unexpected license")
			}
		})

		It("returns a release from the given directory", func() {
			release, err := act()
			Expect(err).NotTo(HaveOccurred())

			Expect(release.Name()).To(BeEmpty())
			Expect(release.Version()).To(BeEmpty())
			Expect(release.CommitHashWithMark("*")).To(BeEmpty())
			Expect(release.Jobs()).To(ConsistOf([]*boshjob.Job{job1, job2}))
			Expect(release.Packages()).To(ConsistOf([]*boshpkg.Package{pkg1, pkg2}))
			Expect(release.CompiledPackages()).To(BeEmpty())
			Expect(release.IsCompiled()).To(BeFalse())
			Expect(release.License()).To(Equal(lic))

			// job and pkg dependencies are resolved
			Expect(job1.Packages).To(Equal([]boshpkg.Compilable{pkg1}))
			Expect(pkg1.Dependencies).To(Equal([]*boshpkg.Package{pkg2}))
		})

		It("orders jobs and packages alphabetically", func() {
			release, err := act()
			Expect(err).NotTo(HaveOccurred())

			Expect(release.Name()).To(BeEmpty())
			Expect(release.Version()).To(BeEmpty())
			Expect(release.CommitHashWithMark("*")).To(BeEmpty())
			Expect(release.Jobs()).To(Equal([]*boshjob.Job{job1, job2}))
			Expect(release.Packages()).To(Equal([]*boshpkg.Package{pkg1, pkg2}))
			Expect(release.CompiledPackages()).To(BeEmpty())
			Expect(release.IsCompiled()).To(BeFalse())
			Expect(release.License()).To(Equal(lic))

			// job and pkg dependencies are resolved
			Expect(job1.Packages).To(Equal([]boshpkg.Compilable{pkg1}))
			Expect(pkg1.Dependencies).To(Equal([]*boshpkg.Package{pkg2}))
		})

		Context("there are no jobs or packages", func() {
			BeforeEach(func() {
				fs.SetGlob(filepath.Join("/", "release", "jobs", "*"), []string{})
				fs.SetGlob(filepath.Join("/", "release", "packages", "*"), []string{})

				licReader.ReadStub = nil
				licReader.ReadReturns(nil, nil)
			})

			It("returns empty release", func() {

				release, err := act()
				Expect(err).NotTo(HaveOccurred())

				Expect(release.Name()).To(BeEmpty())
				Expect(release.Version()).To(BeEmpty())
				Expect(release.CommitHashWithMark("*")).To(BeEmpty())
				Expect(release.Jobs()).To(BeEmpty())
				Expect(release.Packages()).To(BeEmpty())
				Expect(release.CompiledPackages()).To(BeEmpty())
				Expect(release.IsCompiled()).To(BeFalse())
				Expect(release.License()).To(BeNil())
			})
		})

		Context("There are invalid jobs and packages", func() {
			BeforeEach(func() {
				jobReader.ReadReturns(nil, errors.New("job-err"))
				pkgReader.ReadReturns(nil, errors.New("pkg-err"))
			})

			It("returns errors for each invalid job and package", func() {
				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Reading job from '" + filepath.Join("/", "release", "jobs", "job1") + "'"))
				Expect(err.Error()).To(ContainSubstring("Reading job from '" + filepath.Join("/", "release", "jobs", "job2") + "'"))
				Expect(err.Error()).To(ContainSubstring("Reading package from '" + filepath.Join("/", "release", "packages", "pkg1") + "'"))
				Expect(err.Error()).To(ContainSubstring("Reading package from '" + filepath.Join("/", "release", "packages", "pkg2") + "'"))
			})
		})

		Context("a jobs package deps cannot be satisfied", func() {
			BeforeEach(func() {
				job1 = boshjob.NewJob(NewResource("job1", "job1-fp", nil))
				job1.PackageNames = []string{"pkg-with-other-name"}
				jobReader.ReadReturns(job1, nil)

				pkg1 = boshpkg.NewPackage(NewResource("pkg1", "pkg1-fp", nil), nil)
				pkgReader.ReadReturns(pkg1, nil)
			})

			It("returns error", func() {
				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					"Expected to find package 'pkg-with-other-name' since it's a dependency of job 'job1'"))
			})
		})

		Context("the pkg's pkg dependencies cannot be satisfied", func() {
			BeforeEach(func() {
				job1 = boshjob.NewJob(NewResource("job1", "job1-fp", nil))
				jobReader.ReadReturns(job1, nil)

				pkg1 = boshpkg.NewPackage(NewResource("pkg1", "pkg1-fp", nil), []string{"pkg-with-other-name"})
				pkgReader.ReadReturns(pkg1, nil)
			})

			It("returns error", func() {
				_, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					"Expected to find package 'pkg-with-other-name' since it's a dependency of package 'pkg1'"))
			})
		})

		Context("cleanup", func() {
			BeforeEach(func() {
				fs.SetGlob(filepath.Join("/", "release", "jobs", "*"), []string{})
				fs.SetGlob(filepath.Join("/", "release", "packages", "*"), []string{})

				fs.MkdirAll(filepath.Join("/", "release"), os.ModeDir)
			})

			It("returns a release that does nothing", func() {
				release, err := reader.Read(filepath.Join("/", "release"))
				Expect(err).NotTo(HaveOccurred())

				Expect(release.CleanUp()).ToNot(HaveOccurred())
				Expect(fs.FileExists(filepath.Join("/", "release"))).To(BeTrue())
			})
		})

		Context("There is a job that is not a directory", func() {
			BeforeEach(func() {
				fs.SetGlob(filepath.Join("/", "release", "jobs", "*"), []string{
					filepath.Join("/", "release", "jobs", "job1"),
					filepath.Join("/", "release", "jobs", "job2"),
					filepath.Join("/", "release", "jobs", "lol"),
				})

				fs.MkdirAll(filepath.Join("/", "release", "jobs", "job1"), os.ModeDir)
				fs.MkdirAll(filepath.Join("/", "release", "jobs", "job2"), os.ModeDir)

				fs.WriteFileString(filepath.Join("/", "release", "jobs", "lol"), "why did the chicken cross the road?")
			})

			It("ignores the non-dir input", func() {
				release, err := act()
				Expect(err).NotTo(HaveOccurred())

				Expect(release.Jobs()).To(Equal([]*boshjob.Job{job1, job2}))
				// job and pkg dependencies are resolved
				Expect(job1.Packages).To(Equal([]boshpkg.Compilable{pkg1}))
				Expect(pkg1.Dependencies).To(Equal([]*boshpkg.Package{pkg2}))
			})
		})

		Context("There is a package that is not a directory", func() {
			BeforeEach(func() {
				fs.SetGlob(filepath.Join("/", "release", "packages", "*"), []string{
					filepath.Join("/", "release", "packages", "pkg1"),
					filepath.Join("/", "release", "packages", "pkg2"),
					filepath.Join("/", "release", "packages", "lol"),
				})

				fs.MkdirAll(filepath.Join("/", "release", "packages", "pkg1"), os.ModeDir)
				fs.MkdirAll(filepath.Join("/", "release", "packages", "pkg2"), os.ModeDir)

				fs.WriteFileString(filepath.Join("/", "release", "packages", "lol"), "why did the chicken cross the road?")
			})

			It("ignores the non-dir input", func() {
				release, err := act()
				Expect(err).NotTo(HaveOccurred())

				Expect(release.Packages()).To(Equal([]*boshpkg.Package{pkg1, pkg2}))
				// job and pkg dependencies are resolved
				Expect(job1.Packages).To(Equal([]boshpkg.Compilable{pkg1}))
				Expect(pkg1.Dependencies).To(Equal([]*boshpkg.Package{pkg2}))
			})
		})
	})
})
