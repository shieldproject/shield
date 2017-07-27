package cmd_test

import (
	"errors"
	"os"
	"path/filepath"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
)

var _ = Describe("Factory", func() {
	var (
		fs      *fakesys.FakeFileSystem
		factory Factory
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()

		ui := boshui.NewConfUI(logger)
		defer ui.Flush()

		deps := NewBasicDeps(ui, logger)
		deps.FS = fs

		factory = NewFactory(deps)
	})

	Describe("unknown commands, args and flags", func() {
		BeforeEach(func() {
			err := fs.WriteFileString(filepath.Join("/", "file"), "")
			Expect(err).ToNot(HaveOccurred())
		})

		cmds := map[string][]string{
			"help":                  []string{},
			"add-blob":              []string{filepath.Join("/", "file"), "directory"},
			"attach-disk":           []string{"instance/abad1dea", "disk-cid-123"},
			"blobs":                 []string{},
			"interpolate":           []string{filepath.Join("/", "file")},
			"cancel-task":           []string{"1234"},
			"clean-up":              []string{},
			"cloud-check":           []string{},
			"cloud-config":          []string{},
			"create-env":            []string{filepath.Join("/", "file")},
			"sha2ify-release":       []string{filepath.Join("/", "file"), filepath.Join("/", "file2")},
			"create-release":        []string{filepath.Join("/", "file")},
			"delete-deployment":     []string{},
			"delete-disk":           []string{"cid"},
			"delete-env":            []string{filepath.Join("/", "file")},
			"delete-release":        []string{"release-version"},
			"delete-snapshot":       []string{"cid"},
			"delete-snapshots":      []string{},
			"delete-stemcell":       []string{"name/version"},
			"delete-vm":             []string{"cid"},
			"deploy":                []string{filepath.Join("/", "file")},
			"deployment":            []string{},
			"deployments":           []string{},
			"disks":                 []string{},
			"alias-env":             []string{"alias"},
			"environment":           []string{},
			"environments":          []string{},
			"errands":               []string{},
			"events":                []string{},
			"export-release":        []string{"release/version", "os/version"},
			"finalize-release":      []string{filepath.Join("/", "file")},
			"generate-job":          []string{filepath.Join("/", "file")},
			"generate-package":      []string{filepath.Join("/", "file")},
			"init-release":          []string{},
			"inspect-release":       []string{"name/version"},
			"instances":             []string{},
			"locks":                 []string{},
			"log-in":                []string{},
			"log-out":               []string{},
			"logs":                  []string{"slug"},
			"manifest":              []string{},
			"recreate":              []string{"slug"},
			"releases":              []string{},
			"remove-blob":           []string{filepath.Join("/", "file")},
			"reset-release":         []string{},
			"restart":               []string{"slug"},
			"run-errand":            []string{"name"},
			"runtime-config":        []string{},
			"snapshots":             []string{"group/id"},
			"start":                 []string{"slug"},
			"stemcells":             []string{},
			"stop":                  []string{"slug"},
			"sync-blobs":            []string{},
			"take-snapshot":         []string{"group/id"},
			"task":                  []string{"1234"},
			"tasks":                 []string{},
			"update-cloud-config":   []string{filepath.Join("/", "file")},
			"update-resurrection":   []string{"off"},
			"update-runtime-config": []string{filepath.Join("/", "file")},
			"upload-blobs":          []string{},
			"upload-release":        []string{filepath.Join("/", "file")},
			"upload-stemcell":       []string{filepath.Join("/", "file")},
			"vms":                   []string{},
		}

		for cmd, requiredArgs := range cmds {
			cmd, requiredArgs := cmd, requiredArgs // copy

			Describe(cmd, func() {
				It("fails with extra arguments", func() {
					cmdWithArgs := append([]string{cmd}, requiredArgs...)
					cmdWithArgs = append(cmdWithArgs, "extra", "args")

					_, err := factory.New(cmdWithArgs)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("does not support extra arguments: extra, args"))
				})
			})
		}

		Describe("ssh", func() {
			It("uses all remaining arguments as a command", func() {
				cmd, err := factory.New([]string{"ssh", "group", "cmd", "extra", "args"})
				Expect(err).ToNot(HaveOccurred())

				opts := cmd.Opts.(*SSHOpts)
				Expect(opts.Command).To(Equal([]string{"cmd", "extra", "args"}))
			})

			It("uses all remaining arguments as a command even that look like flags", func() {
				cmd, err := factory.New([]string{"ssh", "group", "cmd", "extra", "args", "--", "--gw-disable"})
				Expect(err).ToNot(HaveOccurred())

				opts := cmd.Opts.(*SSHOpts)
				Expect(opts.Command).To(Equal([]string{"cmd", "extra", "args", "--gw-disable"}))
			})

			It("returns error if command is given and extra arguments are specified", func() {
				_, err := factory.New([]string{"ssh", "group", "-c", "command", "--", "extra", "args"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not support extra arguments: extra, args"))
			})
		})

		It("catches unknown commands and lists available commands", func() {
			_, err := factory.New([]string{"unknown-cmd"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unknown command `unknown-cmd'. Please specify one command of: add-blob"))
		})

		It("catches unknown global flags", func() {
			_, err := factory.New([]string{"--unknown-flag"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown flag `unknown-flag'"))
		})

		It("catches unknown command flags", func() {
			_, err := factory.New([]string{"ssh", "--unknown-flag"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown flag `unknown-flag'"))
		})
	})

	Describe("gateway flags", func() {
		It("ssh command has configured gateway flags", func() {
			cmd, err := factory.New([]string{"ssh", "group", "cmd", "extra", "args", "--", "--gw-disable"})
			Expect(err).ToNot(HaveOccurred())

			_, _, err = cmd.Opts.(*SSHOpts).GatewayFlags.AsSSHOpts()
			Expect(err).ToNot(HaveOccurred())
		})

		It("scp command has configured gateway flags", func() {
			cmd, err := factory.New([]string{"scp", "group", "cmd", "extra", "args", "--", "--gw-disable"})
			Expect(err).ToNot(HaveOccurred())

			_, _, err = cmd.Opts.(*SCPOpts).GatewayFlags.AsSSHOpts()
			Expect(err).ToNot(HaveOccurred())
		})

		It("logs -f command has configured gateway flags", func() {
			cmd, err := factory.New([]string{"logs", "-f", "cmd"})
			Expect(err).ToNot(HaveOccurred())

			_, _, err = cmd.Opts.(*LogsOpts).GatewayFlags.AsSSHOpts()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("deploy command", func() {
		BeforeEach(func() {
			err := fs.WriteFileString(filepath.Join("/", "file"), "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("parses multiple skip-drain flags", func() {
			cmd, err := factory.New([]string{"deploy", "--skip-drain=job1", "--skip-drain=job2", filepath.Join("/", "file")})
			Expect(err).ToNot(HaveOccurred())

			slug1, _ := boshdir.NewInstanceGroupOrInstanceSlugFromString("job1")
			slug2, _ := boshdir.NewInstanceGroupOrInstanceSlugFromString("job2")

			opts := cmd.Opts.(*DeployOpts)
			Expect(opts.SkipDrain).To(Equal([]boshdir.SkipDrain{
				{Slug: slug1},
				{Slug: slug2},
			}))
		})

		It("errors when excluding = from --skip-drain", func() {
			_, err := factory.New([]string{"deploy", "--skip-drain", "job1", filepath.Join("/", "file")})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Not found: open job1: no such file or directory"))
		})

		It("defaults --skip-drain option value to all", func() {
			cmd, err := factory.New([]string{"deploy", "--skip-drain", filepath.Join("/", "file")})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*DeployOpts)
			Expect(opts.SkipDrain).To(Equal([]boshdir.SkipDrain{
				{All: true},
			}))
		})
	})

	Describe("create-env command (command that uses FileBytesArg)", func() {
		It("returns *nice error from FileBytesArg* error if it cannot read manifest", func() {
			fs.ReadFileError = errors.New("fake-err")

			_, err := factory.New([]string{"create-env", "manifest.yml"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("open manifest.yml: no such file or directory"))
		})
	})

	Describe("alias-env command", func() {
		It("is passed global environment URL", func() {
			cmd, err := factory.New([]string{"alias-env", "-e", "env", "alias"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*AliasEnvOpts)
			Expect(opts.URL).To(Equal("env"))
		})

		It("is passed the global CA cert", func() {
			cmd, err := factory.New([]string{"alias-env", "--ca-cert", "BEGIN ca-cert", "alias"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*AliasEnvOpts)
			opts.CACert.FS = nil
			Expect(opts.CACert).To(Equal(CACertArg{Content: "BEGIN ca-cert"}))
		})
	})

	Describe("events command", func() {
		It("is passed the deployment flag", func() {
			cmd, err := factory.New([]string{"events", "--deployment", "deployment"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*EventsOpts)
			Expect(opts.Deployment).To(Equal("deployment"))
		})
	})

	Describe("vms command", func() {
		It("is passed the deployment flag", func() {
			cmd, err := factory.New([]string{"vms", "--deployment", "deployment"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*VMsOpts)
			Expect(opts.Deployment).To(Equal("deployment"))
		})
	})

	Describe("instances command", func() {
		It("is passed the deployment flag", func() {
			cmd, err := factory.New([]string{"instances", "--deployment", "deployment"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*InstancesOpts)
			Expect(opts.Deployment).To(Equal("deployment"))
		})
	})

	Describe("tasks command", func() {
		It("is passed the deployment flag", func() {
			cmd, err := factory.New([]string{"tasks", "--deployment", "deployment"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*TasksOpts)
			Expect(opts.Deployment).To(Equal("deployment"))
		})
	})

	Describe("help command", func() {
		It("has a help command", func() {
			cmd, err := factory.New([]string{"help"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*MessageOpts)
			Expect(opts.Message).To(ContainSubstring("Usage:"))
			Expect(opts.Message).To(ContainSubstring("Application Options:"))
			Expect(opts.Message).To(ContainSubstring("Available commands:"))
		})
	})

	Describe("help options", func() {
		It("has a help flag", func() {
			cmd, err := factory.New([]string{"--help"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*MessageOpts)
			Expect(opts.Message).To(ContainSubstring("Usage:"))
			Expect(opts.Message).To(ContainSubstring(
				"SSH into instance(s)                               https://bosh.io/docs/cli-v2#ssh"))
			Expect(opts.Message).To(ContainSubstring("Application Options:"))
			Expect(opts.Message).To(ContainSubstring("Available commands:"))
		})

		It("has a command help flag", func() {
			cmd, err := factory.New([]string{"ssh", "--help"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*MessageOpts)
			Expect(opts.Message).To(ContainSubstring("Usage:"))
			Expect(opts.Message).To(ContainSubstring("SSH into instance(s)\n\nhttps://bosh.io/docs/cli-v2#ssh"))
			Expect(opts.Message).To(ContainSubstring("Application Options:"))
			Expect(opts.Message).To(ContainSubstring("[ssh command options]"))
		})
	})

	Describe("version option", func() {
		It("has a version flag", func() {
			cmd, err := factory.New([]string{"--version"})
			Expect(err).ToNot(HaveOccurred())

			opts := cmd.Opts.(*MessageOpts)
			Expect(opts.Message).To(Equal("version [DEV BUILD]\n"))
		})
	})

	Describe("global options", func() {
		clearNonGlobalOpts := func(boshOpts BoshOpts) BoshOpts {
			boshOpts.VersionOpt = nil   // can't compare functions
			boshOpts.CACertOpt.FS = nil // fs is populated by factory.New
			boshOpts.UploadRelease = UploadReleaseOpts{}
			boshOpts.ExportRelease = ExportReleaseOpts{}
			boshOpts.RunErrand = RunErrandOpts{}
			boshOpts.Logs = LogsOpts{}
			boshOpts.Interpolate = InterpolateOpts{}
			boshOpts.InitRelease = InitReleaseOpts{}
			boshOpts.ResetRelease = ResetReleaseOpts{}
			boshOpts.GenerateJob = GenerateJobOpts{}
			boshOpts.GeneratePackage = GeneratePackageOpts{}
			boshOpts.CreateRelease = CreateReleaseOpts{}
			boshOpts.FinalizeRelease = FinalizeReleaseOpts{}
			boshOpts.Blobs = BlobsOpts{}
			boshOpts.AddBlob = AddBlobOpts{}
			boshOpts.RemoveBlob = RemoveBlobOpts{}
			boshOpts.SyncBlobs = SyncBlobsOpts{}
			boshOpts.UploadBlobs = UploadBlobsOpts{}
			boshOpts.SSH = SSHOpts{}
			boshOpts.SCP = SCPOpts{}
			return boshOpts
		}

		It("has set of default options", func() {
			cmd, err := factory.New([]string{"locks"})
			Expect(err).ToNot(HaveOccurred())

			// Check against entire BoshOpts to avoid future missing assertions
			Expect(clearNonGlobalOpts(cmd.BoshOpts)).To(Equal(BoshOpts{
				ConfigPathOpt: "~/.bosh/config",
			}))
		})

		It("can set variety of options", func() {
			opts := []string{
				"--config", "config",
				"--environment", "env",
				"--ca-cert", "BEGIN ca-cert",
				"--client", "client",
				"--client-secret", "client-secret",
				"--deployment", "dep",
				"--json",
				"--tty",
				"--no-color",
				"--non-interactive",
				"locks",
			}

			cmd, err := factory.New(opts)
			Expect(err).ToNot(HaveOccurred())

			Expect(clearNonGlobalOpts(cmd.BoshOpts)).To(Equal(BoshOpts{
				ConfigPathOpt:     "config",
				EnvironmentOpt:    "env",
				CACertOpt:         CACertArg{Content: "BEGIN ca-cert"},
				ClientOpt:         "client",
				ClientSecretOpt:   "client-secret",
				DeploymentOpt:     "dep",
				JSONOpt:           true,
				TTYOpt:            true,
				NoColorOpt:        true,
				NonInteractiveOpt: true,
			}))
		})

		It("errors when --user is set", func() {
			opts := []string{
				"--user", "foo",
				"--json",
				"--tty",
			}

			_, err := factory.New(opts)
			Expect(err).To(HaveOccurred())
		})

		It("errors when BOSH_USER is set", func() {
			os.Setenv("BOSH_USER", "bar")
			_, err := factory.New([]string{})
			Expect(err).To(HaveOccurred())
		})
	})
})
