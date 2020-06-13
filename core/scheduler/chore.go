package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/db"
)

var next = 0

type Chore struct {
	ID       string
	TaskUUID string

	Do func(chore Chore)

	Stdout chan string
	Stderr chan string
	Exit   chan int
	Cancel chan bool
}

func NewChore(id string, do func(Chore)) Chore {
	next += 1
	return Chore{
		ID:       fmt.Sprintf("%s-%08d", time.Now().Format("20060102-150405"), next),
		TaskUUID: id,
		Do:       do,

		Stdout: make(chan string),
		Stderr: make(chan string),
		Exit:   make(chan int),
		Cancel: make(chan bool),
	}
}

func (chore Chore) String() string {
	return fmt.Sprintf("chore %s", chore.ID)
}

func (chore Chore) Infof(msg string, args ...interface{}) {
	log.Debugf(chore.String()+": stdout: "+msg, args...)
	chore.Stdout <- fmt.Sprintf(msg+"\n", args...)
}

func (chore Chore) Errorf(msg string, args ...interface{}) {
	log.Debugf(chore.String()+": stderr: "+msg, args...)
	chore.Stderr <- fmt.Sprintf(msg+"\n", args...)
}

func (chore Chore) UnixExit(rc int) {
	defer func() {
		recover()
	}()

	chore.Exit <- rc
	close(chore.Exit)
	log.Debugf("%s: exiting %d", chore, rc)
}

func (w *Worker) Execute(chore Chore) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s: %s", chore, err)
			w.db.UpdateTaskLog(chore.TaskUUID, fmt.Sprintf("\n\nERROR: %s\n\n", err))

			log.Errorf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
			w.db.FailTask(chore.TaskUUID, time.Now())
		}
	}()

	var wait sync.WaitGroup

	w.Reserve(chore.TaskUUID)
	defer w.Release()

	log.Infof("%s: %s executing chore for task '%s'", chore, w, chore.TaskUUID)
	w.db.StartTask(chore.TaskUUID, time.Now())

	log.Debugf("%s: spinning up [stderr] goroutine to watch chore stderr and update the task log...", chore)
	wait.Add(1)
	go func() {
		for s := range chore.Stderr {
			w.db.UpdateTaskLog(chore.TaskUUID, s)
		}
		log.Debugf("%s: no more standard error; [stderr] goroutine shutting down...", chore)
		wait.Done()
	}()

	log.Debugf("%s: spinning up [stdout] goroutine to watch chore stdout and accumulate the output...", chore)
	output := ""
	wait.Add(1)
	go func() {
		for s := range chore.Stdout {
			output += s
		}
		log.Debugf("%s: no more standard output; [stdout] goroutine shutting down...", chore)
		wait.Done()
	}()

	log.Debugf("%s: spinning up [exit] goroutine to watch chore exit status and remember it...", chore)
	rc := 0
	wait.Add(1)
	go func() {
		rc = <-chore.Exit
		log.Debugf("%s: rc %d noted; [exit] goroutine shutting down...", chore, rc)
		wait.Done()
	}()

	log.Debugf("%s: spinning up [main] goroutine to execute chore `do' function...", chore)
	wait.Add(1)
	go func() {
		chore.Do(chore)
		log.Debugf("%s: chore execution complete; [main] goroutine shutting down...", chore)

		chore.UnixExit(0) /* catch-all */
		close(chore.Stderr)
		close(chore.Stdout)
		wait.Done()
	}()

	log.Debugf("%s: waiting for chore to complete...", chore)
	wait.Wait()
	w.db.UpdateTaskLog(chore.TaskUUID, "\n\n------\n")

	task, err := w.db.GetTask(chore.TaskUUID)
	if err != nil {
		panic(fmt.Errorf("failed to retrieve task '%s' from database: %s", chore.TaskUUID, err))
	}

	switch task.Op {
	case db.BackupOperation:
		output = strings.TrimSpace(output)
		w.db.UpdateTaskLog(task.UUID, fmt.Sprintf("BACKUP: `%s`\n", output))

		if rc != 0 {
			log.Debugf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
			w.db.FailTask(chore.TaskUUID, time.Now())
			return
		}

		log.Infof("%s: parsing output of %s operation, '%s'", chore, task.Op, output)
		var v struct {
			Key  string `json:"key"`
			Size int64  `json:"archive_size"`
		}
		err := json.Unmarshal([]byte(output), &v)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshal output [%s] from %s operation: %s", output, task.Op, err))
		}

		if v.Key == "" {
			panic(fmt.Errorf("%s: no restore key detected in %s operation output", chore, task.Op))
		}

		w.db.UpdateTaskLog(task.UUID, fmt.Sprintf("BACKUP: restore key  = %s\n", v.Key))
		w.db.UpdateTaskLog(task.UUID, fmt.Sprintf("BACKUP: archive size = %d bytes\n", v.Size))

		log.Infof("%s: restore key for this %s operation is '%s'", chore, task.Op, v.Key)
		_, err = w.db.CreateTaskArchive(task.UUID, task.ArchiveUUID, v.Key, time.Now(),
			v.Size)
		if err != nil {
			panic(fmt.Errorf("failed to create task archive database record '%s': %s", task.ArchiveUUID, err))
		}

	case db.AgentStatusOperation:
		agent, err := w.db.GetAgentByAddress(task.Agent)
		if err != nil {
			log.Debugf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
			w.db.FailTask(chore.TaskUUID, time.Now())
			panic(fmt.Errorf("failed to retrieve agent '%s' from database: %s", task.Agent, err))
		}
		if agent == nil {
			log.Debugf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
			w.db.FailTask(chore.TaskUUID, time.Now())
			panic(fmt.Errorf("failed to retrieve agent '%s' from database: no such agent", task.Agent))
		}

		if rc == 0 {
			var v struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Health  string `json:"health"`
			}

			err = json.Unmarshal([]byte(output), &v)
			if err != nil {
				log.Debugf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
				w.db.FailTask(chore.TaskUUID, time.Now())
				panic(fmt.Errorf("failed to unmarshal output [%s] from %s operation: %s", output, task.Op, err))
			}

			agent.Name = v.Name
			agent.Version = v.Version
			agent.Status = v.Health
			agent.RawMeta = output
			agent.LastCheckedAt = time.Now().Unix()
			agent.LastError = ""
		} else {
			agent.Status = "error"
			agent.LastCheckedAt = time.Now().Unix()
			agent.LastError = fmt.Sprintf("The SHIELD Core was unable to check the status of this agent (see task %s)", chore.TaskUUID)
		}
		err = w.db.UpdateAgent(agent)
		if err != nil {
			log.Debugf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
			w.db.FailTask(chore.TaskUUID, time.Now())
			panic(fmt.Errorf("failed to update agent '%s' record in database: %s", task.Agent, err))
		}

		if rc != 0 {
			log.Debugf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
			w.db.FailTask(chore.TaskUUID, time.Now())
			return
		}
	}

	log.Debugf("%s: completing task '%s' in database", chore, chore.TaskUUID)
	w.db.CompleteTask(chore.TaskUUID, time.Now())
}
