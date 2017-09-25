package windows_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"text/template"
	"time"

	"github.com/cloudfoundry/bosh-agent/agentclient"

	"github.com/cloudfoundry/bosh-agent/agent/action"
	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
	"github.com/cloudfoundry/bosh-agent/agentclient/http"
	"github.com/cloudfoundry/bosh-agent/integration/windows/utils"
	boshfileutil "github.com/cloudfoundry/bosh-utils/fileutil"
	"github.com/nats-io/nats"

	. "github.com/onsi/gomega"
)

type PrepareTemplateConfig struct {
	JobName                             string
	TemplateBlobstoreID                 string
	RenderedTemplatesArchiveBlobstoreID string
	RenderedTemplatesArchiveSHA1        string
	ReplyTo                             string
}

const (
	prepareTemplate = `{
    "arguments": [
        {
            "deployment": "test",
            "job": {
                "name": "test-job",
                "template": "test-job",
                "templates": [
                    {
                        "name": "{{ .JobName }}",
						"blobstore_id": "{{ .TemplateBlobstoreID }}",
						"sha1": "eb9bebdb1f11494b27440ec6ccbefba00e713cd9"
                    }
                ]
            },
            "packages": {},
            "rendered_templates_archive": {
                "blobstore_id": "{{ .RenderedTemplatesArchiveBlobstoreID }}",
                "sha1": "{{ .RenderedTemplatesArchiveSHA1 }}"
            }
        }
    ],
    "method": "prepare",
    "reply_to": "{{ .ReplyTo }}"
}`
	errandTemplate = `
	{"protocol":2,"method":"run_errand","arguments":[],"reply_to":"%s"}
	`
	applyTemplate = `
{
    "arguments": [
        {
            "configuration_hash": "foo",
            "deployment": "hello-world-windows-deployment",
            "id": "62236318-6632-4318-94c7-c3dd6e8e5698",
            "index": 0,
            "job": {
                "blobstore_id": "{{ .TemplateBlobstoreID }}",
                "name": "{{ .JobName }}",
                "sha1": "eb6e6c8bd1b1bc3dd91c741ec5c628b61a4d8f1d",
                "template": "say-hello",
                "templates": [
                    {
                        "blobstore_id": "{{ .TemplateBlobstoreID }}",
                        "name": "{{ .JobName }}",
                        "sha1": "eb6e6c8bd1b1bc3dd91c741ec5c628b61a4d8f1d",
                        "version": "8fe0a4982b28ffe4e59d7c1e573c4f30a526770d"
                    }
                ],
                "version": "8fe0a4982b28ffe4e59d7c1e573c4f30a526770d"
            },
            "networks": {
            	"diego1": {}
            },
			"rendered_templates_archive": {
					"blobstore_id": "{{ .RenderedTemplatesArchiveBlobstoreID }}",
					"sha1": "{{ .RenderedTemplatesArchiveSHA1 }}"
			}
        }
    ],
    "method": "apply",
    "protocol": 2,
    "reply_to": "{{ .ReplyTo }}"
}
	`

	fetchLogsTemplate = `
{
    "arguments": [
        "job",
        null
    ],
    "method": "fetch_logs",
    "protocol": 2,
    "reply_to": "%s"
}
	`

	startTemplate = `
{
	"arguments":[],
	"method":"start",
	"protocol":2,
	"reply_to":"%s"
}
	`
	stopTemplate = `
{
	"protocol":2,
	"method":"stop",
	"arguments":[],
	"reply_to":"%s"
}
	`

	drainTemplate = `
{
	"protocol":2,
	"method":"drain",
	"arguments":[
	  "update",
	  {}
	],
	"reply_to":"%s"
}
	`
	runScriptTemplate = `
{
	"protocol":2,
	"method":"run_script",
	"arguments":[
	  "%s",
	  {}
	],
	"reply_to":"%s"
}
	`
)

type NatCommand struct {
	Protocol  int           `json:"protocol"`
	Method    string        `json:"method"`
	Arguments []interface{} `json:"arguments"`
	ReplyTo   string        `json:"reply_to"`
}

