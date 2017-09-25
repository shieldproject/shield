package integration_test

import (
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	boshrel "github.com/cloudfoundry/bosh-cli/release"
	boshrelman "github.com/cloudfoundry/bosh-cli/release/manifest"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	"os"
)

var _ = Describe("create-release command", func() {
	var (
		ui         *fakeui.FakeUI
		fs         boshsys.FileSystem
		deps       BasicDeps
		cmdFactory Factory
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		logger := boshlog.NewLogger(boshlog.LevelNone)
		confUI := boshui.NewWrappingConfUI(ui, logger)

		fs = boshsys.NewOsFileSystem(logger)
		deps = NewBasicDepsWithFS(confUI, fs, logger)
		cmdFactory = NewFactory(deps)
	})

	execCmd := func(args []string) {
		cmd, err := cmdFactory.New(args)
		Expect(err).ToNot(HaveOccurred())

		err = cmd.Execute()
		Expect(err).ToNot(HaveOccurred())
	}

	removeSHA1s := func(contents string) string {
		matchSHA1s := regexp.MustCompile("sha1: [a-z0-9]{40}\n")
		return matchSHA1s.ReplaceAllString(contents, "sha1: replaced\n")
	}

	expectSha256Checksums := func(filePath string) {
		contents, err := fs.ReadFileString(filePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(MatchRegexp("sha1: sha256:.*"))
	}

	It("can iterate on a basic release", func() {
		tmpDir, err := fs.TempDir("bosh-create-release-int-test")
		Expect(err).ToNot(HaveOccurred())

		defer fs.RemoveAll(tmpDir)

		relName := filepath.Base(tmpDir)

		{
			execCmd([]string{"init-release", "--dir", tmpDir})
			Expect(fs.FileExists(filepath.Join(tmpDir, "config"))).To(BeTrue())
			Expect(fs.FileExists(filepath.Join(tmpDir, "jobs"))).To(BeTrue())
			Expect(fs.FileExists(filepath.Join(tmpDir, "packages"))).To(BeTrue())
			Expect(fs.FileExists(filepath.Join(tmpDir, "src"))).To(BeTrue())
		}

		execCmd([]string{"generate-job", "job1", "--dir", tmpDir})
		execCmd([]string{"generate-package", "pkg1", "--dir", tmpDir})
		execCmd([]string{"generate-package", "pkg2", "--dir", tmpDir})

		err = fs.WriteFileString(filepath.Join(tmpDir, "LICENSE"), "LICENSE")
		Expect(err).ToNot(HaveOccurred())

		{ // pkg1 depends on pkg2 for compilation
			pkg1SpecPath := filepath.Join(tmpDir, "packages", "pkg1", "spec")

			contents, err := fs.ReadFileString(pkg1SpecPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(pkg1SpecPath, strings.Replace(contents, "dependencies: []", "dependencies: [pkg2]", -1))
			Expect(err).ToNot(HaveOccurred())
		}

		{ // job1 depends on both packages
			jobSpecPath := filepath.Join(tmpDir, "jobs", "job1", "spec")

			contents, err := fs.ReadFileString(jobSpecPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(jobSpecPath, strings.Replace(contents, "packages: []", "packages: [pkg1, pkg2]", -1))
			Expect(err).ToNot(HaveOccurred())
		}

		{ // Make empty release
			execCmd([]string{"create-release", "--dir", tmpDir})

			contents, err := fs.ReadFileString(filepath.Join(tmpDir, "dev_releases", relName, relName+"-0+dev.1.yml"))
			Expect(err).ToNot(HaveOccurred())

			Expect(removeSHA1s(contents)).To(Equal(
				"name: " + relName + `
version: 0+dev.1
commit_hash: non-git
uncommitted_changes: false
jobs:
- name: job1
  version: 2587bae8b82982432573d964c4c23bae3403ddee
  fingerprint: 2587bae8b82982432573d964c4c23bae3403ddee
  sha1: replaced
packages:
- name: pkg1
  version: a869c327f5cf345e945f8c8798aca1c34993f66b
  fingerprint: a869c327f5cf345e945f8c8798aca1c34993f66b
  sha1: replaced
  dependencies:
  - pkg2
- name: pkg2
  version: 100bedf6f31da1a4693c446f1ea93348ea7a7a9d
  fingerprint: 100bedf6f31da1a4693c446f1ea93348ea7a7a9d
  sha1: replaced
  dependencies: []
license:
  version: f9d233609f68751f4e3f8fe5ab2ad69e4d534496
  fingerprint: f9d233609f68751f4e3f8fe5ab2ad69e4d534496
  sha1: replaced
`,
			))
		}

		{ // Add a bit of content
			err := fs.WriteFileString(filepath.Join(tmpDir, "src", "in-src"), "in-src")
			Expect(err).ToNot(HaveOccurred())

			randomFile := filepath.Join(tmpDir, "random-file")

			err = fs.WriteFileString(randomFile, "in-blobs")
			Expect(err).ToNot(HaveOccurred())

			execCmd([]string{"add-blob", randomFile, "in-blobs", "--dir", tmpDir})

			pkg1SpecPath := filepath.Join(tmpDir, "packages", "pkg1", "spec")

			contents, err := fs.ReadFileString(pkg1SpecPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(pkg1SpecPath, strings.Replace(contents, "files: []", "files:\n- in-src\n- in-blobs", -1))
			Expect(err).ToNot(HaveOccurred())
		}

		{ // Make release with some contents
			execCmd([]string{"create-release", "--dir", tmpDir})

			rel1File := filepath.Join(tmpDir, "dev_releases", relName, relName+"-0+dev.1.yml")
			rel2File := filepath.Join(tmpDir, "dev_releases", relName, relName+"-0+dev.2.yml")

			contents, err := fs.ReadFileString(rel2File)
			Expect(err).ToNot(HaveOccurred())

			Expect(removeSHA1s(contents)).To(Equal(
				"name: " + relName + `
version: 0+dev.2
commit_hash: non-git
uncommitted_changes: false
jobs:
- name: job1
  version: 2587bae8b82982432573d964c4c23bae3403ddee
  fingerprint: 2587bae8b82982432573d964c4c23bae3403ddee
  sha1: replaced
packages:
- name: pkg1
  version: 9555b8abbcb5180f02f6b7c6027f9d8f49c0e952
  fingerprint: 9555b8abbcb5180f02f6b7c6027f9d8f49c0e952
  sha1: replaced
  dependencies:
  - pkg2
- name: pkg2
  version: 100bedf6f31da1a4693c446f1ea93348ea7a7a9d
  fingerprint: 100bedf6f31da1a4693c446f1ea93348ea7a7a9d
  sha1: replaced
  dependencies: []
license:
  version: f9d233609f68751f4e3f8fe5ab2ad69e4d534496
  fingerprint: f9d233609f68751f4e3f8fe5ab2ad69e4d534496
  sha1: replaced
`,
			))

			man1, err := boshrelman.NewManifestFromPath(rel1File, fs)
			Expect(err).ToNot(HaveOccurred())

			man2, err := boshrelman.NewManifestFromPath(rel2File, fs)
			Expect(err).ToNot(HaveOccurred())

			// Explicitly check that pkg1 changed its fingerprint
			Expect(man1.Packages[0].Name).To(Equal(man2.Packages[0].Name))
			Expect(man1.Packages[0].Fingerprint).ToNot(Equal(man2.Packages[0].Fingerprint))

			// and pkg2 did not change
			Expect(man1.Packages[1].Name).To(Equal(man2.Packages[1].Name))
			Expect(man1.Packages[1].Fingerprint).To(Equal(man2.Packages[1].Fingerprint))
		}

		{ // check contents of index files when sha2 flag is supplied
			execCmd([]string{"create-release", "--sha2", "--dir", tmpDir})

			expectSha256Checksums(filepath.Join(tmpDir, "dev_releases", relName, relName+"-0+dev.3.yml"))
			expectSha256Checksums(filepath.Join(tmpDir, ".dev_builds", "jobs", "job1", "index.yml"))
			expectSha256Checksums(filepath.Join(tmpDir, ".dev_builds", "packages", "pkg1", "index.yml"))
			expectSha256Checksums(filepath.Join(tmpDir, ".dev_builds", "license", "index.yml"))
		}

		{ // Check contents of made release via its tarball
			tgzFile := filepath.Join(tmpDir, "release-3.tgz")

			execCmd([]string{"create-release", "--dir", tmpDir, "--tarball", tgzFile})
			relProvider := boshrel.NewProvider(deps.CmdRunner, deps.Compressor, deps.DigestCalculator, deps.FS, deps.Logger)
			archiveReader := relProvider.NewExtractingArchiveReader()

			release, err := archiveReader.Read(tgzFile)
			Expect(err).ToNot(HaveOccurred())

			defer release.CleanUp()

			pkg1 := release.Packages()[0]
			Expect(fs.ReadFileString(filepath.Join(pkg1.ExtractedPath(), "in-src"))).To(Equal("in-src"))
			Expect(fs.ReadFileString(filepath.Join(pkg1.ExtractedPath(), "in-blobs"))).To(Equal("in-blobs"))
		}

		{ // Check that tarballs will not overwrite a directory
			directoryPath := filepath.Join(tmpDir, "tarball-collision-dir")
			Expect(fs.MkdirAll(directoryPath, os.ModeDir)).To(Succeed())
			_, err := cmdFactory.New([]string{"create-release", "--dir", tmpDir, "--tarball", directoryPath})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Path must not be directory"))
		}

		{ // removes unknown blobs, keeping known blobs
			blobPath := filepath.Join(tmpDir, "blobs", "unknown-blob.tgz")

			fs.WriteFileString(blobPath, "i don't belong here")

			execCmd([]string{"create-release", "--dir", tmpDir})
			Expect(fs.FileExists(blobPath)).To(BeFalse())
			Expect(fs.FileExists(filepath.Join(tmpDir, "blobs", "in-blobs"))).To(BeTrue())
		}
	})
})
