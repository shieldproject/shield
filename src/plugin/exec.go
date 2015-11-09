package plugin

import (
	"fmt"
	"github.com/mattn/go-shellwords"
	"os"
	"os/exec"
)

const NOPIPE = 0
const STDIN = 1
const STDOUT = 2

func ExecWithPipes(cmdString string, stdout *os.File, stderr *os.File, stdin *os.File) (int, error) {
	cmdArgs, err := shellwords.Parse(cmdString)
	if err != nil {
		return EXEC_FAILURE, fmt.Errorf("Could not parse '%s' into exec-able command: %s", cmdString, err.Error)
	}
	DEBUG("Executing '%s' with arguments %v", cmdArgs[0], cmdArgs[1:])

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	if stdin != nil {
		cmd.Stdin = stdin
	}
	err = cmd.Run()
	if err != nil {
		return EXEC_FAILURE, fmt.Errorf("Unable to exec '%s': %s", cmdArgs[0], err.Error())
	}
	return SUCCESS, nil
}

func Exec(cmdString string, flags int) (int, error) {
	var stdout *os.File
	var stdin *os.File
	if flags&STDOUT == STDOUT {
		stdout = os.Stdout
	}
	if flags&STDIN == STDIN {
		stdin = os.Stdin
	}
	return ExecWithPipes(cmdString, stdout, os.Stderr, stdin)
}
