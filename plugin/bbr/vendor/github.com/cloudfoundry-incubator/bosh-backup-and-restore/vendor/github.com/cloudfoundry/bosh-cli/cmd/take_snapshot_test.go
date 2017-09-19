package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("TakeSnapshotCmd", func() {
	var (
		deployment *fakedir.FakeDeployment
		command    TakeSnapshotCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{}
		command = NewTakeSnapshotCmd(deployment)
	})

	Describe("Run", func() {
		var (
			opts TakeSnapshotOpts
		)

		BeforeEach(func() {
			opts = TakeSnapshotOpts{}
		})

		act := func() error { return command.Run(opts) }

		Context("when taking a snapshot of specific instance", func() {
			BeforeEach(func() {
				opts.Args.Slug = boshdir.NewInstanceSlug("some-name", "some-id")
			})

			It("take snapshots for a given instance", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.TakeSnapshotCallCount()).To(Equal(1))
				Expect(deployment.TakeSnapshotsCallCount()).To(Equal(0))

				Expect(deployment.TakeSnapshotArgsForCall(0)).To(Equal(
					boshdir.NewInstanceSlug("some-name", "some-id")))
			})

			It("returns error if taking snapshots failed", func() {
				deployment.TakeSnapshotReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when taking snapshots for the entire deployment", func() {
			It("takes snapshots for the entire deployment", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.TakeSnapshotCallCount()).To(Equal(0))
				Expect(deployment.TakeSnapshotsCallCount()).To(Equal(1))
			})

			It("returns error if taking snapshots failed", func() {
				deployment.TakeSnapshotsReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})
	})
})
