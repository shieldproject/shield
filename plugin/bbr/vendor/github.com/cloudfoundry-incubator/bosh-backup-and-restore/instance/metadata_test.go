package instance_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	It("has BackupName and RestoreName fields", func() {
		metadata := Metadata{
			BackupName:  "foo",
			RestoreName: "bar",
		}

		Expect(metadata.BackupName).To(Equal("foo"))
		Expect(metadata.RestoreName).To(Equal("bar"))
	})

	It("can be created with raw metadata YAML", func() {
		rawMetadata := []byte(`---
backup_name: foo
restore_name: bar`)

		m, err := NewJobMetadata(rawMetadata)

		Expect(err).NotTo(HaveOccurred())
		Expect(m.BackupName).To(Equal("foo"))
		Expect(m.RestoreName).To(Equal("bar"))
	})

	It("fails when provided invalid YAML", func() {
		rawMetadata := []byte(`arrrr`)

		_, err := NewJobMetadata(rawMetadata)

		Expect(err).To(MatchError(ContainSubstring("failed to unmarshal job metadata")))
	})
})
