package supervisor

import (
	"bufio"
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
	var targetCommand string
	var storeCommand string
	if t.Op == BACKUP {
		targetCommand = "backup"
		storeCommand = "store"

	} else {
		targetCommand = "restore"
		storeCommand = "retrieve"
	}

	targetArgs := []string{
		targetCommand,
		"--endpoint",
		t.Target.Endpoint,
	}
	targetCmd := exec.Command(t.Target.Plugin, targetArgs...)
	storeArgs := []string{
		storeCommand,
		"--endpoint",
		t.Store.Endpoint,
	}
	if t.Op != BACKUP {
		storeArgs = append(storeArgs, "--key", "FIXME")
	}
	storeCmd := exec.Command(t.Store.Plugin, storeArgs...)

	var pstdout io.Reader
	if t.Op == BACKUP {
		storeCmd.Stdin, _ = targetCmd.StdoutPipe()
		pstdout, _ = storeCmd.StdoutPipe()
	} else {
		targetCmd.Stdin, _ = storeCmd.StdoutPipe()
		pstdout, _ = targetCmd.StdoutPipe()
	}

	pTargetStderr, _ := targetCmd.StderrPipe()
	pStoreStderr, _ := storeCmd.StderrPipe()

	go drain(pTargetStderr, "stderr", stderr)
	go drain(pStoreStderr, "stderr", stderr)
	go drain(pstdout, "stdout", stdout)

	err := targetCmd.Start()
	if err != nil {
		return err
	}
	err = storeCmd.Start()
	if err != nil {
		return err
	}

	err = storeCmd.Wait()
	if err != nil {
		return err
	}
	err = targetCmd.Wait()
	if err != nil {
		return err
	}
	return nil
}
