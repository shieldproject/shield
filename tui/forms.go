package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Form struct {
	Input map[string]interface{}
}

type FormOptions struct {
	IsInteger  bool
	IsBool     bool
	IsOptional bool
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

func (f *Form) NewField(prompt string, name string) error {
	must_be_bool := false
	must_be_int := false
	is_optional := false

	fmt.Printf("%s:\n", prompt)
	in := bufio.NewReader(os.Stdin)
	v, err := in.ReadString('\n')
	for i := 0; i <= 3; i++ {
		if len(v) > 1 || is_optional || err != nil {
			break
		} else if len(v) <= 1 && !is_optional && i < 3 {
			fmt.Printf("Field cannot be empty. %s:\n", prompt)
			in := bufio.NewReader(os.Stdin)
			v, err = in.ReadString('\n')
		} else if i == 3 {
			fmt.Printf("Non-empty value not provided in 3 attempts. Cancelling request.\n")
		}
	}
	if err != nil {
		return fmt.Errorf("ERROR: Could not read input: %s", err)
	}
	v = strings.TrimSpace(v)

	if must_be_int {
		f.AddAsInt(name, v)
	} else if must_be_bool {
		f.AddAsBool(name, v)
	} else {
		f.Add(name, v)
	}
	return nil
}

func FieldIsOptional(i interface{}) bool {
	if i != nil {
		b, _ := i.(bool)
		return b
	}
	return true
}

func ResultAsInteger(i interface{}) bool {
	if i != nil {
		b, _ := i.(bool)
		return b
	}
	return true
}

func ResultAsBool(i interface{}) bool {
	if i != nil {
		b, _ := i.(bool)
		return b
	}
	return true
}

func (f *Form) Add(k string, v string) {
	f.Input[k] = strings.TrimSpace(v)
}

func (f *Form) AddAsBool(k string, v string) {
	f.Input[k] = false
	if Yes(v) {
		f.Input[k] = true
	}
}

func (f *Form) AddAsInt(k string, v string) {
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
