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

var _ = Describe("CloudConfigCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  CloudConfigCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewCloudConfigCmd(ui, director)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("shows cloud config", func() {
			cloudConfig := boshdir.CloudConfig{
				Properties: "some-properties",
			}

			director.LatestCloudConfigReturns(cloudConfig, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Blocks).To(Equal([]string{"some-properties"}))
		})

		It("returns error if cloud config cannot be retrieved", func() {
			director.LatestCloudConfigReturns(boshdir.CloudConfig{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
