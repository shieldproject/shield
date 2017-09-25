package windows_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/bosh-agent/agent/action"
	"github.com/cloudfoundry/bosh-agent/integration/windows/utils"
	boshfileutil "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	agentGUID = "123-456-789"
	agentID   = "agent." + agentGUID
	senderID  = "director.987-654-321"
)

func natsURI() string {
	natsURL := "nats://172.31.180.3:4222"
	vagrantProvider := os.Getenv("VAGRANT_PROVIDER")
	if vagrantProvider == "aws" {
		natsURL = fmt.Sprintf("nats://%s:4222", os.Getenv("NATS_ELASTIC_IP"))
	}
	return natsURL
}

func blobstoreURI() string {
	blobstoreURI := "http://172.31.180.3:25250"
	vagrantProvider := os.Getenv("VAGRANT_PROVIDER")
	if vagrantProvider == "aws" {
		blobstoreURI = fmt.Sprintf("http://%s:25250", os.Getenv("NATS_ELASTIC_IP"))
	}
	return blobstoreURI
}

var _ = Describe("An Agent running on Windows", func() {
	var (
		fs              boshsys.FileSystem
		natsClient      *NatsClient
		blobstoreClient utils.BlobClient
	)

	BeforeEach(func() {
		message := fmt.Sprintf(`{"method":"ping","arguments":[],"reply_to":"%s"}`, senderID)

		blobstoreClient = utils.NewBlobstore(blobstoreURI())

		logger := boshlog.NewLogger(boshlog.LevelNone)
		cmdRunner := boshsys.NewExecCmdRunner(logger)
		fs = boshsys.NewOsFileSystem(logger)
		compressor := boshfileutil.NewTarballCompressor(cmdRunner, fs)

		natsClient = NewNatsClient(compressor, blobstoreClient)
		err := natsClient.Setup()
		Expect(err).NotTo(HaveOccurred())

		testPing := func() (string, error) {
			response, err := natsClient.SendRawMessage(message)
			return string(response), err
		}

		Eventually(testPing, 30*time.Second, 1*time.Second).Should(Equal(`{"value":"pong"}`))
	})

	AfterEach(func() {
		natsClient.Cleanup()
	})

	It("responds to 'get_state' message over NATS", func() {
		getStateSpecAgentID := func() string {
			message := fmt.Sprintf(`{"method":"get_state","arguments":[],"reply_to":"%s"}`, senderID)
			rawResponse, err := natsClient.SendRawMessage(message)
			Expect(err).NotTo(HaveOccurred())

			response := map[string]action.GetStateV1ApplySpec{}
			err = json.Unmarshal(rawResponse, &response)
			Expect(err).NotTo(HaveOccurred())

			return response["value"].AgentID
		}

		Eventually(getStateSpecAgentID, 30*time.Second, 1*time.Second).Should(Equal(agentGUID))
	})

	It("can run a run_errand action", func() {
		natsClient.PrepareJob("say-hello")

		runErrandResponse, err := natsClient.RunErrand()
		Expect(err).NotTo(HaveOccurred())

		runErrandCheck := natsClient.CheckErrandResultStatus(runErrandResponse["value"]["agent_task_id"])
		Eventually(runErrandCheck, 30*time.Second, 1*time.Second).Should(Equal(action.ErrandResult{
			Stdout:     "hello world\r\n",
			ExitStatus: 0,
		}))
	})

	It("can start a job", func() {
		natsClient.PrepareJob("say-hello")

		runStartResponse, err := natsClient.RunStart()
		Expect(err).NotTo(HaveOccurred())
		Expect(runStartResponse["value"]).To(Equal("started"))

		agentState := natsClient.GetState()
		Expect(agentState.JobState).To(Equal("running"))
	})

	It("can run a drain script", func() {
		natsClient.PrepareJob("say-hello")

		err := natsClient.RunDrain()
		Expect(err).NotTo(HaveOccurred())

		logsDir, err := fs.TempDir("windows-agent-drain-test")
		Expect(err).NotTo(HaveOccurred())
		defer fs.RemoveAll(logsDir)

		natsClient.FetchLogs(logsDir)

		drainLogContents, err := fs.ReadFileString(filepath.Join(logsDir, "say-hello", "drain.log"))
		Expect(err).NotTo(HaveOccurred())

		Expect(drainLogContents).To(ContainSubstring("Hello from drain"))
	})

	It("can unmonitor the job during drain script", func() {
		natsClient.PrepareJob("unmonitor-hello")

		runStartResponse, err := natsClient.RunStart()
		Expect(err).NotTo(HaveOccurred())
		Expect(runStartResponse["value"]).To(Equal("started"))

		agentState := natsClient.GetState()
		Expect(agentState.JobState).To(Equal("running"))

		err = natsClient.RunDrain()
		Expect(err).NotTo(HaveOccurred())

		logsDir, err := fs.TempDir("windows-agent-drain-test")
		Expect(err).NotTo(HaveOccurred())
		defer fs.RemoveAll(logsDir)

		natsClient.FetchLogs(logsDir)

		drainLogContents, err := fs.ReadFileString(filepath.Join(logsDir, "unmonitor-hello", "drain.log"))
		Expect(err).NotTo(HaveOccurred())

		Expect(drainLogContents).To(ContainSubstring("success"))
	})

	It("stops alerting failing jobs when job is stopped", func() {
		natsClient.PrepareJob("crashes-on-start")
		runStartResponse, err := natsClient.RunStart()
		Expect(err).NotTo(HaveOccurred())
		Expect(runStartResponse["value"]).To(Equal("started"))

		Eventually(func() string { return natsClient.GetState().JobState }, 30*time.Second, 1*time.Second).Should(Equal("failing"))

		Eventually(func() (string, error) {
			alert, err := natsClient.GetNextAlert(10 * time.Second)
			if err != nil {
				return "", err
			}
			return alert.Title, nil
		}).Should(MatchRegexp(`crash-service \(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\) - pid failed - Start`))

		err = natsClient.RunStop()
		Expect(err).NotTo(HaveOccurred())

		_, err = natsClient.GetNextAlert(10 * time.Second)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("nats: timeout"))
	})

	It("can run arbitrary user scripts", func() {
		natsClient.PrepareJob("say-hello")

		err := natsClient.RunScript("pre-start")
		Expect(err).NotTo(HaveOccurred())

		logsDir, err := fs.TempDir("windows-agent-prestart-test")
		Expect(err).NotTo(HaveOccurred())
		defer fs.RemoveAll(logsDir)

		natsClient.FetchLogs(logsDir)

		prestartStdoutContents, err := fs.ReadFileString(filepath.Join(logsDir, "say-hello", "pre-start.stdout.log"))
		Expect(err).NotTo(HaveOccurred())

		Expect(prestartStdoutContents).To(ContainSubstring("Hello from stdout"))

		prestartStderrContents, err := fs.ReadFileString(filepath.Join(logsDir, "say-hello", "pre-start.stderr.log"))
		Expect(err).NotTo(HaveOccurred())

		Expect(prestartStderrContents).To(ContainSubstring("Hello from stderr"))
	})

	It("can compile packages", func() {
		const (
			blobName     = "blob.tar"
			fileName     = "output.txt"
			fileContents = "i'm a compiled package!"
		)
		result, err := natsClient.CompilePackage("simple-package")
		Expect(err).NotTo(HaveOccurred())

		tempDir, err := fs.TempDir("windows-agent-compile-test")
		Expect(err).NotTo(HaveOccurred())

		path := filepath.Join(tempDir, blobName)
		Expect(blobstoreClient.Get(result.BlobstoreID, path)).To(Succeed())

		tarPath, err := ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tarPath)

		err = exec.Command("tar", "xf", path, "-C", tarPath).Run()
		Expect(err).NotTo(HaveOccurred())

		out, err := ioutil.ReadFile(filepath.Join(tarPath, fileName))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(out)).To(ContainSubstring(fileContents))
	})

	It("Includes the default IP in the 'get_state' response", func() {
		getNetworkProperty := func(key string) func() string {
			return func() string {
				message := fmt.Sprintf(`{"method":"get_state","arguments":["full"],"reply_to":"%s"}`, senderID)
				rawResponse, err := natsClient.SendRawMessage(message)
				Expect(err).NotTo(HaveOccurred())

				response := map[string]action.GetStateV1ApplySpec{}
				err = json.Unmarshal(rawResponse, &response)
				Expect(err).NotTo(HaveOccurred())

				for _, spec := range response["value"].NetworkSpecs {
					field, ok := spec.Fields[key]
					if !ok {
						return ""
					}
					if val, ok := field.(string); ok {
						return val
					}
				}
				return ""
			}
		}

		Eventually(getNetworkProperty("ip"), 30*time.Second, 1*time.Second).ShouldNot(BeEmpty())
		Eventually(getNetworkProperty("gateway"), 30*time.Second, 1*time.Second).ShouldNot(BeEmpty())
		Eventually(getNetworkProperty("netmask"), 30*time.Second, 1*time.Second).ShouldNot(BeEmpty())
	})
})
