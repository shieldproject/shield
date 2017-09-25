package main

import (
	"os"
	"os/exec"
	"time"
)

func main() {
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Start()

	time.Sleep(time.Second * 1)
}
