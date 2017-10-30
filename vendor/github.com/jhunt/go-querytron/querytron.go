package querytron

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

var (
	True  *bool
	False *bool
)

func init() {
	t := true
	True = &t

	f := false
	False = &f
}

func Int(v int) *int          { return &v }
func Int8(v int8) *int8       { return &v }
func Int16(v int16) *int16    { return &v }
func Int32(v int32) *int32    { return &v }
func Int64(v int64) *int64    { return &v }
func Uint(v uint) *uint       { return &v }
func Uint8(v uint8) *uint8    { return &v }
func Uint16(v uint16) *uint16 { return &v }
func Uint32(v uint32) *uint32 { return &v }
func Uint64(v uint64) *uint64 { return &v }

func override(q url.Values, t reflect.Type, v *reflect.Value) {
	if t.Kind() != reflect.Struct {
		return
	}
	if !v.CanSet() {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if _, set := field.Tag.Lookup("qs"); !set {
			switch field.Type.Kind() {
			case reflect.Struct, reflect.Ptr:
				valu := v.Field(i)
				override(q, t.Field(i).Type, &valu)
			}
			continue
		}
		tag := field.Tag.Get("qs")
		if _, ok := q[tag]; ok {
			e := q.Get(tag)
			switch field.Type.Kind() {
			case reflect.String:
				v.Field(i).Set(reflect.ValueOf(stringify(e)))

			case reflect.Bool:
				v.Field(i).Set(reflect.ValueOf(boolify(e)))

			case reflect.Int:
				v.Field(i).Set(reflect.ValueOf(intify(e, 0)))

			case reflect.Int8:
				v.Field(i).Set(reflect.ValueOf(intify(e, 8)))

			case reflect.Int16:
				v.Field(i).Set(reflect.ValueOf(intify(e, 16)))

			case reflect.Int32:
				v.Field(i).Set(reflect.ValueOf(intify(e, 32)))

			case reflect.Int64:
				v.Field(i).Set(reflect.ValueOf(intify(e, 64)))

			case reflect.Uint:
				v.Field(i).Set(reflect.ValueOf(uintify(e, 0)))

			case reflect.Uint8:
				v.Field(i).Set(reflect.ValueOf(uintify(e, 8)))

			case reflect.Uint16:
				v.Field(i).Set(reflect.ValueOf(uintify(e, 16)))

			case reflect.Uint32:
				v.Field(i).Set(reflect.ValueOf(uintify(e, 32)))

			case reflect.Uint64:
				v.Field(i).Set(reflect.ValueOf(uintify(e, 64)))

			case reflect.Float32:
				v.Field(i).Set(reflect.ValueOf(floatify(e, 32)))

			case reflect.Float64:
				v.Field(i).Set(reflect.ValueOf(floatify(e, 64)))
			}
		}
	}
}

func Override(thing interface{}, q url.Values) {
	t := reflect.TypeOf(thing)
	v := reflect.ValueOf(thing)
	for t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	override(q, t, &v)
}

func generate(q *url.Values, t reflect.Type, v *reflect.Value) {
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

		case reflect.Int:
			fallthrough
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			q.Add(tag, fmt.Sprintf("%v", v.Field(i).Int()))

		case reflect.Uint:
			fallthrough
		case reflect.Uint8:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			fallthrough
		case reflect.Uint64:
			q.Add(tag, fmt.Sprintf("%v", v.Field(i).Uint()))

		/* FIXME: alway-on bool support */
		case reflect.Bool:
			setbool(q, tag, v.Field(i).Bool())
			break

		case reflect.Ptr:
			if v.Field(i).Pointer() != 0 {
				switch field.Type.Elem().Kind() {
				case reflect.String:
					if s := v.Field(i).String(); s != "" {
						q.Add(tag, s)
					}

				case reflect.Int:
					fallthrough
				case reflect.Int8:
					fallthrough
				case reflect.Int16:
					fallthrough
				case reflect.Int32:
					fallthrough
				case reflect.Int64:
					q.Add(tag, fmt.Sprintf("%v", v.Field(i).Elem().Int()))

				case reflect.Uint:
					fallthrough
				case reflect.Uint8:
					fallthrough
				case reflect.Uint16:
					fallthrough
				case reflect.Uint32:
					fallthrough
				case reflect.Uint64:
					q.Add(tag, fmt.Sprintf("%v", v.Field(i).Elem().Uint()))

				case reflect.Bool:
					setbool(q, tag, v.Field(i).Elem().Bool())
				}
			}
			break
		}
	}
}

func Generate(thing interface{}) url.Values {
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

	generate(&q, t, &v)
	return q
}

func setbool(u *url.Values, tag string, tru bool) {
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

func stringify(s string) string {
	return s
}

func boolify(s string) bool {
	switch strings.ToLower(s) {
	case "y", "yes", "1", "true":
		return true
	}
	return false
}

func intify(s string, w int) interface{} {
	i64, err := strconv.ParseInt(s, 10, w)
	if err != nil {
		i64 = 0
	}

	switch w {
	case 0:
		return int(i64)
	case 8:
		return int8(i64)
	case 16:
		return int16(i64)
	case 32:
		return int32(i64)
	case 64:
		return int64(i64)
	}
	return int(0)
}

func uintify(s string, w int) interface{} {
	u64, err := strconv.ParseUint(s, 10, w)
	if err != nil {
		u64 = 0
	}

	switch w {
	case 0:
		return uint(u64)
	case 8:
		return uint8(u64)
	case 16:
		return uint16(u64)
	case 32:
		return uint32(u64)
	case 64:
		return uint64(u64)
	}
	return uint(0)
}

func floatify(s string, w int) interface{} {
	f64, err := strconv.ParseFloat(s, w)
	if err != nil {
		f64 = 0.0
	}

	switch w {
	case 32:
		return float32(f64)
	case 64:
		return float64(f64)
	}
	return float32(0)
}
