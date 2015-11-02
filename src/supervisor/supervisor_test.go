package supervisor_test

import (
	. "supervisor"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Supervisor", func() {
	Describe("Task Executor", func() {
		var t *Task

		BeforeEach(func() {
			t = &Task{
				Op: BACKUP,
				Store: &PluginConfig{
					Plugin:   "test/bin/dummy",
					Endpoint: "{mode:store,endpoint:config}",
				},
				Target: &PluginConfig{
					Plugin:   "test/bin/dummy",
					Endpoint: "{mode:target,endpoint:config}",
				},
			}
		})

		drainTo := func(dst *[]string, ch chan string) {
			for {
				s, ok := <-ch
				if !ok {
					break
				}
				*dst = append(*dst, s)
			}
		}

		It("works", func() {
			var output, errors []string

			stdout := make(chan string)
			stderr := make(chan string)

			go drainTo(&output, stdout)
			go drainTo(&errors, stderr)

			err := t.Run(stdout, stderr)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("collects output from the command pipeline", func() {
			var output, errors []string

			stdout := make(chan string)
			stderr := make(chan string)

			go drainTo(&output, stdout)
			go drainTo(&errors, stderr)

			err := t.Run(stdout, stderr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(output)).Should(BeNumerically(">", 0))
			Ω(len(errors)).Should(BeNumerically(">", 0))
		})
	})
})
