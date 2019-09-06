package db

import (
	"fmt"

	"github.com/shieldproject/shield/plugin"
)

type ConfigItem struct {
	Label    string `json:"label"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Redacted bool   `json:"redacted"`
	Obscure  bool   `json:"obscure"`
}

func DisplayableConfig(typ string, info *plugin.PluginInfo, config map[string]interface{}, private bool) []ConfigItem {
	l := make([]ConfigItem, 0)
	for _, field := range info.Fields {
		if field.Mode != typ {
			continue
		}

		item := ConfigItem{
			Key:     field.Name,
			Label:   field.Title,
			Default: field.Default,
			Value:   fmt.Sprintf("%v", config[field.Name]),
			Type:    field.Type,
		}
		if field.Type == "bool" {
			if config[field.Name] == nil {
				item.Value = "no"
			} else {
				item.Value = "yes"
			}
		} else if field.Type == "password" || field.Type == "pem-rsa-pk" {
			item.Obscure = true
			if !private {
				item.Value = ""
				item.Redacted = true
			}
		} else if config[field.Name] == nil {
			item.Value = ""
		}
		l = append(l, item)
	}

	return l
}
