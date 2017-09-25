package http

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-agent/agentclient/applyspec"
	"github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

type agentClient struct {
	agentRequest        agentRequest
	getTaskDelay        time.Duration
	toleratedErrorCount int
	logger              boshlog.Logger
	logTag              string
}

func NewAgentClient(
	endpoint string,
	directorID string,
	getTaskDelay time.Duration,
	toleratedErrorCount int,
	httpClient httpclient.HTTPClient,
	logger boshlog.Logger,
) agentclient.AgentClient {
	// if this were NATS, we would need the agentID, but since it's http, the endpoint is unique to the agent
	agentEndpoint := fmt.Sprintf("%s/agent", endpoint)
	agentRequest := agentRequest{
		directorID: directorID,
		endpoint:   agentEndpoint,
		httpClient: httpClient,
	}
	return &agentClient{
		agentRequest:        agentRequest,
		getTaskDelay:        getTaskDelay,
		toleratedErrorCount: toleratedErrorCount,
		logger:              logger,
		logTag:              "httpAgentClient",
	}
}

func (c *agentClient) Ping() (string, error) {
	var response SimpleTaskResponse
	err := c.agentRequest.Send("ping", []interface{}{}, &response)
	if err != nil {
		return "", bosherr.WrapError(err, "Sending ping to the agent")
	}

	return response.Value, nil
}

func (c *agentClient) Stop() error {
	_, err := c.sendAsyncTaskMessage("stop", []interface{}{})
	return err
}

func (c *agentClient) Apply(spec applyspec.ApplySpec) error {
	_, err := c.sendAsyncTaskMessage("apply", []interface{}{spec})
	return err
}

func (c *agentClient) Start() error {
	var response SimpleTaskResponse
	err := c.agentRequest.Send("start", []interface{}{}, &response)
	if err != nil {
		return bosherr.WrapError(err, "Starting agent services")
	}

	if response.Value != "started" {
		return bosherr.Errorf("Failed to start agent services with response: '%s'", response)
	}

	return nil
}

func (c *agentClient) GetState() (agentclient.AgentState, error) {
	var response StateResponse

	getStateRetryable := boshretry.NewRetryable(func() (bool, error) {
		err := c.agentRequest.Send("get_state", []interface{}{}, &response)
		if err != nil {
			return true, bosherr.WrapError(err, "Sending get_state to the agent")
		}
		return false, nil
	})

	attemptRetryStrategy := boshretry.NewAttemptRetryStrategy(c.toleratedErrorCount+1, c.getTaskDelay, getStateRetryable, c.logger)
	err := attemptRetryStrategy.Try()
	if err != nil {
		return agentclient.AgentState{}, bosherr.WrapError(err, "Sending get_state to the agent")
	}

	agentState := agentclient.AgentState{
		JobState:     response.Value.JobState,
		NetworkSpecs: response.Value.NetworkSpecs,
	}

	return agentState, err
}

func (c *agentClient) ListDisk() ([]string, error) {
	var response ListResponse
	err := c.agentRequest.Send("list_disk", []interface{}{}, &response)
	if err != nil {
		return []string{}, bosherr.WrapError(err, "Sending 'list_disk' to the agent")
	}

	return response.Value, nil
}

func (c *agentClient) MountDisk(diskCID string) error {
	_, err := c.sendAsyncTaskMessage("mount_disk", []interface{}{diskCID})
	return err
}

func (c *agentClient) UnmountDisk(diskCID string) error {
	_, err := c.sendAsyncTaskMessage("unmount_disk", []interface{}{diskCID})
	return err
}

func (c *agentClient) MigrateDisk() error {
	_, err := c.sendAsyncTaskMessage("migrate_disk", []interface{}{})
	return err
}

func (c *agentClient) UpdateSettings(settings settings.Settings) error {
	_, err := c.sendAsyncTaskMessage("update_settings", []interface{}{settings})
	return err
}

func (c *agentClient) RunScript(scriptName string, options map[string]interface{}) error {
	_, err := c.sendAsyncTaskMessage("run_script", []interface{}{scriptName, options})

	if err != nil && strings.Contains(err.Error(), "unknown message") {
		// ignore 'unknown message' errors for backwards compatibility with older stemcells
		c.logger.Warn(c.logTag, "Ignoring run_script 'unknown message' error from the agent: %s. Received while trying to run: %s", err.Error(), scriptName)
		return nil
	}

	return err
}

