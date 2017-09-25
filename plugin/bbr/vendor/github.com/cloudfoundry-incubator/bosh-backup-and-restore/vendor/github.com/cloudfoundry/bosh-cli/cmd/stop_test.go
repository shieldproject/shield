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

var _ = Describe("StopCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    StopCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewStopCmd(ui, deployment)
	})

	Describe("Run", func() {
		var (
			opts StopOpts
		)

		BeforeEach(func() {
			opts = StopOpts{
				Args: AllOrInstanceGroupOrInstanceSlugArgs{
					Slug: boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", ""),
				},
			}
		})

		act := func() error { return command.Run(opts) }

		It("stops deployment, pool or instances", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StopCallCount()).To(Equal(1))

			slug, stopOpts := deployment.StopArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(stopOpts.Hard).To(BeFalse())
			Expect(stopOpts.SkipDrain).To(BeFalse())
			Expect(stopOpts.Force).To(BeFalse())
		})

		It("stops allowing to detach vms", func() {
			opts.Hard = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StopCallCount()).To(Equal(1))

			slug, stopOpts := deployment.StopArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(stopOpts.Hard).To(BeTrue())
			Expect(stopOpts.SkipDrain).To(BeFalse())
			Expect(stopOpts.Force).To(BeFalse())
		})

		It("stops allowing to skip drain scripts", func() {
			opts.SkipDrain = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StopCallCount()).To(Equal(1))

			slug, stopOpts := deployment.StopArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(stopOpts.Hard).To(BeFalse())
			Expect(stopOpts.SkipDrain).To(BeTrue())
			Expect(stopOpts.Force).To(BeFalse())
		})

		It("stops forcefully", func() {
			opts.Force = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StopCallCount()).To(Equal(1))

			slug, stopOpts := deployment.StopArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(stopOpts.Hard).To(BeFalse())
			Expect(stopOpts.SkipDrain).To(BeFalse())
			Expect(stopOpts.Force).To(BeTrue())
		})

		It("does not stop if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.StopCallCount()).To(Equal(0))
		})

		It("returns error if stop failed", func() {
			deployment.StopReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("can set canaries", func() {
			opts.Canaries = "30%"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StopCallCount()).To(Equal(1))

			_, stopOpts := deployment.StopArgsForCall(0)
			Expect(stopOpts.Canaries).To(Equal("30%"))
		})

		It("can set max_in_flight", func() {
			opts.MaxInFlight = "5"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StopCallCount()).To(Equal(1))

			_, stopOpts := deployment.StopArgsForCall(0)
			Expect(stopOpts.MaxInFlight).To(Equal("5"))
		})
	})
})
