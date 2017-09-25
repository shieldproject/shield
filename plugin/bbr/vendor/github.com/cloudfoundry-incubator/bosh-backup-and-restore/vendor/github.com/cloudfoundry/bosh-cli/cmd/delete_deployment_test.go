package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("DeleteDeploymentCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    DeleteDeploymentCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewDeleteDeploymentCmd(ui, deployment)
	})

	Describe("Run", func() {
		var (
			opts DeleteDeploymentOpts
		)

		BeforeEach(func() {
			opts = DeleteDeploymentOpts{}
		})

		act := func() error { return command.Run(opts) }

		It("deletes deployment", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.DeleteCallCount()).To(Equal(1))
			Expect(deployment.DeleteArgsForCall(0)).To(BeFalse())
		})

		It("deletes deployment forcefully if requested", func() {
			opts.Force = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.DeleteCallCount()).To(Equal(1))
			Expect(deployment.DeleteArgsForCall(0)).To(BeTrue())
		})

		It("does not delete deployment if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.DeleteCallCount()).To(Equal(0))
		})

		It("returns error if deleting deployment failed", func() {
			deployment.DeleteReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
