package main

func FieldIsStoreUUID(name string, value string) (interface{}, error) {
	o, _, err := FindStore(value, false)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsTargetUUID(name string, value string) (interface{}, error) {
	o, _, err := FindTarget(value, false)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsRetentionPolicyUUID(name string, value string) (interface{}, error) {
	o, _, err := FindRetentionPolicy(value, false)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsScheduleUUID(name string, value string) (interface{}, error) {
	o, _, err := FindSchedule(value, false)
	if err != nil {
		return nil, err
	}
	return o.UUID, nil
}

func FieldIsRetentionTimeframe(name string, value string) (interface{}, error) {
	i, err := ParseDuration(value)
	if err != nil {
		return value, err
	}
	return i, nil
}
