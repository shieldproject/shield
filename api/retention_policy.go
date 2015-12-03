package api

type RetentionPoliciesFilter struct {
	Unused YesNo
}

type RetentionPolicy struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Expires uint   `json:"expires"`
}

func GetRetentionPolicies(filter RetentionPoliciesFilter) ([]RetentionPolicy, error) {
	uri := ShieldURI("/v1/retention")
	uri.MaybeAddParameter("unused", filter.Unused)

	var data []RetentionPolicy
	return data, uri.Get(&data)
}

func GetRetentionPolicy(uuid string) (RetentionPolicy, error) {
	var data RetentionPolicy
	return data, ShieldURI("v1/retention/%s", uuid).Get(&data)
}

func CreateRetentionPolicy(contentJSON string) (RetentionPolicy, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v2/retention").Post(&data, contentJSON)
	if err == nil {
		return GetRetentionPolicy(data.UUID)
	}
	return RetentionPolicy{}, err
}

func UpdateRetentionPolicy(uuid string, contentJSON string) (RetentionPolicy, error) {
	err := ShieldURI("/v1/retention/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetRetentionPolicy(uuid)
	}
	return RetentionPolicy{}, err
}

func DeleteRetentionPolicy(uuid string) error {
	return ShieldURI("/v1/retention/%s", uuid).Delete(nil)
}