func (c *agentClient) CompilePackage(packageSource agentclient.BlobRef, compiledPackageDependencies []agentclient.BlobRef) (compiledPackageRef agentclient.BlobRef, err error) {
	dependencies := make(map[string]BlobRef, len(compiledPackageDependencies))
	for _, dependency := range compiledPackageDependencies {
		dependencies[dependency.Name] = BlobRef{
			Name:        dependency.Name,
			Version:     dependency.Version,
			SHA1:        dependency.SHA1,
			BlobstoreID: dependency.BlobstoreID,
		}
	}

	args := []interface{}{
		packageSource.BlobstoreID,
		packageSource.SHA1,
		packageSource.Name,
		packageSource.Version,
		dependencies,
	}

	responseValue, err := c.sendAsyncTaskMessage("compile_package", args)
	if err != nil {
		return agentclient.BlobRef{}, bosherr.WrapError(err, "Sending 'compile_package' to the agent")
	}

	result, ok := responseValue["result"].(map[string]interface{})
	if !ok {
		return agentclient.BlobRef{}, bosherr.Errorf("Unable to parse 'compile_package' response from the agent: %#v", responseValue)
	}

	sha1, ok := result["sha1"].(string)
	if !ok {
		return agentclient.BlobRef{}, bosherr.Errorf("Unable to parse 'compile_package' response from the agent: %#v", responseValue)
	}

	blobstoreID, ok := result["blobstore_id"].(string)
	if !ok {
		return agentclient.BlobRef{}, bosherr.Errorf("Unable to parse 'compile_package' response from the agent: %#v", responseValue)
	}

	compiledPackageRef = agentclient.BlobRef{
		Name:        packageSource.Name,
		Version:     packageSource.Version,
		SHA1:        sha1,
		BlobstoreID: blobstoreID,
	}

	return compiledPackageRef, nil
}

func (c *agentClient) DeleteARPEntries(ips []string) error {
	return c.agentRequest.Send("delete_arp_entries", []interface{}{map[string][]string{"ips": ips}}, &TaskResponse{})
}

func (c *agentClient) SyncDNS(blobID, sha1 string) (string, error) {
	var response SyncDNSResponse
	err := c.agentRequest.Send("sync_dns", []interface{}{blobID, sha1}, &response)
	if err != nil {
		return "", bosherr.WrapError(err, "Sending 'sync_dns' to the agent")
	}

	return response.Value, nil
}

func (c *agentClient) sendAsyncTaskMessage(method string, arguments []interface{}) (value map[string]interface{}, err error) {
	var response TaskResponse
	err = c.agentRequest.Send(method, arguments, &response)
	if err != nil {
		return value, bosherr.WrapErrorf(err, "Sending '%s' to the agent", method)
	}

	agentTaskID, err := response.TaskID()
	if err != nil {
		return value, bosherr.WrapError(err, "Getting agent task id")
	}

	sendErrors := 0
	getTaskRetryable := boshretry.NewRetryable(func() (bool, error) {
		var response TaskResponse
		err = c.agentRequest.Send("get_task", []interface{}{agentTaskID}, &response)
		if err != nil {
			sendErrors++
			shouldRetry := sendErrors <= c.toleratedErrorCount
			err = bosherr.WrapError(err, "Sending 'get_task' to the agent")
			msg := fmt.Sprintf("Error occured sending get_task. Error retry %d of %d", sendErrors, c.toleratedErrorCount)
			c.logger.Debug(c.logTag, msg, err)
			return shouldRetry, err
		}
		sendErrors = 0

		c.logger.Debug(c.logTag, "get_task response value: %#v", response.Value)

		taskState, err := response.TaskState()
		if err != nil {
			return false, bosherr.WrapError(err, "Getting task state")
		}

		if taskState != "running" {
			var ok bool
			value, ok = response.Value.(map[string]interface{})
			if !ok {
				c.logger.Warn(c.logTag, "Unable to parse get_task response value: %#v", response.Value)
			}
			return true, nil
		}

		return true, bosherr.Errorf("Task %s is still running", method)
	})

	getTaskRetryStrategy := boshretry.NewUnlimitedRetryStrategy(c.getTaskDelay, getTaskRetryable, c.logger)
	// cannot call getTaskRetryStrategy.Try in the return statement due to gccgo
	// execution order issues: https://code.google.com/p/go/issues/detail?id=8698&thanks=8698&ts=1410376474
	err = getTaskRetryStrategy.Try()
	return value, err
}
