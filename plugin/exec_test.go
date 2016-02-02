package plugin_test

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/starkandwayne/shield/plugin"
)

var _ = Describe("Plugin Commands", func() {

	drain := func(file *os.File, output chan string) {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			panic(fmt.Sprintf("Error reading from pipe, test is invalid: %s", err.Error()))
		}
		output <- string(data)
	}

	It("Executes commands successfully", func() {
		opts := plugin.ExecOptions{
			Cmd: "test/bin/exec_tester 0",
		}
		err := plugin.ExecWithOptions(opts)
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("Returns errors when the command fails", func() {
		opts := plugin.ExecOptions{
			Cmd: "test/bin/exec_tester 1",
		}
		err := plugin.ExecWithOptions(opts)
		Expect(err).Should(HaveOccurred())
	})
	It("Doesn't return errors when the command returns an expected exit code", func() {
		opts := plugin.ExecOptions{
			Cmd:      "test/bin/exec_tester 1",
			ExpectRC: []int{0, 1},
		}
		err := plugin.ExecWithOptions(opts)
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("Doesn't return errors if the command exits 0 but we forgot to expect it", func() {
		opts := plugin.ExecOptions{
			Cmd:      "test/bin/exec_tester 0",
			ExpectRC: []int{1},
		}
		err := plugin.ExecWithOptions(opts)
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("Does return errors when the cmd returns an unexpected exit code", func() {
		opts := plugin.ExecOptions{
			Cmd:      "test/bin/exec_tester 2",
			ExpectRC: []int{1},
		}
		err := plugin.ExecWithOptions(opts)
		Expect(err).Should(HaveOccurred())
	})
	It("Gets stderr/stdout and uses stdin", func() {
		rStdin, wStdin, err := os.Pipe()
		Expect(err).ShouldNot(HaveOccurred())

		rStderr, wStderr, err := os.Pipe()
		Expect(err).ShouldNot(HaveOccurred())
		stderrC := make(chan string)
		go drain(rStderr, stderrC)

		rStdout, wStdout, err := os.Pipe()
		Expect(err).ShouldNot(HaveOccurred())
		stdoutC := make(chan string)
		go drain(rStdout, stdoutC)

		_, err = wStdin.Write([]byte("This should go to stdout"))
		Expect(err).ShouldNot(HaveOccurred())
		wStdin.Close()

		opts := plugin.ExecOptions{
			Cmd:      "test/bin/exec_tester 0",
			Stdout:   wStdout,
			Stderr:   wStderr,
			Stdin:    rStdin,
			ExpectRC: []int{0},
		}

		err = plugin.ExecWithOptions(opts)
		wStderr.Close() // simulate command exiting + its pipe being closed
		wStdout.Close() // simulate command exiting + its pipe being closed

		stderr := <-stderrC
		stdout := <-stdoutC

		Expect(err).ShouldNot(HaveOccurred())
		Expect(stdout).Should(Equal("This should go to stdout"))
		Expect(stderr).Should(Equal("This goes to stderr\n"))
	})
	It("Returns an error for commands that cannot be parsed", func() {
		err := plugin.Exec("this '\"cannot be parsed", plugin.NOPIPE)
		Expect(err).Should(HaveOccurred())
	})
})
