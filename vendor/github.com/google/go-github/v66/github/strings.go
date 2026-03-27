// Copyright 2013 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"bytes"
	"fmt"
	"reflect"
)

var timestampType = reflect.TypeOf(Timestamp{})

// Stringify attempts to create a reasonable string representation of types in
// the GitHub library. It does things like resolve pointers to their values
// and omits struct fields with nil values.
func Stringify(message any) string {
	var buf bytes.Buffer
	v := reflect.ValueOf(message)
	stringifyValue(&buf, v)
	return buf.String()
}

// stringifyValue was heavily inspired by the goprotobuf library.

func stringifyValue(w *bytes.Buffer, val reflect.Value) {
<<<<<<<< HEAD:vendor/github.com/google/go-github/v76/github/strings.go
	if val.Kind() == reflect.Pointer && val.IsNil() {
========
	if val.Kind() == reflect.Ptr && val.IsNil() {
>>>>>>>> 2bdfd3af (Update go.mod and vendor for v10.0.0 deps):vendor/github.com/google/go-github/v66/github/strings.go
		w.WriteString("<nil>")
		return
	}

	v := reflect.Indirect(val)

	switch v.Kind() {
	case reflect.String:
		fmt.Fprintf(w, `"%v"`, v)
	case reflect.Slice:
		w.WriteByte('[')
<<<<<<<< HEAD:vendor/github.com/google/go-github/v76/github/strings.go
		for i := range v.Len() {
========
		for i := 0; i < v.Len(); i++ {
>>>>>>>> 2bdfd3af (Update go.mod and vendor for v10.0.0 deps):vendor/github.com/google/go-github/v66/github/strings.go
			if i > 0 {
				w.WriteByte(' ')
			}

			stringifyValue(w, v.Index(i))
		}

		w.WriteByte(']')
		return
	case reflect.Struct:
		if v.Type().Name() != "" {
			w.WriteString(v.Type().String())
		}

		// special handling of Timestamp values
		if v.Type() == timestampType {
			fmt.Fprintf(w, "{%v}", v.Interface())
			return
		}

		w.WriteByte('{')

		var sep bool
		for i := range v.NumField() {
			fv := v.Field(i)
			if fv.Kind() == reflect.Pointer && fv.IsNil() {
				continue
			}
			if fv.Kind() == reflect.Slice && fv.IsNil() {
				continue
			}
			if fv.Kind() == reflect.Map && fv.IsNil() {
				continue
			}

			if sep {
				w.WriteString(", ")
			} else {
				sep = true
			}

			w.WriteString(v.Type().Field(i).Name)
			w.WriteByte(':')
			stringifyValue(w, fv)
		}

		w.WriteByte('}')
	default:
		if v.CanInterface() {
			fmt.Fprint(w, v.Interface())
		}
	}
}
