package supervisor

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

type Runner struct {
	task *Task
}

func (t *Task) Runner() (*Runner, error) {
	r := &Runner{task: t}
	return r, nil
}

func drain(io io.Reader, name string, ch chan []byte) {
	s := bufio.NewScanner(io)
	for s.Scan() {
		ch <- s.Bytes()
	}
}

func (r *Runner) Exec(c chan []byte) error {
	var subcommand string
	if r.task.Op == BACKUP {
		subcommand = fmt.Sprintf("%s backup | %s store", r.task.Target.Plugin, r.task.Store.Plugin)
	} else {
		subcommand = fmt.Sprintf("%s retrieve | %s restore", r.task.Store.Plugin, r.task.Target.Plugin)
	}

	cmd := exec.Command("/bin/sh", "-c", subcommand)
	cmd.Env = []string{
		fmt.Sprintf("SHIELD_TARGET_ENDPOINT=%s", r.task.Target.Endpoint),
		fmt.Sprintf("SHIELD_STORE_ENDPOINT=%s", r.task.Store.Endpoint),
	}
	// FIXME: SHIELD_RESTORE_KEY ?

	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	go drain(stderr, "stderr", c)
	go drain(stdout, "stdout", c)

	err := cmd.Run()
	return err
}
