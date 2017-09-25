package cmd_test

import (
	fakecmd "github.com/cloudfoundry/bosh-init/cmd/fakes"
	fakeui "github.com/cloudfoundry/bosh-init/ui/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/cmd"
)

var _ = Describe("HelpCmd", func() {
	var (
		commands CommandList
		ui       *fakeui.FakeUI
		help     Cmd
	)

	BeforeEach(func() {
		commands = CommandList{
			"simple": func() (Cmd, error) {
				meta := Meta{
					Synopsis: "Simple command... sorted to the end of the list",
				}
				return fakecmd.NewFakeCommand("simple", meta), nil
			},
			"complex": func() (Cmd, error) {
				meta := Meta{
					Synopsis: "Complex command... has usage and env",
					Usage:    "[arguments]",
					Env: map[string]MetaEnv{
						"BOSH_ENV_VARIABLE1": {
							Example:     "value",
							Default:     "default",
							Description: "Sets some environment variable to a value",
						},
						"BOSH_ENV_VARIABLE2": {
							Example:     "something-else",
							Description: "Sets another environment variable",
						},
					},
				}
				return fakecmd.NewFakeCommand("complex", meta), nil
			},
			"help": func() (Cmd, error) {
				return NewHelpCmd(ui, commands), nil
			},
		}
		ui = &fakeui.FakeUI{}
		help = NewHelpCmd(ui, commands)
	})

	Describe("Name", func() {
		It("returns 'help'", func() {
			Expect(help.Name()).To(Equal("help"))
		})
	})

	Describe("Run", func() {
		Context("when no arguments were passed in", func() {
			It("prints generic help", func() {
				err := help.Run(fakeui.NewFakeStage(), []string{})
				Expect(err).ToNot(HaveOccurred())

				expectedOutput := `NAME:
    bosh-init - A command line tool to initialize BOSH deployments

USAGE:
    bosh-init [global options] <command> [arguments...]

COMMANDS:
    complex    Complex command... has usage and env
    help       Show help message
    simple     Simple command... sorted to the end of the list

GLOBAL OPTIONS:
    --help, -h       Show help message
    --version, -v    Show version`

				Expect(ui.Said).To(Equal([]string{expectedOutput}))
			})
		})

		Context("when existing command name passed in as first argument", func() {
			Context("given a command without usage and env", func() {
				It("prints requested command help", func() {
					err := help.Run(fakeui.NewFakeStage(), []string{"simple"})
					Expect(err).ToNot(HaveOccurred())
					expectedOutput := `NAME:
    simple - Simple command... sorted to the end of the list

USAGE:
    bosh-init [global options] simple

GLOBAL OPTIONS:
    --help, -h       Show help message
    --version, -v    Show version`

					Expect(ui.Said).To(Equal([]string{expectedOutput}))
				})
			})

			Context("given a command with usage and env", func() {
				It("prints requested command help", func() {
					err := help.Run(fakeui.NewFakeStage(), []string{"complex"})
					Expect(err).ToNot(HaveOccurred())
					expectedOutput := `NAME:
    complex - Complex command... has usage and env

USAGE:
    bosh-init [global options] complex [arguments]

ENVIRONMENT VARIABLES:
    BOSH_ENV_VARIABLE1=value             Sets some environment variable to a value. Default: default
    BOSH_ENV_VARIABLE2=something-else    Sets another environment variable

GLOBAL OPTIONS:
    --help, -h       Show help message
    --version, -v    Show version`

					Expect(ui.Said).To(Equal([]string{expectedOutput}))
				})
			})
		})

		Context("when non-existing command name passed in as first argument", func() {
			It("prints an error", func() {
				err := help.Run(fakeui.NewFakeStage(), []string{"foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ui.Said).To(Equal([]string{"No help found for command `foo'. Run 'bosh-init help' to see all available commands."}))
			})
		})

		Context("when help on help is requested", func() {
			It("returns helps's help", func() {
				err := help.Run(fakeui.NewFakeStage(), []string{"help", "help"})
				Expect(err).ToNot(HaveOccurred())
				expectedOutput := `NAME:
    help - Show help message

USAGE:
    bosh-init [global options] help [command]

GLOBAL OPTIONS:
    --help, -h       Show help message
    --version, -v    Show version`

				Expect(ui.Said).To(Equal([]string{expectedOutput}))
			})
		})
	})
})
