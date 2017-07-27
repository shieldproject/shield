package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("DeleteSnapshotCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    DeleteSnapshotCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewDeleteSnapshotCmd(ui, deployment)
	})

	Describe("Run", func() {
		var (
			opts DeleteSnapshotOpts
		)

		BeforeEach(func() {
			opts = DeleteSnapshotOpts{
				Args: DeleteSnapshotArgs{CID: "some-cid"},
			}
		})

		act := func() error { return command.Run(opts) }

		It("deletes snapshot", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.DeleteSnapshotCallCount()).To(Equal(1))
			Expect(deployment.DeleteSnapshotArgsForCall(0)).To(Equal("some-cid"))
		})

		It("does not delete snapshot if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.DeleteSnapshotCallCount()).To(Equal(0))
		})

		It("returns error if deleting snapshot failed", func() {
			deployment.DeleteSnapshotReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
