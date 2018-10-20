package bus

import (
	"fmt"
	"reflect"
)

func reflectOn(thing interface{}) (map[string] interface{}, error) {
	t := reflect.TypeOf(thing)
	v := reflect.ValueOf(thing)

	for t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	m := make(map[string] interface{})
	return reflectSomeMore(m, t, &v)
}

func reflectSomeMore(out map[string] interface{}, t reflect.Type, v *reflect.Value) (map[string] interface{}, error) {
	if t.Kind() != reflect.Struct {
		return out, fmt.Errorf("bus.ParseEventData() only operates on structures")
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag, set := field.Tag.Lookup("mbus")
		if !set {
			continue
		}

		switch field.Type.Kind() {
		case reflect.String, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:

			out[tag] = v.Field(i).Interface()

		case reflect.Struct:
			vfield := v.Field(i)
			sub := make(map[string] interface{})
			sub, err := reflectSomeMore(sub, vfield.Type(), &vfield)
			if err != nil {
				return out, err
			}
			out[tag] = sub

		default:
			return  out, fmt.Errorf("bus.ParseEventData cannot operate on this type of thing")
		}
	}

	return out, nil
}
