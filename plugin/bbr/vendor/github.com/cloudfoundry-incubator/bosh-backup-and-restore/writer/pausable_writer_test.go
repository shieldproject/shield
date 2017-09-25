package writer_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("The Pausable Writer", func() {
	It("is pausable", func() {
		buf := gbytes.NewBuffer()
		pw := NewPausableWriter(buf)

		By("writing when not paused", func() {
			nb, err := pw.Write([]byte("not paused"))
			Expect(err).NotTo(HaveOccurred())
			Expect(nb).To(Equal(10))
			Expect(string(buf.Contents())).To(Equal("not paused"))
		})

		By("not writing while paused", func() {
			pw.Pause()
			nb, err := pw.Write([]byte(" - paused"))
			Expect(err).NotTo(HaveOccurred())
			Expect(nb).To(Equal(0))
			Expect(string(buf.Contents())).To(Equal("not paused"))
		})

		By("writing when resumed", func() {
			nb, err := pw.Resume()
			Expect(err).NotTo(HaveOccurred())
			Expect(nb).To(Equal(9))
			Expect(string(buf.Contents())).To(Equal("not paused - paused"))
		})

		By("writing when not paused", func() {
			nb, err := pw.Write([]byte(" - not paused"))
			Expect(err).NotTo(HaveOccurred())
			Expect(nb).To(Equal(13))
			Expect(string(buf.Contents())).To(Equal("not paused - paused - not paused"))
		})
	})
})
