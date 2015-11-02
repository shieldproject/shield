package supervisor

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

func drain(io io.Reader, name string, ch chan []byte) {
	s := bufio.NewScanner(io)
	for s.Scan() {
		ch <- s.Bytes()
	}
}

func (t *Task) Run(c chan []byte) error {
	var subcommand string
	if t.Op == BACKUP {
		subcommand = fmt.Sprintf("%s backup | %s store", t.Target.Plugin, t.Store.Plugin)
	} else {
		subcommand = fmt.Sprintf("%s retrieve | %s restore", t.Store.Plugin, t.Target.Plugin)
	}

	cmd := exec.Command("/bin/sh", "-c", subcommand)
	cmd.Env = []string{
		fmt.Sprintf("SHIELD_TARGET_ENDPOINT=%s", t.Target.Endpoint),
		fmt.Sprintf("SHIELD_STORE_ENDPOINT=%s", t.Store.Endpoint),
	}
	// FIXME: SHIELD_RESTORE_KEY ?

	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	go drain(stderr, "stderr", c)
	go drain(stdout, "stdout", c)

	err := cmd.Run()
	return err
}
