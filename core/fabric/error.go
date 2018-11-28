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

func (f ErrorFabric) chore(id string) scheduler.Chore {
	return scheduler.NewChore(id, func(chore scheduler.Chore) {
		chore.Errorf("[ERROR]: %s\n", f)
		chore.UnixExit(1)
	})
}

func (f ErrorFabric) Backup(task *db.Task, _ vault.Parameters) scheduler.Chore {
	return f.chore(task.UUID)
}

func (f ErrorFabric) Restore(task *db.Task, _ vault.Parameters) scheduler.Chore {
	return f.chore(task.UUID)
}

func (f ErrorFabric) Status(task *db.Task) scheduler.Chore {
	return f.chore(task.UUID)
}

func (f ErrorFabric) Purge(task *db.Task) scheduler.Chore {
	return f.chore(task.UUID)
}

func (f ErrorFabric) TestStore(task *db.Task) scheduler.Chore {
	return f.chore(task.UUID)
}
