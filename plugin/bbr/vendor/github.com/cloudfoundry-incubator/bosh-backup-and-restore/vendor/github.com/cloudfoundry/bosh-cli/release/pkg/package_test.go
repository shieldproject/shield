package pkg_test

import (
	"fmt"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/pkg"
	boshres "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("Sorting Packages", func() {
	Describe("a slice of Packages", func() {
		var packages []*Package
		var expectedPackages []*Package
		BeforeEach(func() {
			for i := 5; i >= 0; i-- {
				packages = append(packages, NewPackage(boshres.NewResourceWithBuiltArchive(fmt.Sprintf("name%d", i), "fp", "path", "sha1"), []string{"pkg1"}))
				expectedPackages = append(expectedPackages, NewPackage(boshres.NewResourceWithBuiltArchive(fmt.Sprintf("name%d", 5-i), "fp", "path", "sha1"), []string{"pkg1"}))
			}
		})

		It("can be sorted by package name", func() {
			sort.Sort(ByName(packages))
			Expect(packages).To(Equal(expectedPackages))
		})
	})
})

var _ = Describe("Package", func() {
	Describe("common methods", func() {
		It("delegates to resource", func() {
			pkg := NewPackage(boshres.NewResourceWithBuiltArchive("name", "fp", "path", "sha1"), []string{"pkg1"})
			Expect(pkg.Name()).To(Equal("name"))
			Expect(pkg.String()).To(Equal("name"))
			Expect(pkg.Fingerprint()).To(Equal("fp"))
			Expect(pkg.ArchivePath()).To(Equal("path"))
			Expect(pkg.ArchiveSHA1()).To(Equal("sha1"))
			Expect(pkg.DependencyNames()).To(Equal([]string{"pkg1"}))
		})
	})

	Describe("AttachDependencies", func() {
		It("attaches dependencies based on their names", func() {
			pkg := NewPackage(boshres.NewResourceWithBuiltArchive("name", "fp", "path", "sha1"), []string{"pkg1", "pkg2"})
			pkg1 := NewPackage(boshres.NewResourceWithBuiltArchive("pkg1", "fp", "path", "sha1"), nil)
			pkg2 := NewPackage(boshres.NewResourceWithBuiltArchive("pkg2", "fp", "path", "sha1"), nil)
			unusedPkg := NewPackage(boshres.NewResourceWithBuiltArchive("unused", "fp", "path", "sha1"), nil)

			err := pkg.AttachDependencies([]*Package{pkg1, unusedPkg, pkg2})
			Expect(err).ToNot(HaveOccurred())

			Expect(pkg.Dependencies).To(Equal([]*Package{pkg1, pkg2}))
		})

		It("returns error if dependency cannot be found", func() {
			pkg := NewPackage(boshres.NewResourceWithBuiltArchive("name", "fp", "path", "sha1"), []string{"pkg1"})
			pkg2 := NewPackage(boshres.NewResourceWithBuiltArchive("pkg2", "fp", "path", "sha1"), nil)

			err := pkg.AttachDependencies([]*Package{pkg2})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected to find package 'pkg1' since it's a dependency of package 'name'"))
		})
	})

	Describe("CleanUp", func() {
		It("does nothing by default", func() {
			pkg := NewPackage(boshres.NewResourceWithBuiltArchive("name", "fp", "path", "sha1"), nil)
			Expect(pkg.CleanUp()).ToNot(HaveOccurred())
		})
	})
})
