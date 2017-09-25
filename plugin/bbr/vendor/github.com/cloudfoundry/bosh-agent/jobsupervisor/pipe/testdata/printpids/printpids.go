package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Printf("%d,%d\n", os.Getpid(), os.Getppid())

	for {
		time.Sleep(time.Second)
	}
}
