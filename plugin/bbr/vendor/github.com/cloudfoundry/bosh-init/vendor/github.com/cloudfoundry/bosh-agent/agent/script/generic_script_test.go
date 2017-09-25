package script_test

import (
	"errors"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshscript "github.com/cloudfoundry/bosh-agent/agent/script"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("GenericScript", func() {
	var (
		fs            *fakesys.FakeFileSystem
		cmdRunner     *fakesys.FakeCmdRunner
		genericScript boshscript.GenericScript
		stdoutLogPath string
		stderrLogPath string
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		stdoutLogPath = filepath.Join("base", "stdout", "logdir", "stdout.log")
		stderrLogPath = filepath.Join("base", "stderr", "logdir", "stderr.log")
		genericScript = boshscript.NewScript(
			fs,
			cmdRunner,
			"my-tag",
			"/path-to-script",
			stdoutLogPath,
			stderrLogPath,
		)
	})

	Describe("Tag", func() {
		It("returns path", func() {
			Expect(genericScript.Tag()).To(Equal("my-tag"))
		})
	})

	Describe("Path", func() {
		It("returns path", func() {
			Expect(genericScript.Path()).To(Equal("/path-to-script"))
		})
	})

	Describe("Exists", func() {
		It("returns bool", func() {
			Expect(genericScript.Exists()).To(BeFalse())

			fs.WriteFile("/path-to-script", []byte{})
			Expect(genericScript.Exists()).To(BeTrue())
		})
	})

	Describe("Run", func() {
		It("executes given command", func() {
			err := genericScript.Run()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if it fails to create logs directory", func() {
			fs.MkdirAllError = errors.New("fake-mkdir-all-error")

			err := genericScript.Run()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake-mkdir-all-error"))
		})

		It("returns an error if it fails to open stdout/stderr log file", func() {
			fs.OpenFileErr = errors.New("fake-open-file-error")

			err := genericScript.Run()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake-open-file-error"))
		})

		Context("when command succeeds", func() {
			BeforeEach(func() {
				cmdRunner.AddCmdResult("/path-to-script", fakesys.FakeCmdResult{
					Stdout:     "fake-stdout",
					Stderr:     "fake-stderr",
					ExitStatus: 0,
					Error:      nil,
				})
			})

			It("saves stdout/stderr to log file", func() {
				err := genericScript.Run()
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists(stdoutLogPath)).To(BeTrue())
				Expect(fs.FileExists(stderrLogPath)).To(BeTrue())

				stdout, err := fs.ReadFileString(stdoutLogPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal("fake-stdout"))

				stderr, err := fs.ReadFileString(stderrLogPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(stderr).To(Equal("fake-stderr"))
			})
		})

		Context("when command fails", func() {
			BeforeEach(func() {
				cmdRunner.AddCmdResult("/path-to-script", fakesys.FakeCmdResult{
					Stdout:     "fake-stdout",
					Stderr:     "fake-stderr",
					ExitStatus: 1,
					Error:      errors.New("fake-command-error"),
				})
			})

			It("saves stdout/stderr to log file", func() {
				err := genericScript.Run()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("fake-command-error"))

				Expect(fs.FileExists(stdoutLogPath)).To(BeTrue())
				Expect(fs.FileExists(stderrLogPath)).To(BeTrue())

				stdout, err := fs.ReadFileString(stdoutLogPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal("fake-stdout"))

				stderr, err := fs.ReadFileString(stderrLogPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(stderr).To(Equal("fake-stderr"))
			})
		})
	})
})
