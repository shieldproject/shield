package store

import (
	"github.com/cloudfoundry/config-server/config"
	"strings"

	"github.com/cloudfoundry/bosh-utils/errors"
)

func CreateStore(config config.ServerConfig) (store Store, err error) {
	if strings.EqualFold(config.Store, "database") {
		dbConfig := config.Database

		if strings.EqualFold(dbConfig.Adapter, "postgres") {
			var dbProvider DbProvider
			dbProvider, err = NewConcreteDbProvider(NewSQLWrapper(), dbConfig)
			store = NewPostgresStore(dbProvider)
		} else if strings.EqualFold(dbConfig.Adapter, "mysql") {
			var dbProvider DbProvider
			dbProvider, err = NewConcreteDbProvider(NewSQLWrapper(), dbConfig)
			store = NewMysqlStore(dbProvider)
		} else {
			err = errors.Errorf("Unsupported adapter '%s'", dbConfig.Adapter)
		}
	} else {
		store = NewMemoryStore()
	}

	return
}
