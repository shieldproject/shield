package testutils

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/pivotal-golang/yaml"
)

func MarshalToString(input interface{}) (string, error) {
	bytes, err := yaml.Marshal(input)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Marshaling to string: %#v", input)
	}

	return string(bytes), nil
}
