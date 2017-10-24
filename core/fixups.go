package core

import (
	"github.com/jhunt/go-log"
)

func (core *Core) fixups() error {
	log.Infof("fixups: back-filling purge_agent to stores that have no agent of their own")
	err := core.DB.Exec(`UPDATE stores SET agent = ? WHERE agent IS NULL OR agent = ''`,
		core.purgeAgent)
	if err != nil {
		return err
	}

	return nil
}
