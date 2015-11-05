package plugin_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"plugin"
)

var _ = Describe("ShieldEndpoint", func() {
	endpoint := plugin.ShieldEndpoint{
		"stringVal": "asdf",
		"intVal":    1234,
		"floatVal":  1234.1234,
		"boolVal":   true,
		"arrayVal": []interface{}{
			"asdf",
			"fdsa",
		},
		"mapVal": map[string]interface{}{
			"key": "value",
		},
	}
	Describe("ArrayVal", func() {
		It("returns an array from the endpoint, when provided the right key", func() {
			expected := []interface{}{"asdf", "fdsa"}

			got, err := endpoint.ArrayValue("arrayVal")
			Expect(got).Should(BeEquivalentTo(expected))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("errors out when not pointed at an array", func() {
			got, err := endpoint.ArrayValue("stringVal")
			Expect(got).Should(BeNil())
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("'stringVal' key in endpoint json is a string, not an array"))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.ArrayValue("doesnotexist")
			Expect(got).Should(BeNil())
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("No 'doesnotexist' key specified in the endpoint json"))
		})
	})
	Describe("MapVal", func() {
		It("returns a map from the endpoint, when provided the right key", func() {
			expected := map[string]interface{}{"key": "value"}

			got, err := endpoint.MapValue("mapVal")
			Expect(got).Should(BeEquivalentTo(expected))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("errors out when not pointed at a map", func() {
			got, err := endpoint.MapValue("stringVal")
			Expect(got).Should(BeNil())
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("'stringVal' key in endpoint json is a string, not a map"))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.MapValue("doesnotexist")
			Expect(got).Should(BeNil())
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("No 'doesnotexist' key specified in the endpoint json"))
		})
	})
	Describe("StringVal", func() {
		It("returns an array from the endpoint, when provided the right key", func() {
			expected := "asdf"

			got, err := endpoint.StringValue("stringVal")
			Expect(got).Should(BeEquivalentTo(expected))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("errors out when not pointed at a string", func() {
			got, err := endpoint.StringValue("boolVal")
			Expect(got).Should(Equal(""))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("'boolVal' key in endpoint json is a bool, not a string"))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.StringValue("doesnotexist")
			Expect(got).Should(Equal(""))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("No 'doesnotexist' key specified in the endpoint json"))
		})
	})
	Describe("BooleanVal", func() {
		It("returns a bool from the endpoint, when provided the right key", func() {
			expected := true

			got, err := endpoint.BooleanValue("boolVal")
			Expect(got).Should(BeEquivalentTo(expected))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("errors out when not pointed at a bool", func() {
			got, err := endpoint.BooleanValue("stringVal")
			Expect(got).Should(Equal(false))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("'stringVal' key in endpoint json is a string, not a boolean"))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.BooleanValue("doesnotexist")
			Expect(got).Should(Equal(false))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("No 'doesnotexist' key specified in the endpoint json"))
		})
	})
	Describe("FloatVal", func() {
		It("returns a float from the endpoint, when provided the right key", func() {
			expected := 1234.1234

			got, err := endpoint.FloatValue("floatVal")
			Expect(got).Should(BeEquivalentTo(expected))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("errors out when not pointed at an number", func() {
			got, err := endpoint.FloatValue("stringVal")
			Expect(got).Should(Equal(0.0))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("'stringVal' key in endpoint json is a string, not a numeric"))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.FloatValue("doesnotexist")
			Expect(got).Should(Equal(0.0))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("No 'doesnotexist' key specified in the endpoint json"))
		})
	})
})
