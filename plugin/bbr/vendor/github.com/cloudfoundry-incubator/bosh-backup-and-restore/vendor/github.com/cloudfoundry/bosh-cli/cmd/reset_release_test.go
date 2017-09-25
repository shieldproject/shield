package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
)

var _ = Describe("ResetReleaseCmd", func() {
	var (
		releaseDir *fakereldir.FakeReleaseDir
		command    ResetReleaseCmd
	)

	BeforeEach(func() {
		releaseDir = &fakereldir.FakeReleaseDir{}
		command = NewResetReleaseCmd(releaseDir)
	})

	Describe("Run", func() {
		var (
			opts ResetReleaseOpts
		)

		BeforeEach(func() {
			opts = ResetReleaseOpts{}
		})

		act := func() error { return command.Run(opts) }

		It("resets release", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(releaseDir.ResetCallCount()).To(Equal(1))
		})

		It("returns error if resetting release fails", func() {
			releaseDir.ResetReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
