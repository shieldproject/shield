package installation_test

import (
	. "github.com/cloudfoundry/bosh-init/installation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Target", func() {
	Describe("Paths", func() {
		var target Target
		BeforeEach(func() {
			target = NewTarget("/home/fake/madcow")
		})

		It("returns the blobstore path", func() {
			Expect(target.BlobstorePath()).To(Equal("/home/fake/madcow/blobs"))
		})

		It("returns the compiled packages index path", func() {
			Expect(target.CompiledPackagedIndexPath()).To(Equal("/home/fake/madcow/compiled_packages.json"))
		})

		It("returns the templates index path", func() {
			Expect(target.TemplatesIndexPath()).To(Equal("/home/fake/madcow/templates.json"))
		})

		It("returns the packages path", func() {
			Expect(target.PackagesPath()).To(Equal("/home/fake/madcow/packages"))
		})

		It("returns the temp path", func() {
			Expect(target.TmpPath()).To(Equal("/home/fake/madcow/tmp"))
		})
	})
})
