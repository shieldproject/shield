package http

import (
	"encoding/json"

	"runtime/debug"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type Response interface {
	Unmarshal([]byte) error
	ServerError() error
}

type exception struct {
	Message string
}

type SimpleTaskResponse struct {
	Value     string
	Exception *exception
}

func (r *SimpleTaskResponse) ServerError() error {
	if r.Exception != nil {
		return bosherr.Errorf("Agent responded with error: %s", r.Exception.Message)
	}
	return nil
}

func (r *SimpleTaskResponse) Unmarshal(message []byte) error {
	return json.Unmarshal(message, r)
}

type SyncDNSResponse struct {
	Value     string
	Exception *exception
}

func (r *SyncDNSResponse) ServerError() error {
	if r.Exception != nil {
		return bosherr.Errorf("Agent responded with error: %s", r.Exception.Message)
	}
	return nil
}

func (r *SyncDNSResponse) Unmarshal(message []byte) error {
	return json.Unmarshal(message, r)
}

type ListResponse struct {
	Value     []string
	Exception *exception
}

func (r *ListResponse) ServerError() error {
	if r.Exception != nil {
		return bosherr.Errorf("Agent responded with error: %s", r.Exception.Message)
	}
	return nil
}

func (r *ListResponse) Unmarshal(message []byte) error {
	return json.Unmarshal(message, r)
}

type BlobRef struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	SHA1        string `json:"sha1"`
	BlobstoreID string `json:"blobstore_id"`
}

type BlobResponse struct {
	Value     map[string]string
	Exception *exception
}

func (r *BlobResponse) ServerError() error {
	if r.Exception != nil {
		return bosherr.Errorf("Agent responded with error: %s", r.Exception.Message)
	}
	return nil
}

func (r *BlobResponse) Unmarshal(message []byte) error {
	return json.Unmarshal(message, r)
}

type StateResponse struct {
	Value     AgentState
	Exception *exception
}

func (r *StateResponse) ServerError() error {
	if r.Exception != nil {
		return bosherr.Errorf("Agent responded with error: %s", r.Exception.Message)
	}
	return nil
}

func (r *StateResponse) Unmarshal(message []byte) error {
	return json.Unmarshal(message, r)
}

type AgentState struct {
	JobState     string                             `json:"job_state"`
	NetworkSpecs map[string]agentclient.NetworkSpec `json:"networks"`
}

type TaskResponse struct {
	Value     interface{}
	Exception *exception
}

func (r *TaskResponse) ServerError() error {
	if r.Exception != nil {
		return bosherr.Errorf("Agent responded with error: %s", r.Exception.Message)
	}
	return nil
}

func (r *TaskResponse) Unmarshal(message []byte) error {
	return json.Unmarshal(message, r)
}

func (r *TaskResponse) TaskID() (string, error) {
	complexResponse, ok := r.Value.(map[string]interface{})
	if !ok {
		return "", bosherr.Errorf("Failed to convert agent response to map %#v\n%s", r.Value, debug.Stack())
	}

	agentTaskID, ok := complexResponse["agent_task_id"]
	if !ok {
		return "", bosherr.Errorf("Failed to parse task id from agent response %#v", r.Value)
	}

	return agentTaskID.(string), nil
}

// TaskState returns the state of the task reported by agent.
//
// Agent response to get_task can be in different format based on task state.
// If task state is running agent responds
// with value as {value: { agent_task_id: "task-id", state: "running" }}
// Otherwise the value is a string like "stopped".
func (r *TaskResponse) TaskState() (string, error) {
	complexResponse, ok := r.Value.(map[string]interface{})
	if ok {
		_, ok := complexResponse["agent_task_id"]
		if ok {
			taskState, ok := complexResponse["state"]
			if ok {
				return taskState.(string), nil
			}

			return "", bosherr.Errorf("Failed to parse task state from agent response %#v", r.Value)
		}
	}

	return "finished", nil
}
