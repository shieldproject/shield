package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("DeleteSnapshotsCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    DeleteSnapshotsCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewDeleteSnapshotsCmd(ui, deployment)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("deletes snapshots", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.DeleteSnapshotsCallCount()).To(Equal(1))
		})

		It("does not delete snapshots if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.DeleteSnapshotsCallCount()).To(Equal(0))
		})

		It("returns error if deleting snapshots failed", func() {
			deployment.DeleteSnapshotsReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
