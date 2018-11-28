package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/jhunt/go-log"
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
		log.Debugf("%s: no more standard error; [stderr] gooutine shutting down...", chore)
		wait.Done()
	}()

	log.Debugf("%s: spinning up [stdout] goroutine to watch chore stdout and accumulate the output...", chore)
	output := ""
	wait.Add(1)
	go func() {
		for s := range chore.Stdout {
			output += s
		}
		log.Debugf("%s: no more standard output; [stdout] gooutine shutting down...", chore)
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

	if rc == 0 {
		log.Debugf("%s: completing task '%s' in database", chore, chore.TaskUUID)
		w.db.CompleteTask(chore.TaskUUID, time.Now())
	} else {
		log.Debugf("%s: FAILING task '%s' in database", chore, chore.TaskUUID)
		w.db.FailTask(chore.TaskUUID, time.Now())
	}
}
