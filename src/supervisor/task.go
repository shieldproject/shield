package supervisor

import (
	"bufio"
	"fmt"
	"github.com/pborman/uuid"
	"io"
	"os/exec"
	"time"
)

type Operation int

const (
	BACKUP Operation = iota
	RESTORE
)

type Status int

const (
	PENDING Status = iota
	RUNNING
	CANCELED
	DONE
)

type PluginConfig struct {
	Plugin   string
	Endpoint string
}

type Task struct {
	uuid uuid.UUID

	Store  *PluginConfig
	Target *PluginConfig

	Op     Operation
	status Status

	startedAt time.Time
	stoppedAt time.Time

	output []string
}

func drain(io io.Reader, name string, ch chan string) {
	s := bufio.NewScanner(io)
	for s.Scan() {
		ch <- s.Text()
	}
}

func (t *Task) Run(stdout chan string, stderr chan string) error {
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

	pstderr, _ := cmd.StderrPipe()
	pstdout, _ := cmd.StdoutPipe()
	go drain(pstderr, "stderr", stderr)
	go drain(pstdout, "stdout", stdout)

	err := cmd.Run()
	return err
}
