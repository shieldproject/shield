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

		drainTo := func(dst *[]byte, ch chan []byte) {
			for {
				b, ok := <-ch
				if !ok {
					break
				}
				*dst = append(*dst, b...)
			}
		}

		It("works", func() {
			var output, errors []byte

			stdout := make(chan []byte)
			stderr := make(chan []byte)

			go drainTo(&output, stdout)
			go drainTo(&errors, stderr)

			err := t.Run(stdout, stderr)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("collects output from the command pipeline", func() {
			var output, errors []byte

			stdout := make(chan []byte)
			stderr := make(chan []byte)

			go drainTo(&output, stdout)
			go drainTo(&errors, stderr)

			err := t.Run(stdout, stderr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(output)).Should(BeNumerically(">", 0))
			Ω(len(errors)).Should(BeNumerically(">", 0))
		})
	})
})
