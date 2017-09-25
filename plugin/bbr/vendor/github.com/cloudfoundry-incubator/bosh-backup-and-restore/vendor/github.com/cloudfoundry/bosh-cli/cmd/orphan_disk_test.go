package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("OrphanDiskCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  OrphanDiskCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewOrphanDiskCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts OrphanDiskOpts
		)

		BeforeEach(func() {
			opts = OrphanDiskOpts{
				Args: OrphanDiskArgs{CID: "disk-cid"},
			}
		})

		act := func() error { return command.Run(opts) }

		It("orphans disk", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.OrphanDiskArgsForCall(0)).To(Equal("disk-cid"))
			Expect(director.OrphanDiskCallCount()).To(Equal(1))
		})

		It("returns error if orphaning disk failed", func() {
			director.OrphanDiskReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("does not orphan disk if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(director.OrphanDiskCallCount()).To(Equal(0))
		})
	})
})
