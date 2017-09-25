// +build windows

package jobsupervisor_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
	"github.com/cloudfoundry/bosh-agent/jobsupervisor/winsvc"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	. "github.com/cloudfoundry/bosh-agent/jobsupervisor"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func init() {
	// Make sure we don't use 'vcap' as the service description,
	// otherwise we may destroy BOSH deployed Concourse workers.
	//
	// This is set in windows_job_supervisor_export_test.go
	if ServiceDescription == "vcap" {
		panic("ServiceDescription == 'vcap' - This is not allowed for tests!")
	}
}

const (
	jobFailuresServerPort = 5000
	DefaultMachineIP      = "127.0.0.1"
	DefaultTimeout        = time.Second * 15
	DefaultInterval       = time.Millisecond * 500
	DefaultEventPort      = 2825
)

var (
	StartStopExe string
	HelloExe     string
	WaitSvcExe   string
	FlapStartExe string
	TempDir      string

	ServiceDescription = GetServiceDescription()
)

var _ = AfterSuite(func() {
	os.RemoveAll(TempDir)
	gexec.CleanupBuildArtifacts()

	match := func(s string) bool {
		return s == ServiceDescription
	}
	m, err := winsvc.Connect(match)
	Expect(err).To(Succeed())
	defer m.Disconnect()

	Expect(m.Delete()).To(Succeed())
})

var _ = BeforeSuite(func() {
	var err error
	TempDir, err = ioutil.TempDir("", "bosh-")
	Expect(err).ToNot(HaveOccurred())

	StartStopExe, err = gexec.Build("github.com/cloudfoundry/bosh-agent/jobsupervisor/testdata/StartStop")
	Expect(err).ToNot(HaveOccurred())

	HelloExe, err = gexec.Build("github.com/cloudfoundry/bosh-agent/jobsupervisor/testdata/Hello")
	Expect(err).ToNot(HaveOccurred())

	FlapStartExe, err = gexec.Build("github.com/cloudfoundry/bosh-agent/jobsupervisor/testdata/FlapStart")
	Expect(err).ToNot(HaveOccurred())

	WaitSvcExe, err = gexec.Build("github.com/cloudfoundry/bosh-agent/jobsupervisor/testdata/WaitSvc")
	Expect(err).ToNot(HaveOccurred())
})

func testWindowsConfigs(jobName string) (WindowsProcessConfig, bool) {
	var procs []WindowsProcess
	switch jobName {
	case "say-hello":
		procs = []WindowsProcess{
			{
				Name:       fmt.Sprintf("say-hello-1-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Hello-1"},
			},
			{
				Name:       fmt.Sprintf("say-hello-2-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Hello-2"},
			},
		}
	case "say-hello-syslog":
		procs = []WindowsProcess{
			{
				Name:       fmt.Sprintf("say-hello-1-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Hello"},
				Env: map[string]string{
					"__PIPE_SYSLOG_HOST":      "localhost",
					"__PIPE_SYSLOG_PORT":      "10202",
					"__PIPE_SYSLOG_TRANSPORT": "udp",
				},
			},
		}
	case "flapping":
		procs = []WindowsProcess{
			{
				Name:       fmt.Sprintf("flapping-1-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Flapping-1", "-loop", "100ms", "-die", "2s", "-exit", "2"},
			},
			{
				Name:       fmt.Sprintf("flapping-2-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Flapping-2", "-loop", "100ms", "-die", "3s", "-exit", "2"},
			},
			{
				Name:       fmt.Sprintf("flapping-3-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Flapping-3", "-loop", "100ms", "-die", "4s", "-exit", "2"},
			},
		}
	case "looping":
		procs = []WindowsProcess{
			{
				Name:       fmt.Sprintf("looping-1-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Looping"},
			},
			{
				Name:       fmt.Sprintf("looping-2-%d", time.Now().UnixNano()),
				Executable: HelloExe,
				Args:       []string{"-message", "Looping-Subprocess-Creator", "-subproc"},
			},
		}
	case "stop-executable":
		// create temp file - used by stop-start jobs
		f, err := ioutil.TempFile(TempDir, "stopfile-")
		Expect(err).ToNot(HaveOccurred())
		tmpFileName := f.Name()
		f.Close()
		procs = []WindowsProcess{
			{
				Name:       fmt.Sprintf("stop-executable-1-%d", time.Now().UnixNano()),
				Executable: StartStopExe,
				Args:       []string{"start", tmpFileName},
				Stop: &StopCommand{
					Executable: StartStopExe,
					Args:       []string{"stop", tmpFileName},
				},
			},
		}
	default:
		return WindowsProcessConfig{}, false
	}

	return WindowsProcessConfig{Processes: procs}, true
}

