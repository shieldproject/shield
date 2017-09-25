package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"

	"github.com/cloudfoundry/bosh-agent/jobsupervisor/pipe/syslog"
)

const ServiceName = "jimbob"
const MachineIP = "1.2.3.4"

func FindOpenPort() (int, error) {
	const Base = 5000
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 50; i++ {
		port := Base + rand.Intn(10000)
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			return 0, err
		}
		l, err := net.ListenUDP("udp", addr)
		if err != nil {
			continue
		}
		l.Close()
		return port, nil
	}
	return 0, errors.New("could not find open port to listen on")
}

var _ = Describe("Main", func() {
	It("should run the echo", func() {
		var stdout bytes.Buffer
		cmd := exec.Command(pathToPipeCLI, echoCmdArgs...)

		cmd.Stdout = &stdout

		Expect(cmd.Run()).To(Succeed())
		Expect(strings.TrimSpace(stdout.String())).To(Equal(echoOutput))
	})

	It("should return the exit code", func() {
		cmd := exec.Command(pathToPipeCLI, ExitCodePath, "-exitcode", "23")
		Expect(cmd.Run()).ToNot(Succeed())
		code, err := ExitCode(cmd)
		Expect(err).To(Succeed())
		Expect(code).To(Equal(23))
	})

	Context("HTTP", func() {
		var server *ghttp.Server
		var bodyCh chan []byte
		BeforeEach(func() {
			server = ghttp.NewServer()
			bodyCh = make(chan []byte, 100)
			server.RouteToHandler("POST", "/", func(w http.ResponseWriter, r *http.Request) {
				b, err := ioutil.ReadAll(r.Body)
				Expect(err).To(Succeed())
				r.Body.Close()
				bodyCh <- b
			})
		})
		AfterEach(func() {
			server.Close()
		})

		testNotifyHTTP := func() {
			cmd := exec.Command(pathToPipeCLI, ExitCodePath, "-exitcode", "23")
			cmd.Env = cmdEnv(
				joinEnv("NOTIFY_HTTP", server.URL()),
				joinEnv("SERVICE_NAME", "foo"),
			)
			Expect(cmd.Run()).ToNot(Succeed())

			code, err := ExitCode(cmd)
			Expect(err).To(Succeed())
			Expect(code).To(Equal(23))

			Expect(server.ReceivedRequests()).To(HaveLen(1))

			Expect(bodyCh).To(HaveLen(1))
			b := <-bodyCh
			var event Event
			Expect(json.Unmarshal(b, &event)).To(Succeed())
			Expect(event.ExitCode).To(Equal(23))
			Expect(event.ProcessName).To(Equal("foo"))
		}

		// On Concourse tests on Windows may be ran in a Pipe, which
		// will already have it's env vars set.  Make sure we don't
		// pass those env vars to the Pipe we are testing.
		It("overwrites Pipe specific NOTIFY_HTTP env vars during testing", func() {
			defer invalidatePipeEnvVars()()
			testNotifyHTTP()
		})

		It("should notify over http upon exit NEW", func() {
			testNotifyHTTP()
		})

		It("should notify when given nothing to run", func() {
			cmd := exec.Command(pathToPipeCLI)
			cmd.Env = cmdEnv(
				joinEnv("NOTIFY_HTTP", server.URL()),
				joinEnv("SERVICE_NAME", "foo"),
			)
			Expect(cmd.Run()).ToNot(Succeed())

			Expect(server.ReceivedRequests()).To(HaveLen(1))

			Expect(bodyCh).To(HaveLen(1))
			b := <-bodyCh
			var event Event
			Expect(json.Unmarshal(b, &event)).To(Succeed())
			Expect(event.ExitCode).To(Equal(1))
			Expect(event.ProcessName).To(Equal("foo"))
		})
	})

	Context("log dir", func() {
		var tempDir string
		BeforeEach(func() {
			var err error
			tempDir, err = ioutil.TempDir("", "something")
			Expect(err).To(Succeed())
		})
		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("never logs own behaviour to stdout/err", func() {
			cmd := exec.Command(pathToPipeCLI, echoCmdArgs...)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Env = os.Environ()
			Expect(cmd.Run()).To(Succeed())

			Expect(strings.TrimSpace(stdout.String())).To(Equal(echoOutput))
			Expect(stderr.Len()).To(Equal(0))
		})

		testLogToFile := func() {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd := exec.Command(pathToPipeCLI, echoCmdArgs...)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Env = cmdEnv(joinEnv("LOG_DIR", tempDir))
			Expect(cmd.Run()).To(Succeed())

			files, err := filepath.Glob(path.Join(tempDir, "*.log"))
			Expect(err).To(Succeed())
			Expect(files).To(HaveLen(1))

			pipeLog, err := ioutil.ReadFile(files[0])
			Expect(err).To(Succeed())
			Expect(string(pipeLog)).To(ContainSubstring("pipe:"))

			Expect(strings.TrimSpace(stdout.String())).To(Equal(echoOutput))
			Expect(stderr.Len()).To(Equal(0))
		}

		It("overwrites Pipe specific LOG_DIR env vars during testing", func() {
			defer invalidatePipeEnvVars()()
			testLogToFile()
		})

		It("logs own behaviour to file", func() {
			testLogToFile()
		})

		It("does not log when given an invalid log directory", func() {
			var invalidLogDir string
			randString := func() string {
				b := make([]byte, 8)
				n, _ := rand.Read(b)
				return fmt.Sprintf("%X", b[:n])
			}
			for i := 0; i < 1000; i++ {
				path := filepath.Join("/", randString(), randString(), randString())
				if _, err := os.Stat(path); err != nil {
					invalidLogDir = path
					break
				}
			}
			Expect(invalidLogDir).ToNot(Equal(""))

			cmd := exec.Command(pathToPipeCLI, echoCmdArgs...)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Env = cmdEnv(joinEnv("LOG_DIR", invalidLogDir))
			Expect(cmd.Run()).To(Succeed())

			_, err := os.Stat(invalidLogDir)
			Expect(err).ToNot(Succeed())

			Expect(strings.TrimSpace(stdout.String())).To(Equal(echoOutput))
			Expect(stderr.Len()).To(Equal(0))
		})
	})

	Context("syslog provided", func() {
		const Interval = time.Millisecond * 100
		const Start = 1
		const End = 5

		var ServerConn *net.UDPConn
		var ServerAddr *net.UDPAddr
		var syslogPort string
		var syslogReceived chan (string)

		var done chan struct{}
		var wg *sync.WaitGroup

		BeforeEach(func() {
			done = make(chan struct{})
			wg = new(sync.WaitGroup)

			port, err := FindOpenPort()
			Expect(err).To(Succeed())
			ServerAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
			Expect(err).To(Succeed())
			ServerConn, err = net.ListenUDP("udp", ServerAddr)
			Expect(err).To(Succeed())
			syslogPort = fmt.Sprintf("%d", ServerAddr.Port)

			syslogReceived = make(chan string, 100)
			wg.Add(1)
			go func() {
				defer wg.Done()
				buf := make([]byte, 65535) // theoretical max size of UDP datagram
				for {
					select {
					case <-done:
						return
					default:
						n, _, err := ServerConn.ReadFromUDP(buf)
						if err != nil {
							return
						}
						syslogReceived <- string(buf[0:n])
					}
				}
			}()
		})
		AfterEach(func() {
			ServerConn.Close()
			close(done)
			wg.Wait()
		})

		checkSequenceOutput := func(buf *bytes.Buffer, start, end int) error {
			out := strings.TrimSpace(buf.String())
			lines := strings.Split(out, "\n")
			expLen := end - start + 1
			if len(lines) != expLen {
				return fmt.Errorf("expected (%d) lines got: %d", expLen, len(lines))
			}
			Expect(lines).To(HaveLen(end - start + 1))
			var n int
			for i := start; i <= end; i++ {
				exp := strconv.Itoa(i)
				if lines[n] != exp {
					return fmt.Errorf("expected line (%d) to equal (%s) got: %s", i, exp, lines[n])
				}
				n++
			}
			return nil
		}

		It("ignores errors writing to syslog, allowing the app to continue functioning", func() {
			cmd := exec.Command(pathToPipeCLI, GoSequencePath,
				"-start", strconv.Itoa(Start),
				"-end", strconv.Itoa(End),
				"-int", Interval.String(),
			)
			cmd.Env = cmdEnv(
				joinEnv("SYSLOG_HOST", "localhost"),
				joinEnv("SYSLOG_PORT", syslogPort),
				joinEnv("SYSLOG_TRANSPORT", "udp"),
				joinEnv("SERVICE_NAME", ServiceName),
			)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout

			Expect(cmd.Start()).To(Succeed())
			go func() {
				time.Sleep((Interval / 2) * 3) // * 1.5
				ServerConn.Close()
			}()
			Expect(cmd.Wait()).To(Succeed())

			checkSequenceOutput(&stdout, Start, End)
		})

		testStdoutToSyslog := func() {
			cmd := exec.Command(pathToPipeCLI, GoSequencePath,
				"-start", strconv.Itoa(Start),
				"-end", strconv.Itoa(End),
				"-int", Interval.String(),
			)
			cmd.Env = cmdEnv(
				joinEnv("SYSLOG_HOST", "localhost"),
				joinEnv("SYSLOG_PORT", syslogPort),
				joinEnv("SYSLOG_TRANSPORT", "udp"),
				joinEnv("SERVICE_NAME", ServiceName),
				joinEnv("MACHINE_IP", MachineIP),
			)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout

			Expect(cmd.Run()).To(Succeed())
			for i := Start; i <= End; i++ {
				select {
				case out := <-syslogReceived:
					Expect(check(syslog.LOG_INFO, strconv.Itoa(i), out)).To(Succeed())
				case <-time.After(time.Second):
					Fail(fmt.Sprintf("timed out waiting for syslogReceived after: %s loop: %d", time.Second, i))
				}
			}
			Eventually(syslogReceived).ShouldNot(Receive())

			// test that stdout was still written to
			Expect(checkSequenceOutput(&stdout, Start, End)).To(Succeed())
		}

		It("overwrites Pipe specific SYSLOG env vars during testing", func() {
			defer invalidatePipeEnvVars()()
			testStdoutToSyslog()
		})

		It("logs stdout output to syslog", func() {
			testStdoutToSyslog()
		})

		It("logs stderr output to syslog", func() {
			cmd := exec.Command(pathToPipeCLI, GoSequencePath,
				"-start", strconv.Itoa(Start),
				"-end", strconv.Itoa(End),
				"-int", Interval.String(),
				"-out", "stderr",
			)
			cmd.Env = cmdEnv(
				joinEnv("SYSLOG_HOST", "localhost"),
				joinEnv("SYSLOG_PORT", syslogPort),
				joinEnv("SYSLOG_TRANSPORT", "udp"),
				joinEnv("SERVICE_NAME", ServiceName),
				joinEnv("MACHINE_IP", MachineIP),
			)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			Expect(cmd.Run()).To(Succeed())

			for i := Start; i <= End; i++ {
				select {
				case out := <-syslogReceived:
					Expect(check(syslog.LOG_WARNING, strconv.Itoa(i), out)).To(Succeed())
				case <-time.After(time.Second):
					Fail(fmt.Sprintf("timed out waiting for syslogReceived after: %s loop: %d", time.Second, i))
				}
			}
			Eventually(syslogReceived).ShouldNot(Receive())

			// test that stderr was still written to
			Expect(checkSequenceOutput(&stderr, Start, End)).To(Succeed())
		})
	})
})

