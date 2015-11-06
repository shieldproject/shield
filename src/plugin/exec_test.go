package plugin_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"plugin"
)

var _ = Describe("Plugin Commands", func() {
	It("Executes commands successfully", func() {
		rc, err := plugin.Exec(plugin.NOPIPE, "test/bin/exec_tester 0")
		Expect(rc).Should(Equal(plugin.SUCCESS))
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("Returns errors when the command fails", func() {
		rc, err := plugin.Exec(plugin.NOPIPE, "test/bin/exec_tester 1")
		Expect(rc).Should(Equal(plugin.EXEC_FAILURE))
		Expect(err).Should(HaveOccurred())
	})
	It("Hooks up stderr to the caller's stderr", func() {
		Skip("Test not implemented yet :( PRs welcome ;)")
	})
	It("Hooks up stdin to the callers stdin when requested", func() {
		Skip("Test not implemented yet :( PRs welcome ;)")
	})
	It("Hooks up stdout to the callers stdout when requested", func() {
		Skip("Test not implemented yet :( PRs welcome ;)")
	})
	It("Does not hook up stdout to the callers stdout when not requested", func() {
		Skip("Test not implemented yet :( PRs welcome ;)")
	})
	It("Does not hook up stdin to the callers stdin when not requested", func() {
		Skip("Test not implemented yet :( PRs welcome ;)")
	})
	It("Returns an error for commands that cannot be parsed", func() {
		Skip("Test not implemented yet :( PRs welcome ;)")
	})
})
