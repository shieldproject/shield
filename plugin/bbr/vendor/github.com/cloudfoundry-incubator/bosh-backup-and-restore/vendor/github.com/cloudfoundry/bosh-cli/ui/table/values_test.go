package table_test

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("ValueString", func() {
	It("returns string", func() {
		Expect(ValueString{"val"}.String()).To(Equal("val"))
	})

	It("returns itself", func() {
		Expect(ValueString{"val"}.Value()).To(Equal(ValueString{"val"}))
	})

	It("returns int based on string compare", func() {
		Expect(ValueString{"a"}.Compare(ValueString{"a"})).To(Equal(0))
		Expect(ValueString{"a"}.Compare(ValueString{"b"})).To(Equal(-1))
		Expect(ValueString{"b"}.Compare(ValueString{"a"})).To(Equal(1))
	})
})

var _ = Describe("ValueStrings", func() {
	It("returns new line joined strings", func() {
		Expect(ValueStrings{[]string{"val1", "val2"}}.String()).To(Equal("val1\nval2"))
	})

	It("returns itself", func() {
		Expect(ValueStrings{[]string{"val1"}}.Value()).To(Equal(ValueStrings{[]string{"val1"}}))
	})

	It("returns int based on string compare", func() {
		Expect(ValueStrings{[]string{"val1"}}.Compare(ValueStrings{[]string{"val1"}})).To(Equal(0))
		Expect(ValueStrings{[]string{"val1"}}.Compare(ValueStrings{[]string{"val1", "val2"}})).To(Equal(-1))
		Expect(ValueStrings{[]string{"val1", "val2"}}.Compare(ValueStrings{[]string{"val1"}})).To(Equal(1))
	})
})

var _ = Describe("ValueInt", func() {
	It("returns string", func() {
		Expect(ValueInt{1}.String()).To(Equal("1"))
	})

	It("returns itself", func() {
		Expect(ValueInt{1}.Value()).To(Equal(ValueInt{1}))
	})

	It("returns int based on int compare", func() {
		Expect(ValueInt{1}.Compare(ValueInt{1})).To(Equal(0))
		Expect(ValueInt{1}.Compare(ValueInt{2})).To(Equal(-1))
		Expect(ValueInt{2}.Compare(ValueInt{1})).To(Equal(1))
	})
})

var _ = Describe("ValueBytes", func() {
	It("returns formatted bytes", func() {
		Expect(ValueBytes{1}.String()).To(Equal("1 B"))
	})

	It("returns formatted mebibytes", func() {
		Expect(NewValueMegaBytes(1).String()).To(Equal("1.0 MiB"))
	})

	It("returns formatted gibibytes", func() {
		Expect(NewValueMegaBytes(131072).String()).To(Equal("128 GiB"))
	})

	It("returns itself", func() {
		Expect(ValueBytes{1}.Value()).To(Equal(ValueBytes{1}))
	})

	It("returns int based on int compare", func() {
		Expect(ValueBytes{1}.Compare(ValueBytes{1})).To(Equal(0))
		Expect(ValueBytes{1}.Compare(ValueBytes{2})).To(Equal(-1))
		Expect(ValueBytes{2}.Compare(ValueBytes{1})).To(Equal(1))
	})
})

var _ = Describe("ValueTime", func() {
	t1 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	t2 := time.Date(2009, time.November, 10, 23, 0, 0, 1, time.UTC)

	It("returns formatted full time", func() {
		Expect(ValueTime{t1}.String()).To(Equal("Tue Nov 10 23:00:00 UTC 2009"))
	})

	It("returns itself", func() {
		Expect(ValueTime{t1}.Value()).To(Equal(ValueTime{t1}))
	})

	It("returns int based on time compare", func() {
		Expect(ValueTime{t1}.Compare(ValueTime{t1})).To(Equal(0))
		Expect(ValueTime{t1}.Compare(ValueTime{t2})).To(Equal(-1))
		Expect(ValueTime{t2}.Compare(ValueTime{t1})).To(Equal(1))
	})
})

var _ = Describe("ValueBool", func() {
	It("returns true/false as string", func() {
		Expect(ValueBool{true}.String()).To(Equal("true"))
		Expect(ValueBool{false}.String()).To(Equal("false"))
	})

	It("returns itself", func() {
		Expect(ValueBool{true}.Value()).To(Equal(ValueBool{true}))
	})

	It("returns int based on bool compare", func() {
		Expect(ValueBool{true}.Compare(ValueBool{true})).To(Equal(0))
		Expect(ValueBool{false}.Compare(ValueBool{true})).To(Equal(-1))
		Expect(ValueBool{true}.Compare(ValueBool{false})).To(Equal(1))
	})
})