func concurrentStopConfig() WindowsProcessConfig {
	// Five jobs that print in a loop
	const JobCount = 5

	// Two wait jobs that use stop scripts to until the
	// other jobs have stopped before stopping themselves
	const WaitCount = 2

	// Job config

	var conf WindowsProcessConfig
	for i := 0; i < JobCount; i++ {
		p := WindowsProcess{
			Name:       fmt.Sprintf("job-%d-%d", i, time.Now().UnixNano()),
			Executable: HelloExe,
			Args:       []string{"-message", fmt.Sprintf("Job-%d", i)},
		}
		conf.Processes = append(conf.Processes, p)
	}

	// Wait config

	createStopFile := func() string {
		f, err := ioutil.TempFile(TempDir, "stopfile-")
		Expect(err).ToNot(HaveOccurred())
		path := f.Name()
		f.Close()
		return path
	}

	for i := 0; i < WaitCount; i++ {
		stopFile := createStopFile()
		wait := WindowsProcess{
			Name:       fmt.Sprintf("wait-%d-%d", i, time.Now().UnixNano()),
			Executable: WaitSvcExe,
			Args:       []string{"wait", stopFile},
			Stop: &StopCommand{
				Executable: WaitSvcExe,
				Args: []string{
					"-count", strconv.Itoa(WaitCount),
					"-description", ServiceDescription,
					"stop", stopFile,
				},
			},
		}
		conf.Processes = append(conf.Processes, wait)
	}

	return conf
}

// If the interval is less than 1 the default is used
func flappingStartConfig(flapCount, jobCount int) (WindowsProcessConfig, error) {
	var conf WindowsProcessConfig

	if flapCount < 1 {
		return conf, fmt.Errorf("flappingStartConfig: invalid flap count: %d", flapCount)
	}
	if jobCount < 1 {
		return conf, fmt.Errorf("flappingStartConfig: invalid job count: %d", jobCount)
	}

	for i := 0; i < jobCount; i++ {
		f, err := ioutil.TempFile(TempDir, "flapping-")
		if err != nil {
			return conf, err
		}
		defer f.Close()

		if _, err := f.WriteString(strconv.Itoa(flapCount)); err != nil {
			return conf, err
		}

		p := WindowsProcess{
			Name:       fmt.Sprintf("flap-start-%d-%d", i, time.Now().UnixNano()),
			Executable: FlapStartExe,
			Args:       []string{"-file", f.Name()},
		}
		conf.Processes = append(conf.Processes, p)
	}

	return conf, nil
}

func buildPipeExe() error {
	pathToPipeCLI, err := gexec.Build("github.com/cloudfoundry/bosh-agent/jobsupervisor/pipe")
	if err != nil {
		return err
	}
	SetPipeExePath(pathToPipeCLI)
	return nil
}

type AlertHandler struct {
	mu    sync.Mutex
	alert boshalert.MonitAlert
}

func (a *AlertHandler) Set(alert boshalert.MonitAlert) {
	a.mu.Lock()
	a.alert = alert
	a.mu.Unlock()
}

func (a *AlertHandler) Get() boshalert.MonitAlert {
	a.mu.Lock()
	alert := a.alert
	a.mu.Unlock()
	return alert
}

