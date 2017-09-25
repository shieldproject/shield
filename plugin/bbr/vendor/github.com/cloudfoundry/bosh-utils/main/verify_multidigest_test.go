package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"io/ioutil"
	"os"
	"os/exec"
)

var _ = Describe("Verify_multidigest", func() {
	var session *gexec.Session
	var act func(arg ...string)
	var tempFile *os.File

	BeforeEach(func() {
		var err error
		tempFile, err = ioutil.TempFile("", "multi-digest-test")
		Expect(err).ToNot(HaveOccurred())
		_, err = tempFile.WriteString("sample content")
		Expect(err).ToNot(HaveOccurred())

		act = func(argCommands ...string) {
			var err error
			command := exec.Command(pathToBoshUtils, argCommands...)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
		}
	})

	AfterEach(func() {
		os.Remove(tempFile.Name())
	})

	Describe("version option", func() {
		It("has a version flag", func() {
			act("--version")
			Eventually(session).Should(gexec.Exit(0))
			Eventually(session.Out).Should(gbytes.Say("version \\[DEV BUILD\\]"))
		})
	})

	Context("verification", func() {
		Context("when correct args are passed to verify-multi-digest command", func() {
			It("exits 0", func() {
				act("verify-multi-digest", tempFile.Name(), "c4f246e2d6f84ee61a699d68a4bd1a2e43ec40f6")
				Eventually(session).Should(gexec.Exit(0))
			})
		})

		Context("when passing incorrect args", func() {
			It("exits 1 when digest does not match", func() {
				act("verify-multi-digest", tempFile.Name(), "incorrectdigest")
				Eventually(session).Should(gexec.Exit(1))
				Eventually(session.Err).Should(gbytes.Say("Expected stream to have digest 'incorrectdigest' but was 'c4f246e2d6f84ee61a699d68a4bd1a2e43ec40f6'"))
			})

			It("exits 1 when file does not exist", func() {
				act("verify-multi-digest", "potato", "c4f246e2d6f84ee61a699d68a4bd1a2e43ec40f6")
				Eventually(session).Should(gexec.Exit(1))
				Eventually(session.Err).Should(gbytes.Say("open potato:"))
			})
		})
	})

	Context("digest creation", func() {
		Context("when correct args are passed to create-multi-digest command", func() {
			It("sha1", func() {
				act("create-multi-digest", "sha1", tempFile.Name())
				Eventually(session).Should(gexec.Exit(0))
				Eventually(session).Should(gbytes.Say("c4f246e2d6f84ee61a699d68a4bd1a2e43ec40f6"))
			})

			It("sha256", func() {
				act("create-multi-digest", "sha256", tempFile.Name())
				Eventually(session).Should(gexec.Exit(0))
				Eventually(session).Should(gbytes.Say("sha256:571ca3b4ef92a81f8c062f2c2437b9116435d1575589a7b64a5c607d058fde0d"))
			})

			It("sha512", func() {
				act("create-multi-digest", "sha512", tempFile.Name())
				Eventually(session).Should(gexec.Exit(0))
				Eventually(session).Should(gbytes.Say("sha512:bd9686023e9b5ddca02fe00ca0fcfe4dccbee6470ff90795aa005809c374b3a9f00cde7eba1a8266b715a0789041d08650d5cc4182856091ed93cfd3dd1195c8"))
			})

			It("sha1,sha256", func() {
				act("create-multi-digest", "sha1,sha256", tempFile.Name())
				Eventually(session).Should(gexec.Exit(0))
				Eventually(session).Should(gbytes.Say("c4f246e2d6f84ee61a699d68a4bd1a2e43ec40f6;sha256:571ca3b4ef92a81f8c062f2c2437b9116435d1575589a7b64a5c607d058fde0d"))
			})

			It("does not emit any newlines or other whitespace", func() {
				act("create-multi-digest", "sha1,sha256", tempFile.Name())
				Eventually(session).Should(gexec.Exit(0))
				Eventually(session).Should(gbytes.Say(`\Ac4f246e2d6f84ee61a699d68a4bd1a2e43ec40f6;sha256:571ca3b4ef92a81f8c062f2c2437b9116435d1575589a7b64a5c607d058fde0d\z`))

				act("create-multi-digest", "sha1", tempFile.Name())
				Eventually(session).Should(gexec.Exit(0))
				Eventually(session).Should(gbytes.Say(`\Ac4f246e2d6f84ee61a699d68a4bd1a2e43ec40f6\z`))
			})
		})

		Context("when passing incorrect args", func() {
			It("exits 1 when file does not exist", func() {
				act("create-multi-digest", "sha1", "potato")
				Eventually(session).Should(gexec.Exit(1))
				Eventually(session.Err).Should(gbytes.Say("Calculating digest of 'potato': open potato:"))
			})

			It("exits 1 when the algorithm is unknown", func() {
				act("create-multi-digest", "potoato", tempFile.Name())
				Eventually(session).Should(gexec.Exit(1))
				Eventually(session.Err).Should(gbytes.Say("unknown algorithm 'potoato'"))
			})
		})
	})

})
