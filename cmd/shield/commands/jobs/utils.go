package jobs

import (
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

func maybeGCSchedule(scheduleUUID string) error {
	shouldGC, err := shouldGCSchedule(scheduleUUID)
	if err != nil {
		return err
	}
	if shouldGC {
		log.DEBUG("Deleting unreferenced schedule")
		err = api.DeleteSchedule(uuid.Parse(scheduleUUID))
		if err != nil {
			return err
		}
		log.DEBUG("Schedule deleted")
	}
	return nil
}

func shouldGCSchedule(scheduleUUID string) (bool, error) {
	jobs, err := api.GetJobs(api.JobFilter{Schedule: scheduleUUID})
	return err == nil && len(jobs) == 0, err
}
