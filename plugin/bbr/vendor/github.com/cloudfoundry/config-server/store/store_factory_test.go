package store_test

import (
	. "github.com/cloudfoundry/config-server/store"

	"github.com/cloudfoundry/config-server/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateStore", func() {

	Describe("Given a store as Database", func() {
		var serverConfig config.ServerConfig

		BeforeEach(func() {
			serverConfig = config.ServerConfig{
				Store:    "database",
				Database: config.DBConfig{},
			}
		})

		Context("adapter is postgres", func() {
			BeforeEach(func() {
				serverConfig.Database.Adapter = "postgres"
			})

			It("should return a Postgres Store", func() {
				store, _ := CreateStore(serverConfig)
				Expect(store).To(BeAssignableToTypeOf(NewPostgresStore(nil)))
			})
		})

		Context("adapter is mysql", func() {
			BeforeEach(func() {
				serverConfig.Database.Adapter = "mysql"
			})

			It("should return a Mysql Store", func() {
				store, _ := CreateStore(serverConfig)
				Expect(store).To(BeAssignableToTypeOf(NewMysqlStore(nil)))
			})
		})

		Context("adapter is unknown/invalid", func() {
			BeforeEach(func() {
				serverConfig.Database.Adapter = "foo"
			})

			It("should return a Mysql Store", func() {
				_, err := CreateStore(serverConfig)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("Unsupported adapter 'foo'"))
			})
		})
	})
})
