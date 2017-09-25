package pkg_test

import (
	"errors"

	biindex "github.com/cloudfoundry/bosh-init/index"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/state/pkg"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("CompiledPackageRepo", func() {
	var (
		index               biindex.Index
		compiledPackageRepo CompiledPackageRepo
		fakeFS              *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		fakeFS = fakesys.NewFakeFileSystem()
		index = biindex.NewFileIndex("/index_file", fakeFS)
		compiledPackageRepo = NewCompiledPackageRepo(index)
	})

	Context("Save and Find", func() {
		var (
			record     CompiledPackageRecord
			dependency birelpkg.Package
			pkg        birelpkg.Package
		)

		BeforeEach(func() {
			record = CompiledPackageRecord{}
			dependency = birelpkg.Package{
				Name:        "fake-dependency-package",
				Fingerprint: "fake-dependency-fingerprint",
			}
			pkg = birelpkg.Package{
				Name:         "fake-package-name",
				Fingerprint:  "fake-package-fingerprint",
				Dependencies: []*birelpkg.Package{&dependency},
			}
		})

		It("saves the compiled package to the index", func() {
			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			result, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(result).To(Equal(record))
		})

		It("returns false when finding before saving", func() {
			pkg := birelpkg.Package{
				Name: "fake-package-name",
			}
			_, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("returns false if package dependencies have changed after saving", func() {
			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			_, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			dependency.Fingerprint = "new-fake-dependency-fingerprint"

			_, found, err = compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("returns true if dependency order changed", func() {
			dependency1 := birelpkg.Package{
				Name:        "fake-package-1",
				Fingerprint: "fake-dependency-fingerprint-1",
			}
			dependency2 := birelpkg.Package{
				Name:        "fake-package-2",
				Fingerprint: "fake-dependency-fingerprint-2",
			}

			pkg.Dependencies = []*birelpkg.Package{&dependency1, &dependency2}

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			pkg.Dependencies = []*birelpkg.Package{&dependency2, &dependency1}

			result, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(result).To(Equal(record))
		})

		It("returns false if a transitive dependency has changed after saving", func() {
			transitive := birelpkg.Package{
				Name:        "fake-transitive-package",
				Fingerprint: "fake-transitive-fingerprint",
			}
			dependency.Dependencies = []*birelpkg.Package{&transitive}

			err := compiledPackageRepo.Save(pkg, record)
			Expect(err).ToNot(HaveOccurred())

			_, found, err := compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			transitive.Fingerprint = "new-fake-dependency-fingerprint"

			_, found, err = compiledPackageRepo.Find(pkg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		Context("when saving to index fails", func() {
			It("returns error", func() {
				fakeFS.WriteFileError = errors.New("Could not save")
				record := CompiledPackageRecord{
					BlobID:   "fake-blob-id",
					BlobSHA1: "fake-sha1",
				}

				pkg := birelpkg.Package{
					Name: "fake-package-name",
				}

				err := compiledPackageRepo.Save(pkg, record)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Saving compiled package"))
			})
		})

		Context("when reading from index fails", func() {
			It("returns error", func() {
				err := compiledPackageRepo.Save(pkg, record)
				fakeFS.ReadFileError = errors.New("fake-error")

				_, _, err = compiledPackageRepo.Find(pkg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Finding compiled package"))
			})
		})
	})
})
