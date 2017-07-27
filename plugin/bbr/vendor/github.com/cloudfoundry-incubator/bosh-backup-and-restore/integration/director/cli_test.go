package director

import (
	"io/ioutil"
	"os"

	"time"

	"fmt"

	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

var _ = Describe("CLI Interface", func() {

	var director *mockhttp.Server
	var backupWorkspace string

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	Context("bbr director", func() {
		Describe("backup with invalid command line arguments", func() {
			Context("private-key-path flag", func() {
				var (
					keyFile       *os.File
					err           error
					session       *gexec.Session
					sessionOutput string
				)

				BeforeEach(func() {
					keyFile, err = ioutil.TempFile("", time.Now().String())
					Expect(err).NotTo(HaveOccurred())
					fmt.Fprintf(keyFile, "this is not a valid key")

					session = binary.Run(backupWorkspace,
						[]string{},
						"director",
						"-u", "admin",
						"--host", "10.0.0.5",
						"--private-key-path", keyFile.Name(),
						"backup")
					Eventually(session).Should(gexec.Exit())
					sessionOutput = string(session.Err.Contents())
				})

				It("fails", func() {
					By("printing a meaningful message when the key is invalid", func() {
						Expect(sessionOutput).To(ContainSubstring("ssh.NewConnection.ParsePrivateKey failed"))
					})

					By("not printing a stack trace", func() {
						Expect(sessionOutput).NotTo(ContainSubstring("main.go"))
					})

					By("saving the stack trace into a file", func() {
						files, err := filepath.Glob(filepath.Join(backupWorkspace, "bbr-*.err.log"))
						Expect(err).NotTo(HaveOccurred())
						logFilePath := files[0]
						_, err = os.Stat(logFilePath)
						Expect(os.IsNotExist(err)).To(BeFalse())
						stackTrace, err := ioutil.ReadFile(logFilePath)
						Expect(err).ToNot(HaveOccurred())
						Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
					})
				})
			})
		})

		Describe("restore with incorrect artifact-path", func() {
			Context("restore command with missing artifact-path", func() {
				var (
					session       *gexec.Session
					sessionOutput string
				)

				BeforeEach(func() {
					session = binary.Run(backupWorkspace,
						[]string{},
						"director",
						"-u", "admin",
						"--host", "10.0.0.5",
						"--private-key-path", "doesn't matter",
						"restore")
					Eventually(session).Should(gexec.Exit())
					sessionOutput = string(session.Err.Contents())
				})

				It("fails", func() {
					By("erroring", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})
					By("printing a meaningful message about the missing parameter", func() {
						Expect(sessionOutput).To(ContainSubstring("--artifact-path flag is required"))
					})
				})
			})
			Context("restore command with artifact-path pointing to non-existent file", func() {
				var (
					session       *gexec.Session
					sessionOutput string
				)

				BeforeEach(func() {
					session = binary.Run(backupWorkspace,
						[]string{},
						"director",
						"-u", "admin",
						"--host", "10.0.0.5",
						"--private-key-path", "doesn't matter",
						"restore",
						"--artifact-path", "non-existent-file")
					Eventually(session).Should(gexec.Exit())
					sessionOutput = string(session.Err.Contents())
				})

				It("fails", func() {
					By("erroring", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})
					By("printing a meaningful message about the missing parameter", func() {
						Expect(sessionOutput).To(ContainSubstring("non-existent-file: no such file or directory"))
					})
				})
			})
		})
	})

})
