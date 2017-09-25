package license_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/license"
	boshres "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("License", func() {
	Describe("Name/Fingerprint/ArchivePath/ArchiveSHA1", func() {
		It("delegates to resource", func() {
			job := NewLicense(boshres.NewResourceWithBuiltArchive("name", "fp", "path", "sha1"))
			Expect(job.Name()).To(Equal("name"))
			Expect(job.Fingerprint()).To(Equal("fp"))
			Expect(job.ArchivePath()).To(Equal("path"))
			Expect(job.ArchiveSHA1()).To(Equal("sha1"))
		})
	})
})
