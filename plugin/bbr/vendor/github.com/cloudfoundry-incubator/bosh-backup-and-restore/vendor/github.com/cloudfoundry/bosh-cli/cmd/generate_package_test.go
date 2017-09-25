package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
)

var _ = Describe("GeneratePackageCmd", func() {
	var (
		releaseDir *fakereldir.FakeReleaseDir
		command    GeneratePackageCmd
	)

	BeforeEach(func() {
		releaseDir = &fakereldir.FakeReleaseDir{}
		command = NewGeneratePackageCmd(releaseDir)
	})

	Describe("Run", func() {
		var (
			opts GeneratePackageOpts
		)

		BeforeEach(func() {
			opts = GeneratePackageOpts{Args: GeneratePackageArgs{Name: "pkg"}}
		})

		act := func() error { return command.Run(opts) }

		It("generates package", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(releaseDir.GeneratePackageCallCount()).To(Equal(1))
			Expect(releaseDir.GeneratePackageArgsForCall(0)).To(Equal("pkg"))
		})

		It("returns error if generating package fails", func() {
			releaseDir.GeneratePackageReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
