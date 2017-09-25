package commands_test

import (
	. "github.com/starkandwayne/shield/cmd/shield/commands"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dispatcher", func() {

	var nop = func(_ *Options, _ ...string) error { return nil }
	var nopHelp = &HelpInfo{}
	var nopGroup = &HelpGroup{}

	Context("When commands where one name is a substring of the other are registered", func() {
		var cmd *Command
		var cmdname string
		var cmdargs []string
		var testinput []string
		const shortName = "restore"
		var shortCommand = &Command{
			Summary: "Restore something",
			Help:    nopHelp,
			RunFn:   nop,
			Group:   nopGroup,
		}
		const longName = "restore archive"
		var longCommand = &Command{
			Summary: "Restore... archive... something",
			Help:    nopHelp,
			RunFn:   nop,
			Group:   nopGroup,
		}

		BeforeEach(func() {
			Add(shortName, shortCommand)
			Add(longName, longCommand)
		})

		JustBeforeEach(func() {
			cmd, cmdname, cmdargs = ParseCommand(testinput...)
		})

		AfterEach(func() {
			Reset()
			cmd, cmdname, cmdargs, testinput = nil, "", nil, nil
		})

		Context("When trying to call the shorter command", func() {
			BeforeEach(func() {
				testinput = []string{shortName, "argument"}
			})

			It("should give back the correct command structure", func() {
				Expect(cmd).To(BeIdenticalTo(shortCommand))
			})

			It("should parse the correct command name", func() {
				Expect(cmdname).To(Equal(shortName))
			})

			It("should parse the correct args", func() {
				Expect(cmdargs).To(Equal([]string{"argument"}))
			})
		})

		Context("When trying to call the longer command", func() {
			BeforeEach(func() {
				testinput = []string{longName, "argument"}
			})

			It("should give back the correct command structure", func() {
				Expect(cmd).To(BeIdenticalTo(longCommand))
			})

			It("should parse the correct command name", func() {
				Expect(cmdname).To(Equal(longName))
			})

			It("should parse the correct args", func() {
				Expect(cmdargs).To(Equal([]string{"argument"}))
			})
		})
	})
})
