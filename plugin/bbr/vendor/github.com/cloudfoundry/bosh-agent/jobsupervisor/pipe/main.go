package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cloudfoundry/bosh-agent/jobsupervisor/pipe/syslog"
)

type noopWriter struct{}

func (n noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// set log output to noop writer on program initialization.  We do not
// want to write any logs to stderr - only the underlying program should
// write to stderr and stdout.
func init() { log.SetOutput(noopWriter{}) }

const EnvPrefix = "__PIPE_"

type Config struct {
	ServiceName     string // "SERVICE_NAME"
	LogDir          string // "LOG_DIR"
	NotifyHTTP      string // "NOTIFY_HTTP"
	DisableNotify   bool   // "DISABLE_NOTIFY"
	SyslogHost      string // "SYSLOG_HOST"
	SyslogPort      string // "SYSLOG_PORT"
	SyslogTransport string // "SYSLOG_TRANSPORT"
	MachineIP       string // "MACHINE_IP"
}

func ParseConfig() *Config {
	c := &Config{
		ServiceName:     os.Getenv(EnvPrefix + "SERVICE_NAME"),
		LogDir:          os.Getenv(EnvPrefix + "LOG_DIR"),
		NotifyHTTP:      os.Getenv(EnvPrefix + "NOTIFY_HTTP"),
		SyslogHost:      os.Getenv(EnvPrefix + "SYSLOG_HOST"),
		SyslogPort:      os.Getenv(EnvPrefix + "SYSLOG_PORT"),
		SyslogTransport: os.Getenv(EnvPrefix + "SYSLOG_TRANSPORT"),
		MachineIP:       os.Getenv(EnvPrefix + "MACHINE_IP"),
	}
	if c.ServiceName == "" {
		c.ServiceName = os.Args[0]
	}
	if c.NotifyHTTP == "" {
		c.NotifyHTTP = "http://127.0.0.1:2825"
	}
	if s := os.Getenv(EnvPrefix + "DISABLE_NOTIFY"); s != "" {
		disable, err := strconv.ParseBool(s)
		c.DisableNotify = err == nil && disable
	}
	return c
}

func (c *Config) InitLog() (logEnabled bool) {
	dir := c.LogDir
	if dir == "" {
		return false
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return false
	}
	name := filepath.Join(dir, "pipe.log")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return false
	}
	log.SetOutput(f)
	return true
}

// errIncompleteSyslogConfig is returned if the syslog configuration is missing or incomplete.
var errIncompleteSyslogConfig = errors.New("syslog: configuration missing or incomplete")

func (c *Config) validSyslog() error {
	if c.SyslogHost == "" || c.SyslogPort == "" || c.SyslogTransport == "" || c.ServiceName == "" {
		return errIncompleteSyslogConfig
	}
	return nil
}

func (c *Config) Syslog() (outw *syslog.Writer, errw *syslog.Writer, err error) {
	if err = c.validSyslog(); err != nil {
		return
	}
	addr := c.SyslogHost + ":" + c.SyslogPort
	outw, err = syslog.DialHostname(c.SyslogTransport, addr, syslog.LOG_INFO, c.ServiceName, c.MachineIP)
	if err != nil {
		return
	}
	errw, err = syslog.DialHostname(c.SyslogTransport, addr, syslog.LOG_WARNING, c.ServiceName, c.MachineIP)
	if err != nil {
		return
	}
	return
}

type Event struct {
	Event       string `json:"event"`
	ProcessName string `json:"processName"`
	ExitCode    int    `json:"exitCode"`
}

