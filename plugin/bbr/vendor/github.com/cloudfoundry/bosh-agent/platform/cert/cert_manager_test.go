package cert_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/bosh-agent/platform/cert"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	"github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/system"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

const cert1 string = `-----BEGIN CERTIFICATE-----
MIIEJDCCAwygAwIBAgIJAO+CqgiJnCgpMA0GCSqGSIb3DQEBBQUAMGkxCzAJBgNV
BAYTAkNBMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX
qokoSBXzJCJTt2P681gyqBDr/hUYzqpoXUsOTRisScbEbaSv8hTiTeFJUMyNQAqn
DtmvI8bXKxU=
-----END CERTIFICATE-----`

var _ = Describe("Certificate Management", func() {
	var log logger.Logger
	BeforeEach(func() {
		log = logger.NewLogger(logger.LevelNone)
	})

	Describe("CertificateSplitting", func() {
		It("splits 2 back-to-back certificates", func() {
			certs := fmt.Sprintf("%s\n%s\n", cert1, cert1)

			result := cert.SplitCerts(certs)
			Expect(result[0]).To(Equal(cert1))
			Expect(result[1]).To(Equal(cert1))
			Expect(len(result)).To(Equal(2))
		})

		It("splits 2 back-to-back certificates without trailing newline", func() {
			certs := fmt.Sprintf("%s\n%s", cert1, cert1)

			result := cert.SplitCerts(certs)
			Expect(result[0]).To(Equal(cert1))
			Expect(result[1]).To(Equal(cert1))
			Expect(len(result)).To(Equal(2))
		})

		It("splits 2 back-to-back certificates ignoring junk between them", func() {
			certs := fmt.Sprintf("%s\n abcdefghij %s\n", cert1, cert1)

			result := cert.SplitCerts(certs)
			Expect(result[0]).To(Equal(cert1))
			Expect(result[1]).To(Equal(cert1))
			Expect(len(result)).To(Equal(2))
		})

		It("handles 1 certificate with trailing newline", func() {
			certs := fmt.Sprintf("%s\n", cert1)

			result := cert.SplitCerts(certs)
			Expect(result[0]).To(Equal(cert1))
			Expect(len(result)).To(Equal(1))
		})

		It("handles 1 certificate without trailing newline", func() {
			result := cert.SplitCerts(cert1)
			Expect(result[0]).To(Equal(cert1))
			Expect(len(result)).To(Equal(1))
		})

		It("ignores junk before the first certicate", func() {
			certs := fmt.Sprintf("abcdefg %s\n%s\n", cert1, cert1)

			result := cert.SplitCerts(certs)
			Expect(result[0]).To(Equal(cert1))
			Expect(result[1]).To(Equal(cert1))
			Expect(len(result)).To(Equal(2))
		})

		It("ignores junk after the last certicate", func() {
			certs := fmt.Sprintf("%s\n%s\n abcdefghij", cert1, cert1)

			result := cert.SplitCerts(certs)
			Expect(result[0]).To(Equal(cert1))
			Expect(result[1]).To(Equal(cert1))
			Expect(len(result)).To(Equal(2))
		})

		It("returns an empty slice for an empty string", func() {
			result := cert.SplitCerts("")
			Expect(len(result)).To(Equal(0))
		})

		It("returns an empty slice for an non-empty string that does not contain any certificates", func() {
			result := cert.SplitCerts("abcdefghij")
			Expect(len(result)).To(Equal(0))
		})
	})

	Describe("DeleteFile()", func() {
		var (
			fakeFs *fakesys.FakeFileSystem
		)

		BeforeEach(func() {
			fakeFs = fakesys.NewFakeFileSystem()
		})

		It("only deletes the files with the given prefix", func() {
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_1.foo", "goodbye")
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_2.foo", "goodbye")
			fakeFs.WriteFileString("/path/to/delete/stuff/in/different_file_1.bar", "goodbye")
			fakeFs.SetGlob("/path/to/delete/stuff/in/delete_me_*", []string{
				"/path/to/delete/stuff/in/delete_me_1.foo",
				"/path/to/delete/stuff/in/delete_me_2.foo",
			})
			count, err := cert.DeleteFiles(fakeFs, "/path/to/delete/stuff/in/", "delete_me_")
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(countFiles(fakeFs, "/path/to/delete/stuff/in/")).To(Equal(1))
		})

		It("only deletes the files in the given path", func() {
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_1.foo", "goodbye")
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_2.foo", "goodbye")
			fakeFs.WriteFileString("/path/to/other/things/in/delete_me_3.foo", "goodbye")
			fakeFs.SetGlob("/path/to/delete/stuff/in/delete_me_*", []string{
				"/path/to/delete/stuff/in/delete_me_1.foo",
				"/path/to/delete/stuff/in/delete_me_2.foo",
			})
			count, err := cert.DeleteFiles(fakeFs, "/path/to/delete/stuff/in/", "delete_me_")
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(countFiles(fakeFs, "/path/to/delete/stuff/in/")).To(Equal(0))
			Expect(countFiles(fakeFs, "/path/to/other/things/in/")).To(Equal(1))
		})

		It("returns an error when glob fails", func() {
			fakeFs.GlobErr = errors.New("couldn't walk")
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_1.foo", "goodbye")
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_2.bar", "goodbye")
			count, err := cert.DeleteFiles(fakeFs, "/path/to/delete/stuff/in/", "delete_me_")
			Expect(err).To(HaveOccurred())
			Expect(count).To(Equal(0))
		})

		It("returns an error when RemoveAll() fails", func() {
			fakeFs.RemoveAllStub = func(_ string) error {
				return errors.New("couldn't delete")
			}
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_1.foo", "goodbye")
			fakeFs.WriteFileString("/path/to/delete/stuff/in/delete_me_2.bar", "goodbye")
			fakeFs.SetGlob("/path/to/delete/stuff/in/delete_me_*", []string{
				"/path/to/delete/stuff/in/delete_me_1.foo",
				"/path/to/delete/stuff/in/delete_me_2.bar",
			})
			count, err := cert.DeleteFiles(fakeFs, "/path/to/delete/stuff/in/", "delete_me_")
			Expect(err).To(HaveOccurred())
			Expect(count).To(Equal(0))
		})
	})

	Describe("cert.Manager implementations", func() {
		var (
			fakeFs        *fakesys.FakeFileSystem
			fakeCmdRunner *fakesys.FakeCmdRunner
			certManager   cert.Manager
		)

		SharedLinuxCertManagerExamples := func(certBasePath, certUpdateProgram string) {
			It("writes 1 cert to a file", func() {
				err := certManager.UpdateCertificates(cert1)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeFs.FileExists(fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath))).To(BeTrue())
			})

			It("writes each cert to its own file", func() {
				certs := fmt.Sprintf("%s\n%s\n", cert1, cert1)

				err := certManager.UpdateCertificates(certs)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeFs.FileExists(fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath))).To(BeTrue())
				Expect(fakeFs.FileExists(fmt.Sprintf("%s/bosh-trusted-cert-2.crt", certBasePath))).To(BeTrue())
				Expect(countFiles(fakeFs, certBasePath)).To(Equal(2))
			})

			It("deletes all certs when passed an empty string", func() {
				fakeFs.WriteFileString(fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath), "goodbye")
				fakeFs.SetGlob(fmt.Sprintf("%s/bosh-trusted-cert-*", certBasePath), []string{
					fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath),
				})
				err := certManager.UpdateCertificates("")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeFs.FileExists(fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath))).To(BeFalse())
			})

			It("deletes exisitng cert files before writing new ones", func() {
				certs := fmt.Sprintf("%s\n%s\n", cert1, cert1)
				err := certManager.UpdateCertificates(certs)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeFs.FileExists(fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath))).To(BeTrue())
				Expect(fakeFs.FileExists(fmt.Sprintf("%s/bosh-trusted-cert-2.crt", certBasePath))).To(BeTrue())
				Expect(countFiles(fakeFs, certBasePath)).To(Equal(2))

				fakeFs.SetGlob(fmt.Sprintf("%s/bosh-trusted-cert-*", certBasePath), []string{
					fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath),
					fmt.Sprintf("%s/bosh-trusted-cert-2.crt", certBasePath),
				})
				certManager.UpdateCertificates(cert1)
				Expect(fakeFs.FileExists(fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath))).To(BeTrue())
				Expect(countFiles(fakeFs, certBasePath)).To(Equal(1))
			})

			It("returns an error when writing new cert files fails", func() {
				fakeFs.WriteFileError = errors.New("NOT ALLOW")
				err := certManager.UpdateCertificates(cert1)
				Expect(err).To(HaveOccurred())
			})

			It("returns an error when deleting old certs fails", func() {
				fakeFs.RemoveAllStub = func(_ string) error {
					return errors.New("NOT ALLOW")
				}
				fakeFs.WriteFileString(fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath), "goodbye")
				fakeFs.SetGlob(fmt.Sprintf("%s/bosh-trusted-cert-*", certBasePath), []string{
					fmt.Sprintf("%s/bosh-trusted-cert-1.crt", certBasePath),
				})

				err := certManager.UpdateCertificates("")
				Expect(err).To(HaveOccurred())
			})
		}

		Context("Ubuntu", func() {
			var (
				fakeResult   boshsys.Result
				fakeProcess1 *fakesys.FakeProcess
				fakeProcess2 *fakesys.FakeProcess
				fakeProcess3 *fakesys.FakeProcess
			)
			BeforeEach(func() {
				fakeFs = fakesys.NewFakeFileSystem()
				fakeCmdRunner = fakesys.NewFakeCmdRunner()
				fakeCmdRunner.AddCmdResult("/usr/sbin/update-ca-certificates", fakesys.FakeCmdResult{
					Stdout:     "",
					Stderr:     "",
					ExitStatus: 0,
					Sticky:     true,
				})
				certManager = cert.NewUbuntuCertManager(fakeFs, fakeCmdRunner, 1, log)
				fakeResult = boshsys.Result{
					Stdout:     "",
					Stderr:     "",
					ExitStatus: 0,
					Error:      nil,
				}

				fakeProcess1 = &fakesys.FakeProcess{WaitResult: fakeResult}
				fakeProcess2 = &fakesys.FakeProcess{WaitResult: fakeResult}
				fakeProcess3 = &fakesys.FakeProcess{WaitResult: fakeResult}

				fakeCmdRunner.AddProcess("/usr/sbin/update-ca-certificates -f", fakeProcess1)
				fakeCmdRunner.AddProcess("/usr/sbin/update-ca-certificates -f", fakeProcess2)
				fakeCmdRunner.AddProcess("/usr/sbin/update-ca-certificates -f", fakeProcess3)
			})

			SharedLinuxCertManagerExamples("/usr/local/share/ca-certificates", "/usr/sbin/update-ca-certificates")

			It("updates certs", func() {
				err := certManager.UpdateCertificates(cert1)

				Expect(fakeProcess1.Waited).To(BeTrue())
				Expect(fakeProcess1.TerminatedNicely).To(BeFalse())

				Expect(err).ToNot(HaveOccurred())
			})

			It("fails at first try and succeeds by killing and re-run", func() {
				fakeResult.ExitStatus = 143
				fakeResult.Error = errors.New("command failed")

				fakeProcess1.TerminatedNicelyCallBack = func(p *fakesys.FakeProcess) {}

				err := certManager.UpdateCertificates(cert1)

				Expect(fakeProcess1.Waited).To(BeTrue())
				Expect(fakeProcess1.TerminatedNicely).To(BeTrue())
				Expect(fakeProcess1.TerminateNicelyKillGracePeriod).To(Equal(5 * time.Second))
				Expect(fakeProcess2.Waited).To(BeTrue())
				Expect(fakeProcess2.TerminatedNicely).To(BeFalse())
				Expect(fakeProcess3.Waited).To(BeFalse())

				Expect(err).ToNot(HaveOccurred())
			})

			It("terminates update cert command nicely upon time-out", func() {
				fakeResult.ExitStatus = 143
				fakeResult.Error = errors.New("command failed")

				fakeProcess1.TerminatedNicelyCallBack = func(p *fakesys.FakeProcess) {}
				fakeProcess2.TerminatedNicelyCallBack = func(p *fakesys.FakeProcess) {}
				fakeProcess3.TerminatedNicelyCallBack = func(p *fakesys.FakeProcess) {}

				err := certManager.UpdateCertificates(cert1)

				Expect(fakeProcess1.Waited).To(BeTrue())
				Expect(fakeProcess1.TerminatedNicely).To(BeTrue())
				Expect(fakeProcess1.TerminateNicelyKillGracePeriod).To(Equal(5 * time.Second))
				Expect(fakeProcess2.Waited).To(BeTrue())
				Expect(fakeProcess2.TerminatedNicely).To(BeTrue())
				Expect(fakeProcess2.TerminateNicelyKillGracePeriod).To(Equal(5 * time.Second))
				Expect(fakeProcess3.Waited).To(BeTrue())
				Expect(fakeProcess3.TerminatedNicely).To(BeTrue())
				Expect(fakeProcess3.TerminateNicelyKillGracePeriod).To(Equal(5 * time.Second))

				Expect(err).To(HaveOccurred())
			})
		})

		Context("CentOS", func() {
			BeforeEach(func() {
				fakeFs = fakesys.NewFakeFileSystem()
				fakeCmdRunner = fakesys.NewFakeCmdRunner()
				fakeCmdRunner.AddCmdResult("/usr/bin/update-ca-trust", fakesys.FakeCmdResult{
					Stdout:     "",
					Stderr:     "",
					ExitStatus: 0,
					Sticky:     true,
				})
				certManager = cert.NewCentOSCertManager(fakeFs, fakeCmdRunner, 0, log)
			})

			SharedLinuxCertManagerExamples("/etc/pki/ca-trust/source/anchors", "/usr/bin/update-ca-trust")

			It("executes update cert command", func() {
				fakeCmdRunner = fakesys.NewFakeCmdRunner()
				fakeCmdRunner.AddCmdResult("/usr/bin/update-ca-trust", fakesys.FakeCmdResult{
					Stdout:     "",
					Stderr:     "",
					ExitStatus: 2,
					Error:      errors.New("command failed"),
				})
				certManager = cert.NewCentOSCertManager(fakeFs, fakeCmdRunner, 0, log)

				err := certManager.UpdateCertificates(cert1)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("OpenSUSE", func() {
			BeforeEach(func() {
				fakeFs = fakesys.NewFakeFileSystem()
				fakeCmdRunner = fakesys.NewFakeCmdRunner()
				fakeCmdRunner.AddCmdResult("/usr/bin/update-ca-trust", fakesys.FakeCmdResult{
					Stdout:     "",
					Stderr:     "",
					ExitStatus: 0,
					Sticky:     true,
				})
				certManager = cert.NewOpensuseOSCertManager(fakeFs, fakeCmdRunner, 0, log)
			})

			SharedLinuxCertManagerExamples("/usr/lib/ca-certificates", "/usr/sbin/update-ca-certificates")

			It("executes update cert command", func() {
				fakeCmdRunner = fakesys.NewFakeCmdRunner()
				fakeCmdRunner.AddCmdResult("/usr/sbin/update-ca-certificates", fakesys.FakeCmdResult{
					Stdout:     "",
					Stderr:     "",
					ExitStatus: 2,
					Error:      errors.New("command failed"),
				})
				certManager = cert.NewOpensuseOSCertManager(fakeFs, fakeCmdRunner, 0, log)

				err := certManager.UpdateCertificates(cert1)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Windows", func() {
			const validCerts string = `-----BEGIN CERTIFICATE-----
MIIC0jCCAboCCQCuQJScK+G0WzANBgkqhkiG9w0BAQsFADArMQswCQYDVQQGEwJV
UzENMAsGA1UECBMEQk9TSDENMAsGA1UEChMEQk9TSDAeFw0xNjA4MDIxNDQ2MTla
Fw0xNzA4MDIxNDQ2MTlaMCsxCzAJBgNVBAYTAlVTMQ0wCwYDVQQIEwRCT1NIMQ0w
CwYDVQQKEwRCT1NIMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA22ea
D3XBlXOLDzcJOKKICYrkoHxT4wg+9ybRS9r/oAx+5xEwTdoUNvK3j7hUP4ttfCgT
qx9TUN+h4HzjDZQQ9oj8aUOhV83BLawUxDUOZbyDGUrHCKXkE5UKeiMjVtfmZNd0
0t+zepLF+helT0p+ogXFGFM6pKgfNoPHrf5R+KUqzvCoeMiL9nxO/yypfR+fnKOQ
KYGo55BlH0nYLAwKfefiUkaqAOMyQ7mdLf+iWT6CqfZ83OdNSXe8SmaDspnHkipu
/9+/VBEABv+IiAgLrosynSIA0DFP4vPYuV6PzHW8pXpTB6CSl8QwhPQv3SpgjXoB
O3rMc0pJ/2sSRIXKvQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBnbY4FOo28yWAJ
G5hkOReWl6f6y/LNa+W5B7zqoPuUpiYwujdDSGA+wsig46EK65mEK2NdGO2PnTKw
hP27FHbagskiu9h0PtEfBcRi6lNySOgQNFEqpB+maOzwOwYRRxdABBu0ieSaxYXI
TINuBZ/Fi1igmL4Auwl4mFLYn6ofrtZFOLp7a1vGDewZFG75V4t2IdKvN8HsCnPW
vHfs34+z5ZdCHWY7uQFmC1K+4oqKanG7Lw78bZ+HaU5fLb8CpvkiDmCDA/KXXpCS
En4cZ4+CJRoyzjaooDDOo/+9P7Mx1O12Ev/lna2laLLueUyTN3aVPbLvWsUrCr/1
NrjpvLIP
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIC0jCCAboCCQC/JcYWmGS6OTANBgkqhkiG9w0BAQsFADArMQswCQYDVQQGEwJV
UzENMAsGA1UECBMEQk9TSDENMAsGA1UEChMEQk9TSDAeFw0xNjA4MDIyMTQyMzJa
Fw0xNzA4MDIyMTQyMzJaMCsxCzAJBgNVBAYTAlVTMQ0wCwYDVQQIEwRCT1NIMQ0w
CwYDVQQKEwRCT1NIMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA7Z4R
8dSoipPja6cnjs5x3bk2zfuHwSFW6XHOASNVQXxdSgbRixXeiSh0cJoT0FUvGnQX
ptU2WeMtx7ZrXp3YcO312bVxjyEBhzlvLhdqWHaATOucuvXi+sH+I4EXVhlHlbr7
+OhR85q0DCdF9x7U3xVJm/JG/cNXHtNB0aaYUZ9HXpVpt8yMdVGQCE8FMqNQ4DsU
/WHRCaTkoP3BXbza090yoGMSCT8IilrKUnwmtNZiDerWwTJfVz6oqIN8Ei+myJ4M
qvis48OQkOgg/e1RbrCGuF2L7q7Ja3j1RQWgEXrNiK45Eae3W6uhbTV6RXPrk9Xk
Si8Atvw03rkuqJjXYwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQC6HK25lvP2PLmF
KRQ4z7qOvIVXNl9m4scHCsINF+VpZo+miXK2kMhOk6Bade+PG76dYRNhPXv0vWqe
QNHDW2J85dF1h0Dbdl84irCijSb1WOPHdRgqSMooTaRxn0mpRMKgUdOSuJTUj6N7
yHdf1gYNB8vt/NzTfl1gKc0KjK9L8I2Y0myq9Hu1aHVELFAKskhZJpnToZn1w6O0
WDtlweO/jTmDwyeIqzA/60LXAv7xfJMRoyNElqWHC+EeeuMnh6BJPSdwC8ynTP3R
SDRQj6MXyyS4LBMZA56DYXaXyR6pDTpmvBUNQ4FR0UgYm1GeGWo1kOUPjfs7sUQz
aAzOWRDC
-----END CERTIFICATE-----`

			var certThumbprints []string = []string{
				"23AC7706D032651BE146388FA8DF7B0B2DD7CFA6",
				"73C0BFD7BB53EC299B289CB86A010AE485F6D49B",
			}

			const getCertScript string = `
(Get-ChildItem Cert:\LocalMachine\Root | where { $_.Subject -eq "O=BOSH, S=BOSH, C=US" }).Length`

			const removeCertScript string = `
if (Test-Path %[1]s) {
	Remove-Item %[1]s
}`

			var tempDir string
			var dirProvider boshdir.Provider
			var fs boshsys.FileSystem

			BeforeEach(func() {
				if runtime.GOOS != "windows" {
					Skip("Only run on Windows")
				}

				fs = boshsys.NewOsFileSystem(log)
				var err error
				tempDir, err = fs.TempDir("")
				Expect(err).To(BeNil())
				dirProvider = boshdir.NewProvider(tempDir)
				certManager = cert.NewWindowsCertManager(fs, boshsys.NewExecCmdRunner(log), dirProvider, log)
			})

			AfterEach(func() {
				for _, thumbprint := range certThumbprints {
					cmd := exec.Command("powershell", "-Command", fmt.Sprintf(removeCertScript, `Cert:\LocalMachine\Root\`+thumbprint))
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).To(BeNil())
					Eventually(session).Should(gexec.Exit(0))
				}
				os.RemoveAll(tempDir)
			})

			It("should create the tmpDir if doesn't exist", func() {
				_, err := os.Stat(dirProvider.TmpDir())
				fmt.Println("BEfore", dirProvider.TmpDir(), err)
				missing := os.IsNotExist(err)
				Expect(missing).To(BeTrue())
				err = certManager.UpdateCertificates(validCerts)
				Expect(err).To(BeNil())
				_, err = os.Stat(dirProvider.TmpDir())
				Expect(err).To(BeNil())
			})

			Context("When TempDir exists", func() {
				BeforeEach(func() {
					err := fs.MkdirAll(dirProvider.TmpDir(), os.FileMode(0777))
					Expect(err).To(BeNil())
				})
				It("adds certs to the trusted cert chain", func() {
					err := certManager.UpdateCertificates(validCerts)
					Expect(err).To(BeNil())

					cmd := exec.Command("powershell", "-Command", getCertScript)
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).To(BeNil())

					Eventually(session).Should(gexec.Exit(0))
					Eventually(session.Out).Should(gbytes.Say("2"))
				})

				It("returns an error when passed an invalid cert", func() {
					err := certManager.UpdateCertificates(cert1)
					Expect(err).NotTo(BeNil())
				})

				It("deletes all certs when passed an empty string", func() {
					err := certManager.UpdateCertificates(validCerts)
					Expect(err).To(BeNil())

					err = certManager.UpdateCertificates("")
					Expect(err).To(BeNil())

					cmd := exec.Command("powershell", "-Command", getCertScript)
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).To(BeNil())

					Eventually(session).Should(gexec.Exit(0))
					Eventually(session.Out).Should(gbytes.Say("0"))
				})
			})

		})
	})
})

func countFiles(fs system.FileSystem, dir string) (count int) {
	fs.Walk(dir, func(path string, info os.FileInfo, err error) error {
		count++
		return nil
	})
	return
}