func (n NatCommand) Marshal() (string, error) {
	n.Protocol = 2
	b, err := json.Marshal(n)
	return string(b), err
}

type NatsClient struct {
	nc       *nats.Conn
	sub      *nats.Subscription
	alertSub *nats.Subscription

	compressor      boshfileutil.Compressor
	blobstoreClient utils.BlobClient
}

func NewNatsClient(
	compressor boshfileutil.Compressor,
	blobstoreClient utils.BlobClient,
) *NatsClient {
	return &NatsClient{
		compressor:      compressor,
		blobstoreClient: blobstoreClient,
	}
}

func (n *NatsClient) Setup() error {
	var err error
	n.nc, err = nats.Connect(natsURI())
	if err != nil {
		return err
	}

	n.sub, err = n.nc.SubscribeSync(senderID)
	n.alertSub, err = n.nc.SubscribeSync("hm.agent.alert." + agentGUID)
	return err
}

func (n *NatsClient) Cleanup() {
	err := n.RunStop()
	Expect(err).NotTo(HaveOccurred())

	n.nc.Close()
}

func (n *NatsClient) PrepareJob(jobName string) {
	templateID, sha1, err := n.uploadJob(jobName)
	Expect(err).NotTo(HaveOccurred())

	prepareTemplateConfig := PrepareTemplateConfig{
		JobName:                             jobName,
		TemplateBlobstoreID:                 templateID,
		RenderedTemplatesArchiveBlobstoreID: templateID,
		RenderedTemplatesArchiveSHA1:        sha1,
		ReplyTo: senderID,
	}

	buffer := bytes.NewBuffer([]byte{})
	t := template.Must(template.New("prepare").Parse(prepareTemplate))
	err = t.Execute(buffer, prepareTemplateConfig)
	Expect(err).NotTo(HaveOccurred())
	prepareResponse, err := n.SendMessage(buffer.String())
	Expect(err).NotTo(HaveOccurred())

	_, err = n.WaitForTask(prepareResponse["value"]["agent_task_id"], -1)
	Expect(err).ToNot(HaveOccurred())

	buffer.Reset()
	t = template.Must(template.New("apply").Parse(applyTemplate))
	err = t.Execute(buffer, prepareTemplateConfig)
	Expect(err).NotTo(HaveOccurred())
	applyResponse, err := n.SendMessage(buffer.String())
	Expect(err).NotTo(HaveOccurred())

	_, err = n.WaitForTask(applyResponse["value"]["agent_task_id"], -1)
	Expect(err).ToNot(HaveOccurred())
}

type CompileTemplate struct {
	BlobstoreID  string
	SHA1         string
	Name         string
	Version      string
	Dependencies map[string]agentclient.BlobRef
}

func (c CompileTemplate) Arguments() []interface{} {
	return []interface{}{
		c.BlobstoreID,
		c.SHA1,
		c.Name,
		c.Version,
		c.Dependencies,
	}
}

func (n *NatsClient) CompilePackage(packageName string) (*agentclient.BlobRef, error) {
	blobID, err := n.uploadPackage(packageName)
	if err != nil {
		return nil, err
	}

	template := CompileTemplate{
		BlobstoreID:  blobID,
		SHA1:         "902d0e7690d45738681f9d0c1ecee19e40dec507",
		Name:         packageName,
		Version:      "1.2.3",
		Dependencies: nil,
	}

	command := NatCommand{
		Method:    "compile_package",
		ReplyTo:   senderID,
		Arguments: template.Arguments(),
	}
	msg, err := command.Marshal()
	if err != nil {
		return nil, err
	}
	pkgResponse, err := n.SendMessage(msg)
	if err != nil {
		return nil, err
	}

	taskID := pkgResponse["value"]["agent_task_id"]
	response, err := n.WaitForTask(taskID, -1)
	if err != nil {
		return nil, err
	}

	value, ok := response.Value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(`CompilePackage invalid response value: %#v`, value)
	}
	result, ok := value["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(`CompilePackage invalid 'result' field: %#v`, value)
	}
	blobstoreID, ok := result["blobstore_id"].(string)
	if !ok {
		return nil, fmt.Errorf(`CompilePackage missing 'blobstore_id' field: %#v`, result)
	}
	sha1, ok := result["sha1"].(string)
	if !ok {
		return nil, fmt.Errorf(`CompilePackage missing 'sha1' field: %#v`, result)
	}
	compiledPackageRef := agentclient.BlobRef{
		Name:        template.Name,
		Version:     template.Version,
		SHA1:        sha1,
		BlobstoreID: blobstoreID,
	}
	return &compiledPackageRef, nil
}

