package shield

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

func generateQueryString(thing interface{}) url.Values {
	q := make(url.Values)
	if thing == nil {
		return q
	}

	t := reflect.TypeOf(thing)
	v := reflect.ValueOf(thing)
	for t.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		t = v.Type()
	}

	qsGenerate(&q, t, &v)
	return q
}

func qsGenerate(q *url.Values, t reflect.Type, v *reflect.Value) {
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if _, set := field.Tag.Lookup("qs"); !set {
			continue
		}
		tag := field.Tag.Get("qs")
		switch field.Type.Kind() {
		case reflect.String:
			if s := v.Field(i).String(); s != "" {
				q.Add(tag, s)
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			q.Add(tag, fmt.Sprintf("%v", v.Field(i).Int()))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			q.Add(tag, fmt.Sprintf("%v", v.Field(i).Uint()))

		case reflect.Bool:
			qsSetBool(q, tag, v.Field(i).Bool())

		case reflect.Ptr:
			if v.Field(i).Pointer() != 0 {
				switch field.Type.Elem().Kind() {
				case reflect.String:
					if s := v.Field(i).Elem().String(); s != "" {
						q.Add(tag, s)
					}

				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					q.Add(tag, fmt.Sprintf("%v", v.Field(i).Elem().Int()))

				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					q.Add(tag, fmt.Sprintf("%v", v.Field(i).Elem().Uint()))

				case reflect.Bool:
					qsSetBool(q, tag, v.Field(i).Elem().Bool())
				}
			}
		}
	}
}

func qsSetBool(u *url.Values, tag string, tru bool) {
	tags := strings.Split(tag, ":")
	if len(tags) == 3 && !tru {
		u.Add(tags[0], tags[2])
		return
	}
	if len(tags) > 1 && tru {
		u.Add(tags[0], tags[1])
		return
	}
	if tru {
		u.Add(tags[0], "")
	}
}
