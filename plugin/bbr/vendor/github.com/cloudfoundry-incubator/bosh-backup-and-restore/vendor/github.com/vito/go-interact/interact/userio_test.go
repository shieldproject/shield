package interact_test

import (
	"github.com/kr/pty"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/go-interact/interact"
)

var _ = Describe("User IO", func() {
	Describe("fetching input from the user", func() {
		Context("when the terminal reports Ctrl-C was pressed", func() {
			It("returns ErrKeyboardInterrupt", func() {
				aPty, tty, err := pty.Open()
				Expect(err).NotTo(HaveOccurred())

				interaction := interact.NewInteraction("What is the air-speed of a Swallow?")
				interaction.Input = aPty
				interaction.Output = aPty

				go func() {
					defer GinkgoRecover()

					_, err = tty.Write([]byte{03})
					Expect(err).NotTo(HaveOccurred())
				}()

				var thing string
				err = interaction.Resolve(&thing)

				Expect(err).To(Equal(interact.ErrKeyboardInterrupt))
			})
		})
	})
})
