package metrics

import (
	"net/http"

	"github.com/jhunt/go-log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shieldproject/shield/core/bus"
)

type Config struct {
	Namespace        string
	TenantCount      int
	AgentCount       int
	TargetCount      int
	StoreCount       int
	JobCount         int
	TaskCount        int
	ArchiveCount     int
	StorageUsedCount int64
}

type Metrics struct {
	bus                   *bus.Bus
	tenantsGauge          prometheus.Gauge
	agentsGauge           prometheus.Gauge
	targetsGauge          prometheus.Gauge
	storesGauge           prometheus.Gauge
	jobsGauge             prometheus.Gauge
	tasksGauge            prometheus.Gauge
	archivesGauge         prometheus.Gauge
	storageUsedBytesGauge prometheus.Gauge
}

const (
	tenantsTotal      = "tenants_total"
	agentsTotal       = "agents_total"
	targetsTotal      = "targets_total"
	storesTotal       = "stores_total"
	jobsTotal         = "jobs_total"
	tasksTotal        = "tasks_total"
	archivesTotal     = "archives_total"
	storageBytesTotal = "storage_used_bytes"
	/* TODO
	targetHealthStatus = "target_health_status"
	storeHealthStatus  = "storeHealthStatus"
	coreStatus         = "core_status" */
)

func New(config Config) *Metrics {
	metrics := &Metrics{}
	namespace := config.Namespace
	metrics.tenantsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      tenantsTotal,
			Help:      "How many Tenants exist",
		})

	metrics.agentsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      agentsTotal,
			Help:      "How many SHIELD Agents have been registered",
		})

	metrics.targetsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      targetsTotal,
			Help:      "How many Target Systems have been defined",
		})

	metrics.storesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      storesTotal,
			Help:      "How many Cloud Storage Systems have been defined",
		})

	metrics.jobsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      jobsTotal,
			Help:      "How many backup jobs have been defined",
		})

	metrics.tasksGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      tasksTotal,
			Help:      "How many tasks have been created",
		})

	metrics.archivesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      archivesTotal,
			Help:      "How many Backup Archives have been generated",
		})

	metrics.storageUsedBytesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      storageBytesTotal,
			Help:      "How much storage has been used, in bytes.",
		})

	prometheus.MustRegister(metrics.tenantsGauge, metrics.agentsGauge,
		metrics.targetsGauge, metrics.storesGauge, metrics.jobsGauge,
		metrics.tasksGauge, metrics.archivesGauge, metrics.storageUsedBytesGauge)

	metrics.tenantsGauge.Set(float64(config.TenantCount))
	metrics.agentsGauge.Set(float64(config.AgentCount))
	metrics.targetsGauge.Set(float64(config.TargetCount))
	metrics.storesGauge.Set(float64(config.StoreCount))
	metrics.jobsGauge.Set(float64(config.JobCount))
	metrics.tasksGauge.Set(float64(config.TaskCount))
	metrics.archivesGauge.Set(float64(config.ArchiveCount))
	metrics.storageUsedBytesGauge.Set(float64(config.StorageUsedCount))

	return metrics
}

func (m *Metrics) Inform(mbus *bus.Bus) {
	m.bus = mbus
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) createObjectCount(typeThing string, data interface{}) {
	interfaceData := data.(map[string]interface{})
	switch typeThing {
	case "tenant":
		m.tenantsGauge.Inc()
	case "agent":
		m.agentsGauge.Inc()
	case "target":
		m.targetsGauge.Inc()
	case "store":
		m.storesGauge.Inc()
	case "job":
		m.jobsGauge.Inc()
	case "task":
		m.tasksGauge.Inc()
	case "archive":
		m.archivesGauge.Inc()
		m.storageUsedBytesGauge.Add(float64(interfaceData["size"].(int64)))
	default:
		log.Debugf("Metrics ignoring create event for object type `%s'", typeThing)
	}
}

func (m *Metrics) updateObjectCount(typeThing string, data interface{}) {
	interfaceData := data.(map[string]interface{})
	switch typeThing {
	case "archive":
		if interfaceData["status"] == "manually purged" {
			m.archivesGauge.Dec()
			m.storageUsedBytesGauge.Sub(float64(interfaceData["size"].(int64)))
		}
	default:
		log.Debugf("Metrics ignoring update event for object type `%s'", typeThing)
	}
}

func (m *Metrics) deleteObjectCount(typeThing string) {
	switch typeThing {
	case "tenant":
		m.tenantsGauge.Dec()
	case "agent":
		m.agentsGauge.Dec()
	case "target":
		m.targetsGauge.Dec()
	case "store":
		m.storesGauge.Dec()
	case "job":
		m.jobsGauge.Dec()
	default:
		log.Debugf("Metrics ignoring delete event for object type `%s'", typeThing)
	}
}

/* TODO
func (m *Metrics) UpdateCoreStatus(value float64) {
	coreStatusGauge.Set(value)
} */

func (m *Metrics) Watch(queues ...string) {
	ch, _, err := m.bus.Register(queues)
	if err != nil {
		log.Infof("bus didn't register for mbus")
		return
	}

	for eventObject := range ch {
		event := eventObject.Event
		typeThing := eventObject.Type
		var data interface{} = eventObject.Data
		data = eventObject.Data
		if event == "lock-core" {
			log.Infof("TODO")
		} else if event == "unlock-core" {
			log.Infof("TODO")
		} else if event == "create-object" {
			m.createObjectCount(typeThing, data)
		} else if event == "update-object" {
			m.updateObjectCount(typeThing, data)
		} else if event == "delete-object" {
			m.deleteObjectCount(typeThing)
		} else {
			log.Debugf("Metrics ignoring event of type `%s'", event)
		}
	}
}