var _ = Describe("WindowsJobSupervisor", func() {
	Context("add jobs and control services", func() {
		BeforeEach(func() {
			if runtime.GOOS != "windows" {
				Skip("Pending on non-Windows")
			}
		})

		var (
			once              sync.Once
			fs                boshsys.FileSystem
			logger            boshlog.Logger
			basePath          string
			logDir            string
			exePath           string
			jobDir            string
			processConfigPath string
			jobSupervisor     JobSupervisor
			runner            boshsys.CmdRunner
			logOut            *bytes.Buffer
		)

		BeforeEach(func() {
			once.Do(func() { Expect(buildPipeExe()).To(Succeed()) })

			const testExtPath = "testdata/job-service-wrapper"

			logOut = bytes.NewBufferString("")

			logger = boshlog.NewWriterLogger(boshlog.LevelDebug, logOut)
			fs = boshsys.NewOsFileSystem(logger)

			var err error
			basePath, err = ioutil.TempDir(TempDir, "")
			Expect(err).ToNot(HaveOccurred())
			fs.MkdirAll(basePath, 0755)

			binPath := filepath.Join(basePath, "bosh", "bin")
			fs.MkdirAll(binPath, 0755)

			logDir = path.Join(basePath, "sys", "log")
			fs.MkdirAll(binPath, 0755)

			exePath = filepath.Join(binPath, "job-service-wrapper.exe")

			err = fs.CopyFile(testExtPath, exePath)
			Expect(err).ToNot(HaveOccurred())

			logDir = path.Join(basePath, "sys", "log")
		})

		WriteJobConfig := func(configContents WindowsProcessConfig) (string, error) {
			dirProvider := boshdirs.NewProvider(basePath)
			runner = boshsys.NewExecCmdRunner(logger)
			jobSupervisor = NewWindowsJobSupervisor(runner, dirProvider, fs, logger, jobFailuresServerPort,
				make(chan bool), DefaultMachineIP)
			if err := jobSupervisor.RemoveAllJobs(); err != nil {
				return "", err
			}
			processConfigContents, err := json.Marshal(configContents)
			if err != nil {
				return "", err
			}

			jobDir, err = fs.TempDir("testWindowsJobSupervisor")
			processConfigPath = filepath.Join(jobDir, "monit")

			err = fs.WriteFile(processConfigPath, processConfigContents)
			return processConfigPath, err
		}

		AddJob := func(jobName string) (WindowsProcessConfig, error) {
			conf, ok := testWindowsConfigs(jobName)
			if !ok {
				return conf, fmt.Errorf("Invalid Windows Config Process name: %s", jobName)
			}
			confPath, err := WriteJobConfig(conf)
			if err != nil {
				return conf, err
			}
			return conf, jobSupervisor.AddJob(jobName, 0, confPath)
		}

		// AddFlappingJob adds and starts flapping job jobName and waits for it
		// to fail at least once.
		AddFlappingJob := func(jobName string) (*WindowsProcessConfig, error) {
			conf, err := AddJob(jobName)
			if err != nil {
				return nil, err
			}
			if err := jobSupervisor.Start(); err != nil {
				return nil, err
			}
			m, err := mgr.Connect()
			if err != nil {
				return nil, err
			}
			defer m.Disconnect()

			var svcs []*mgr.Service
			defer func() {
				for _, s := range svcs {
					s.Close()
				}
			}()

			for _, proc := range conf.Processes {
				s, err := m.OpenService(proc.Name)
				if err != nil {
					return nil, err
				}
				svcs = append(svcs, s)
			}

			timeout := time.After(time.Second * 10)
			tick := time.NewTicker(time.Millisecond * 50)
			defer tick.Stop()

		OuterLoop:
			for {
				select {
				case <-tick.C:
					for _, s := range svcs {
						st, err := s.Query()
						if err != nil {
							return nil, err
						}
						if st.State == svc.Stopped || st.State == svc.StopPending {
							break OuterLoop
						}
					}
				case <-timeout:
					return nil, errors.New("AddFlappingJob: timed out waiting for job to fail")
				}
			}

			return &conf, nil
		}

		AddFlappingStartJob := func(flapCount, jobCount int) (*WindowsProcessConfig, error) {
			conf, err := flappingStartConfig(flapCount, jobCount)
			if err != nil {
				return nil, err
			}
			confPath, err := WriteJobConfig(conf)
			if err != nil {
				return nil, err
			}
			return &conf, jobSupervisor.AddJob("flap-start", 0, confPath)
		}

		GetServiceState := func(serviceName string) (svc.State, error) {
			m, err := mgr.Connect()
			if err != nil {
				return 0, err
			}
			defer m.Disconnect()
			s, err := m.OpenService(serviceName)
			if err != nil {
				return 0, err
			}
			defer s.Close()
			st, err := s.Query()
			if err != nil {
				return 0, err
			}
			return st.State, nil
		}

		GetServiceInfo := func(svcName string) (svc.Status, mgr.Config, error) {
			m, err := mgr.Connect()
			if err != nil {
				return svc.Status{}, mgr.Config{}, err
			}
			defer m.Disconnect()

			s, err := m.OpenService(svcName)
			if err != nil {
				return svc.Status{}, mgr.Config{}, err
			}
			defer s.Close()

			status, err := s.Query()
			if err != nil {
				return svc.Status{}, mgr.Config{}, err
			}

			conf, err := s.Config()
			if err != nil {
				return svc.Status{}, mgr.Config{}, err
			}

			return status, conf, nil
		}

		AfterEach(func() {
			Expect(jobSupervisor.Stop()).To(Succeed())
			Expect(jobSupervisor.RemoveAllJobs()).To(Succeed())
			Expect(fs.RemoveAll(jobDir)).To(Succeed())
			Expect(fs.RemoveAll(logDir)).To(Succeed())
		})

		Describe("Processes", func() {
			It("list the process under vcap description", func() {

				conf, err := AddJob("say-hello")
				names := make(map[string]bool)
				for _, p := range conf.Processes {
					names[p.Name] = true
				}

				Expect(err).ToNot(HaveOccurred())
				Expect(jobSupervisor.Start()).To(Succeed())

				allProcsAreRunning := func() bool {
					procs, err := jobSupervisor.Processes()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(procs)).To(Equal(len(conf.Processes)))
					for _, p := range procs {
						Expect(names).To(HaveKey(p.Name))
						if p.State != "running" && p.State != "starting" {
							return false
						}
					}
					return true
				}
				Eventually(allProcsAreRunning, DefaultTimeout, DefaultInterval).Should(BeTrue())
			})

			It("lists the status of stopped process under vcap description", func() {

				conf, err := AddJob("say-hello")
				Expect(err).ToNot(HaveOccurred())
				Expect(jobSupervisor.Start()).To(Succeed())
				Expect(jobSupervisor.Stop()).To(Succeed())

				procs, err := jobSupervisor.Processes()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(procs)).To(Equal(len(conf.Processes)))

				names := make(map[string]bool)
				for _, p := range conf.Processes {
					names[p.Name] = true
				}
				for _, p := range procs {
					Expect(names).To(HaveKey(p.Name))
					Expect(p.State).To(Equal("stopped"))
					Expect(int(p.CPU.Total)).To(Equal(0))
					Expect(int(p.CPU.Total)).To(Equal(0))
					Expect(int(p.Memory.Kb)).To(Equal(0))
				}
			})
		})

		Describe("AddJob", func() {
			It("creates a service with vcap description", func() {
				conf, err := AddJob("say-hello")
				Expect(err).ToNot(HaveOccurred())

				for _, proc := range conf.Processes {
					status, conf, err := GetServiceInfo(proc.Name)
					Expect(err).ToNot(HaveOccurred())

					Expect(conf.Description).To(Equal(ServiceDescription))
					Expect(status.State).To(Equal(svc.Stopped))
				}
			})

			Context("when monit file is empty", func() {
				BeforeEach(func() {
					Expect(fs.WriteFileString(processConfigPath, "")).To(Succeed())
				})

				It("does not return an error", func() {
					_, err := AddJob("say-hello")
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Describe("Start", func() {
			var conf WindowsProcessConfig
			BeforeEach(func() {
				var err error
				conf, err = AddJob("say-hello")
				Expect(err).ToNot(HaveOccurred())
			})

			It("will start all the services", func() {
				Expect(jobSupervisor.Start()).To(Succeed())

				for _, proc := range conf.Processes {
					state, err := GetServiceState(proc.Name)
					Expect(err).ToNot(HaveOccurred())
					Expect(state).To(Equal(svc.Running))
				}
			})

			It("writes logs to job log directory", func() {
				Expect(jobSupervisor.Start()).To(Succeed())

				for i, proc := range conf.Processes {
					readLogFile := func() (string, error) {
						return fs.ReadFileString(path.Join(logDir, "say-hello", proc.Name, "job-service-wrapper.out.log"))
					}
					Eventually(readLogFile, DefaultTimeout, DefaultInterval).Should(ContainSubstring(fmt.Sprintf("Hello-%d", i+1)))
				}
			})

			It("sets the LOG_DIR env variable for the pipe", func() {
				Expect(jobSupervisor.Start()).To(Succeed())

				validFile := func(name string) func() error {
					return func() error {
						fi, err := os.Stat(name)
						if err != nil {
							return err
						}
						if fi.Size() == 0 {
							return fmt.Errorf("empty file: %s", name)
						}
						return nil
					}
				}

				for _, proc := range conf.Processes {
					pipeLogPath := filepath.Join(logDir, "say-hello", proc.Name, "pipe.log")
					Eventually(validFile(pipeLogPath), DefaultTimeout, DefaultInterval).Should(Succeed())
				}
			})

			It("sets the SERVICE_NAME env variable for the pipe", func() {
				Expect(jobSupervisor.Start()).To(Succeed())

				fileContains := func(filename, substring string) func() error {
					return func() error {
						b, err := ioutil.ReadFile(filename)
						if err != nil {
							return err
						}
						if !bytes.Contains(b, []byte(substring)) {
							return fmt.Errorf("file %s does not contain substring: %s", filename, substring)
						}
						return nil
					}
				}

				for _, proc := range conf.Processes {
					pipeLogPath := filepath.Join(logDir, "say-hello", proc.Name, "pipe.log")
					Eventually(fileContains(pipeLogPath, proc.Name), DefaultTimeout, DefaultInterval).Should(Succeed())
				}
			})
		})

		Describe("Start Flapping Jobs", func() {

			testStartFlappingJobs := func(flapCount, jobCount int) {
				// Prevent Pipe.exe from sending failure notifications as there
				// is nothing listening and the delay of trying to send the
				// notification causes WinSW to think the process is actually
				// running when it is not.  That is because WinSW monitors
				// pipe.exe - not the underlying process.
				os.Setenv("__PIPE_DISABLE_NOTIFY", strconv.FormatBool(true))
				defer os.Unsetenv("__PIPE_DISABLE_NOTIFY")

				conf, err := AddFlappingStartJob(flapCount, jobCount)
				Expect(err).ToNot(HaveOccurred())

				Expect(jobSupervisor.Start()).To(Succeed())

				for i := 0; i < 5; i++ {
					for _, p := range conf.Processes {
						st, err := GetServiceState(p.Name)
						Expect(err).To(Succeed())
						Expect(SvcStateString(st)).To(Equal(SvcStateString(svc.Running)))
					}
					time.Sleep(time.Second)
				}
			}

			It("starts one flapping service", func() {
				testStartFlappingJobs(3, 1)
			})

			It("starts many flapping services", func() {
				testStartFlappingJobs(1, 5)
			})
		})

		Describe("Status", func() {
			Context("with jobs", func() {
				BeforeEach(func() {
					_, err := AddJob("say-hello")
					Expect(err).ToNot(HaveOccurred())
				})

				Context("when running", func() {
					It("reports that the job is 'Running'", func() {
						Expect(jobSupervisor.Start()).To(Succeed())

						Expect(jobSupervisor.Status()).To(Equal("running"))
					})
				})

				Context("when stopped", func() {
					It("reports that the job is 'Stopped'", func() {
						Expect(jobSupervisor.Start()).To(Succeed())

						Expect(jobSupervisor.Stop()).To(Succeed())

						Expect(jobSupervisor.Status()).To(Equal("stopped"))
					})
				})
			})

			Context("with no jobs", func() {
				Context("when running", func() {
					It("reports that the job is 'Running'", func() {
						Expect(jobSupervisor.Start()).To(Succeed())

						Expect(jobSupervisor.Status()).To(Equal("running"))
					})
				})
			})
		})

		Describe("Unmonitor", func() {
			var conf WindowsProcessConfig
			BeforeEach(func() {
				var err error
				conf, err = AddJob("say-hello")
				Expect(err).ToNot(HaveOccurred())
			})

			It("sets service status to Disabled", func() {
				Expect(jobSupervisor.Unmonitor()).To(Succeed())

				for _, proc := range conf.Processes {
					_, conf, err := GetServiceInfo(proc.Name)
					Expect(err).ToNot(HaveOccurred())
					Expect(uint(conf.StartType)).To(Equal(uint(mgr.StartDisabled)))
				}
			})
		})

		Describe("StopAndWait", func() {
			It("waits for the services to be stopped", func() {
				conf, err := AddJob("looping")
				Expect(err).ToNot(HaveOccurred())

				Expect(jobSupervisor.Start()).To(Succeed())
				Expect(jobSupervisor.StopAndWait()).To(Succeed())

				for _, proc := range conf.Processes {
					st, err := GetServiceState(proc.Name)
					Expect(err).To(Succeed())
					Expect(SvcStateString(st)).To(Equal(SvcStateString(svc.Stopped)))
				}
			})

			It("stops flapping service", func() {
				conf, err := AddFlappingJob("flapping")
				Expect(err).To(Succeed())
				Expect(jobSupervisor.StopAndWait()).To(Succeed())

				Consistently(func() bool {
					stopped := true
					for _, proc := range conf.Processes {
						st, err := GetServiceState(proc.Name)
						if err != nil || st != svc.Stopped {
							stopped = false
						}
					}
					return stopped
				}, time.Second*6, time.Millisecond*10).Should(BeTrue())
			})
		})

		Describe("Stop", func() {

			It("sets service status to Stopped", func() {
				conf, err := AddJob("say-hello")
				Expect(err).ToNot(HaveOccurred())

				Expect(jobSupervisor.Start()).To(Succeed())
				Expect(jobSupervisor.Stop()).To(Succeed())

				for _, proc := range conf.Processes {
					Eventually(func() (string, error) {
						st, err := GetServiceState(proc.Name)
						return SvcStateString(st), err
					}).Should(Equal(SvcStateString(svc.Stopped)))
				}
			})

			It("can start a stopped service", func() {
				conf, err := AddJob("say-hello")
				Expect(err).ToNot(HaveOccurred())

				Expect(jobSupervisor.Start()).To(Succeed())
				Expect(jobSupervisor.Stop()).To(Succeed())

				for _, proc := range conf.Processes {
					Eventually(func() (string, error) {
						st, err := GetServiceState(proc.Name)
						return SvcStateString(st), err
					}).Should(Equal(SvcStateString(svc.Stopped)))
				}

				Expect(jobSupervisor.Start()).To(Succeed())
				for _, proc := range conf.Processes {
					Eventually(func() (string, error) {
						st, err := GetServiceState(proc.Name)
						return SvcStateString(st), err
					}).Should(Equal(SvcStateString(svc.Running)))
				}
			})

			It("stops flapping services", func() {
				conf, err := AddFlappingJob("flapping")
				Expect(err).ToNot(HaveOccurred())

				Expect(jobSupervisor.Stop()).To(Succeed())

				proc := conf.Processes[0]
				Eventually(func() (string, error) {
					st, err := GetServiceState(proc.Name)
					return SvcStateString(st), err
				}, time.Second*6).Should(Equal(SvcStateString(svc.Stopped)))

				Consistently(func() (string, error) {
					st, err := GetServiceState(proc.Name)
					return SvcStateString(st), err
				}, time.Second*6, time.Millisecond*100).Should(Equal(SvcStateString(svc.Stopped)))
			})

			It("stops flapping services and gives a status of stopped", func() {
				const wait = time.Second * 6
				const freq = time.Millisecond * 100
				const loops = int(time.Second * 10 / freq)

				_, err := AddFlappingJob("flapping")
				Expect(err).ToNot(HaveOccurred())

				Expect(jobSupervisor.Stop()).To(Succeed())
				for i := 0; i < loops && jobSupervisor.Status() != "stopped"; i++ {
					time.Sleep(freq)
				}
				Consistently(jobSupervisor.Status, wait).Should(Equal("stopped"))
			})

			It("stops services concurrently", func() {
				conf := concurrentStopConfig()
				confPath, err := WriteJobConfig(conf)
				Expect(err).To(Succeed())
				Expect(jobSupervisor.AddJob("ConcurrentWait", 0, confPath)).To(Succeed())

				defer jobSupervisor.RemoveAllJobs()
				Expect(jobSupervisor.Start()).To(Succeed())

				// WARN WARN WARN
				time.Sleep(time.Second * 10)

				Expect(jobSupervisor.Stop()).To(Succeed())
			})
		})

		Describe("StopCommand", func() {

			It("uses the stop executable to stop the process", func() {
				conf, err := AddJob("stop-executable")
				Expect(err).ToNot(HaveOccurred())

				Expect(jobSupervisor.Start()).To(Succeed())
				Expect(jobSupervisor.Stop()).To(Succeed())

				for _, proc := range conf.Processes {
					Eventually(func() (string, error) {
						st, err := GetServiceState(proc.Name)
						return SvcStateString(st), err
					}, DefaultTimeout, DefaultInterval).Should(Equal(SvcStateString(svc.Stopped)))
				}
			})
		})

		Describe("MonitorJobFailures", func() {
			var cancelServer chan bool
			var dirProvider boshdirs.Provider
			const failureRequest = `{
				"event": "pid failed",
				"exitCode": 55,
				"processName": "nats"
			}`
			BeforeEach(func() {
				dirProvider = boshdirs.NewProvider(basePath)
				runner = boshsys.NewExecCmdRunner(logger)
				cancelServer = make(chan bool)
				jobSupervisor = NewWindowsJobSupervisor(runner, dirProvider, fs, logger, jobFailuresServerPort,
					cancelServer, DefaultMachineIP)
			})

			AfterEach(func() {
				close(cancelServer)
			})

			doJobFailureRequest := func(payload string, port int) error {
				url := fmt.Sprintf("http://localhost:%d", port)
				_, err := http.Post(url, "application/json", strings.NewReader(payload))
				return err
			}

			expectedMonitAlert := func(recieved boshalert.MonitAlert) interface{} {
				date, err := time.Parse(time.RFC1123Z, recieved.Date)
				if err != nil {
					return err
				}
				return boshalert.MonitAlert{
					ID:          "nats",
					Service:     "nats",
					Event:       "pid failed",
					Action:      "Start",
					Date:        date.Format(time.RFC1123Z),
					Description: "exited with code 55",
				}
			}

			It("sends alerts for a flapping service", func() {

				var handledAlert AlertHandler
				alertReceived := make(chan bool, 1)
				failureHandler := func(alert boshalert.MonitAlert) (err error) {
					alertReceived <- true
					handledAlert.Set(alert)
					return
				}
				go jobSupervisor.MonitorJobFailures(failureHandler)

				_, err := AddFlappingJob("flapping")
				Expect(err).To(Succeed())
				Eventually(alertReceived, time.Second*6).Should(Receive())

				Expect(handledAlert.Get().ID).To(ContainSubstring("flapping"))
				Expect(handledAlert.Get().Event).To(Equal("pid failed"))
			})

			It("receives job failures from the service wrapper via HTTP", func() {
				var handledAlert AlertHandler
				failureHandler := func(alert boshalert.MonitAlert) (err error) {
					handledAlert.Set(alert)
					return
				}

				go jobSupervisor.MonitorJobFailures(failureHandler)

				err := doJobFailureRequest(failureRequest, jobFailuresServerPort)
				Expect(err).ToNot(HaveOccurred())

				alert := handledAlert.Get()
				Expect(alert).To(Equal(expectedMonitAlert(alert)))
			})

			It("stops sending failures after a call to Unmonitor", func() {
				var handledAlert AlertHandler
				failureHandler := func(alert boshalert.MonitAlert) (err error) {
					handledAlert.Set(alert)
					return
				}
				go jobSupervisor.MonitorJobFailures(failureHandler)

				// Unmonitor jobs
				Expect(jobSupervisor.Unmonitor()).To(Succeed())

				err := doJobFailureRequest(failureRequest, jobFailuresServerPort)
				Expect(err).ToNot(HaveOccurred())

				// Should match empty MonitAlert
				Expect(handledAlert.Get()).To(Equal(boshalert.MonitAlert{}))
			})

			It("re-monitors all jobs after a call to start", func() {
				var handledAlert AlertHandler
				failureHandler := func(alert boshalert.MonitAlert) (err error) {
					handledAlert.Set(alert)
					return
				}
				go jobSupervisor.MonitorJobFailures(failureHandler)

				// Unmonitor jobs
				Expect(jobSupervisor.Unmonitor()).To(Succeed())

				err := doJobFailureRequest(failureRequest, jobFailuresServerPort)
				Expect(err).ToNot(HaveOccurred())

				// Should match empty MonitAlert
				Expect(handledAlert.Get()).To(Equal(boshalert.MonitAlert{}))

				// Start should re-monitor all jobs
				Expect(jobSupervisor.Start()).To(Succeed())

				err = doJobFailureRequest(failureRequest, jobFailuresServerPort)
				Expect(err).ToNot(HaveOccurred())

				alert := handledAlert.Get()
				Expect(alert).To(Equal(expectedMonitAlert(alert)))
			})

			It("ignores unknown requests", func() {
				var didHandleAlert int32
				failureHandler := func(alert boshalert.MonitAlert) (err error) {
					atomic.StoreInt32(&didHandleAlert, 1)
					return
				}
				go jobSupervisor.MonitorJobFailures(failureHandler)

				err := doJobFailureRequest(`some bad request`, jobFailuresServerPort)
				Expect(err).ToNot(HaveOccurred())
				Expect(int(atomic.LoadInt32(&didHandleAlert))).To(Equal(0))
				Expect(logOut.Bytes()).To(ContainSubstring("MonitorJobFailures: received unknown request"))
			})

			It("returns an error when it fails to bind", func() {
				failureHandler := func(alert boshalert.MonitAlert) (err error) { return }

				go jobSupervisor.MonitorJobFailures(failureHandler)
				time.Sleep(50 * time.Millisecond)
				err := jobSupervisor.MonitorJobFailures(failureHandler)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the WindowsProcess has syslog environment variables", func() {
			var ServerConn *net.UDPConn
			var syslogReceived chan (string)
			BeforeEach(func() {
				ServerAddr, err := net.ResolveUDPAddr("udp", ":10202")
				Expect(err).To(Succeed())
				ServerConn, err = net.ListenUDP("udp", ServerAddr)
				Expect(err).To(Succeed())

				syslogReceived = make(chan (string), 1)
				go func() {
					buf := make([]byte, 1024)
					for {
						n, _, err := ServerConn.ReadFromUDP(buf)
						if err == nil {
							syslogReceived <- string(buf[0:n])
						} else {
							return
						}
					}
				}()
			})
			AfterEach(func() {
				ServerConn.Close()
			})

			// Test that the syslog message s matches pattern:
			// <6>2017-02-01T10:14:58-05:00 127.0.0.1 say-hello-1-123[100]: Hello 1
			matchSyslogMsg := func(s string) {
				const tmpl = "<6>%s %s say-hello-1-%d[%d]: Hello"
				var (
					id        int
					pid       int
					timeStamp string
					ipAddr    string
				)
				s = strings.TrimSpace(s)
				n, err := fmt.Sscanf(s, tmpl, &timeStamp, &ipAddr, &id, &pid)
				if n != 4 || err != nil {
					Expect(fmt.Errorf("Got %q, does not match template %q (%d %s)",
						s, tmpl, n, err)).To(Succeed())
				}

				_, err = time.Parse(time.RFC3339, timeStamp)
				Expect(err).To(Succeed())

				Expect(ipAddr).To(Equal(DefaultMachineIP))

				Expect(id).ToNot(Equal(0))
				Expect(pid).ToNot(Equal(0))
			}

			It("report the logs", func(done Done) {
				_, err := AddJob("say-hello-syslog")
				Expect(err).To(Succeed())
				Expect(jobSupervisor.Start()).To(Succeed())
				syslogMsg := <-syslogReceived
				matchSyslogMsg(syslogMsg)
				close(done)
			}, 20)
		})
	})

	Describe("WindowsProcess#ServiceWrapperConfig", func() {
		It("adds the pipe.exe environment variables to the winsw XML", func() {
			proc := WindowsProcess{
				Name:       "Name",
				Executable: "Executable",
				Args:       []string{"A"},
			}
			srvc := proc.ServiceWrapperConfig("LogPath", 123, DefaultMachineIP)
			envs := make(map[string]string)
			for _, e := range srvc.Env {
				envs[e.Name] = e.Value
			}
			Expect(envs["__PIPE_LOG_DIR"]).To(Equal("LogPath"))
			Expect(envs["__PIPE_NOTIFY_HTTP"]).To(Equal(fmt.Sprintf("http://localhost:%d", 123)))
		})

		Context("when the WindowsProcess has environment variables", func() {
			It("adds them to the marshalled WindowsServiceWrapperConfig XML", func() {
				proc := WindowsProcess{
					Name:       "Name",
					Executable: "Executable",
					Args:       []string{"A", "B"},
					Env: map[string]string{
						"Key_1": "Val_1",
						"Key_2": "Val_2",
					},
				}
				srvc := proc.ServiceWrapperConfig("LogPath", 0, DefaultMachineIP)
				srvcHash := map[string]string{}
				for _, e := range srvc.Env {
					srvcHash[e.Name] = e.Value
				}

				for key, value := range proc.Env {
					Expect(value).To(Equal(srvcHash[key]))
				}
			})
		})

		Context("when stop arguments or executable are provided", func() {
			var proc WindowsProcess
			BeforeEach(func() {
				proc = WindowsProcess{
					Name:       "Name",
					Executable: "Executable",
					Args:       []string{"Start_1", "Start_2"},
				}
			})

			It("uses 'startargument' instead of 'argment'", func() {
				proc.Stop = &StopCommand{
					Executable: "STOPPER",
					Args:       []string{"Stop_1", "Stop_2"},
				}
				srvc := proc.ServiceWrapperConfig("LogPath", 0, DefaultMachineIP)
				Expect(srvc.Arguments).To(HaveLen(0))
				args := append([]string{proc.Executable}, proc.Args...)
				Expect(srvc.StartArguments).To(Equal(args))
				Expect(srvc.StopArguments).To(Equal(proc.Stop.Args))
				Expect(srvc.StopExecutable).To(Equal(proc.Stop.Executable))
			})

			// FIXME (CEV & MH): This is temporary workaround until this is fixed
			// in WinSW.
			It("it only adds stop executable if stop args are supplied", func() {
				proc.Stop = &StopCommand{
					Executable: "STOPPER",
				}
				srvc := proc.ServiceWrapperConfig("LogPath", 0, DefaultMachineIP)
				args := append([]string{proc.Executable}, proc.Args...)
				Expect(srvc.Arguments).To(Equal(args))
				Expect(srvc.StartArguments).To(HaveLen(0))
				Expect(srvc.StopArguments).To(HaveLen(0))
			})

			It("uses the process executable when no stop executable is provided - not pipe.exe", func() {
				proc.Stop = &StopCommand{
					Args: []string{"Stop_1"},
				}
				srvc := proc.ServiceWrapperConfig("LogPath", 0, DefaultMachineIP)
				Expect(srvc.StopExecutable).To(Equal(proc.Executable))
			})
		})
	})
})

/*
func XXX_testWindowsConfigs(jobName string) (WindowsProcessConfig, bool) {
	// create temp file - used by stop-start jobs
	f, err := ioutil.TempFile(TempDir, "stopfile-")
	Expect(err).ToNot(HaveOccurred())
	tmpFileName := f.Name()
	f.Close()

	m := map[string]WindowsProcessConfig{
		"say-hello": WindowsProcessConfig{
			Processes: []WindowsProcess{
				{
					Name:       fmt.Sprintf("say-hello-1-%d", time.Now().UnixNano()),
					Executable: "cmd.exe",
					Args:       []string{"/C", `@echo off & for /l %x in (1, 1, 10000) do echo "Hello-1" & timeout /t 10 /nobreak > NUL`},
				},
				{
					Name:       fmt.Sprintf("say-hello-2-%d", time.Now().UnixNano()),
					Executable: "cmd.exe",
					Args:       []string{"/C", `@echo off & for /l %x in (1, 1, 10000) do echo "Hello-2" & timeout /t 10 /nobreak > NUL`},
				},
			},
		},
		"say-hello-syslog": WindowsProcessConfig{
			Processes: []WindowsProcess{
				{
					Name:       fmt.Sprintf("say-hello-1-%d", time.Now().UnixNano()),
					Executable: "cmd.exe",
					Args:       []string{"/C", `@echo off & for /l %x in (1, 1, 10000) do echo "Hello" & timeout /t 10 /nobreak > NUL`},
					Env: map[string]string{
						"__PIPE_SYSLOG_HOST":      "localhost",
						"__PIPE_SYSLOG_PORT":      "10202",
						"__PIPE_SYSLOG_TRANSPORT": "udp",
					},
				},
			},
		},
		"flapping": WindowsProcessConfig{
			Processes: []WindowsProcess{
				{
					Name:       fmt.Sprintf("flapping-1-%d", time.Now().UnixNano()),
					Executable: "cmd.exe",
					Args:       []string{"/C", `@echo off & echo "Flapping-1" & timeout /t 2 /nobreak > NUL & EXIT /B 2`},
				},
				{
					Name:       fmt.Sprintf("flapping-2-%d", time.Now().UnixNano()),
					Executable: "cmd.exe",
					Args:       []string{"/C", `@echo off & echo "Flapping-2" & timeout /t 2 /nobreak > NUL & EXIT /B 2`},
				},
				{
					Name:       fmt.Sprintf("flapping-3-%d", time.Now().UnixNano()),
					Executable: "cmd.exe",
					Args:       []string{"/C", `@echo off & echo "Flapping-3" & timeout /t 2 /nobreak > NUL & EXIT /B 2`},
				},
			},
		},
		"looping": WindowsProcessConfig{
			Processes: []WindowsProcess{
				{
					Name:       fmt.Sprintf("looping-1-%d", time.Now().UnixNano()),
					Executable: "cmd.exe",
					Args:       []string{"/C", `@echo off & for /l %x in (1, 1, 10000) do echo "Looping" & timeout /t 1 /nobreak > NUL`},
				},
				{
					Name:       fmt.Sprintf("looping-2-%d", time.Now().UnixNano()),
					Executable: "powershell.exe",
					Args:       []string{"/C", "While($true) { Start-Process -NoNewWindow powershell.exe -ArgumentList 'Start-Sleep','50'; Start-Sleep 1 }"},
				},
			},
		},
		"stop-executable": WindowsProcessConfig{
			Processes: []WindowsProcess{
				{
					Name:       fmt.Sprintf("stop-executable-1-%d", time.Now().UnixNano()),
					Executable: StartStopExe,
					Args:       []string{"start", tmpFileName},
					Stop: &StopCommand{
						Executable: StartStopExe,
						Args:       []string{"stop", tmpFileName},
					},
				},
			},
		},
	}
	conf, ok := m[jobName]
	return conf, ok
}
*/
