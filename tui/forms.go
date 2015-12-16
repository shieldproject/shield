package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FieldValidator func(name string, value string) error

type Form struct {
	Fields []*Field
}

type Field struct {
	Label     string
	Name      string
	Value     interface{}
	Validator FieldValidator
}

func NewForm() *Form {
	f := Form{}
	return &f
}

func (f *Form) NewField(label string, name string, fcn FieldValidator) error {
	f.Fields = append(f.Fields, &Field{Label: label, Name: name, Value: "", Validator: fcn})
	return nil
}

func (field *Field) Prompt() error {
	in := bufio.NewReader(os.Stdin)
	fmt.Printf("%s:\n", field.Label)
	v, err := in.ReadString('\n')
	if err != nil {
		fmt.Printf("Could not read input: %s", err)
	}
	v = strings.TrimSpace(v)
	max := 3
	for i := 0; i < max; i++ {
		err = field.Validator(field.Name, v)
		if err != nil && i < (max-1) {
			fmt.Printf("Invalid input '%s' resulted in error: %s", v, err)
			fmt.Printf("%s:\n", field.Label)
			v, err = in.ReadString('\n')
			if err != nil {
				fmt.Printf("Could not read input: %s", err)
			}
			v = strings.TrimSpace(v)
		}
		if err != nil && i == (max-1) {
			return fmt.Errorf("Valid value not provided in %d attempts. Cancelling request.\n", max)
		}
	}
	field.Value = strings.TrimSpace(v)
	return nil
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

func FieldIsRequired(name string, value string) error {
	if len(value) < 1 {
		return fmt.Errorf("Field %s is a required field.\n", name)
	}
	return nil
}

func FieldIsOptional(name string, value string) error {
	return nil
}

func InputIsInteger(name string, value string) error {
	i, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("input value cannot be converted to an integer: %s\n", err)
	}
	if i < 3600 {
		return fmt.Errorf("input value must be greater than 1 hour (3600 seconds)\n")
	}
	return nil
}

func (f *Form) ConvertFieldValueToInteger(fname string) {
	for _, field := range f.Fields {
		if field.Name == fname {
			tmp, _ := field.Value.(string)
			field.Value, _ = strconv.Atoi(tmp)
		}
	}
}

func YesOrNo(v string) (string, error) {
	v = strings.TrimSpace(v)
	v = strings.ToLower(v)
	if v == "y" || v == "yes" {
		return "y", nil
	} else if v == "n" || v == "no" || v == "" {
		return "n", nil
	}
	return "", fmt.Errorf("Input '%s' cannot be converted to bool. Accepted responses are (y)es or (n)o.\n", v)
}

func InputCanBeBool(name string, value string) error {
	_, err := YesOrNo(value)
	if err != nil {
		return err
	}
	return nil
}

func (f *Form) ConvertFieldValueToBool(fname string) {
	for _, field := range f.Fields {
		if field.Name == fname {
			tmp, _ := YesOrNo(field.Value.(string))
			field.Value = false
			if tmp == "y" {
				field.Value = true
			}
		}
	}
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
