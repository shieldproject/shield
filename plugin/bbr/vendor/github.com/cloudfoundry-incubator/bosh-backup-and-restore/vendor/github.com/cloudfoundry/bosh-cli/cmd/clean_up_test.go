package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("CleanUpCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  CleanUpCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewCleanUpCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts CleanUpOpts
		)

		BeforeEach(func() {
			opts = CleanUpOpts{}
		})

		act := func() error { return command.Run(opts) }

		It("cleans up director resources", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.CleanUpCallCount()).To(Equal(1))
			Expect(director.CleanUpArgsForCall(0)).To(BeFalse())
		})

		It("cleans up *all* director resources", func() {
			opts.All = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.CleanUpCallCount()).To(Equal(1))
			Expect(director.CleanUpArgsForCall(0)).To(BeTrue())
		})

		It("does not clean up if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(director.CleanUpCallCount()).To(Equal(0))
		})

		It("returns error if cleaning up fails", func() {
			director.CleanUpReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