var _ = Describe("ValueVersion", func() {
	v1 := semver.MustNewVersionFromString("1.1")
	v2 := semver.MustNewVersionFromString("1.2")

	It("returns formatted version", func() {
		Expect(ValueVersion{v1}.String()).To(Equal("1.1"))
	})

	It("returns itself", func() {
		Expect(ValueVersion{v1}.Value()).To(Equal(ValueVersion{v1}))
	})

	It("returns int based on version compare", func() {
		Expect(ValueVersion{v1}.Compare(ValueVersion{v1})).To(Equal(0))
		Expect(ValueVersion{v2}.Compare(ValueVersion{v1})).To(Equal(1))
		Expect(ValueVersion{v1}.Compare(ValueVersion{v2})).To(Equal(-1))
	})
})

var _ = Describe("ValueError", func() {
	It("returns empty string or error description", func() {
		Expect(ValueError{}.String()).To(Equal(""))
		Expect(ValueError{errors.New("err")}.String()).To(Equal("err"))
	})

	It("returns itself", func() {
		Expect(ValueError{errors.New("err")}.Value()).To(Equal(ValueError{errors.New("err")}))
	})

	It("does not allow comparison", func() {
		f := func() { ValueError{}.Compare(ValueError{}) }
		Expect(f).To(Panic())
	})
})

var _ = Describe("ValueNone", func() {
	It("returns empty string", func() {
		Expect(ValueNone{}.String()).To(Equal(""))
	})

	It("returns itself", func() {
		Expect(ValueNone{}.Value()).To(Equal(ValueNone{}))
	})

	It("does not allow comparison", func() {
		f := func() { ValueNone{}.Compare(ValueNone{}) }
		Expect(f).To(Panic())
	})
})

var _ = Describe("ValueFmt", func() {
	fmtFunc := func(pattern string, vals ...interface{}) string {
		return fmt.Sprintf(">%s<", fmt.Sprintf(pattern, vals...))
	}

	It("returns plain string (not formatted with fmt func)", func() {
		Expect(ValueFmt{V: ValueInt{1}, Func: fmtFunc}.String()).To(Equal("1"))
	})

	It("returns wrapped value", func() {
		Expect(ValueFmt{V: ValueInt{1}, Func: fmtFunc}.Value()).To(Equal(ValueInt{1}))
	})

	It("does not allow comparison", func() {
		f := func() { ValueFmt{V: ValueInt{1}, Func: fmtFunc}.Compare(ValueFmt{}) }
		Expect(f).To(Panic())
	})

	It("writes out value using custom Fprintf", func() {
		buf := bytes.NewBufferString("")
		ValueFmt{V: ValueInt{1}, Func: fmtFunc}.Fprintf(buf, "%s,%s", "val1", "val2")
		Expect(buf.String()).To(Equal(">val1,val2<"))
	})

	It("uses fmt.Fprintf if fmt func is not set", func() {
		buf := bytes.NewBufferString("")
		ValueFmt{V: ValueInt{1}}.Fprintf(buf, "%s,%s", "val1", "val2")
		Expect(buf.String()).To(Equal("val1,val2"))
	})
})

type failsToYAMLMarshal struct{}

func (s failsToYAMLMarshal) MarshalYAML() (interface{}, error) {
	return nil, errors.New("marshal-err")
}

var _ = Describe("ValueInterface", func() {
	It("returns map as a string", func() {
		i := map[string]interface{}{"key": "value", "num": 123}
		Expect(ValueInterface{I: i}.String()).To(Equal("key: value\nnum: 123"))
	})

	It("returns nested items as a string", func() {
		i := map[string]interface{}{"key": map[string]interface{}{"nested_key": "nested_value"}}
		Expect(ValueInterface{I: i}.String()).To(Equal("key:\n  nested_key: nested_value"))
	})

	It("returns nested items as a string", func() {
		i := failsToYAMLMarshal{}
		Expect(ValueInterface{I: i}.String()).To(Equal(`<serialization error> : table_test.failsToYAMLMarshal{}`))
	})

	It("returns nil items as blank string", func() {
		Expect(ValueInterface{I: nil}.String()).To(Equal(""))
	})

	It("returns an empty map as blank string", func() {
		i := map[string]interface{}{}
		Expect(ValueInterface{I: i}.String()).To(Equal(""))
	})

	It("returns an empty slice as blank string", func() {
		i := []string{}
		Expect(ValueInterface{I: i}.String()).To(Equal(""))
	})
})

var _ = Describe("ValueSuffix", func() {
	It("returns formatted string with suffix", func() {
		Expect(ValueSuffix{ValueInt{1}, "*"}.String()).To(Equal("1*"))
		Expect(ValueSuffix{ValueString{"val"}, "*"}.String()).To(Equal("val*"))
	})

	It("returns wrapped value", func() {
		Expect(ValueSuffix{ValueInt{1}, "*"}.Value()).To(Equal(ValueInt{1}))
	})

	It("does not allow comparison", func() {
		f := func() { ValueSuffix{ValueInt{1}, ""}.Compare(ValueSuffix{}) }
		Expect(f).To(Panic())
	})
})
