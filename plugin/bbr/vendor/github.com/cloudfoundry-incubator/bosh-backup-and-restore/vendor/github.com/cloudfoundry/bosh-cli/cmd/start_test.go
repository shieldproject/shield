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

var _ = Describe("StartCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    StartCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewStartCmd(ui, deployment)
	})

	Describe("Run", func() {
		var (
			opts StartOpts
		)

		BeforeEach(func() {
			opts = StartOpts{
				Args: AllOrInstanceGroupOrInstanceSlugArgs{
					Slug: boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", ""),
				},
			}
		})

		act := func() error { return command.Run(opts) }

		It("starts deployment, pool or instances", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StartCallCount()).To(Equal(1))
			Expect(deployment.StartArgsForCall(0)).To(Equal(
				boshdir.NewAllOrInstanceGroupOrInstanceSlug("some-name", "")))
		})

		It("does not start if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(deployment.StartCallCount()).To(Equal(0))
		})

		It("returns error if start failed", func() {
			deployment.StartReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("can set canaries", func() {
			opts.Canaries = "100%"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StartCallCount()).To(Equal(1))

			_, opts := deployment.StartArgsForCall(0)
			Expect(opts.Canaries).To(Equal("100%"))
		})

		It("can set max_in_flight", func() {
			opts.MaxInFlight = "5"

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.StartCallCount()).To(Equal(1))

			_, opts := deployment.StartArgsForCall(0)
			Expect(opts.MaxInFlight).To(Equal("5"))
		})
	})
})