func (c *Config) SendEvent(code int) error {
	if c.DisableNotify {
		return nil
	}
	v := Event{
		Event:       "pid failed",
		ProcessName: c.ServiceName,
		ExitCode:    code,
	}
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	res, err := http.Post(c.NotifyHTTP, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func watchParent(sigCh chan os.Signal) error {
	parent, err := os.FindProcess(os.Getppid())
	if err != nil {
		log.Printf("Failed to find parent process : %v\n", err)
		return err
	}
	go func() {
		_, err := parent.Wait()
		if err != nil {
			log.Printf("Failed to wait for parent process : %v\n", err)
			return
		}

		for {
			sigCh <- os.Kill
			time.Sleep(time.Millisecond * 10)
		}
	}()

	return nil
}

func closeChannel(haltCh chan struct{}) {
	// safely close the channel
	select {
	case _, open := <-haltCh: // channel likely closed
		if open {
			close(haltCh) // nope still open
		}
	default: // channel is open
		close(haltCh)
	}
}

func (c *Config) Run(path string, args []string, stdout, stderr io.Writer) (exitCode int, err error) {
	exitCode = 1
	defer func() {
		if e := recover(); e != nil {
			if err == nil {
				err = fmt.Errorf("run: panic: %v", e)
			} else {
				err = fmt.Errorf("run: panic: %v - previous error: %s", e, err)
			}
		}
	}()

	// Start collecting all signals immediately.  Any buffered signals will be
	// sent to the command once it starts.
	//
	// We want to be an invisible layer between the service wrapper and the
	// command - so we should not dictate how signals are handled, but instead
	// rely upon the commands signal handling - if it exits, we exit - if it
	// ignores the signal, we ignore it.
	sigCh := make(chan os.Signal, 64)
	signal.Notify(sigCh)

	err = watchParent(sigCh)
	if err != nil {
		log.Printf("watchParent: %v", err)
		return 1, err
	}
	haltCh := make(chan struct{})

	cmd := exec.Command(path, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = Environ()

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("starting command (%s): %s", path, err)
	}
	go func() {
		cmd.Wait()
		closeChannel(haltCh)
	}()

	// Critical section: make sure we do not miss an exiting process (fast or otherwise).
	go func(pid int) {
		select {
		case <-time.After(time.Millisecond * 250):
			break // assuming the cmd has started successfully
		case <-haltCh:
			return // process exited
		}
		tick := time.NewTicker(time.Millisecond * 100)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if err := FindProcess(pid); err != nil {
					// give wait a moment to return as it sets the ProcessState
					select {
					case <-time.After(time.Millisecond * 250):
						break
					case <-haltCh:
						return // wait returned
					}
					closeChannel(haltCh)
					return
				}
			case <-haltCh:
				return
			}
		}
	}(cmd.Process.Pid)

	// Send any buffered and all future Signals to command.
	go func() {
		for {
			select {
			case sig := <-sigCh:
				cmd.Process.Signal(sig)
			case <-haltCh:
				return
			}
		}
	}()

	// Wait for termination.
	<-haltCh

	exitCode, err = ExitCode(cmd)
	if err != nil {
		log.Printf("run: %s", err)
	}
	log.Printf("exit: %d", exitCode)

	return exitCode, err
}

func LookPath(file string) (string, error) {
	if filepath.Base(file) != file {
		return file, nil
	}
	lp, err := exec.LookPath(file)
	if err != nil {
		return "", err
	}
	return lp, nil
}

// Environ strips program specific variables from the environment.
func Environ() []string {
	e := os.Environ()
	for i, n := 0, 0; i < len(e); i++ {
		if !strings.HasPrefix(e[i], EnvPrefix) {
			e[n] = e[i]
			n++
		}
	}
	return e
}

// ExitCode returns the exit code for command cmd.
//
// TODO: document and standardize exit codes
func ExitCode(cmd *exec.Cmd) (int, error) {
	if cmd.ProcessState == nil {
		return 3, errors.New("exit code: nil process state")
	}
	switch v := cmd.ProcessState.Sys().(type) {
	case syscall.WaitStatus:
		return v.ExitStatus(), nil
	default:
		return 4, fmt.Errorf("exit code: unsuported type: %T", v)
	}
}

func FindProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err == nil {
		p.Release() // Close process handle
	}
	return err
}

type BulletproofWriter struct {
	w io.Writer
}

func (w *BulletproofWriter) Write(p []byte) (int, error) {
	if w.w != nil {
		w.w.Write(p)
	}
	return len(p), nil
}

func ParseArgs() (path string, args []string, err error) {
	if len(os.Args) < 2 {
		err = fmt.Errorf("usage: %s: [utility [argument ...]]", filepath.Base(os.Args[0]))
		return
	}
	path, err = LookPath(os.Args[1])
	if err != nil {
		return
	}
	args = os.Args[2:]
	return
}

func main() {
	conf := ParseConfig()
	conf.InitLog()
	log.Printf("pipe: configuration: %+v", conf)

	exit := func(code int) {
		if err := conf.SendEvent(code); err != nil {
			log.Printf("event: error %s", err)
		}
		log.Printf("pipe: exiting with code: %d", code)
		os.Exit(code)
	}

	// check after initializing logs
	path, args, err := ParseArgs()
	if err != nil {
		log.Printf("pipe: parsing args: %s", err)
		exit(1)
	}

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr

	outw, errw, err := conf.Syslog()
	switch err {
	case nil:
		stdout = io.MultiWriter(os.Stdout, &BulletproofWriter{w: outw})
		stderr = io.MultiWriter(os.Stderr, &BulletproofWriter{w: errw})
	case errIncompleteSyslogConfig:
		log.Println(err) // log and ignore
	default:
		log.Printf("syslog: error connecting: %s", err)
	}

	log.Println("pipe: starting")
	exitCode, err := conf.Run(path, args, stdout, stderr)
	if err != nil {
		log.Printf("pipe: error running command: %s", err)
	}
	exit(exitCode)
}
