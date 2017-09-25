package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("DeleteVMCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    DeleteVMCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewDeleteVMCmd(ui, deployment)
	})

	Describe("Run", func() {
		var (
			opts DeleteVMOpts
		)

		BeforeEach(func() {
			opts = DeleteVMOpts{
				Args: DeleteVMArgs{CID: "some-cid"},
			}
		})

		act := func() error { return command.Run(opts) }

		It("deletes vm", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.DeleteVMCallCount()).To(Equal(1))
			Expect(deployment.DeleteVMArgsForCall(0)).To(Equal("some-cid"))
		})

		It("does not delete snapshot if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.DeleteVMCallCount()).To(Equal(0))
		})

		It("returns error if deleting snapshot failed", func() {
			deployment.DeleteVMReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
