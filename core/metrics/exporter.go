package metrics

import (
	"fmt"
	"net/http"

	"github.com/jhunt/go-log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shieldproject/shield/core/bus"
)

type Metrics struct {
	bus *bus.Bus
}

const (
	namespace     = "shield"
	tenantsTotal  = "tenants_total"
	agentsTotal   = "agents_total"
	targetsTotal  = "targets_total"
	storesTotal   = "stores_total"
	jobsTotal     = "jobs_total"
	tasksTotal    = "tasks_total"
	archivesTotal = "archives_total"
	coreStatus    = "core_status"
)

var (
	tenantsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      tenantsTotal,
			Help:      "How many Tenants exist",
		})

	agentsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      agentsTotal,
			Help:      "How many SHIELD Agents have been registered",
		})

	targetsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      targetsTotal,
			Help:      "How many Target Systems have been defined",
		})

	storesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      storesTotal,
			Help:      "How many Cloud Storage Systems have been defined",
		})

	jobsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      jobsTotal,
			Help:      "How many backup jobs have been defined",
		})

	tasksGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      tasksTotal,
			Help:      "How many tasks have been created",
		})

	archivesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      archivesTotal,
			Help:      "How many Backup Archives have been generated",
		})

	coreStatusGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      coreStatus,
			Help:      "The global initialized / locked status of the SHIELD core; (0=uninitialized, 1=locked, 2=unlocked)",
		})
)

func InitalizeMetrics() *Metrics {
	metrics := &Metrics{}
	return metrics
}

func (m *Metrics) Inform(mbus *bus.Bus) {
	m.bus = mbus
}

func (m *Metrics) RegisterExporter(tenantCount, agentsCount, targetsCount, storesCount, jobsCount, tasksCount, archivesCount, coreStatusInitial float64) {
	prometheus.MustRegister(tenantsGauge)
	prometheus.MustRegister(agentsGauge)
	prometheus.MustRegister(targetsGauge)
	prometheus.MustRegister(storesGauge)
	prometheus.MustRegister(jobsGauge)
	prometheus.MustRegister(tasksGauge)
	prometheus.MustRegister(archivesGauge)
	prometheus.MustRegister(coreStatusGauge)

	tenantsGauge.Set(tenantCount)
	agentsGauge.Set(agentsCount)
	targetsGauge.Set(targetsCount)
	storesGauge.Set(storesCount)
	jobsGauge.Set(jobsCount)
	tasksGauge.Set(tasksCount)
	archivesGauge.Set(archivesCount)
	coreStatusGauge.Set(coreStatusInitial)
}

func (m *Metrics) ServeExporter() http.Handler {
	return promhttp.Handler()
}

func CreateObjectCount(typeThing string) {
	fmt.Printf("\n")
	fmt.Printf("Prometheus Data: Increasing count for type ---> %s\n", typeThing)
	fmt.Printf("\n")
	switch typeThing {
	case "tenant":
		tenantsGauge.Inc()
	case "agent":
		agentsGauge.Inc()
	case "target":
		targetsGauge.Inc()
	case "store":
		storesGauge.Inc()
	case "job":
		jobsGauge.Inc()
	case "task":
		tasksGauge.Inc()
	case "archive":
		archivesGauge.Inc()
	default:
		log.Infof("Event type not recognized or not implemented yet.")
	}
}

func UpdateObjectCount(typeThing string, data interface{}) {
	interfaceData := data.(map[string]interface{})
	switch typeThing {
	case "archive":
		if interfaceData["status"] == "manually purged" {
			archivesGauge.Dec()
		}
	}
}

func DeleteObjectCount(typeThing string) {
	switch typeThing {
	case "tenant":
		tenantsGauge.Dec()
	case "agent":
		agentsGauge.Dec()
	case "target":
		targetsGauge.Dec()
	case "store":
		storesGauge.Dec()
	case "job":
		jobsGauge.Dec()
	default:
		log.Infof("Event type not recognized or not implemented yet.")
	}
}

func (m *Metrics) UpdateCoreStatus(value float64) {
	coreStatusGauge.Set(value)
}

func (m *Metrics) RegisterBusEvents(queues ...string) {
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
			coreStatusGauge.Set(1)
		} else if event == "unlock-core" {
			coreStatusGauge.Set(2)
		} else if event == "create-object" {
			CreateObjectCount(typeThing)
		} else if event == "update-object" {
			UpdateObjectCount(typeThing, data)
		} else if event == "delete-object" {
			DeleteObjectCount(typeThing)
		}
	}
}
