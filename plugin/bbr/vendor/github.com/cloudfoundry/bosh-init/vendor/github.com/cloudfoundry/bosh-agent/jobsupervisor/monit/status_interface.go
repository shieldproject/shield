package monit

type Status interface {
	GetIncarnation() (int, error)
	ServicesInGroup(name string) (services []Service)
}

type Service struct {
	Name                 string
	Monitored            bool
	Status               string
	Uptime               int
	MemoryPercentTotal   float64
	MemoryKilobytesTotal int
	CPUPercentTotal      float64
}
