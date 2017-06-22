package core

type Request struct {
	Operation      string `json:"operation"`
	TargetPlugin   string `json:"target_plugin,omitempty"`
	TargetEndpoint string `json:"target_endpoint,omitempty"`
	StorePlugin    string `json:"store_plugin,omitempty"`
	StoreEndpoint  string `json:"store_endpoint,omitempty"`
	RestoreKey     string `json:"restore_key,omitempty"`
}
