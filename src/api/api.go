// Jamie: This contains the go source code that will become shield.

package api

import (
	"db"

	"fmt"
	"os"
	"net/http"
)

func Run(bind string, template *db.DB) {
	db := template.Copy()
	if err := db.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to %s database at %s: %s\n",
			db.Driver, db.DSN, err)
		return
	}

	http.Handle("/v1/ping", &PingAPI{})

	http.Handle("/v1/jobs", &JobAPI{Data: db})
	http.Handle("/v1/job", &JobAPI{Data: db})

	http.Handle("/v1/retention", &RetentionAPI{Data: db})

	http.Handle("/v1/archives", &ArchiveAPI{Data: db})
	http.Handle("/v1/archive", &ArchiveAPI{Data: db})

	http.Handle("/v1/schedules", &ScheduleAPI{Data: db})
	http.Handle("/v1/schedule", &ScheduleAPI{Data: db})

	http.Handle("/v1/stores", &StoreAPI{Data: db})
	http.Handle("/v1/store", &StoreAPI{Data: db})

	http.Handle("/v1/targets", &TargetAPI{Data: db})
	http.Handle("/v1/target", &TargetAPI{Data: db})

	http.ListenAndServe(bind, nil)
}
