package fabric

import (
	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

func Error(err error) ErrorFabric {
	return ErrorFabric{e: err}
}

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

func (f ErrorFabric) Backup(*db.Task, vault.Parameters) scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) Restore(*db.Task, vault.Parameters) scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) Status() scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) Purge(*db.Task) scheduler.Chore {
	return f.chore()
}

func (f ErrorFabric) TestStore(*db.Task) scheduler.Chore {
	return f.chore()
}
