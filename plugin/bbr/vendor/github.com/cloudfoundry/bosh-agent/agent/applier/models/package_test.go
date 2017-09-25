package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	"github.com/cloudfoundry/bosh-utils/crypto"
)

var _ = Describe("Package", func() {
	Describe("BundleName", func() {
		It("returns name", func() {
			pkg := Package{Name: "fake-name"}
			Expect(pkg.BundleName()).To(Equal("fake-name"))
		})
	})

	Describe("BundleVersion", func() {
		It("returns version plus sha1 of source to make packages unique", func() {
			pkg := Package{
				Version: "fake-version",
				Source:  Source{Sha1: crypto.NewDigest(crypto.DigestAlgorithmSHA1, "fake-sha1")},
			}
			Expect(pkg.BundleVersion()).To(Equal("fake-version-fake-sha1"))
		})
	})
})

var _ = Describe("LocalPackage", func() {
	Describe("BundleName", func() {
		It("returns name", func() {
			pkg := LocalPackage{Name: "fake-name"}
			Expect(pkg.BundleName()).To(Equal("fake-name"))
		})
	})

	Describe("BundleVersion", func() {
		It("returns version plus sha1 of source to make packages unique", func() {
			pkg := LocalPackage{Version: "fake-version"}
			Expect(pkg.BundleVersion()).To(Equal("fake-version"))
		})
	})
})
