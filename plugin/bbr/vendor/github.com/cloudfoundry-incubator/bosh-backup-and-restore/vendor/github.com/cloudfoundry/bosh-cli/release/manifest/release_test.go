package manifest_test

import (
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/manifest"
)

var _ = Describe("NewManifestFromPath", func() {
	var (
		fs *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
	})

	It("decodes release with base64 job/package sha1s and fingerprints", func() {
		fs.WriteFileString("/release.yml", `---
name: release
version: version
commit_hash: commit
uncommitted_changes: true

jobs:
- name: job1
  sha1: !binary |-
    am9iMS1zaGEx
  fingerprint: !binary |-
    am9iMS1mcA==
  version: !binary |-
    am9iMS12ZXJzaW9u

packages:
- name: pkg1
  sha1: !binary |-
    cGtnMS1zaGEx
  fingerprint: !binary |-
    cGtnMS1mcA==
  version: !binary |-
    cGtnMS12ZXJzaW9u
`)

		manifest, err := NewManifestFromPath("/release.yml", fs)
		Expect(err).NotTo(HaveOccurred())

		Expect(manifest.Jobs[0].SHA1).To(Equal("job1-sha1"))
		Expect(manifest.Jobs[0].Fingerprint).To(Equal("job1-fp"))
		Expect(manifest.Jobs[0].Version).To(Equal("job1-version"))

		Expect(manifest.Packages[0].SHA1).To(Equal("pkg1-sha1"))
		Expect(manifest.Packages[0].Fingerprint).To(Equal("pkg1-fp"))
		Expect(manifest.Packages[0].Version).To(Equal("pkg1-version"))
	})
})