func (n *NatsClient) WaitForTask(id string, timeout time.Duration) (*http.TaskResponse, error) {
	if timeout <= 0 {
		timeout = time.Second * 5
	}
	const finished = "finished" // TaskResponse final state
	start := time.Now()
	for time.Since(start) < timeout {
		response, err := n.GetTask(id)
		if err != nil {
			return nil, err
		}
		state, err := response.TaskState()
		if err != nil {
			return nil, err
		}
		if state == finished {
			return response, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("WaitForTask: timed out after: %s", timeout)
}

func (n *NatsClient) GetTask(id string) (*http.TaskResponse, error) {
	var b []byte
	const msgFmt = `{"method": "get_task", "arguments": ["%s"], "reply_to": "%s"}`
	b, err := n.SendRawMessage(fmt.Sprintf(msgFmt, id, senderID))
	if err != nil {
		return nil, err
	}

	var result http.TaskResponse
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	if err := result.ServerError(); err != nil {
		return nil, err
	}
	return &result, nil
}

func (n *NatsClient) getTask(taskID string) ([]byte, error) {
	message := fmt.Sprintf(`{"method": "get_task", "arguments": ["%s"], "reply_to": "%s"}`, taskID, senderID)
	return n.SendRawMessage(message)
}

func (n *NatsClient) RunDrain() error {
	message := fmt.Sprintf(drainTemplate, senderID)
	drainResponse, err := n.SendMessage(message)
	if err != nil {
		return err
	}

	taskResponse, err := n.WaitForTask(drainResponse["value"]["agent_task_id"], time.Second*30)
	magicNumber, ok := taskResponse.Value.(float64)
	if !ok {
		return fmt.Errorf("RunDrain got invalid taskResponse %s", reflect.TypeOf(taskResponse.Value))
	}
	Expect(int(magicNumber)).To(Equal(0))

	return nil
}

func (n *NatsClient) RunScript(scriptName string) error {
	message := fmt.Sprintf(runScriptTemplate, scriptName, senderID)
	response, err := n.SendMessage(message)
	if err != nil {
		return err
	}

	_, err = n.WaitForTask(response["value"]["agent_task_id"], 30*time.Second)

	return err
}

func (n *NatsClient) RunStart() (map[string]string, error) {
	message := fmt.Sprintf(startTemplate, senderID)
	rawResponse, err := n.SendRawMessageWithTimeout(message, time.Minute)
	if err != nil {
		return map[string]string{}, err
	}

	response := map[string]string{}
	err = json.Unmarshal(rawResponse, &response)
	return response, err
}

func (n *NatsClient) GetState() action.GetStateV1ApplySpec {
	message := fmt.Sprintf(`{"method":"get_state","arguments":[],"reply_to":"%s"}`, senderID)
	rawResponse, err := n.SendRawMessage(message)
	Expect(err).NotTo(HaveOccurred())

	getStateResponse := map[string]action.GetStateV1ApplySpec{}
	err = json.Unmarshal(rawResponse, &getStateResponse)
	Expect(err).NotTo(HaveOccurred())

	return getStateResponse["value"]
}

func (n *NatsClient) RunStop() error {
	message := fmt.Sprintf(stopTemplate, senderID)
	rawResponse, err := n.SendRawMessage(message)
	if err != nil {
		return err
	}

	response := map[string]map[string]string{}

	err = json.Unmarshal(rawResponse, &response)
	if err != nil {
		return err
	}

	_, err = n.WaitForTask(response["value"]["agent_task_id"], -1)
	return err
}

func (n *NatsClient) RunErrand() (map[string]map[string]string, error) {
	message := fmt.Sprintf(errandTemplate, senderID)
	return n.SendMessage(message)
}

func (n *NatsClient) FetchLogs(destinationDir string) {
	message := fmt.Sprintf(fetchLogsTemplate, senderID)
	fetchLogsResponse, err := n.SendMessage(message)
	var fetchLogsResult map[string]string

	fetchLogsCheckFunc := func() (map[string]string, error) {
		var err error
		var taskResult map[string]map[string]string

		valueResponse, err := n.getTask(fetchLogsResponse["value"]["agent_task_id"])
		if err != nil {
			return map[string]string{}, err
		}

		err = json.Unmarshal(valueResponse, &taskResult)
		if err != nil {
			return map[string]string{}, err
		}

		fetchLogsResult = taskResult["value"]

		return fetchLogsResult, nil
	}

	Eventually(fetchLogsCheckFunc, 30*time.Second, 1*time.Second).Should(HaveKey("blobstore_id"))

	fetchedLogFile := filepath.Join(destinationDir, "log.tgz")
	err = n.blobstoreClient.Get(fetchLogsResult["blobstore_id"], fetchedLogFile)
	Expect(err).NotTo(HaveOccurred())

	err = n.compressor.DecompressFileToDir(fetchedLogFile, destinationDir, boshfileutil.CompressorOptions{})
	Expect(err).NotTo(HaveOccurred())
}

func (n *NatsClient) CheckErrandResultStatus(taskID string) func() (action.ErrandResult, error) {
	return func() (action.ErrandResult, error) {
		var result map[string]action.ErrandResult
		valueResponse, err := n.getTask(taskID)
		if err != nil {
			return action.ErrandResult{}, err
		}

		err = json.Unmarshal(valueResponse, &result)
		if err != nil {
			return action.ErrandResult{}, err
		}

		return result["value"], nil
	}
}

func (n *NatsClient) SendRawMessageWithTimeout(message string, timeout time.Duration) ([]byte, error) {
	err := n.nc.Publish(agentID, []byte(message))
	if err != nil {
		return nil, err
	}

	raw, err := n.sub.NextMsg(timeout)
	if err != nil {
		return nil, err
	}

	return raw.Data, nil
}

func (n *NatsClient) SendRawMessage(message string) ([]byte, error) {
	return n.SendRawMessageWithTimeout(message, 5*time.Second)
}

func (n *NatsClient) SendMessage(message string) (map[string]map[string]string, error) {
	rawMessage, err := n.SendRawMessage(message)
	if err != nil {
		return nil, err
	}

	response := map[string]map[string]string{}
	err = json.Unmarshal(rawMessage, &response)
	return response, err
}

func (n *NatsClient) GetNextAlert(timeout time.Duration) (*boshalert.Alert, error) {
	raw, err := n.alertSub.NextMsg(timeout)
	if err != nil {
		return nil, err
	}
	var alert boshalert.Alert
	err = json.Unmarshal(raw.Data, &alert)
	return &alert, err
}

func (n *NatsClient) uploadJob(jobName string) (templateID, renderedTemplateSha string, err error) {
	var dirname string
	dirname, err = ioutil.TempDir("", "templates")
	if err != nil {
		return
	}
	defer os.RemoveAll(dirname)

	tarfile := filepath.Join(dirname, jobName+".tar")
	chdir := "fixtures/templates"
	dir := filepath.Join(chdir, jobName)

	renderedTemplateSha, err = utils.TarDirectory(dir, chdir, tarfile)
	if err != nil {
		return
	}
	templateID, err = n.blobstoreClient.Create(tarfile)
	if err != nil {
		return
	}
	return
}

func (n *NatsClient) uploadPackage(packageName string) (blobID string, err error) {
	var dirname string
	dirname, err = ioutil.TempDir("", "templates")
	if err != nil {
		return
	}
	defer os.RemoveAll(dirname)

	tarfile := filepath.Join(dirname, packageName+".tar")
	dir := filepath.Join("fixtures/templates", packageName)
	_, err = utils.TarDirectory(dir, dir, tarfile)
	if err != nil {
		return
	}

	return n.blobstoreClient.Create(tarfile)
}
