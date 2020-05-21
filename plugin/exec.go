package plugin

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"

	"github.com/mattn/go-shellwords"
)

type ExecOptions struct {
	Stdout   io.Writer
	Stdin    io.Reader
	Stderr   io.Writer
	Cmd      string
	ExpectRC []int
}

func ExecWithOptions(opts ExecOptions) error {
	cmdArgs, err := shellwords.Parse(opts.Cmd)
	if err != nil {
		return ExecFailure{Err: fmt.Sprintf("Could not parse '%s' into exec-able command: %s", opts.Cmd, err.Error())}
	}
	DEBUG("Executing '%s' with arguments %v", cmdArgs[0], cmdArgs[1:])

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	}
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}

	if len(opts.ExpectRC) == 0 {
		opts.ExpectRC = []int{0}
	}

	err = cmd.Run()
	if err != nil {
		// make sure we got an Exit error
		if exitErr, ok := err.(*exec.ExitError); ok {
			sys := exitErr.ProcessState.Sys()
			// os.ProcessState.Sys() may not return syscall.WaitStatus on non-UNIX machines,
			// so currently this feature only works on UNIX, but shouldn't crash on other OSes
			if rc, ok := sys.(syscall.WaitStatus); ok {
				code := rc.ExitStatus()
				// -1 indicates signals, stops, or traps, so force an error
				if code >= 0 {
					for _, expect := range opts.ExpectRC {
						if code == expect {
							return nil
						}
					}
				}
			}
		}
		return ExecFailure{Err: fmt.Sprintf("Unable to exec '%s': %s", cmdArgs[0], err.Error())}
	}
	return nil
}

func Exec(cmdString string, in io.Reader, out io.Writer, err io.Writer) error {
	return ExecWithOptions(ExecOptions{
		Cmd:    cmdString,
		Stdin:  in,
		Stdout: out,
		Stderr: err,
	})
}
