package metrics

import (
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
	namespace         = "shield"
	tenantsTotal      = "tenants_total"
	agentsTotal       = "agents_total"
	targetsTotal      = "targets_total"
	storesTotal       = "stores_total"
	jobsTotal         = "jobs_total"
	tasksTotal        = "tasks_total"
	archivesTotal     = "archives_total"
	storageBytesTotal = "storage_used_bytes"

	targetHealthStatus = "target_health_status"
	storeHealthStatus  = "storeHealthStatus"
	coreStatus         = "core_status"
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

	storageUsedBytesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      storageBytesTotal,
			Help:      "How much storage has been used, in bytes.",
		})

	targetHealthGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      targetHealthStatus,
			Help:      "A generic health status for targets, that is differentiated into different contexts based on labels. For example, the label target=x tenant=y means that the metric applies to the health of a given target, owned by a tenant.",
		}, []string{"tenant_name", "tenant_uuid", "target_name", "target_uuid"})

	storeHealthGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      storeHealthStatus,
			Help:      "A generic health status for stores, that is differentiated into different contexts based on labels. For example, the label store=x tenant=y means that the metric applies to the health of a given store, owned by a tenant.",
		}, []string{"tenant_name", "tenant_uuid", "store_name", "store_uuid"})

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
	prometheus.MustRegister(storageUsedBytesGauge)

	prometheus.MustRegister(targetHealthGaugeVec)
	prometheus.MustRegister(storeHealthGaugeVec)
	prometheus.MustRegister(coreStatusGauge)

	tenantsGauge.Set(tenantCount)
	agentsGauge.Set(agentsCount)
	targetsGauge.Set(targetsCount)
	storesGauge.Set(storesCount)
	jobsGauge.Set(jobsCount)
	tasksGauge.Set(tasksCount)
	archivesGauge.Set(archivesCount)
	coreStatusGauge.Set(coreStatusInitial)
	storageUsedBytesGauge.Set(0)
}

func (m *Metrics) ServeExporter() http.Handler {
	return promhttp.Handler()
}

func CreateObjectCount(typeThing string) {
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
	case "tenant":
		storageUsedBytesGauge.Set(float64(interfaceData["storage_used"].(int64)))
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

func UpdateTargetHealth (type thing, data interface{}) {
    
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
		} else if event == "update-task-status" {

        }
	}
}
