package cmd_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bicmd "github.com/cloudfoundry/bosh-init/cmd"

	fakebicmd "github.com/cloudfoundry/bosh-init/cmd/fakes"
	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("Runner", func() {
	var (
		runner      *bicmd.Runner
		factory     *fakebicmd.FakeFactory
		fakeCommand *fakebicmd.FakeCommand
		fakeStage   *fakebiui.FakeStage
	)

	BeforeEach(func() {
		fakeCommand = fakebicmd.NewFakeCommand("fake-command-name", bicmd.Meta{})
		factory = &fakebicmd.FakeFactory{PresetCommand: fakeCommand}
		fakeStage = fakebiui.NewFakeStage()
	})

	JustBeforeEach(func() {
		runner = bicmd.NewRunner(factory)
	})

	Context("Run", func() {
		Context("valid args", func() {
			It("extracts command name from the arguments", func() {
				err := runner.Run(fakeStage, "fake-command-name", "/fake/manifest_path")
				Expect(err).ToNot(HaveOccurred())
				Expect(factory.CommandName).To(Equal("fake-command-name"))
			})

			It("creates and run a non nil Command with remaining args", func() {
				err := runner.Run(fakeStage, "fake-command-name", "/fake/manifest_path")
				Expect(err).ToNot(HaveOccurred())
				Expect(factory.CommandName).To(Equal("fake-command-name"))
				Expect(factory.PresetCommand).ToNot(BeNil())
				Expect(factory.PresetCommand.GetArgs()).To(Equal([]string{"/fake/manifest_path"}))
			})
		})

		Context("when no arguments were passed in", func() {
			It("prints the generic help command", func() {
				err := runner.Run(fakeStage)
				Expect(err).ToNot(HaveOccurred())
				Expect(factory.CommandName).To(Equal("help"))
			})
		})

		Context("when help option is passed in", func() {
			testCases := [][]string{
				[]string{"fake-command-name", "help"},
				[]string{"fake-command-name", "-h"},
				[]string{"fake-command-name", "--help"},
				[]string{"help", "fake-command-name"},
				[]string{"-h", "fake-command-name"},
				[]string{"--help", "fake-command-name"},
			}

			It("prints command help", func() {
				for _, testCase := range testCases {
					err := runner.Run(fakeStage, testCase[0], testCase[1])
					Expect(err).ToNot(HaveOccurred())
					Expect(factory.CommandName).To(Equal("help"))
					Expect(factory.PresetCommand.GetArgs()).To(Equal([]string{"fake-command-name"}))
				}
			})
		})

		Context("when an unknown command name was passed in", func() {
			var fakeCommandName string

			BeforeEach(func() {
				fakeCommandName = "fake-command-name"
				factory.PresetError = fmt.Errorf("Command '%s' unknown", fakeCommandName)
			})

			It("fails with error with unknown command name", func() {
				err := runner.Run(fakeStage, "fake-command-name", "/fake/manifest_path")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Command '%s' unknown", fakeCommandName)))
				Expect(factory.CommandName).To(Equal("fake-command-name"))
			})
		})
	})
})
