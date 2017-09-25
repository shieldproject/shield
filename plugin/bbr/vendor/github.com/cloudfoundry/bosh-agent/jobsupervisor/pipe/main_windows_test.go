package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	Context("when parent exits", func() {
		It("kills children and exits", func(done Done) {
			cmd := exec.Command(ExitRunnerPath, pathToPipeCLI, PrintPidsPath)
			cmd.Env = append(os.Environ(),
				joinEnv("SERVICE_NAME", ServiceName),
			)

			s, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).To(Succeed())
			Eventually(func() int { return len(s.Out.Contents()) }).Should(BeNumerically(">", 0))
			pids := strings.Split(strings.TrimSpace(string(s.Out.Contents())), ",")

			i, err := strconv.Atoi(pids[1])
			Expect(err).To(Succeed())
			pipeProc, err := os.FindProcess(i)
			Expect(err).To(Succeed())

			i, err = strconv.Atoi(pids[0])
			Expect(err).To(Succeed())
			childProc, err := os.FindProcess(i)
			Expect(err).To(Succeed())

			_, err = pipeProc.Wait()
			Expect(err).To(Succeed())

			_, err = childProc.Wait()
			Expect(err).To(Succeed())

			close(done)
		}, 10)
	})
})
