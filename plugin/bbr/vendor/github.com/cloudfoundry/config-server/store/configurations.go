package store

import (
	"fmt"
	"strings"
)

type Configurations []Configuration

func (c Configurations) StringifiedJSON() (string, error) {
	var stringifiedConfigs []string
	for _, config := range c {
		configJSON, err := config.StringifiedJSON()
		if err != nil {
			return "", err
		}
		stringifiedConfigs = append(stringifiedConfigs, configJSON)
	}
	return fmt.Sprintf(`{"data":[%s]}`, strings.Join(stringifiedConfigs, ", ")), nil
}

func (c Configurations) Len() int           { return len(c) }
func (c Configurations) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c Configurations) Less(i, j int) bool { return c[i].ID > c[j].ID }
