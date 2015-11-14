package supervisor

import (
	"bufio"
	"fmt"
	"github.com/pborman/uuid"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Operation int

const (
	BACKUP Operation = iota
	RESTORE
)

func (o Operation) String() string {
	switch o {
	case BACKUP:
		return "backup"
	case RESTORE:
		return "restore"
	default:
		return "UNKNOWN"
	}
}

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
	UUID uuid.UUID

	Store  *PluginConfig
	Target *PluginConfig

	Op     Operation
	Status Status

	StartedAt time.Time
	StoppedAt time.Time

	Output []string
}

func (t *Task) Run(output chan string, errors chan string) error {
	cmd := exec.Command("shield-pipe")
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("USER=%s", os.Getenv("USER")),
		fmt.Sprintf("LANG=%s", os.Getenv("LANG")),

		fmt.Sprintf("SHIELD_OP=%s", t.Op),
		fmt.Sprintf("SHIELD_STORE_PLUGIN=%s", t.Store.Plugin),
		fmt.Sprintf("SHIELD_STORE_ENDPOINT=%s", t.Store.Endpoint),
		fmt.Sprintf("SHIELD_TARGET_PLUGIN=%s", t.Target.Plugin),
		fmt.Sprintf("SHIELD_TARGET_ENDPOINT=%s", t.Target.Endpoint),
		fmt.Sprintf("SHIELD_RESTORE_KEY=%s", "FIXME"), // FIXME
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	drain := func(rd io.Reader, c chan string) {
		defer wg.Done()
		s := bufio.NewScanner(rd)
		for s.Scan() {
			c <- s.Text()
		}
		close(c)
	}

	wg.Add(2)
	go drain(stdout, output)
	go drain(stderr, errors)

	err = cmd.Start()
	if err != nil {
		return err
	}

	wg.Wait()

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
