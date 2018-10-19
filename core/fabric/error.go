package fabric

import (
	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

type ErrorFabric struct {
	e error
}

func (f ErrorFabric) Error() string {
	return f.e.Error()
}

func (f ErrorFabric) chore() scheduler.Chore {
	return scheduler.NewChore(func(chore scheduler.Chore) {
		chore.Errorf("[ERROR]: %s\n", f)
		chore.UnixExit(1)
	})
}

func (f ErrorFabric) Backup(*db.Target, *db.Store, string, *vault.Parameters) scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) Restore(*db.Archive, *db.Target, *vault.Parameters) scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) Status() scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) Purge(*db.Archive) scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) TestStore(*db.Store) scheduler.Chore {
	return f.chore()
}
