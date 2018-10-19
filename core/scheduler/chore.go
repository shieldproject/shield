package scheduler

import (
	"fmt"
)

type Chore struct {
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
	chore.Stdout <- fmt.Sprintf(msg, args...)
}

func (chore Chore) Errorf(msg string, args ...interface{}) {
	chore.Stderr <- fmt.Sprintf(msg, args...)
}

func (chore Chore) UnixExit(rc int) {
	chore.Exit <- [2]int{rc, 0}
	close(chore.Exit)
}

func (w *Worker) Execute(chore Chore) {
	w.Reserve()
	defer w.Release()

	chore.Do(chore)
}
