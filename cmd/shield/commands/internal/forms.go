package internal

import (
	"fmt"
	"strings"
)

func FieldIsStoreUUID(name string, value string) (interface{}, error) {
	o, _, err := FindStore(value, false)
	if err != nil {
		return nil, err
	}
	return NewReference(o.UUID, o.Name), nil
}

func FieldIsTargetUUID(name string, value string) (interface{}, error) {
	o, _, err := FindTarget(value, false)
	if err != nil {
		return nil, err
	}
	return NewReference(o.UUID, o.Name), nil
}

func FieldIsScheduleUUID(name string, value string) (interface{}, error) {
	o, _, err := FindSchedule(value, false)
	if err != nil {
		return nil, err
	}
	return NewReference(o.UUID, o.Name), nil
}

func FieldIsRetentionPolicyUUID(name string, value string) (interface{}, error) {
	o, _, err := FindRetentionPolicy(value, false)
	if err != nil {
		return nil, err
	}
	return NewReference(o.UUID, o.Name), nil
}

func FieldIsRetentionTimeframe(name string, value string) (interface{}, error) {
	i, err := ParseDuration(value)
	if err != nil {
		return value, err
	}
	i.text = strings.TrimSuffix(i.text, "d")
	return i, nil
}

func FieldIsPluginName(name string, value string) (interface{}, error) {
	if len(value) < 1 {
		return value, fmt.Errorf("Field %s is a required field.\n", name)
	}
	return strings.ToLower(value), nil
}
