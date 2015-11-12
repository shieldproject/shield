// Jamie: This contains the go source code that will become shield.

package api

import (
	"fmt"
	"github.com/starkandwayne/shield/db"
	"net/http"
	"os"
)

func Run(bind string, template *db.DB, c chan int) {
	db := template.Copy()
	if err := db.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to %s database at %s: %s\n",
			db.Driver, db.DSN, err)
		return
	}

	ping := &PingAPI{}
	http.Handle("/v1/ping", ping)

	jobs := &JobAPI{Data: db, SuperChan: c}
	http.Handle("/v1/jobs", jobs)
	http.Handle("/v1/job", jobs)

	retention := &RetentionAPI{Data: db, SuperChan: c}
	http.Handle("/v1/retention", retention)

	archives := &ArchiveAPI{Data: db, SuperChan: c}
	http.Handle("/v1/archives", archives)
	http.Handle("/v1/archive", archives)

	schedules := &ScheduleAPI{Data: db, SuperChan: c}
	http.Handle("/v1/schedules", schedules)
	http.Handle("/v1/schedule", schedules)

	stores := &StoreAPI{Data: db, SuperChan: c}
	http.Handle("/v1/stores", stores)
	http.Handle("/v1/store", stores)

	targets := &TargetAPI{Data: db, SuperChan: c}
	http.Handle("/v1/targets", targets)
	http.Handle("/v1/target", targets)

	http.ListenAndServe(bind, nil)
}
