package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
)

var _ = Describe("InitReleaseCmd", func() {
	var (
		releaseDir *fakereldir.FakeReleaseDir
		command    InitReleaseCmd
	)

	BeforeEach(func() {
		releaseDir = &fakereldir.FakeReleaseDir{}
		command = NewInitReleaseCmd(releaseDir)
	})

	Describe("Run", func() {
		var (
			opts InitReleaseOpts
		)

		BeforeEach(func() {
			opts = InitReleaseOpts{}
		})

		act := func() error { return command.Run(opts) }

		It("inits release", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(releaseDir.InitCallCount()).To(Equal(1))
			Expect(releaseDir.InitArgsForCall(0)).To(BeFalse())
		})

		It("inits release with git as true", func() {
			opts.Git = true

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(releaseDir.InitCallCount()).To(Equal(1))
			Expect(releaseDir.InitArgsForCall(0)).To(BeTrue())
		})

		It("returns error if initing release fails", func() {
			releaseDir.InitReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
