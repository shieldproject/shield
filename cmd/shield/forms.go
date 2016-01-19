package main

import (
	"fmt"
	"strconv"
)

func FieldIsStoreUUID(name string, value string) (interface{}, error) {
	o, _, err := FindStore(value)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsTargetUUID(name string, value string) (interface{}, error) {
	o, _, err := FindTarget(value)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsRetentionPolicyUUID(name string, value string) (interface{}, error) {
	o, _, err := FindRetentionPolicy(value)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsScheduleUUID(name string, value string) (interface{}, error) {
	o, _, err := FindSchedule(value)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsRetentionTimeframe(name string, value string) (interface{}, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		return value, fmt.Errorf("'%s' is not an integer: %s", value, err)
	}
	if i < 0 {
		return value, fmt.Errorf("retention timeframe must be at least 1 day")
	}
	return i * 86400, nil
}
