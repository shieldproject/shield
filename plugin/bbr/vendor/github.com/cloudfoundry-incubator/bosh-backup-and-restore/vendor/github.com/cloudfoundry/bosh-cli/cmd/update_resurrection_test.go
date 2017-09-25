package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("UpdateResurrectionCmd", func() {
	var (
		director *fakedir.FakeDirector
		command  UpdateResurrectionCmd
	)

	BeforeEach(func() {
		director = &fakedir.FakeDirector{}
		command = NewUpdateResurrectionCmd(director)
	})

	Describe("Run", func() {
		var (
			opts UpdateResurrectionOpts
		)

		BeforeEach(func() {
			opts = UpdateResurrectionOpts{}
		})

		act := func() error { return command.Run(opts) }

		It("enables resurrection", func() {
			opts.Args.Enabled = BoolArg(true)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.EnableResurrectionCallCount()).To(Equal(1))
			Expect(director.EnableResurrectionArgsForCall(0)).To(BeTrue())
		})

		It("disables resurrection", func() {
			opts.Args.Enabled = BoolArg(false)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.EnableResurrectionCallCount()).To(Equal(1))
			Expect(director.EnableResurrectionArgsForCall(0)).To(BeFalse())
		})

		It("returns error if changing resurrection fails", func() {
			director.EnableResurrectionReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
