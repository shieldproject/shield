package pkg_test

import (
	. "github.com/cloudfoundry/bosh-init/release/pkg"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PackageRepo", func() {
	It("creates an new Package when package is not in repo", func() {
		packageRepo := &PackageRepo{}
		pkg := packageRepo.FindOrCreatePackage("fake-package-1")
		Expect(*pkg).To(Equal(Package{Name: "fake-package-1"}))
	})

	It("finds existing Package when package is in repo", func() {
		packageRepo := &PackageRepo{}
		pkg1 := packageRepo.FindOrCreatePackage("fake-package-1")
		pkg2 := packageRepo.FindOrCreatePackage("fake-package-1")
		Expect(pkg1.Name).To(Equal("fake-package-1"))
		Expect(pkg2.Name).To(Equal("fake-package-1"))
		Expect(pkg1).To(Equal(pkg2))
	})
})
