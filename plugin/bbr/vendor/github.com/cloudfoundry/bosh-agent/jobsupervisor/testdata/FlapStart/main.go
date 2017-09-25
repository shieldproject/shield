package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

const HardTimeout = 5 * time.Minute

var FlapFile string

func init() {
	flag.StringVar(&FlapFile, "file", "", "File to record the current flap count")
	flag.StringVar(&FlapFile, "f", "Hello", "File to record the current flap count (shorthand)")
}

func realMain() error {
	flag.Parse()

	f, err := os.OpenFile(FlapFile, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return fmt.Errorf("Error: empty flap count file:", FlapFile)
	}

	n, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}

	if n > 0 {
		if err := f.Truncate(0); err != nil {
			return err
		}
		if _, err := f.Seek(0, 0); err != nil {
			return err
		}
		if _, err := f.WriteString(strconv.Itoa(n - 1)); err != nil {
			return err
		}
	}
	f.Close()

	// Flap then exit
	if n > 0 {
		return fmt.Errorf("Exiting count: %d", n)
	}

	t := time.Now()
	for time.Since(t) < HardTimeout {
		time.Sleep(time.Second * 30)
	}

	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(5)
	}
}
