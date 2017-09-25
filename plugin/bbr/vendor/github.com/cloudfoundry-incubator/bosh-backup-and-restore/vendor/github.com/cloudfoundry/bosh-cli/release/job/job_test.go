package job_test

import (
	"fmt"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/job"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("Sorting Jobs", func() {
	Describe("a slice of jobs", func() {
		var jobs []*Job
		var expectedJobs []*Job
		BeforeEach(func() {
			for i := 5; i >= 0; i-- {
				jobs = append(jobs, NewJob(NewResourceWithBuiltArchive(fmt.Sprintf("name%d", i), "fp", "path", "sha1")))
				expectedJobs = append(expectedJobs, NewJob(NewResourceWithBuiltArchive(fmt.Sprintf("name%d", 5-i), "fp", "path", "sha1")))
			}
		})

		It("can be sorted by job name", func() {
			sort.Sort(ByName(jobs))
			Expect(jobs).To(Equal(expectedJobs))
		})
	})
})

var _ = Describe("Job", func() {
	Describe("Name/Fingerprint/ArchivePath/ArchiveSHA1", func() {
		It("delegates to resource", func() {
			job := NewJob(NewResourceWithBuiltArchive("name", "fp", "path", "sha1"))
			Expect(job.Name()).To(Equal("name"))
			Expect(job.Fingerprint()).To(Equal("fp"))
			Expect(job.ArchivePath()).To(Equal("path"))
			Expect(job.ArchiveSHA1()).To(Equal("sha1"))
		})
	})

	Describe("FindTemplateByValue", func() {
		Context("when a template with the value exists", func() {
			It("returns the template and true", func() {
				job := Job{
					Templates: map[string]string{"src": "dst"},
				}

				tpl, ok := job.FindTemplateByValue("dst")
				Expect(ok).To(BeTrue())
				Expect(tpl).To(Equal("src"))
			})
		})

		Context("when the template does not exist", func() {
			It("returns nil and false", func() {
				job := Job{
					Templates: map[string]string{"src": "dst"},
				}

				_, ok := job.FindTemplateByValue("other")
				Expect(ok).To(BeFalse())
			})
		})
	})

	Describe("AttachPackages", func() {
		It("attaches packages based on their names", func() {
			job := NewJob(NewResource("name", "fp", nil))
			job.PackageNames = []string{"pkg1", "pkg2"}

			pkg1 := boshpkg.NewPackage(NewResourceWithBuiltArchive("pkg1", "fp", "path", "sha1"), nil)
			pkg2 := boshpkg.NewPackage(NewResourceWithBuiltArchive("pkg2", "fp", "path", "sha1"), nil)
			unusedPkg := boshpkg.NewPackage(NewResourceWithBuiltArchive("unused", "fp", "path", "sha1"), nil)

			err := job.AttachPackages([]*boshpkg.Package{pkg1, unusedPkg, pkg2})
			Expect(err).ToNot(HaveOccurred())

			Expect(job.Packages).To(Equal([]boshpkg.Compilable{pkg1, pkg2}))
		})

		It("returns error if package cannot be found", func() {
			job := NewJob(NewResource("name", "fp", nil))
			job.PackageNames = []string{"pkg1"}

			pkg2 := boshpkg.NewPackage(NewResourceWithBuiltArchive("pkg2", "fp", "path", "sha1"), nil)

			err := job.AttachPackages([]*boshpkg.Package{pkg2})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected to find package 'pkg1' since it's a dependency of job 'name'"))
		})
	})

	Describe("AttachCompilablePackages", func() {
		It("attaches compiled packages based on their names", func() {
			job := NewJob(NewResource("name", "fp", nil))
			job.PackageNames = []string{"pkg1", "pkg2"}

			pkg1 := boshpkg.NewCompiledPackageWithArchive("pkg1", "fp", "", "path", "sha1", nil)
			pkg2 := boshpkg.NewCompiledPackageWithArchive("pkg2", "fp", "", "path", "sha1", nil)
			unusedPkg := boshpkg.NewCompiledPackageWithArchive("unused", "fp", "", "path", "sha1", nil)

			err := job.AttachCompilablePackages([]boshpkg.Compilable{pkg1, unusedPkg, pkg2})
			Expect(err).ToNot(HaveOccurred())

			Expect(job.Packages).To(Equal([]boshpkg.Compilable{pkg1, pkg2}))
		})

		It("returns error if compiled package cannot be found", func() {
			job := NewJob(NewResource("name", "fp", nil))
			job.PackageNames = []string{"pkg1"}

			pkg2 := boshpkg.NewCompiledPackageWithArchive("pkg2", "fp", "", "path", "sha1", nil)

			err := job.AttachCompilablePackages([]boshpkg.Compilable{pkg2})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected to find package 'pkg1' since it's a dependency of job 'name'"))
		})
	})

	Describe("CleanUp", func() {
		It("does nothing by default", func() {
			job := NewJob(NewResourceWithBuiltArchive("name", "fp", "path", "sha1"))
			Expect(job.CleanUp()).ToNot(HaveOccurred())
		})
	})
})
