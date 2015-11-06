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

func Exec(flags int, cmdString string) (int, error) {
	cmdArgs, err := shellwords.Parse(cmdString)
	if err != nil {
		return EXEC_FAILURE, fmt.Errorf("Could not parse '%s' into exec-able command: %s", cmdString, err.Error)
	}
	DEBUG("Executing '%s' with arguments %v", cmdArgs[0], cmdArgs[1:])

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	if flags&STDOUT == STDOUT {
		cmd.Stdout = os.Stdout
	}
	if flags&STDIN == STDIN {
		cmd.Stdin = os.Stdin
	}
	err = cmd.Run()
	if err != nil {
		return EXEC_FAILURE, fmt.Errorf("Unable to exec '%s': %s", cmdArgs[0], err.Error())
	}
	return SUCCESS, nil
}
