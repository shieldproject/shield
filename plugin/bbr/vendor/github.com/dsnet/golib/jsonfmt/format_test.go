// Copyright 2017, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package jsonfmt

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		in  string
		out string
		err error
	}{{
		in:  "",
		out: "",
		err: stringError("unable to parse value"),
	}, {
		in:  `/*comment*/"Hello, 世界"/*comment*/ // some comment`,
		out: `"Hello, 世界"`,
	}, {
		in:  "5 /*comment*/trailing",
		out: "5trailing",
		err: stringError("unexpected trailing input"),
	}, {
		in:  "\"\xff\\r\\n\\u4444\\tfewfew\"",
		out: "\"\xff\\r\\n\\u4444\\tfewfew\"",
	}, {
		in:  "[]",
		out: "[]",
	}, {
		in:  "[,]",
		out: "[,]",
		err: stringError("unable to parse value"),
	}, {
		in:  "[5;3]",
		out: "[5;3]",
		err: stringError("unable to parse array"),
	}, {
		in:  "/*comment*/[/*comment*/null/*comment*/]/*comment*/",
		out: "[null]",
	}, {
		in:  "/*comment*/[/*comment*/false/*comment*/,/*comment*/]/*comment*/",
		out: "[false]",
	}, {
		in:  "[1,2,3,]",
		out: "[1,2,3]",
	}, {
		in:  `/*comment*/{/*comment*/"foo"/*comment*/:/*comment*/"bar"/*comment*/,/*comment*/}//comment`,
		out: `{"foo":"bar"}`,
	}, {
		in: `// comment /*
			{/*multi
			line// fewafewa
			comment*/"key"/*multi
			line*/://comment


			"value"/* /* /*
			comment
			*/,//comment
			}// /*comment

			`,
		out: `{"key":"value"}`,
	}}

	for i, tt := range tests {
		got, err := Format([]byte(tt.in), Minify())
		if string(got) != tt.out || err != tt.err {
			t.Errorf("test %d, Format(%q, Minify()):\ngot  (%q, %v)\nwant (%q, %v)", i, tt.in, got, err, tt.out, tt.err)
		}
	}
}
