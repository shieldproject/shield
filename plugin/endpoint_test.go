package plugin_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/starkandwayne/shield/plugin"
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
			Expect(err).Should(MatchError(plugin.EndpointDataTypeMismatchError{Key: "stringVal", DesiredType: "array"}))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.ArrayValue("doesnotexist")
			Expect(got).Should(BeNil())
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(MatchError(plugin.EndpointMissingRequiredDataError{Key: "doesnotexist"}))
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
			Expect(err).Should(MatchError(plugin.EndpointDataTypeMismatchError{Key: "stringVal", DesiredType: "map"}))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.MapValue("doesnotexist")
			Expect(got).Should(BeNil())
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(MatchError(plugin.EndpointMissingRequiredDataError{Key: "doesnotexist"}))
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
			Expect(err).Should(MatchError(plugin.EndpointDataTypeMismatchError{Key: "boolVal", DesiredType: "string"}))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.StringValue("doesnotexist")
			Expect(got).Should(Equal(""))
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(MatchError(plugin.EndpointMissingRequiredDataError{Key: "doesnotexist"}))
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
			Expect(err).Should(MatchError(plugin.EndpointDataTypeMismatchError{Key: "stringVal", DesiredType: "boolean"}))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.BooleanValue("doesnotexist")
			Expect(got).Should(Equal(false))
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(MatchError(plugin.EndpointMissingRequiredDataError{Key: "doesnotexist"}))
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
			Expect(err).Should(MatchError(plugin.EndpointDataTypeMismatchError{Key: "stringVal", DesiredType: "numeric"}))
		})
		It("errors out when pointed at a nonexistant key", func() {
			got, err := endpoint.FloatValue("doesnotexist")
			Expect(got).Should(Equal(0.0))
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(MatchError(plugin.EndpointMissingRequiredDataError{Key: "doesnotexist"}))
		})
	})
})
