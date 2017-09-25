package monit

const (
	StatusUnknown  = "unknown"
	StatusStarting = "starting"
	StatusRunning  = "running"
	StatusFailing  = "failing"
)

type Status interface {
	GetIncarnation() (int, error)
	ServicesInGroup(name string) (services []Service)
}

type Service struct {
	Name                 string
	Monitored            bool
	Errored              bool
	Pending              bool
	Status               string
	StatusMessage        string
	Uptime               int
	MemoryPercentTotal   float64
	MemoryKilobytesTotal int
	CPUPercentTotal      float64
}
