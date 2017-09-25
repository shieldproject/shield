package pkg_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	biindex "github.com/cloudfoundry/bosh-cli/index"
	boshrelpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/state/pkg"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("CompiledPackageRepo", func() {
	var (
		index               biindex.Index
		compiledPackageRepo CompiledPackageRepo
		fs                  *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		index = biindex.NewFileIndex("/index_file", fs)
		compiledPackageRepo = NewCompiledPackageRepo(index)
	})

	Context("Save/Find", func() {
		var (
			record CompiledPackageRecord
		)

		BeforeEach(func() {
			record = CompiledPackageRecord{}
		})

		It("saves the compiled package to the index", func() {
			pkg := newPkg("pkg-name", "pkg-fp", nil)

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			result, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(result).To(Equal(record))
		})

		It("returns false when finding before saving", func() {
			pkg := newPkg("pkg-name", "pkg-fp", nil)

			_, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("returns false if package dependencies have changed after saving", func() {
			dependency := newPkg("dep-name", "dep-fp", nil)
			pkg := newPkg("pkg-name", "pkg-fp", []string{"dep-name"})
			pkg.AttachDependencies([]*boshrelpkg.Package{dependency})

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			_, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			dependency = newPkg("dep-name", "new-dep-fp", nil)
			pkg = newPkg("pkg-name", "pkg-fp", []string{"dep-name"})
			pkg.AttachDependencies([]*boshrelpkg.Package{dependency})

			_, found, err = compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("returns true if dependency order changed", func() {
			dependency1 := newPkg("dep1-name", "dep1-fp", nil)
			dependency2 := newPkg("dep2-name", "dep2-fp", nil)
			pkg := newPkg("pkg-name", "pkg-fp", []string{"dep1-name", "dep2-name"})
			pkg.AttachDependencies([]*boshrelpkg.Package{dependency1, dependency2})

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			pkg = newPkg("pkg-name", "pkg-fp", []string{"dep2-name", "dep1-name"})
			pkg.AttachDependencies([]*boshrelpkg.Package{dependency2, dependency1})

			result, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(result).To(Equal(record))
		})

		It("returns false if a transitive dependency has changed after saving", func() {
			dependency1 := newPkg("dep1-name", "dep1-fp", []string{"dep3-name"})
			dependency2 := newPkg("dep2-name", "dep2-fp", nil)
			dependency3 := newPkg("dep3-name", "dep3-fp", nil)
			dependency1.AttachDependencies([]*boshrelpkg.Package{dependency3})
			pkg := newPkg("pkg-name", "pkg-fp", []string{"dep1-name", "dep2-name"})
			pkg.AttachDependencies([]*boshrelpkg.Package{dependency1, dependency2})

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			_, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			dependency1 = newPkg("dep1-name", "dep1-fp", []string{"dep3-name"})
			dependency2 = newPkg("dep2-name", "dep2-fp", nil)
			dependency3 = newPkg("dep3-name", "new-dep3-fp", nil)
			dependency1.AttachDependencies([]*boshrelpkg.Package{dependency3})
			pkg = newPkg("pkg-name", "pkg-fp", []string{"dep1-name", "dep2-name"})
			pkg.AttachDependencies([]*boshrelpkg.Package{dependency1, dependency2})

			_, found, err = compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("returns error when saving to index fails", func() {
			fs.WriteFileError = errors.New("Could not save")

			record := CompiledPackageRecord{
				BlobID:   "fake-blob-id",
				BlobSHA1: "fake-sha1",
			}

			pkg := newPkg("pkg-name", "pkg-fp", nil)

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Saving compiled package"))
		})

		It("returns error when reading from index fails", func() {
			pkg := newPkg("pkg-name", "pkg-fp", nil)

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			fs.ReadFileError = errors.New("fake-error")

			_, _, err = compiledPackageRepo.Find(pkg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding compiled package"))
		})
	})
})
