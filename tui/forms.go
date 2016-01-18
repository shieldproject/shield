package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FieldProcessor func(name string, value string) (interface{}, error)

type Form struct {
	Fields []*Field
}

type Field struct {
	Label     string
	Name      string
	ShowAs    string
	Value     interface{}
	Processor FieldProcessor
}

func NewForm() *Form {
	f := Form{}
	return &f
}

func (f *Form) NewField(label string, name string, value interface{}, showas string, fn FieldProcessor) error {
	f.Fields = append(f.Fields, &Field{
		Label: label,
		Name: name,
		ShowAs: showas,
		Value: value,
		Processor: fn,
	})
	return nil
}

func (field *Field) PromptString() string {
	if field.ShowAs != "" {
		return fmt.Sprintf("%s (%s)", field.Label, field.ShowAs)
	}
	if field.Value != nil {
		return fmt.Sprintf("%s (%v)", field.Label, field.Value)
	}
	return field.Label
}

func (field *Field) Prompt() error {
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s: ", field.PromptString())
		v, err := in.ReadString('\n')
		if err != nil {
			return err
		}

		v = field.OrDefault(strings.TrimSpace(v))
		final, err := field.Processor(field.Name, v)
		if err != nil {
			fmt.Printf("!! %s\n", err)
			continue
		}

		field.Value = final
		return nil
	}
}

func (field *Field) OrDefault(v string) string {
	if v == "" {
		return fmt.Sprintf("%v", field.Value)
	}
	return v
}

func (f *Form) Show() error {
	for _, field := range f.Fields {
		err := field.Prompt()
		if err != nil {
			return fmt.Errorf("%s", err)
		}
	}
	return nil
}

func FieldIsRequired(name string, value string) (interface{}, error) {
	if len(value) < 1 {
		return value, fmt.Errorf("Field %s is a required field.\n", name)
	}
	return value, nil
}

func FieldIsOptional(name string, value string) (interface{}, error) {
	return value, nil
}

func FieldIsRetentionTimeframe(name string, value string) (interface{}, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		return value, fmt.Errorf("'%s' is not an integer: %s", value, err)
	}
	if i < 3600 {
		return value, fmt.Errorf("retention timeframe must be greater than 1h (3600)")
	}
	return i, nil
}

func FieldIsBoolean(name string, value string) (interface{}, error) {
	switch strings.ToLower(value) {
	case "y": fallthrough
	case "yes": return true, nil

	case "n":
	case "no": return false, nil
	}

	return "", fmt.Errorf("'%s' is not a boolean value.  Acceptable values are (y)es or (n)o.", value)
}

func (f *Form) BuildContent() (string, error) {
	c := make(map[string]interface{})
	for z := 0; z < len(f.Fields); z++ {
		field := f.Fields[z]
		c[field.Name] = field.Value
	}
	j, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("Could not marshal into JSON\nmapped input:%v\nerror:%s", c, err)
	}
	return string(j), nil
}
