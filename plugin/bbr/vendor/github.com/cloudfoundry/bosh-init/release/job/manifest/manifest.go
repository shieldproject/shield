package manifest

type Manifest struct {
	Name       string                        `yaml:"name"`
	Templates  map[string]string             `yaml:"templates"`
	Packages   []string                      `yaml:"packages"`
	Properties map[string]PropertyDefinition `yaml:"properties"`
}

type PropertyDefinition struct {
	Description string      `yaml:"description"`
	Default     interface{} `yaml:"default"`
}
