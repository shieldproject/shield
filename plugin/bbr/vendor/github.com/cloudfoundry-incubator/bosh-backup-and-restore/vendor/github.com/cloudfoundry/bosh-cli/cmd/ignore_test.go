package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"

	"errors"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("IgnoreCmd", func() {
	var (
		deployment *fakedir.FakeDeployment
		command    IgnoreCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{}
		command = NewIgnoreCmd(deployment)
	})

	Describe("Run", func() {
		var (
			opts IgnoreOpts
		)

		BeforeEach(func() {
			opts = IgnoreOpts{}
		})

		act := func() error {
			return command.Run(opts)
		}

		Context("when ignoring an instance", func() {
			BeforeEach(func() {
				opts.Args.Slug = boshdir.NewInstanceSlug("some-name", "some-id")
			})

			It("ignores the instance", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.IgnoreCallCount()).To(Equal(1))

				slugArg, ignoreArg := deployment.IgnoreArgsForCall(0)
				Expect(slugArg).To(Equal(boshdir.NewInstanceSlug("some-name", "some-id")))
				Expect(ignoreArg).To(Equal(true))
			})

			Context("when ignoring fails", func() {

				BeforeEach(func() {
					deployment.IgnoreReturns(errors.New("nope nope nope"))
				})

				It("returns the error", func() {
					err := act()
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
