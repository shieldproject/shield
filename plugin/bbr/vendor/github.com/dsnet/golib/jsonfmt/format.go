// Copyright 2017, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package jsonfmt provides helper functions related to JSON.
package jsonfmt

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
)

// Option configures how to format JSON.
type Option interface {
	option()
}

type (
	minify      struct{ Option }
	standardize struct{ Option }
)

// Minify configures Format to produce the minimal representation of the input.
// If Format returns no error, then the output is guaranteed to be valid JSON,
func Minify() Option {
	return minify{}
}

// Standardize configures Format to produce valid JSON according to ECMA-404.
// This strips any comments and trailing commas.
func Standardize() Option {
	return standardize{}
}

// EncodeStrings configures Format to encode string literals in a specific way.
// By default, Format re-encodes string literals as UTF-8.
func EncodeStrings() Option {
	return nil
}

// Format parses and formats the input JSON according to provided Options.
// If err is non-nil, then the output is a best effort at processing the input.
//
// This function accepts a superset of the JSON specification that allows
// comments and trailing commas after the last element in an object or array.
func Format(s []byte, opts ...Option) (out []byte, err error) {
	if len(opts) != 1 {
		return s, errors.New("jsonfmt: only Minify option is currently allowed")
	}
	if _, ok := opts[0].(minify); !ok {
		return s, errors.New("jsonfmt: only Minify option is currently allowed")
	}

	m := minifier{in: s}
	defer m.errRecover(&out, &err)
	m.parseIgnored()
	m.parseValue()
	m.parseIgnored()
	if len(m.in) > 0 {
		m.errPanic("unexpected trailing input")
	}
	return m.out, nil
}

type minifier struct {
	in, out []byte
}

var (
	stringRegex  = regexp.MustCompile(`^"(\\(["\\\/bfnrt]|u[a-fA-F0-9]{4})|[^"\\\x00-\x1f\x7f]+)*"`)
	numberRegex  = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?`)
	literalRegex = regexp.MustCompile(`^(true|false|null)`)

	commentRegex = regexp.MustCompile(`^(/\*([^\n]|\n)*?\*/|//[^\n]*\n?)`)
	spaceRegex   = regexp.MustCompile(`^[ \r\n\t]*`)
)

func (m *minifier) parseValue() {
	if len(m.in) == 0 {
		m.errPanic("unable to parse value")
	}
	switch m.in[0] {
	case '{':
		m.parseObject()
	case '[':
		m.parseArray()
	case '"':
		m.parseRegexp(stringRegex, "string")
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		m.parseRegexp(numberRegex, "number")
	case 't', 'f', 'n':
		m.parseRegexp(literalRegex, "literal")
	default:
		m.errPanic("unable to parse value")
	}
}

func (m *minifier) parseObject() {
	m.parseChar('{', "object")
	for {
		m.parseIgnored()
		if len(m.in) > 0 && m.in[0] == '}' {
			break
		}
		m.parseRegexp(stringRegex, "string")
		m.parseIgnored()
		m.parseChar(':', "object")
		m.parseIgnored()
		m.parseValue()
		m.parseIgnored()
		if len(m.in) > 0 && m.in[0] == '}' {
			break
		}
		m.parseChar(',', "object")
	}
	m.out = append(bytes.TrimRight(m.out, ","), '}')
	m.in = m.in[1:]
}

func (m *minifier) parseArray() {
	m.parseChar('[', "array")
	for {
		m.parseIgnored()
		if len(m.in) > 0 && m.in[0] == ']' {
			break
		}
		m.parseValue()
		m.parseIgnored()
		if len(m.in) > 0 && m.in[0] == ']' {
			break
		}
		m.parseChar(',', "array")
	}
	m.out = append(bytes.TrimRight(m.out, ","), ']')
	m.in = m.in[1:]
}

func (m *minifier) parseIgnored() {
	for {
		n := len(commentRegex.Find(m.in)) + len(spaceRegex.Find(m.in))
		if n == 0 {
			return
		}
		m.in = m.in[n:]
	}
}

func (m *minifier) parseRegexp(r *regexp.Regexp, what string) {
	n := len(r.Find(m.in))
	if n == 0 {
		m.errPanic("unable to parse %s", what)
	}
	m.out = append(m.out, m.in[:n]...)
	m.in = m.in[n:]
}

func (m *minifier) parseChar(c uint8, what string) {
	if len(m.in) == 0 || m.in[0] != c {
		m.errPanic("unable to parse %s", what)
	}
	m.out = append(m.out, m.in[0])
	m.in = m.in[1:]
}

type stringError string

func (es stringError) Error() string {
	return "jsonfmt: " + string(es)
}

func (m *minifier) errPanic(f string, x ...interface{}) {
	m.out = append(m.out, m.in...)
	panic(stringError(fmt.Sprintf(f, x...)))
}

func (m *minifier) errRecover(out *[]byte, err *error) {
	if ex := recover(); ex != nil {
		if es, ok := ex.(stringError); ok {
			*err = es
			*out = m.out
		} else {
			panic(ex)
		}
	}
}
