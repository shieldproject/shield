package releasedir_test

import (
	"errors"

	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/releasedir"
)

var _ = Describe("GitRepo", func() {
	var (
		cmdRunner *fakesys.FakeCmdRunner
		fs        *fakesys.FakeFileSystem
		gitRepo   GitRepo
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		gitRepo = NewFSGitRepo("/dir", cmdRunner, fs)
	})

	Describe("Init", func() {
		It("inits directory as git repo", func() {
			err := gitRepo.Init()
			Expect(err).ToNot(HaveOccurred())

			Expect(cmdRunner.RunCommands).To(Equal([][]string{{"git", "init", "/dir"}}))

			Expect(fs.ReadFileString("/dir/.gitignore")).To(Equal(`config/private.yml
blobs
dev_releases
releases/*.tgz
releases/**/*.tgz
.dev_builds
.final_builds/jobs/**/*.tgz
.final_builds/packages/**/*.tgz
.DS_Store
.idea
*.swp
*~
*#
#*
`))
		})

		It("returns error if git init fails", func() {
			cmdRunner.AddCmdResult("git init /dir", fakesys.FakeCmdResult{
				Error: errors.New("fake-err"),
			})
			err := gitRepo.Init()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if writing gitignore fails", func() {
			fs.WriteFileError = errors.New("fake-err")

			err := gitRepo.Init()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})

	Describe("LastCommitSHA", func() {
		cmd := "git rev-parse --short HEAD"

		It("returns last commit", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{
				Stdout: "commit\n",
			})
			commit, err := gitRepo.LastCommitSHA()
			Expect(err).ToNot(HaveOccurred())
			Expect(commit).To(Equal("commit"))

			Expect(cmdRunner.RunComplexCommands).To(Equal([]boshsys.Command{{
				Name:       "git",
				Args:       []string{"rev-parse", "--short", "HEAD"},
				WorkingDir: "/dir",
			}}))
		})

		It("returns 'non-git' if it's not a git repo", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{
				Stderr: "fatal: Not a git repository: '/dir/.git'\n",
				Error:  errors.New("fake-err"),
			})
			commit, err := gitRepo.LastCommitSHA()
			Expect(err).ToNot(HaveOccurred())
			Expect(commit).To(Equal("non-git"))
		})

		It("returns 'empty' if there are no commits", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{
				Stderr: "fatal: Needed a single revision: '/dir/.git'\n",
				Error:  errors.New("fake-err"),
			})
			commit, err := gitRepo.LastCommitSHA()
			Expect(err).ToNot(HaveOccurred())
			Expect(commit).To(Equal("empty"))
		})

		It("returns error if cannot check last commit", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{
				Error: errors.New("fake-err"),
			})
			_, err := gitRepo.LastCommitSHA()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})

	Describe("MustNotBeDirty", func() {
		cmd := "git status --short"

		It("returns false if there are no changes", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{Stdout: ""})
			dirty, err := gitRepo.MustNotBeDirty(false)
			Expect(err).ToNot(HaveOccurred())
			Expect(dirty).To(BeFalse())

			Expect(cmdRunner.RunComplexCommands).To(Equal([]boshsys.Command{{
				Name:       "git",
				Args:       []string{"status", "--short"},
				WorkingDir: "/dir",
			}}))
		})

		It("returns true if there are changes", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{Stdout: "?? change"})
			dirty, err := gitRepo.MustNotBeDirty(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Git repository has local modifications:\n\n?? change"))
			Expect(dirty).To(BeTrue())
		})

		It("returns false if there are changes but being forceful", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{Stdout: "?? change"})
			dirty, err := gitRepo.MustNotBeDirty(true)
			Expect(err).ToNot(HaveOccurred())
			Expect(dirty).To(BeTrue())
		})

		It("returns false if it's not a git repo", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{
				Stderr: "fatal: Not a git repository: '/dir/.git'\n",
				Error:  errors.New("fake-err"),
			})
			dirty, err := gitRepo.MustNotBeDirty(false)
			Expect(err).ToNot(HaveOccurred())
			Expect(dirty).To(BeFalse())
		})

		It("returns error if cannot check dirty state", func() {
			cmdRunner.AddCmdResult(cmd, fakesys.FakeCmdResult{
				Error: errors.New("fake-err"),
			})
			_, err := gitRepo.MustNotBeDirty(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
