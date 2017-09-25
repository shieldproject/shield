package cmd_test

import (
	fakeui "github.com/cloudfoundry/bosh-init/ui/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/cmd"
)

var _ = Describe("VersionCmd", func() {
	var (
		ui      *fakeui.FakeUI
		command Cmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		command = NewVersionCmd(ui)
	})

	Describe("Name", func() {
		It("returns 'version'", func() {
			Expect(command.Name()).To(Equal("version"))
		})
	})

	Describe("Run", func() {
		It("prints the version", func() {
			err := command.Run(fakeui.NewFakeStage(), []string{})
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Said).To(Equal([]string{"version [DEV BUILD]"}))
		})
	})
})
