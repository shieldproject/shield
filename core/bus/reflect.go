package bus

import (
	"reflect"
)

func reflectOn(thing interface{}) map[string] interface{} {
	t := reflect.TypeOf(thing)
	v := reflect.ValueOf(thing)

	for t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	m := make(map[string] interface{})
	return reflectSomeMore(m, t, &v)
}

func reflectSomeMore(out map[string] interface{}, t reflect.Type, v *reflect.Value) map[string] interface{} {
	if t.Kind() != reflect.Struct {
		panic("bus.ParseEventData() only operates on structures")
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
		default:
			out[tag] = v.Field(i).Interface()

		case reflect.Struct:
			vfield := v.Field(i)
			out[tag] = reflectSomeMore(make(map[string] interface{}), vfield.Type(), &vfield)
		}
	}

	return out
}
