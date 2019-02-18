package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func (c *Core) TaskErrored(task *db.Task, fail string, args ...interface{}) {
	if len(args) != 0 {
		fail = fmt.Sprintf(fail, args...)
	}

	log.Infof("  %s> %s", task.UUID, strings.Trim(fail, "\n"))
	if err := c.db.UpdateTaskLog(task.UUID, fail); err != nil {
		log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
	}

	log.Warnf("  %s: task failed!", task.UUID)
	if err := c.db.FailTask(task.UUID, time.Now()); err != nil {
		log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
	}
}