func check(p syslog.Priority, in, out string) error {
	tmpl := fmt.Sprintf("<%d>%%s %%s %s[%%d]: %s\n", p, ServiceName, in)

	var parsedHostname, timestamp string
	var pid int

	n, err := fmt.Sscanf(out, tmpl, &timestamp, &parsedHostname, &pid)
	if n != 3 || err != nil || parsedHostname != MachineIP {
		return fmt.Errorf("Got %q, does not match template %q (%d %s)", out, tmpl, n, err)
	}
	return nil
}

func joinEnv(key, value string) string {
	return EnvPrefix + strings.TrimPrefix(key, EnvPrefix) + "=" + value
}

// cmdEnv returns a union of the current environment and envVars, and removes
// any Pipe specific variables not present in envVars.
func cmdEnv(envVars ...string) []string {
	env := os.Environ()
	if len(envVars) == 0 {
		return env
	}
	n := 0
	for i, s := range env {
		if !strings.HasPrefix(s, EnvPrefix) {
			env[n] = env[i]
			n++
		}
	}
	return append(env[:n], envVars...)
}

// invalidatePipeEnvVars stores invalid values in the Pipe specific variables
// of the current environment and returns a function to the reset any modified
// varaibles.
//
// Example:
//
//   defer invalidatePipeEnvVars()()
//   doWork...
//
func invalidatePipeEnvVars() (restore func()) {
	type val struct {
		s  string
		ok bool
	}
	lookupEnv := func(key string) val {
		s, ok := os.LookupEnv(key)
		return val{s, ok}
	}
	envVars := map[string]val{
		"__PIPE_LOG_DIR":          lookupEnv("__PIPE_LOG_DIR"),
		"__PIPE_MACHINE_IP":       lookupEnv("__PIPE_MACHINE_IP"),
		"__PIPE_NOTIFY_HTTP":      lookupEnv("__PIPE_NOTIFY_HTTP"),
		"__PIPE_SERVICE_NAME":     lookupEnv("__PIPE_SERVICE_NAME"),
		"__PIPE_SYSLOG_HOST":      lookupEnv("__PIPE_SYSLOG_HOST"),
		"__PIPE_SYSLOG_PORT":      lookupEnv("__PIPE_SYSLOG_PORT"),
		"__PIPE_SYSLOG_TRANSPORT": lookupEnv("__PIPE_SYSLOG_TRANSPORT"),
	}
	// set empty env vars
	for k := range envVars {
		os.Setenv(k, "")
	}
	// function to reset restore environemnt
	return func() {
		for k, v := range envVars {
			if v.ok {
				os.Setenv(k, v.s)
			} else {
				os.Unsetenv(k)
			}
		}
	}
}
