package drain

import (
	"encoding/json"

	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

// presentedJobState exposes only limited subset of apply spec to drain scripts.
// New fields should only be exposed once concrete use cases develop.
type presentedJobState struct {
	applySpec *boshas.V1ApplySpec
}

type presentedJobStateEnvelope struct {
	// PersistentDisk is exposed to determine if data needs to be migrated off
	// when disk is completely removed or shrinks in size
	PersistentDisk int `json:"persistent_disk"`
}

func newPresentedJobState(applySpec *boshas.V1ApplySpec) presentedJobState {
	return presentedJobState{applySpec: applySpec}
}

func (js presentedJobState) MarshalToJSONString() (string, error) {
	if js.applySpec == nil {
		return "", nil
	}

	envelope := presentedJobStateEnvelope{
		PersistentDisk: js.applySpec.PersistentDisk,
	}

	bytes, err := json.Marshal(envelope)
	if err != nil {
		return "", bosherr.WrapError(err, "Marshalling job state")
	}

	return string(bytes), nil
}
