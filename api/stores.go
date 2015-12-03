package api

type Store struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
}

type StoreFilter struct {
	Plugin string
	Unused YesNo
}

func FetchStoresList(plugin, unused string) ([]Store, error) {
	return GetStores(StoreFilter{
		Plugin: plugin,
		Unused: MaybeString(unused),
	})
}

func GetStores(filter StoreFilter) ([]Store, error) {
	uri := ShieldURI("/v1/stores")
	uri.MaybeAddParameter("plugin", filter.Plugin)
	uri.MaybeAddParameter("unused", filter.Unused)
	var data []Store
	return data, uri.Get(&data)
}

func GetStore(uuid string) (Store, error) {
	var data Store
	return data, ShieldURI("/v1/store/%s", uuid).Get(&data)
}

func CreateStore(contentJSON string) (Store, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/stores").Post(&data, contentJSON)
	if err == nil {
		return GetStore(data.UUID)
	}
	return Store{}, err
}

func UpdateStore(uuid string, contentJSON string) (Store, error) {
	err := ShieldURI("/v1/store/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetStore(uuid)
	}
	return Store{}, err
}

func DeleteStore(uuid string) error {
	return ShieldURI("/v1/store/%s", uuid).Delete(nil)
}
