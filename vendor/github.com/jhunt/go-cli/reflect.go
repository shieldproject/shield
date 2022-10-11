package cli

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

func reflectOnIt(thing interface{}) (context, error) {
	t := reflect.TypeOf(thing)
	v := reflect.ValueOf(thing)

	for t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	c := context{
		Options: make([]*option, 0),
		Subs:    make(map[string]context),
	}
	return reflectSomeMore(c, t, &v)
}

func reflectSomeMore(c context, t reflect.Type, v *reflect.Value) (context, error) {

	if t.Kind() != reflect.Struct {
		return c, fmt.Errorf("go-cli only operates on structures")
	}
	if !v.CanSet() {
		return c, fmt.Errorf("go-cli requires a writable structure")
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if _, set := field.Tag.Lookup("cli"); !set {
			continue
		}

		tag := field.Tag.Get("cli")

		t := field.Type
		v := v.Field(i)
		for t.Kind() == reflect.Ptr && !v.IsNil() {
			t = v.Elem().Type()
			v = v.Elem()
		}

		switch t.Kind() {
		case reflect.Slice:
			if !v.IsValid() {
				return c, fmt.Errorf("go-cli requires slice ([]thing) options to be initialized first")
			}
			fallthrough

		case reflect.String, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:

			o, err := newOption(t, t.Kind(), &v, tag)
			if err != nil {
				return c, err
			}
			c.Options = append(c.Options, o)
			break

		case reflect.Ptr:
			if t.Elem().Kind() == reflect.Bool {
				o, err := newOption(t, t.Kind(), &v, tag)
				if err != nil {
					return c, err
				}
				c.Options = append(c.Options, o)
			} else {
				return c, fmt.Errorf("go-cli cannot operate on this type of thing")
			}
			break

		case reflect.Struct:
			sub := context{
				Options: make([]*option, 0),
				Subs:    make(map[string]context),
			}
			sub, err := reflectSomeMore(sub, v.Type(), &v)
			if err != nil {
				return c, err
			}

			for _, cmd := range regexp.MustCompile(" *, *").Split(tag, -1) {
				if strings.HasSuffix(cmd, "!") {
					sub.Stop = true
					cmd = cmd[:len(cmd)-1]
				}
				if sub.Command == "" {
					sub.Command = cmd
				}
				c.Subs[cmd] = sub
			}
			break

		default:
			return c, fmt.Errorf("go-cli cannot operate on this type of thing")
		}
	}

	return c, nil
}

func newOption(typ reflect.Type, kind reflect.Kind, value *reflect.Value, tag string) (*option, error) {
	splitter := regexp.MustCompile(" *, *")
	short := regexp.MustCompile("^-([a-zA-Z0-9?])$")
	long := regexp.MustCompile("^--([a-zA-Z0-9?][a-zA-Z0-9?-]+)$")

	o := &option{
		Init:   false,
		Type:   typ,
		Kind:   kind,
		Value:  value,
		Shorts: "",
		Longs:  make([]string, 0),
	}

	seen := make(map[string]bool) /* to de-dupe inside the tag spec */
	for _, opt := range splitter.Split(tag, -1) {
		if m := short.FindStringSubmatch(opt); m != nil {
			if _, ok := seen[m[1]]; !ok {
				o.Shorts = o.Shorts + m[1]
				seen[m[1]] = true
			}
			continue
		}
		if m := long.FindStringSubmatch(opt); m != nil {
			if _, ok := seen[m[1]]; !ok {
				o.Longs = append(o.Longs, m[1])
				seen[m[1]] = true
			}
			continue
		}
		return o, fmt.Errorf("invalid option flag '%s'", opt)
	}
	return o, nil
}
