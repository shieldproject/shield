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

var _ = Describe("RuntimeConfigCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  RuntimeConfigCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewRuntimeConfigCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts RuntimeConfigOpts
		)
		BeforeEach(func() {
			opts = RuntimeConfigOpts{
				Name: "some-foo-config",
			}
		})
		act := func() error { return command.Run(opts) }

		It("shows runtime config", func() {
			runtimeConfig := boshdir.RuntimeConfig{
				Properties: "some-properties",
			}

			director.LatestRuntimeConfigReturns(runtimeConfig, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(director.LatestRuntimeConfigCallCount()).To(Equal(1))
			Expect(director.LatestRuntimeConfigArgsForCall(0)).To(Equal("some-foo-config"))

			Expect(ui.Blocks).To(Equal([]string{"some-properties"}))
		})

		It("returns error if runtime config cannot be retrieved", func() {
			director.LatestRuntimeConfigReturns(boshdir.RuntimeConfig{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
