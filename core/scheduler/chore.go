package scheduler

import (
	"fmt"

	"github.com/jhunt/go-log"
)

type Chore struct {
	TaskUUID string

	Do func(chore Chore)

	Stdout chan string
	Stderr chan string
	Exit   chan [2]int
	Cancel chan bool
}

func NewChore(do func(Chore)) Chore {
	return Chore{
		Do: do,

		Stdout: make(chan string),
		Stderr: make(chan string),
		Exit:   make(chan [2]int),
		Cancel: make(chan bool),
	}
}

func (chore Chore) Infof(msg string, args ...interface{}) {
	log.Debugf("scheduler INFO:  "+msg, args...)
	chore.Stdout <- fmt.Sprintf(msg, args...)
}

func (chore Chore) Errorf(msg string, args ...interface{}) {
	log.Debugf("scheduler ERROR: "+msg, args...)
	chore.Stderr <- fmt.Sprintf(msg, args...)
}

func (chore Chore) UnixExit(rc int) {
	log.Debugf("schedule chore exiting %d", rc)
	chore.Exit <- [2]int{rc, 0}
	close(chore.Exit)
}

func (w *Worker) Execute(chore Chore) {
	w.Reserve()
	defer w.Release()

	log.Infof("%s: executing chore for task '%s'", w, chore.TaskUUID)
	chore.Do(chore)
}
