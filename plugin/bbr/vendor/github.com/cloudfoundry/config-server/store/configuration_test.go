package store_test

import (
	"github.com/cloudfoundry/config-server/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configuration", func() {

	Describe("StringifiedJSON", func() {
		Context("When value is a string", func() {
			It("returns json string from the given db result", func() {
				configuration := store.Configuration{
					ID:    "123",
					Name:  "smurf",
					Value: `{"value": "blue"}`,
				}

				jsonString, _ := configuration.StringifiedJSON()

				Expect(jsonString).To(Equal(`{"id":"123","name":"smurf","value":"blue"}`))
			})
		})

		Context("When value is a number", func() {
			It("returns json string from the given db result", func() {
				configuration := store.Configuration{
					ID:    "123",
					Name:  "smurf",
					Value: `{"value": 123}`,
				}

				jsonString, _ := configuration.StringifiedJSON()

				Expect(jsonString).To(Equal(`{"id":"123","name":"smurf","value":123}`))
			})
		})

		Context("When value is complex", func() {
			It("returns json string from the given db result", func() {
				configuration := store.Configuration{
					ID:    "123",
					Name:  "smurf",
					Value: `{"value": {"smurf":"gargamel"}}`,
				}

				jsonString, _ := configuration.StringifiedJSON()

				Expect(jsonString).To(Equal(`{"id":"123","name":"smurf","value":{"smurf":"gargamel"}}`))
			})
		})

	})
})
