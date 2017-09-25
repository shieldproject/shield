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

var _ = Describe("RestartCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    RestartCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewRestartCmd(ui, deployment)
	})

	Describe("Run", func() {
		var (
			opts RestartOpts
		)

		BeforeEach(func() {
			opts = RestartOpts{
				Args: AllOrInstanceGroupOrInstanceSlugArgs{
					Slug: boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", ""),
				},
			}
		})

		act := func() error { return command.Run(opts) }

		It("restarts deployment, pool or instances", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RestartCallCount()).To(Equal(1))

			slug, restartOpts := deployment.RestartArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(restartOpts.SkipDrain).To(BeFalse())
			Expect(restartOpts.Force).To(BeFalse())
		})

		It("restarts allowing to skip drain scripts", func() {
			opts.SkipDrain = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RestartCallCount()).To(Equal(1))

			slug, restartOpts := deployment.RestartArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(restartOpts.SkipDrain).To(BeTrue())
			Expect(restartOpts.Force).To(BeFalse())
		})

		It("can set canaries", func() {
			opts.Canaries = "3"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RestartCallCount()).To(Equal(1))

			_, restartOpts := deployment.RestartArgsForCall(0)
			Expect(restartOpts.Canaries).To(Equal("3"))
		})

		It("can set max_in_flight", func() {
			opts.MaxInFlight = "5"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RestartCallCount()).To(Equal(1))

			_, restartOpts := deployment.RestartArgsForCall(0)
			Expect(restartOpts.MaxInFlight).To(Equal("5"))
		})

		It("restarts forcefully", func() {
			opts.Force = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.RestartCallCount()).To(Equal(1))

			slug, restartOpts := deployment.RestartArgsForCall(0)
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
			Expect(restartOpts.SkipDrain).To(BeFalse())
			Expect(restartOpts.Force).To(BeTrue())
		})

		It("does not restart if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.RestartCallCount()).To(Equal(0))
		})

		It("returns error if restart failed", func() {
			deployment.RestartReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
