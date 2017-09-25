package deployment

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

	Context("bbr deployment", func() {
		AssertDeploymentCLIBehaviour := func(cmd string, extraArgs ...string) {
			Context("params", func() {
				It("can invoke command with short names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{},
						append([]string{
							"deployment",
							"--ca-cert", sslCertPath,
							"-u", "admin",
							"-p", "admin",
							"-t", director.URL,
							"-d", "my-new-deployment",
							cmd}, extraArgs...)...)

					director.VerifyMocks()
				})

				It("can invoke command with long names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{},
						append([]string{
							"deployment",
							"--ca-cert", sslCertPath,
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd}, extraArgs...)...)

					director.VerifyMocks()
				})
			})

			Context("password is supported from env", func() {
				It("can invoke command with long names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--ca-cert", sslCertPath,
							"--username", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd}, extraArgs...)...)

					director.VerifyMocks()
				})
			})

			Context("Hostname is malformed", func() {
				var output helpText
				var session *gexec.Session
				BeforeEach(func() {
					badDirectorURL := "https://:25555"
					session = binary.Run(backupWorkspace,
						[]string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--username", "admin",
							"--password", "admin",
							"--target", badDirectorURL,
							"--deployment", "my-new-deployment",
							cmd}, extraArgs...)...)
					output.output = session.Err.Contents()
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(output.outputString()).To(ContainSubstring("invalid bosh URL"))
				})
			})

			Context("Custom CA cert cannot be read", func() {
				var output helpText
				var session *gexec.Session
				BeforeEach(func() {
					session = binary.Run(backupWorkspace,
						[]string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--ca-cert", "/tmp/whatever",
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd}, extraArgs...)...)
					output.output = session.Err.Contents()
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(output.outputString()).To(ContainSubstring("open /tmp/whatever: no such file or directory"))
				})
			})

			Context("Wrong global args", func() {
				var output helpText
				var session *gexec.Session
				BeforeEach(func() {
					session = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--dave", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd}, extraArgs...)...)
					output.output = session.Out.Contents()
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(output.outputString()).To(ContainSubstring("Incorrect Usage"))
				})

				It("displays the usable flags", func() {
					ShowsTheDeploymentHelpText(&output)
				})
			})

			Context("when any required flags are missing", func() {
				var output helpText
				var session *gexec.Session
				var command []string
				var env []string
				BeforeEach(func() {
					env = []string{"BOSH_CLIENT_SECRET=admin"}
				})
				JustBeforeEach(func() {
					session = binary.Run(backupWorkspace, env, command...)
					output.output = session.Out.Contents()
				})

				Context("Missing target", func() {
					BeforeEach(func() {
						command = append([]string{"deployment", "--username", "admin", "--password", "admin", "--deployment", "my-new-deployment", cmd}, extraArgs...)
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--target flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})

				Context("Missing username", func() {
					BeforeEach(func() {
						command = append([]string{"deployment", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd}, extraArgs...)
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--username flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})

				Context("Missing password in args", func() {
					BeforeEach(func() {
						env = []string{}
						command = append([]string{"deployment", "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd}, extraArgs...)
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--password flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})

				Context("Missing deployment", func() {
					BeforeEach(func() {
						command = append([]string{"deployment", "--username", "admin", "--password", "admin", "--target", director.URL, cmd}, extraArgs...)
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--deployment flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})
			})

			Context("with debug flag set", func() {
				It("outputs verbose HTTP logs", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					session := binary.Run(backupWorkspace, []string{},
						append([]string{
							"deployment",
							"--debug", "--ca-cert",
							sslCertPath, "--username",
							"admin", "--password",
							"admin", "--target",
							director.URL, "--deployment", "my-new-deployment", cmd}, extraArgs...)...)

					Expect(string(session.Out.Contents())).To(ContainSubstring("Sending GET request to endpoint"))

					director.VerifyMocks()
				})
			})
		}

		Context("backup", func() {
			AssertDeploymentCLIBehaviour("backup")
		})

		Context("restore", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(backupWorkspace+"/"+"my-new-deployment", 0777)).To(Succeed())
				createFileWithContents(backupWorkspace+"/"+"my-new-deployment"+"/"+"metadata", []byte(`---
instances: []`))
			})

			AssertDeploymentCLIBehaviour("restore", "--artifact-path", "my-new-deployment")

			Context("when artifact-path is not specified", func() {
				var session *gexec.Session

				BeforeEach(func() {
					session = binary.Run(backupWorkspace, []string{},
						"deployment",
						"--ca-cert", sslCertPath,
						"--username", "admin",
						"--password", "admin",
						"--target", director.URL,
						"--deployment", "my-new-deployment",
						"restore")
					Eventually(session).Should(gexec.Exit())
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("--artifact-path flag is required"))
				})
			})
		})

		Context("--help", func() {
			var output helpText

			BeforeEach(func() {
				output.output = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--help").Out.Contents()
			})

			It("displays the usable flags", func() {
				ShowsTheDeploymentHelpText(&output)
			})
		})

		Context("no arguments", func() {
			var output helpText

			BeforeEach(func() {
				output.output = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment").Out.Contents()
			})

			It("displays the usable flags", func() {
				ShowsTheDeploymentHelpText(&output)
			})
		})

	})

	Context("bbr with no arguments", func() {
		var output helpText

		BeforeEach(func() {
			output.output = binary.Run(backupWorkspace, []string{""}).Out.Contents()
		})

		It("displays the usable flags", func() {
			ShowsTheMainHelpText(&output)
		})
	})

})
