package integration_test

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("upload-release command", func() {
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

	It("can upload release via git protocol", func() {
		tmpDir, err := fs.TempDir("bosh-upload-release-int-test")
		Expect(err).ToNot(HaveOccurred())

		defer fs.RemoveAll(tmpDir)

		relName := filepath.Base(tmpDir)

		{
			execCmd([]string{"init-release", "--git", "--dir", tmpDir})
			execCmd([]string{"generate-job", "job1", "--dir", tmpDir})
			execCmd([]string{"generate-package", "pkg1", "--dir", tmpDir})
		}

		{ // job1 depends on both packages
			jobSpecPath := filepath.Join(tmpDir, "jobs", "job1", "spec")

			contents, err := fs.ReadFileString(jobSpecPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(jobSpecPath, strings.Replace(contents, "packages: []", "packages: [pkg1]", -1))
			Expect(err).ToNot(HaveOccurred())
		}

		{ // Add a bit of content
			err := fs.WriteFileString(filepath.Join(tmpDir, "src", "in-src"), "in-src")
			Expect(err).ToNot(HaveOccurred())

			pkg1SpecPath := filepath.Join(tmpDir, "packages", "pkg1", "spec")

			contents, err := fs.ReadFileString(pkg1SpecPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(pkg1SpecPath, strings.Replace(contents, "files: []", "files:\n- in-src", -1))
			Expect(err).ToNot(HaveOccurred())
		}

		{ // Create release with local blobstore
			blobstoreDir := filepath.Join(tmpDir, ".blobstore")

			err := fs.MkdirAll(blobstoreDir, 0777)
			Expect(err).ToNot(HaveOccurred())

			finalYaml := "name: " + relName + `
blobstore:
  provider: local
  options:
    blobstore_path: ` + blobstoreDir

			err = fs.WriteFileString(filepath.Join(tmpDir, "config", "final.yml"), finalYaml)
			Expect(err).ToNot(HaveOccurred())

			execGit := func(args []string) {
				cmd := boshsys.Command{
					Name:           "git",
					Args:           args,
					UseIsolatedEnv: true,
					WorkingDir:     tmpDir, // --git-dir/--work-tree/etc. dont work great
				}
				_, _, _, err := deps.CmdRunner.RunComplexCommand(cmd)
				Expect(err).ToNot(HaveOccurred())
			}

			execGit([]string{"config", "--local", "user.email", "bosh-upload-release-int-test"})
			execGit([]string{"config", "--local", "user.name", "bosh-upload-release-int-test"})

			execGit([]string{"add", "-A"})
			execGit([]string{"commit", "-m", "init"})

			execCmd([]string{"create-release", "--dir", tmpDir, "--final"})

			execGit([]string{"add", "-A"})
			execGit([]string{"commit", "-m", "Final release 1"})
		}

		uploadedReleaseFile := filepath.Join(tmpDir, "release-3.tgz")

		{
			directorCACert, director := BuildHTTPSServer()
			defer director.Close()

			director.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.RespondWith(http.StatusOK, `{"user_authentication":{"type":"basic","options":{}}}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/releases"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/packages/matches"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/releases"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
					func(w http.ResponseWriter, req *http.Request) {
						defer req.Body.Close()

						body, err := ioutil.ReadAll(req.Body)
						Expect(err).ToNot(HaveOccurred())

						err = fs.WriteFile(uploadedReleaseFile, body)
						Expect(err).ToNot(HaveOccurred())
					},
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=result"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			execCmd([]string{"upload-release", "git+file://" + tmpDir, "-e", director.URL(), "--ca-cert", directorCACert})
		}

		{ // Check contents of uploaded release
			relProvider := boshrel.NewProvider(deps.CmdRunner, deps.Compressor, deps.DigestCalculator, deps.FS, deps.Logger)
			archiveReader := relProvider.NewExtractingArchiveReader()

			release, err := archiveReader.Read(uploadedReleaseFile)
			Expect(err).ToNot(HaveOccurred())

			defer release.CleanUp()

			pkg1 := release.Packages()[0]
			Expect(fs.ReadFileString(filepath.Join(pkg1.ExtractedPath(), "in-src"))).To(Equal("in-src"))
		}
	})
})
