package metrics

import (
	"net/http"

	"github.com/jhunt/go-log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shieldproject/shield/core/bus"
)

type Exporter struct {
	Namespace    string
	AgentCount   int
	TargetCount  int
	JobCount     int
	TaskCount    int
	ArchiveCount int

	Username string
	Password string
	Realm    string

	bus                   *bus.Bus
	agentsGauge           prometheus.Gauge
	targetsGauge          prometheus.Gauge
	jobsGauge             prometheus.Gauge
	tasksGauge            prometheus.Gauge
	archivesGauge         prometheus.Gauge
	storageUsedBytesGauge prometheus.Gauge
}

const (
	agentsTotal       = "agents_total"
	targetsTotal      = "targets_total"
	jobsTotal         = "jobs_total"
	tasksTotal        = "tasks_total"
	archivesTotal     = "archives_total"
	storageBytesTotal = "storage_used_bytes"
	/* TODO
	targetHealthStatus = "target_health_status"
	coreStatus         = "core_status" */
)

func New(endpoint *Exporter) *Exporter {
	if endpoint == nil {
		endpoint = &Exporter{}
	}

	if endpoint.Username == "" {
		endpoint.Username = "prometheus"
	}
	if endpoint.Password == "" {
		endpoint.Password = "shield"
	}
	if endpoint.Realm == "" {
		endpoint.Realm = "SHIELD Prometheus Exporter"
	}
	if endpoint.Namespace == "" {
		endpoint.Namespace = "shield"
	}

	endpoint.agentsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: endpoint.Namespace,
			Name:      agentsTotal,
			Help:      "How many SHIELD Agents have been registered",
		})

	endpoint.targetsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: endpoint.Namespace,
			Name:      targetsTotal,
			Help:      "How many Target Systems have been defined",
		})

	endpoint.jobsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: endpoint.Namespace,
			Name:      jobsTotal,
			Help:      "How many backup jobs have been defined",
		})

	endpoint.tasksGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: endpoint.Namespace,
			Name:      tasksTotal,
			Help:      "How many tasks have been created",
		})

	endpoint.archivesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: endpoint.Namespace,
			Name:      archivesTotal,
			Help:      "How many Backup Archives have been generated",
		})

	endpoint.storageUsedBytesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: endpoint.Namespace,
			Name:      storageBytesTotal,
			Help:      "How much storage has been used, in bytes.",
		})

	prometheus.MustRegister(
		endpoint.agentsGauge,
		endpoint.targetsGauge,
		endpoint.jobsGauge,
		endpoint.tasksGauge,
		endpoint.archivesGauge,
		endpoint.storageUsedBytesGauge,
	)

	endpoint.agentsGauge.Set(float64(endpoint.AgentCount))
	endpoint.targetsGauge.Set(float64(endpoint.TargetCount))
	endpoint.jobsGauge.Set(float64(endpoint.JobCount))
	endpoint.tasksGauge.Set(float64(endpoint.TaskCount))
	endpoint.archivesGauge.Set(float64(endpoint.ArchiveCount))

	return endpoint
}

func (e *Exporter) Inform(mbus *bus.Bus) {
	e.bus = mbus
}

func (e *Exporter) Handler() http.Handler {
	return BasicAuthenticator{
		username: e.Username,
		password: e.Password,
		realm:    e.Realm,
		handler:  promhttp.Handler(),
	}
}

func (e *Exporter) createObjectCount(typ string, raw interface{}) {
	data := raw.(map[string]interface{})
	switch typ {
	case "agent":
		e.agentsGauge.Inc()
	case "target":
		e.targetsGauge.Inc()
	case "job":
		e.jobsGauge.Inc()
	case "task":
		e.tasksGauge.Inc()
	case "archive":
		e.archivesGauge.Inc()
		e.storageUsedBytesGauge.Add(float64(data["size"].(int64)))
	default:
		log.Debugf("ignoring create event for object type `%s'", typ)
	}
}

func (e *Exporter) updateObjectCount(typ string, raw interface{}) {
	data := raw.(map[string]interface{})
	switch typ {
	case "archive":
		if data["status"] == "manually purged" {
			e.archivesGauge.Dec()
			e.storageUsedBytesGauge.Sub(float64(data["size"].(int64)))
		}
	default:
		log.Debugf("ignoring update event for object type `%s'", typ)
	}
}

func (e *Exporter) deleteObjectCount(typ string) {
	switch typ {
	case "agent":
		e.agentsGauge.Dec()
	case "target":
		e.targetsGauge.Dec()
	case "job":
		e.jobsGauge.Dec()
	default:
		log.Debugf("ignoring delete event for object type `%s'", typ)
	}
}

/* TODO
func (e *Exporter) UpdateCoreStatus(value float64) {
	coreStatusGauge.Set(value)
} */

func (e *Exporter) Watch(queues ...string) {
	ch, _, err := e.bus.Register(queues)
	if err != nil {
		log.Infof("bus didn't register for mbus")
		return
	}

	for eventObject := range ch {
		event := eventObject.Event
		typ := eventObject.Type
		var data interface{} = eventObject.Data
		data = eventObject.Data
		if event == "lock-core" {
			log.Infof("TODO")
		} else if event == "unlock-core" {
			log.Infof("TODO")
		} else if event == "create-object" {
			e.createObjectCount(typ, data)
		} else if event == "update-object" {
			e.updateObjectCount(typ, data)
		} else if event == "delete-object" {
			e.deleteObjectCount(typ)
		} else {
			log.Debugf("ignoring event of type `%s'", event)
		}
	}
}
