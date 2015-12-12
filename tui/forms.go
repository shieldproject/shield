package tui

import (
	"fmt"
	"strings"
	"strconv"
	"encoding/json"
)

type Form struct {
	Input map[string]interface{}
}

func Yes(v string) bool {
	v = strings.TrimSpace(v)
	v = strings.ToLower(v)
	if v == "y" || v == "yes" {
		return true
	}
	return false
}

func NewForm() *Form {
	f := Form{
		Input: make(map[string]interface{}),
	}
	return &f
}

func (f *Form) Add(k string, v string) {
	f.Input[k] = strings.TrimSpace(v)
}

func (f *Form) AddAsBool(k string, v string)  {
	f.Input[k] = false
	if Yes(v) {
		f.Input[k] = true
	}
}

func (f *Form) AddAsInt(k string, v string)  {
	v = strings.TrimSpace(v)
	i, _ := strconv.Atoi(v)
	f.Input[k] = i
}

func (f *Form) BuildContent() (string, error) {
	j, err := json.Marshal(f.Input)
	if err != nil {
		return "", fmt.Errorf("ERROR: Could not marshal into JSON\nmapped input:%v\nerror:%s", f.Input, err)
	}
	return string(j), nil
}
