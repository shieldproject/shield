package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const Timeout = time.Minute * 2

var (
	SvcCount    int
	Description string
	Filename    string
)

func init() {
	flag.IntVar(&SvcCount, "count", 1, "Wait until this many services that match "+
		"the description are left before exiting")

	flag.StringVar(&Description, "description", "vcap_test", "Service description")
}

func DeleteServices(m *mgr.Mgr) (err error) {
	if m == nil {
		m, err = mgr.Connect()
		if err != nil {
			return err
		}
		defer m.Disconnect()
	}
	names, err := m.ListServices()
	if err != nil {
		return err
	}
	for _, name := range names {
		s, err := m.OpenService(name)
		if err != nil {
			continue
		}
		c, err := s.Config()
		if err == nil && c.Description == Description {
			s.Delete()
			st, err := s.Query()
			if err != nil {
				continue
			}
			if st.State != svc.Stopped || st.State != svc.StopPending {
				s.Control(svc.Stop)
			}
		}
		s.Close()
	}
	return nil
}

func ServiceNames(m *mgr.Mgr) ([]string, error) {
	list, err := m.ListServices()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, name := range list {
		s, err := m.OpenService(name)
		if err != nil {
			continue
		}
		c, err := s.Config()
		if err == nil && c.Description == Description {
			names = append(names, name)
		}
		s.Close()
	}
	return names, nil
}

func StopScript(interval time.Duration) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	names, err := ServiceNames(m)
	if err != nil {
		return err
	}

	start := time.Now()
	for {
		n := 0
		for _, name := range names {
			if s, err := m.OpenService(name); err == nil {
				s.Close()
				n++
			}
			if n == SvcCount {
				return nil
			}
		}
		time.Sleep(interval)
		if time.Since(start) > Timeout {
			DeleteServices(m)
		}
	}
	return nil
}

func Wait(interval time.Duration) error {
	start := time.Now()
	for {
		fmt.Printf("Waiting on file: %s\n", Filename)
		if _, err := os.Stat(Filename); err != nil {
			return nil
		}
		time.Sleep(interval)
		if time.Since(start) > Timeout {
			return errors.New("TIMEOUT")
		}
	}
	return nil
}

func Usage() {
	fmt.Fprintf(os.Stderr, "%s USAGE: [FLAGS] MODE STOPFILE\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 2 {
		Usage()
	}
	mode := flag.Arg(0)
	Filename = flag.Arg(1)

	if mode == "stop" {
		if SvcCount < 1 {
			fmt.Fprintln(os.Stderr, "Invalid 'count' argument:", SvcCount)
			Usage()
		}
		if Description == "" {
			fmt.Fprintln(os.Stderr, "Invalid 'description' argument:", Description)
			Usage()
		}
	}

	fmt.Println("Mode:", mode)
	fmt.Println("Filename:", Filename)
	fmt.Println("Description:", Description)
	fmt.Println("SvcCount:", SvcCount)

	switch mode {
	case "wait":
		if err := Wait(time.Millisecond * 100); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
	case "stop":
		if err := os.Remove(Filename); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		if err := StopScript(time.Millisecond * 500); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
	default:
		fmt.Fprintln(os.Stderr, "USAGE: MODE STOPFILE")
		os.Exit(1)
	}
	fmt.Println("Okay!")
}
