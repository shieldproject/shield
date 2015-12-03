package api

type Target struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
	Agent    string `json:"agent"`
}

type TargetFilter struct {
	Plugin string
	Unused YesNo
}

func FetchTargetsList(plugin, unused string) ([]Target, error) {
	return GetTargets(TargetFilter{
		Plugin: plugin,
		Unused: MaybeString(unused),
	})
}

func GetTargets(filter TargetFilter) ([]Target, error) {
	uri := ShieldURI("/v1/targets")
	uri.MaybeAddParameter("plugin", filter.Plugin)
	uri.MaybeAddParameter("unused", filter.Unused)

	var data []Target
	return data, uri.Get(&data)
}

func GetTarget(uuid string) (Target, error) {
	var data Target
	return data, ShieldURI("/v1/targets/%s", uuid).Get(&data)
}

func CreateTarget(contentJSON string) (Target, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/targets").Post(&data, contentJSON)
	if err == nil {
		return GetTarget(data.UUID)
	}
	return Target{}, err
}

func UpdateTarget(uuid string, contentJSON string) (Target, error) {
	err := ShieldURI("/v1/target/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetTarget(uuid)
	}
	return Target{}, err
}

func DeleteTarget(uuid string) error {
	return ShieldURI("/v1/target/%s", uuid).Delete(nil)
}
