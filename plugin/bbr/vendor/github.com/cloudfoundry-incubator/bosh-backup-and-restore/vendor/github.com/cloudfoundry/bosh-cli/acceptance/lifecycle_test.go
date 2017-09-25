package acceptance_test

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"

	. "github.com/cloudfoundry/bosh-cli/acceptance"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bitestutils "github.com/cloudfoundry/bosh-cli/testutils"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const (
	stageTimePattern                   = "\\(\\d{2}:\\d{2}:\\d{2}\\)"
	stageFinishedPattern               = "\\.\\.\\. Finished " + stageTimePattern + "$"
	stageCompiledPackageSkippedPattern = "\\.\\.\\. Skipped \\[Package already compiled\\] " + stageTimePattern + "$"
)

var _ = Describe("bosh", func() {
	var (
		logger       boshlog.Logger
		fileSystem   boshsys.FileSystem
		sshCmdRunner CmdRunner
		cmdEnv       map[string]string
		quietCmdEnv  map[string]string
		testEnv      Environment
		config       *Config

		instanceSSH      InstanceSSH
		instanceUsername = "vcap"
		instancePassword = "sshpassword" // encrypted value must be in the manifest: resource_pool.env.bosh.password
		instanceIP       = "10.244.0.42"
	)

	var readLogFile = func(logPath string) (stdout string) {
		stdout, _, exitCode, err := sshCmdRunner.RunCommand(cmdEnv, "cat", logPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
		return stdout
	}

	var deleteLogFile = func(logPath string) {
		_, _, exitCode, err := sshCmdRunner.RunCommand(cmdEnv, "rm", "-f", logPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
	}

	var flushLog = func(logPath string) {
		logString := readLogFile(logPath)
		_, err := GinkgoWriter.Write([]byte(logString))
		Expect(err).ToNot(HaveOccurred())

		// only delete after successfully writing to GinkgoWriter
		deleteLogFile(logPath)
	}

	type manifestContext struct {
		CPIReleaseURL            string
		CPIReleaseSHA1           string
		DummyReleasePath         string
		DummyTooReleasePath      string
		DummyCompiledReleasePath string
		StemcellURL              string
		StemcellSHA1             string
	}

	var prepareDeploymentManifest = func(context manifestContext, sourceManifestPath string) []byte {
		if config.IsLocalCPIRelease() {
			context.CPIReleaseURL = "file://" + testEnv.Path("cpi-release.tgz")
		} else {
			context.CPIReleaseURL = config.CPIReleaseURL
			context.CPIReleaseSHA1 = config.CPIReleaseSHA1
		}

		if config.IsLocalStemcell() {
			context.StemcellURL = "file://" + testEnv.Path("stemcell.tgz")
		} else {
			context.StemcellURL = config.StemcellURL
			context.StemcellSHA1 = config.StemcellSHA1
		}

		buffer := &bytes.Buffer{}
		t := template.Must(template.ParseFiles(sourceManifestPath))
		err := t.Execute(buffer, context)
		Expect(err).ToNot(HaveOccurred())

		return buffer.Bytes()
	}

	// updateDeploymentManifest copies a source manifest from assets to <workspace>/manifest
	var updateDeploymentManifest = func(sourceManifestPath string) {
		context := manifestContext{
			DummyReleasePath:    testEnv.Path("dummy-release.tgz"),
			DummyTooReleasePath: testEnv.Path("dummy-too-release.tgz"),
		}

		buffer := prepareDeploymentManifest(context, sourceManifestPath)
		testEnv.WriteContent("test-manifest.yml", buffer)
	}

	var updateCompiledReleaseDeploymentManifest = func(sourceManifestPath string) {
		context := manifestContext{
			DummyCompiledReleasePath: testEnv.Path("sample-release-compiled.tgz"),
		}

		buffer := prepareDeploymentManifest(context, sourceManifestPath)
		testEnv.WriteContent("test-compiled-manifest.yml", buffer)
	}

	var deploy = func(manifestFile string) string {
		fmt.Fprintf(GinkgoWriter, "\n--- DEPLOY ---\n")

		stdout := &bytes.Buffer{}
		multiWriter := io.MultiWriter(stdout, GinkgoWriter)

		_, _, exitCode, err := sshCmdRunner.RunStreamingCommand(multiWriter, cmdEnv, testEnv.Path("bosh"), "create-env", "--tty", testEnv.Path(manifestFile))
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))

		return stdout.String()
	}

	var expectDeployToError = func() string {
		fmt.Fprintf(GinkgoWriter, "\n--- DEPLOY ---\n")

		stdout := &bytes.Buffer{}
		multiWriter := io.MultiWriter(stdout, GinkgoWriter)

		_, _, exitCode, err := sshCmdRunner.RunStreamingCommand(multiWriter, cmdEnv, testEnv.Path("bosh"), "create-env", "--tty", testEnv.Path("test-manifest.yml"))
		Expect(err).To(HaveOccurred())
		Expect(exitCode).To(Equal(1))

		return stdout.String()
	}

	var deleteDeployment = func() string {
		fmt.Fprintf(GinkgoWriter, "\n--- DELETE DEPLOYMENT ---\n")

		stdout := &bytes.Buffer{}
		multiWriter := io.MultiWriter(stdout, GinkgoWriter)

		_, _, exitCode, err := sshCmdRunner.RunStreamingCommand(multiWriter, cmdEnv, testEnv.Path("bosh"), "delete-env", "--tty", testEnv.Path("test-manifest.yml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))

		return stdout.String()
	}

	var shutdownAgent = func() {
		_, _, exitCode, err := instanceSSH.RunCommandWithSudo("sv stop agent")
		Expect(err).ToNot(HaveOccurred())
		Expect(exitCode).To(Equal(0))
	}

	var findStage = func(outputLines []string, stageName string, zeroIndex int) (steps []string, stopIndex int) {
		startLine := fmt.Sprintf("Started %s", stageName)
		startIndex := -1
		for i, line := range outputLines[zeroIndex:] {
			if line == startLine {
				startIndex = zeroIndex + i
				break
			}
		}
		if startIndex < 0 {
			Fail("Failed to find stage start: " + stageName + ". Lines: start>>\n" + strings.Join(outputLines, "\n") + "<<end")
		}

		stopLinePattern := fmt.Sprintf("^Finished %s %s$", stageName, stageTimePattern)
		stopLineRegex, err := regexp.Compile(stopLinePattern)
		Expect(err).ToNot(HaveOccurred())

		stopIndex = -1
		for i, line := range outputLines[startIndex:] {
			if stopLineRegex.MatchString(line) {
				stopIndex = startIndex + i
				break
			}
		}
		if stopIndex < 0 {
			Fail("Failed to find stage stop: " + stageName)
		}

		return outputLines[startIndex+1 : stopIndex], stopIndex
	}

	BeforeSuite(func() {
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, GinkgoWriter, GinkgoWriter)
		fileSystem = boshsys.NewOsFileSystem(logger)

		var err error
		config, err = NewConfig(fileSystem)
		Expect(err).NotTo(HaveOccurred())

		err = config.Validate()
		Expect(err).NotTo(HaveOccurred())

		testEnv = NewRemoteTestEnvironment(
			config.VMUsername,
			config.VMIP,
			config.VMPort,
			config.PrivateKeyPath,
			fileSystem,
			logger,
		)

		sshCmdRunner = NewSSHCmdRunner(
			config.VMUsername,
			config.VMIP,
			config.VMPort,
			config.PrivateKeyPath,
			logger,
		)
		cmdEnv = map[string]string{
			"TMPDIR":         testEnv.Home(),
			"BOSH_LOG_LEVEL": "DEBUG",
			"BOSH_LOG_PATH":  testEnv.Path("bosh-init.log"),
		}
		quietCmdEnv = map[string]string{
			"TMPDIR":         testEnv.Home(),
			"BOSH_LOG_LEVEL": "ERROR",
			"BOSH_LOG_PATH":  testEnv.Path("bosh-init-cleanup.log"),
		}

		// clean up from previous failed tests
		deleteLogFile(cmdEnv["BOSH_LOG_PATH"])
		deleteLogFile(quietCmdEnv["BOSH_LOG_PATH"])

		err = bitestutils.BuildExecutableForArch("linux-amd64")
		Expect(err).NotTo(HaveOccurred())

		boshCliPath := "./../out/bosh"
		Expect(fileSystem.FileExists(boshCliPath)).To(BeTrue())
		err = testEnv.Copy("bosh", boshCliPath)
		Expect(err).NotTo(HaveOccurred())

		instanceSSH = NewInstanceSSH(
			config.VMUsername,
			config.VMIP,
			config.VMPort,
			config.PrivateKeyPath,
			instanceUsername,
			instanceIP,
			instancePassword,
			fileSystem,
			logger,
		)

		if config.IsLocalStemcell() {
			err = testEnv.Copy("stemcell.tgz", config.StemcellPath)
			Expect(err).NotTo(HaveOccurred())
		}
		if config.IsLocalCPIRelease() {
			err = testEnv.Copy("cpi-release.tgz", config.CPIReleasePath)
			Expect(err).NotTo(HaveOccurred())
		}
		err = testEnv.Copy("dummy-release.tgz", config.DummyReleasePath)
		Expect(err).NotTo(HaveOccurred())

		err = testEnv.Copy("dummy-too-release.tgz", config.DummyTooReleasePath)
		Expect(err).NotTo(HaveOccurred())

		err = testEnv.Copy("sample-release-compiled.tgz", config.DummyCompiledReleasePath)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when deploying with a compiled release", func() {
		AfterEach(func() {
			flushLog(cmdEnv["BOSH_LOG_PATH"])

			// quietly delete the deployment
			_, _, exitCode, err := sshCmdRunner.RunCommand(quietCmdEnv, testEnv.Path("bosh"), "delete-env", "--tty", testEnv.Path("test-compiled-manifest.yml"))
			if exitCode != 0 || err != nil {
				// only flush the delete log if the delete failed
				flushLog(quietCmdEnv["BOSH_LOG_PATH"])
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(exitCode).To(Equal(0))
		})

		It("is able to deploy given many variances with compiled releases", func() {
			updateCompiledReleaseDeploymentManifest("./assets/sample-release-compiled-manifest.yml")

			By("deploying compiled releases successfully with expected output")
			stdout := deploy("test-compiled-manifest.yml")
			outputLines := strings.Split(stdout, "\n")
			numOutputLines := len(outputLines)

			doneIndex := 0
			stepIndex := -1
			nextStep := func() int { stepIndex++; return stepIndex }

			validatingSteps, doneIndex := findStage(outputLines, "validating", doneIndex)
			if !config.IsLocalCPIRelease() {
				Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Downloading release 'bosh-warden-cpi'"))
			}
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating release 'bosh-warden-cpi'" + stageFinishedPattern))
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating release 'sample-release'" + stageFinishedPattern))
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating cpi release" + stageFinishedPattern))
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating deployment manifest" + stageFinishedPattern))
			if !config.IsLocalStemcell() {
				Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Downloading stemcell"))
			}
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating stemcell" + stageFinishedPattern))

			installingSteps, doneIndex := findStage(outputLines, "installing CPI", doneIndex+1)
			numInstallingSteps := len(installingSteps)
			for _, line := range installingSteps[:numInstallingSteps-3] {
				Expect(line).To(MatchRegexp("^  Compiling package '.*/.*'" + stageFinishedPattern))
			}
			Expect(installingSteps[numInstallingSteps-3]).To(MatchRegexp("^  Installing packages" + stageFinishedPattern))
			Expect(installingSteps[numInstallingSteps-2]).To(MatchRegexp("^  Rendering job templates" + stageFinishedPattern))
			Expect(installingSteps[numInstallingSteps-1]).To(MatchRegexp("^  Installing job 'warden_cpi'" + stageFinishedPattern))

			Expect(outputLines[doneIndex+2]).To(MatchRegexp("^Starting registry" + stageFinishedPattern))
			Expect(outputLines[doneIndex+3]).To(MatchRegexp("^Uploading stemcell '.*/.*'" + stageFinishedPattern))

			deployingSteps, doneIndex := findStage(outputLines, "deploying", doneIndex+1)
			numDeployingSteps := len(deployingSteps)
			Expect(deployingSteps[0]).To(MatchRegexp("^  Creating VM for instance 'dummy_compiled_job/0' from stemcell '.*'" + stageFinishedPattern))
			Expect(deployingSteps[1]).To(MatchRegexp("^  Waiting for the agent on VM '.*' to be ready" + stageFinishedPattern))
			Expect(deployingSteps[2]).To(MatchRegexp("^  Creating disk" + stageFinishedPattern))
			Expect(deployingSteps[3]).To(MatchRegexp("^  Attaching disk '.*' to VM '.*'" + stageFinishedPattern))
			Expect(deployingSteps[4]).To(MatchRegexp("^  Rendering job templates" + stageFinishedPattern))

			for _, line := range deployingSteps[5 : numDeployingSteps-3] {
				Expect(line).To(MatchRegexp("^  Compiling package '.*/.*'" + stageCompiledPackageSkippedPattern))
			}

			Expect(deployingSteps[numDeployingSteps-3]).To(MatchRegexp("^  Updating instance 'dummy_compiled_job/0'" + stageFinishedPattern))
			Expect(deployingSteps[numDeployingSteps-2]).To(MatchRegexp("^  Waiting for instance 'dummy_compiled_job/0' to be running" + stageFinishedPattern))
			Expect(deployingSteps[numDeployingSteps-1]).To(MatchRegexp("^  Running the post-start scripts 'dummy_compiled_job/0'" + stageFinishedPattern))

			Expect(outputLines[numOutputLines-4]).To(MatchRegexp("^Cleaning up rendered CPI jobs" + stageFinishedPattern))

			By("setting the ssh password")
			stdout, _, exitCode, err := instanceSSH.RunCommand("echo ssh-succeeded")
			Expect(err).ToNot(HaveOccurred())
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("ssh-succeeded"))

			By("skipping the deploy if there are no changes")
			stdout = deploy("test-compiled-manifest.yml")

			Expect(stdout).To(ContainSubstring("No deployment, stemcell or release changes. Skipping deploy."))
			Expect(stdout).ToNot(ContainSubstring("Started installing CPI jobs"))
			Expect(stdout).ToNot(ContainSubstring("Started deploying"))
		})
	})

	Context("when the deploying with valid usage", func() {
		deploymentManifest := "test-manifest.yml"

		AfterEach(func() {
			flushLog(cmdEnv["BOSH_LOG_PATH"])

			// quietly delete the deployment
			_, _, exitCode, err := sshCmdRunner.RunCommand(quietCmdEnv, testEnv.Path("bosh"), "delete-env", "--tty", testEnv.Path("test-manifest.yml"))
			if exitCode != 0 || err != nil {
				// only flush the delete log if the delete failed
				flushLog(quietCmdEnv["BOSH_LOG_PATH"])
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(exitCode).To(Equal(0))
		})

		It("is able to deploy given many variances", func() {
			updateDeploymentManifest("./assets/manifest.yml")

			By("deploying sucessfully with the expected output")
			stdout := deploy(deploymentManifest)
			outputLines := strings.Split(stdout, "\n")
			numOutputLines := len(outputLines)

			doneIndex := 0
			stepIndex := -1
			nextStep := func() int { stepIndex++; return stepIndex }

			validatingSteps, doneIndex := findStage(outputLines, "validating", doneIndex)
			if !config.IsLocalCPIRelease() {
				Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Downloading release 'bosh-warden-cpi'"))
			}
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating release 'bosh-warden-cpi'" + stageFinishedPattern))
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating release 'dummy'" + stageFinishedPattern))
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating release 'dummyToo'" + stageFinishedPattern))
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating cpi release" + stageFinishedPattern))
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating deployment manifest" + stageFinishedPattern))
			if !config.IsLocalStemcell() {
				Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Downloading stemcell"))
			}
			Expect(validatingSteps[nextStep()]).To(MatchRegexp("^  Validating stemcell" + stageFinishedPattern))

			installingSteps, doneIndex := findStage(outputLines, "installing CPI", doneIndex+1)
			numInstallingSteps := len(installingSteps)
			for _, line := range installingSteps[:numInstallingSteps-3] {
				Expect(line).To(MatchRegexp("^  Compiling package '.*/.*'" + stageFinishedPattern))
			}
			Expect(installingSteps[numInstallingSteps-3]).To(MatchRegexp("^  Installing packages" + stageFinishedPattern))
			Expect(installingSteps[numInstallingSteps-2]).To(MatchRegexp("^  Rendering job templates" + stageFinishedPattern))
			Expect(installingSteps[numInstallingSteps-1]).To(MatchRegexp("^  Installing job 'warden_cpi'" + stageFinishedPattern))

			Expect(outputLines[doneIndex+2]).To(MatchRegexp("^Starting registry" + stageFinishedPattern))
			Expect(outputLines[doneIndex+3]).To(MatchRegexp("^Uploading stemcell '.*/.*'" + stageFinishedPattern))

			deployingSteps, doneIndex := findStage(outputLines, "deploying", doneIndex+1)
			numDeployingSteps := len(deployingSteps)
			Expect(deployingSteps[0]).To(MatchRegexp("^  Creating VM for instance 'dummy_job/0' from stemcell '.*'" + stageFinishedPattern))
			Expect(deployingSteps[1]).To(MatchRegexp("^  Waiting for the agent on VM '.*' to be ready" + stageFinishedPattern))
			Expect(deployingSteps[2]).To(MatchRegexp("^  Creating disk" + stageFinishedPattern))
			Expect(deployingSteps[3]).To(MatchRegexp("^  Attaching disk '.*' to VM '.*'" + stageFinishedPattern))
			Expect(deployingSteps[4]).To(MatchRegexp("^  Rendering job templates" + stageFinishedPattern))

			for _, line := range deployingSteps[5 : numDeployingSteps-3] {
				Expect(line).To(MatchRegexp("^  Compiling package '.*/.*'" + stageFinishedPattern))
			}

			Expect(deployingSteps[numDeployingSteps-3]).To(MatchRegexp("^  Updating instance 'dummy_job/0'" + stageFinishedPattern))
			Expect(deployingSteps[numDeployingSteps-2]).To(MatchRegexp("^  Waiting for instance 'dummy_job/0' to be running" + stageFinishedPattern))
			Expect(deployingSteps[numDeployingSteps-1]).To(MatchRegexp("^  Running the post-start scripts 'dummy_job/0'" + stageFinishedPattern))

			Expect(outputLines[numOutputLines-4]).To(MatchRegexp("^Cleaning up rendered CPI jobs" + stageFinishedPattern))

			By("setting the ssh password")
			stdout, _, exitCode, err := instanceSSH.RunCommand("echo ssh-succeeded")
			Expect(err).ToNot(HaveOccurred())
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("ssh-succeeded"))

			By("skipping the deploy if there are no changes")
			stdout = deploy(deploymentManifest)

			Expect(stdout).To(ContainSubstring("No deployment, stemcell or release changes. Skipping deploy."))
			Expect(stdout).ToNot(ContainSubstring("Started installing CPI jobs"))
			Expect(stdout).ToNot(ContainSubstring("Started deploying"))

			By("deleting the old VM if updating with a property change")
			updateDeploymentManifest("./assets/modified_manifest.yml")

			stdout = deploy(deploymentManifest)

			Expect(stdout).To(ContainSubstring("Deleting VM"))
			Expect(stdout).To(ContainSubstring("Stopping jobs on instance 'unknown/0'"))
			Expect(stdout).To(ContainSubstring("Unmounting disk"))

			Expect(stdout).ToNot(ContainSubstring("Creating disk"))

			By("migrating the disk if the disk size has changed")
			updateDeploymentManifest("./assets/modified_disk_manifest.yml")

			stdout = deploy(deploymentManifest)

			Expect(stdout).To(ContainSubstring("Deleting VM"))
			Expect(stdout).To(ContainSubstring("Stopping jobs on instance 'unknown/0'"))
			Expect(stdout).To(ContainSubstring("Unmounting disk"))

			Expect(stdout).To(ContainSubstring("Creating disk"))
			Expect(stdout).To(ContainSubstring("Migrating disk"))
			Expect(stdout).To(ContainSubstring("Deleting disk"))

			By("deleting the agent when deploying without a working agent")
			shutdownAgent()
			updateDeploymentManifest("./assets/modified_manifest.yml")

			stdout = deploy(deploymentManifest)

			Expect(stdout).To(MatchRegexp("Waiting for the agent on VM '.*'\\.\\.\\. Failed " + stageTimePattern))
			Expect(stdout).To(ContainSubstring("Deleting VM"))
			Expect(stdout).To(ContainSubstring("Creating VM for instance 'dummy_job/0' from stemcell"))
			Expect(stdout).To(ContainSubstring("Finished deploying"))

			By("deleting all VMs, disks, and stemcells")
			stdout = deleteDeployment()

			Expect(stdout).To(ContainSubstring("Stopping jobs on instance"))
			Expect(stdout).To(ContainSubstring("Deleting VM"))
			Expect(stdout).To(ContainSubstring("Deleting disk"))
			Expect(stdout).To(ContainSubstring("Deleting stemcell"))
			Expect(stdout).To(ContainSubstring("Finished deleting deployment"))
		})

		It("delete the vm even without a working agent", func() {
			updateDeploymentManifest("./assets/manifest.yml")

			deploy(deploymentManifest)
			shutdownAgent()

			stdout := deleteDeployment()

			Expect(stdout).To(MatchRegexp("Waiting for the agent on VM '.*'\\.\\.\\. Failed " + stageTimePattern))
			Expect(stdout).To(ContainSubstring("Deleting VM"))
			Expect(stdout).To(ContainSubstring("Deleting disk"))
			Expect(stdout).To(ContainSubstring("Deleting stemcell"))
			Expect(stdout).To(ContainSubstring("Finished deleting deployment"))
		})

		It("deploys & deletes without registry and ssh tunnel", func() {
			updateDeploymentManifest("./assets/manifest_without_registry.yml")

			stdout := deploy(deploymentManifest)
			Expect(stdout).To(ContainSubstring("Finished deploying"))

			stdout = deleteDeployment()
			Expect(stdout).To(ContainSubstring("Finished deleting deployment"))
		})

		It("prints multiple validation errors at the same time", func() {
			updateDeploymentManifest("./assets/invalid_manifest.yml")

			stdout := expectDeployToError()

			Expect(stdout).To(ContainSubstring("Validating deployment manifest... Failed"))
			Expect(stdout).To(ContainSubstring("Failed validating"))

			Expect(stdout).To(ContainSubstring("jobs[0].templates[0].release 'unknown-release' must refer to release in releases"))
		})
	})

	Context("when deploying with all network types", func() {
		AfterEach(func() {
			flushLog(cmdEnv["BOSH_LOG_PATH"])

			// quietly delete the deployment
			_, _, exitCode, err := sshCmdRunner.RunCommand(quietCmdEnv, testEnv.Path("bosh"), "delete-env", "--tty", testEnv.Path("test-manifest.yml"))
			if exitCode != 0 || err != nil {
				// only flush the delete log if the delete failed
				flushLog(quietCmdEnv["BOSH_LOG_PATH"])
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(exitCode).To(Equal(0))
		})

		It("is successful", func() {
			updateDeploymentManifest("./assets/manifest_with_all_network_types.yml")

			stdout := deploy("test-manifest.yml")
			Expect(stdout).To(ContainSubstring("Finished deploying"))
		})
	})

	It("exits early if there's no deployment state to delete", func() {
		updateDeploymentManifest("./assets/manifest.yml")
		stdout := deleteDeployment()

		Expect(stdout).To(ContainSubstring("No deployment state file found"))
	})
})
