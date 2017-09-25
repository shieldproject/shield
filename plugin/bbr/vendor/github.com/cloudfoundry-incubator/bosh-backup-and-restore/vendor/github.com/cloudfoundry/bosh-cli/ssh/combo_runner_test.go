package ssh_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	. "github.com/cloudfoundry/bosh-cli/ssh"
	fakessh "github.com/cloudfoundry/bosh-cli/ssh/sshfakes"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("ComboRunner", func() {
	var (
		cmdRunner   *fakesys.FakeCmdRunner
		session     *fakessh.FakeSession
		signalCh    chan<- os.Signal
		writer      Writer
		fs          *fakesys.FakeFileSystem
		ui          *fakeui.FakeUI
		logger      boshlog.Logger
		comboRunner ComboRunner
	)

	BeforeEach(func() {
		cmdRunner = fakesys.NewFakeCmdRunner()

		session = &fakessh.FakeSession{}
		sessFactory := func(_ ConnectionOpts, _ boshdir.SSHResult) Session { return session }

		signalCh = nil
		signalNotifyFunc := func(ch chan<- os.Signal, s ...os.Signal) { signalCh = ch }

		ui = &fakeui.FakeUI{}

		writer = NewStreamingWriter(boshui.NewComboWriter(ui))

		fs = fakesys.NewFakeFileSystem()
		fs.ReturnTempFilesByPrefix = map[string]boshsys.File{
			"ssh-priv-key":    fakesys.NewFakeFile("/tmp/priv-key", fs),
			"ssh-known-hosts": fakesys.NewFakeFile("/tmp/known-hosts", fs),
		}

		logger = boshlog.NewLogger(boshlog.LevelNone)

		comboRunner = NewComboRunner(
			cmdRunner, sessFactory, signalNotifyFunc, writer, fs, ui, logger)
	})

	Describe("Run", func() {
		var (
			connOpts   ConnectionOpts
			result     boshdir.SSHResult
			cmdFactory func(host boshdir.Host) boshsys.Command
		)

		BeforeEach(func() {
			connOpts = ConnectionOpts{}
			result = boshdir.SSHResult{}
			cmdFactory = func(host boshdir.Host) boshsys.Command {
				return boshsys.Command{Name: "cmd", Args: []string{host.Host}}
			}
		})

		It("returns without error when there are no hosts", func() {
			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).ToNot(HaveOccurred())

			Expect(session.FinishCallCount()).To(Equal(1))
		})

		It("returns without error when there is only one host", func() {
			result.Hosts = []boshdir.Host{{Host: "127.0.0.1"}}

			cmdRunner.AddProcess("cmd 127.0.0.1", &fakesys.FakeProcess{})

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).ToNot(HaveOccurred())

			Expect(session.FinishCallCount()).To(Equal(1))
		})

		It("returns without error when there are multiple hosts", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1"},
				{Host: "127.0.0.2"},
			}

			cmdRunner.AddProcess("cmd 127.0.0.1", &fakesys.FakeProcess{})
			cmdRunner.AddProcess("cmd 127.0.0.2", &fakesys.FakeProcess{})

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).ToNot(HaveOccurred())

			Expect(session.FinishCallCount()).To(Equal(1))
		})

		// Varies ssh opts length, to make sure that
		// they are *copied* before being used in cmd.Args append.
		for optsLen := 1; optsLen < 50; optsLen++ {
			opts := []string{}

			for i := 0; i < optsLen; i++ {
				opts = append(opts, fmt.Sprintf("ssh-opt-%d", i))
			}

			It(fmt.Sprintf("adds ssh opts (len %d) after command before other arguments", optsLen), func() {
				result.Hosts = []boshdir.Host{
					{Host: "127.0.0.1"},
					{Host: "127.0.0.2"},
				}

				session.StartReturns(opts, nil)

				cmdRunner.AddProcess(fmt.Sprintf("cmd %s 127.0.0.1", strings.Join(opts, " ")), &fakesys.FakeProcess{})
				cmdRunner.AddProcess(fmt.Sprintf("cmd %s 127.0.0.2", strings.Join(opts, " ")), &fakesys.FakeProcess{})

				err := comboRunner.Run(connOpts, result, cmdFactory)
				Expect(err).ToNot(HaveOccurred())
			})
		}

		It("writes to ui with a instance prefix", func() {
			result.Hosts = []boshdir.Host{
				{Job: "job1", IndexOrID: "id1", Host: "127.0.0.1"},
				{Job: "job2", IndexOrID: "id2", Host: "127.0.0.2"},
			}

			proc1 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.1", proc1)

			proc2 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.2", proc2)

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).ToNot(HaveOccurred())

			proc1.Stdout.Write([]byte("stdout1\n"))
			proc1.Stderr.Write([]byte("stderr1\n"))

			proc2.Stdout.Write([]byte("stdout2\n"))
			proc2.Stderr.Write([]byte("stderr2\n"))

			Expect(ui.Blocks).To(Equal([]string{
				"job1/id1: stdout | ", "stdout1", "\n",
				"job1/id1: stderr | ", "stderr1", "\n",
				"job2/id2: stdout | ", "stdout2", "\n",
				"job2/id2: stderr | ", "stderr2", "\n",
			}))
		})

		It("writes to ui with a ? prefix when job name is not known", func() {
			result.Hosts = []boshdir.Host{
				{Job: "", IndexOrID: "id1", Host: "127.0.0.1"},
				{Job: "job2", IndexOrID: "id2", Host: "127.0.0.2"},
			}

			proc1 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.1", proc1)

			proc2 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.2", proc2)

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).ToNot(HaveOccurred())

			proc1.Stdout.Write([]byte("stdout1\n"))
			proc2.Stdout.Write([]byte("stdout2\n"))

			proc1.Stderr.Write([]byte("stderr1\n"))
			proc2.Stderr.Write([]byte("stderr2\n"))

			Expect(ui.Blocks).To(Equal([]string{
				"?/id1: stdout | ", "stdout1", "\n",
				"job2/id2: stdout | ", "stdout2", "\n",
				"?/id1: stderr | ", "stderr1", "\n",
				"job2/id2: stderr | ", "stderr2", "\n",
			}))
		})

		It("uses provided stdout/stderr if given", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1"},
			}

			proc1 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.1", proc1)

			stdout := bytes.NewBufferString("")
			stderr := bytes.NewBufferString("")

			cmdFactory = func(host boshdir.Host) boshsys.Command {
				return boshsys.Command{
					Name:   "cmd",
					Args:   []string{host.Host},
					Stdout: stdout,
					Stderr: stderr,
				}
			}

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).ToNot(HaveOccurred())

			proc1.Stdout.Write([]byte("stdout"))
			proc1.Stderr.Write([]byte("stderr"))

			Expect(stdout.String()).To(Equal("stdout"))
			Expect(stderr.String()).To(Equal("stderr"))
		})

		It("ultimately returns an error if any processes fail to start", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1"},
				{Host: "127.0.0.2"},
			}

			proc1 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.1", proc1)

			proc2 := &fakesys.FakeProcess{
				StartErr: errors.New("fake-err"),
			}
			cmdRunner.AddProcess("cmd 127.0.0.2", proc2)

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(session.FinishCallCount()).To(Equal(1))
		})

		It("ultimately returns an error if any processes fail during execution", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1"},
				{Host: "127.0.0.2"},
			}

			proc1 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.1", proc1)

			proc2 := &fakesys.FakeProcess{
				WaitResult: boshsys.Result{Error: errors.New("fake-err")},
			}
			cmdRunner.AddProcess("cmd 127.0.0.2", proc2)

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(session.FinishCallCount()).To(Equal(1))
		})

		It("includes all errors if any processes fail", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1"},
				{Host: "127.0.0.2"},
				{Host: "127.0.0.3"},
			}

			proc1 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.1", proc1)

			proc2 := &fakesys.FakeProcess{
				WaitResult: boshsys.Result{Error: errors.New("fake-err2")},
			}
			cmdRunner.AddProcess("cmd 127.0.0.2", proc2)

			proc3 := &fakesys.FakeProcess{
				StartErr: errors.New("fake-err3"),
			}
			cmdRunner.AddProcess("cmd 127.0.0.3", proc3)

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err2"))
			Expect(err.Error()).To(ContainSubstring("fake-err3"))
		})

		It("terminates processes nicely upon interrupt", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1"},
				{Host: "127.0.0.2"},
				{Host: "127.0.0.3"},
			}

			proc1 := &fakesys.FakeProcess{
				TerminatedNicelyCallBack: func(p *fakesys.FakeProcess) {
					p.WaitCh <- boshsys.Result{}
				},
			}
			cmdRunner.AddProcess("cmd 127.0.0.1", proc1)

			proc2 := &fakesys.FakeProcess{}
			cmdRunner.AddProcess("cmd 127.0.0.2", proc2)

			proc3 := &fakesys.FakeProcess{
				TerminatedNicelyCallBack: func(p *fakesys.FakeProcess) {
					p.WaitCh <- boshsys.Result{Error: errors.New("term-err")}
				},
			}
			cmdRunner.AddProcess("cmd 127.0.0.3", proc3)

			go func() {
				// Wait for interrupt goroutine to set channel
				for signalCh == nil {
					time.Sleep(0 * time.Millisecond)
				}
				signalCh <- os.Interrupt
			}()

			logger.Debug("test", "LOL")

			err := comboRunner.Run(connOpts, result, cmdFactory)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("term-err"))

			Expect(session.FinishCallCount()).To(Equal(2))
		})
	})
})
