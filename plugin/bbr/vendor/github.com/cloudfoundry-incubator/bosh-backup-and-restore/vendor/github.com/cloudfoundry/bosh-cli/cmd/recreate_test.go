package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("RecreateCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    RecreateCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewRecreateCmd(ui, deployment)
	})

	Describe("Run", func() {
		var (
			opts RecreateOpts
		)

		BeforeEach(func() {
			opts = RecreateOpts{
				Args: AllOrInstanceGroupOrInstanceSlugArgs{
					Slug: boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", ""),
				},
			}
		})

		act := func() error { return command.Run(opts) }

		It("recreate deployment, pool or instances", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RecreateCallCount()).To(Equal(1))

			slug, recreateOpts := deployment.RecreateArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(recreateOpts.SkipDrain).To(BeFalse())
			Expect(recreateOpts.Force).To(BeFalse())
		})

		It("recreate allowing to skip drain scripts", func() {
			opts.SkipDrain = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RecreateCallCount()).To(Equal(1))

			slug, recreateOpts := deployment.RecreateArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(recreateOpts.SkipDrain).To(BeTrue())
			Expect(recreateOpts.Force).To(BeFalse())
		})

		It("can set canaries", func() {
			opts.Canaries = "3"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RecreateCallCount()).To(Equal(1))

			_, recreateOpts := deployment.RecreateArgsForCall(0)
			Expect(recreateOpts.Canaries).To(Equal("3"))
		})

		It("can set max_in_flight", func() {
			opts.MaxInFlight = "5"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RecreateCallCount()).To(Equal(1))

			_, recreateOpts := deployment.RecreateArgsForCall(0)
			Expect(recreateOpts.MaxInFlight).To(Equal("5"))
		})

		It("can set dry_run", func() {
			opts.DryRun = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RecreateCallCount()).To(Equal(1))

			_, recreateOpts := deployment.RecreateArgsForCall(0)
			Expect(recreateOpts.DryRun).To(BeTrue())
		})

		It("can set fix", func() {
			opts.Fix = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RecreateCallCount()).To(Equal(1))

			_, recreateOpts := deployment.RecreateArgsForCall(0)
			Expect(recreateOpts.Fix).To(BeTrue())
		})

		It("recreate forcefully", func() {
			opts.Force = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RecreateCallCount()).To(Equal(1))

			slug, recreateOpts := deployment.RecreateArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(recreateOpts.SkipDrain).To(BeFalse())
			Expect(recreateOpts.Force).To(BeTrue())
		})

		It("does not recreate if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.RecreateCallCount()).To(Equal(0))
		})

		It("returns error if restart failed", func() {
			deployment.RecreateReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
