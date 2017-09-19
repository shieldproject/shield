package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"path/filepath"
	"regexp"
	"strings"
)

var _ = Describe("sha2ify-release", func() {

	var (
		ui                  *fakeui.FakeUI
		fs                  boshsys.FileSystem
		deps                BasicDeps
		cmdFactory          Factory
		releaseProvider     boshrel.Provider
		createSimpleRelease func() string
		removeSHA1s         func(string) string
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		logger := boshlog.NewLogger(boshlog.LevelNone)
		confUI := boshui.NewWrappingConfUI(ui, logger)

		fs = boshsys.NewOsFileSystem(logger)
		deps = NewBasicDepsWithFS(confUI, fs, logger)
		cmdFactory = NewFactory(deps)

		releaseProvider = boshrel.NewProvider(
			deps.CmdRunner, deps.Compressor, deps.DigestCalculator, deps.FS, deps.Logger)

	})

	execCmd := func(args []string) {
		cmd, err := cmdFactory.New(args)
		Expect(err).ToNot(HaveOccurred())

		err = cmd.Execute()

		Expect(err).ToNot(HaveOccurred())

	}

	It("converts the SHA1s into SHA2s for packages and jobs", func() {
		sha2ifyReleasePath := createSimpleRelease()
		defer fs.RemoveAll(filepath.Dir(sha2ifyReleasePath))

		dirtyPath, err := fs.TempDir("sha2release")
		Expect(err).ToNot(HaveOccurred())

		outFile := filepath.Join(dirtyPath, "small-sha256-release.tgz")

		execCmd([]string{"sha2ify-release", sha2ifyReleasePath, outFile})

		extractor := releaseProvider.NewExtractingArchiveReader()

		release, err := extractor.Read(outFile)
		Expect(err).ToNot(HaveOccurred())

		By("keeping all the jobs and packages")
		Expect(release.Jobs()).To(HaveLen(1))
		Expect(release.Packages()).To(HaveLen(1))
		Expect(release.License()).ToNot(BeNil())

		By("converting the SHAs to 256")
		jobArchiveSha := release.Jobs()[0].ArchiveSHA1()
		Expect(removeSHA1s(jobArchiveSha)).To(Equal("sha256:replaced"))

		packageArchiveSha := release.Packages()[0].ArchiveSHA1()
		Expect(removeSHA1s(packageArchiveSha)).To(Equal("sha256:replaced"))

		licenseArchiveSha := release.License().ArchiveSHA1()
		Expect(removeSHA1s(licenseArchiveSha)).To(Equal("sha256:replaced"))

		By("preserving the version string exactly")
		Expect(release.Version()).To(Equal("0+dev.1"))
	})

	It("converts the SHA1s into SHA2s for packages and jobs", func() {
		dirtyPath, err := fs.TempDir("sha2release")
		Expect(err).ToNot(HaveOccurred())

		outFile := filepath.Join(dirtyPath, "small-sha256-release.tgz")

		execCmd([]string{"sha2ify-release", "assets/small-sha128-compiled-release.tgz", outFile})

		extractor := releaseProvider.NewExtractingArchiveReader()

		release, err := extractor.Read(outFile)
		Expect(err).ToNot(HaveOccurred())

		By("keeping all the jobs and packages")
		Expect(release.Jobs()).To(HaveLen(1))
		Expect(release.CompiledPackages()).To(HaveLen(1))

		By("converting the SHAs to 256")
		jobArchiveSha := release.Jobs()[0].ArchiveSHA1()
		Expect(removeSHA1s(jobArchiveSha)).To(Equal("sha256:replaced"))
		compiledPackageSha := release.CompiledPackages()[0].ArchiveSHA1()
		Expect(removeSHA1s(compiledPackageSha)).To(Equal("sha256:replaced"))

		By("preserving the version string exactly")
		Expect(release.Version()).To(Equal("0+dev.3"))
	})

	removeSHA1s = func(contents string) string {
		matchSHA1s := regexp.MustCompile("sha256:[a-z0-9]{64}")
		return matchSHA1s.ReplaceAllString(contents, "sha256:replaced")
	}

	createSimpleRelease = func() string {
		tmpDir, err := fs.TempDir("bosh-create-release-int-test")
		Expect(err).ToNot(HaveOccurred())

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

		err = fs.WriteFileString(filepath.Join(tmpDir, "LICENSE"), "LICENSE")
		Expect(err).ToNot(HaveOccurred())

		{
			pkg1SpecPath := filepath.Join(tmpDir, "packages", "pkg1", "spec")

			contents, err := fs.ReadFileString(pkg1SpecPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(pkg1SpecPath, strings.Replace(contents, "dependencies: []", "dependencies: []", -1))
			Expect(err).ToNot(HaveOccurred())
		}

		{
			jobSpecPath := filepath.Join(tmpDir, "jobs", "job1", "spec")

			contents, err := fs.ReadFileString(jobSpecPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(jobSpecPath, strings.Replace(contents, "packages: []", "packages: [pkg1]", -1))
			Expect(err).ToNot(HaveOccurred())
		}

		sha2ifyReleasePath := filepath.Join(tmpDir, "sha2ify-release.tgz")

		{ // Make empty release
			execCmd([]string{"create-release", "--dir", tmpDir, "--tarball", sha2ifyReleasePath})

			_, err := fs.ReadFileString(filepath.Join(tmpDir, "dev_releases", relName, relName+"-0+dev.1.yml"))
			Expect(err).ToNot(HaveOccurred())
		}

		return sha2ifyReleasePath
	}
})
